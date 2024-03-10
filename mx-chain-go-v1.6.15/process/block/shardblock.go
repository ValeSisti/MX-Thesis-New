package block

import (
	"bytes"
	"fmt"
	"math/big"
	"time"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/core/check"
	"github.com/multiversx/mx-chain-core-go/data"
	"github.com/multiversx/mx-chain-core-go/data/block"
	"github.com/multiversx/mx-chain-core-go/data/headerVersionData"
	"github.com/multiversx/mx-chain-go/common"
	"github.com/multiversx/mx-chain-go/dataRetriever"
	processOutport "github.com/multiversx/mx-chain-go/outport/process"
	"github.com/multiversx/mx-chain-go/process"
	"github.com/multiversx/mx-chain-go/process/block/bootstrapStorage"
	"github.com/multiversx/mx-chain-go/process/block/processedMb"
	"github.com/multiversx/mx-chain-go/state"
	logger "github.com/multiversx/mx-chain-logger-go"
	//! -------------------- NEW CODE --------------------
	crypto "github.com/multiversx/mx-chain-crypto-go"
	"github.com/multiversx/mx-chain-core-go/data/transaction"
	"errors"
	"encoding/hex"
	//! ---------------- END OF NEW CODE -----------------
)

var _ process.BlockProcessor = (*shardProcessor)(nil)

const (
	timeBetweenCheckForEpochStart = 100 * time.Millisecond
	pruningDelay                  = 10
)

type createAndProcessMiniBlocksDestMeInfo struct {
	currMetaHdr                 data.HeaderHandler
	currMetaHdrHash             []byte
	currProcessedMiniBlocksInfo map[string]*processedMb.ProcessedMiniBlockInfo
	allProcessedMiniBlocksInfo  map[string]*processedMb.ProcessedMiniBlockInfo
	haveTime                    func() bool
	haveAdditionalTime          func() bool
	miniBlocks                  block.MiniBlockSlice
	hdrAdded                    bool
	numTxsAdded                 uint32
	numHdrsAdded                uint32
	scheduledMode               bool
	//! -------------------- NEW CODE --------------------
	problematicMiniBlocksForCurrentRoundTxHashes map[string]*data.MbInfo
	problematicMiniBlocksForCurrentRoundFullInfo map[string]*data.ProblematicMBInfo //TODO: inviare insieme al blocco (consensus message) e svuotare una volta che il blocco viene committato
	//! ---------------- END OF NEW CODE -----------------	
}

//! -------------------- NEW CODE --------------------
type WaitingMiniBlockInfo struct {
	originalProblematicTxsInfo map[string]*AATInfo
	numProblematicTxs int
	numNotarizedAATs int //! DIFFERENT
}

type AATInfo struct {
	aatHash string
	includedInBlock bool
	notarizedOnDestination bool
}
//! ---------------- END OF NEW CODE -----------------


// shardProcessor implements shardProcessor interface, and actually it tries to execute block
type shardProcessor struct {
	*baseProcessor
	metaBlockFinality uint32
	chRcvAllMetaHdrs  chan bool
	//! -------------------- NEW CODE --------------------
	txSingleSigner crypto.SingleSigner
	txPrivateKey crypto.PrivateKey
	txPublicKey crypto.PublicKey
	coreComponents coreComponentsHolder
	problematicMiniBlocksForCurrentRound map[string]*data.ProblematicMBInfo //TODO: inviare insieme al blocco (consensus message) e svuotare una volta che il blocco viene committato
	readyMbsInsertedInCurrentBlock map[string]bool
	//! ---------------- END OF NEW CODE -----------------	
}

// NewShardProcessor creates a new shardProcessor object
func NewShardProcessor(arguments ArgShardProcessor) (*shardProcessor, error) {
	err := checkProcessorParameters(arguments.ArgBaseProcessor)
	if err != nil {
		return nil, err
	}

	if check.IfNil(arguments.DataComponents.Datapool()) {
		return nil, process.ErrNilDataPoolHolder
	}
	if check.IfNil(arguments.DataComponents.Datapool().Headers()) {
		return nil, process.ErrNilHeadersDataPool
	}
	if check.IfNil(arguments.DataComponents.Datapool().Transactions()) {
		return nil, process.ErrNilTransactionPool
	}
	genesisHdr := arguments.DataComponents.Blockchain().GetGenesisHeader()
	if check.IfNil(genesisHdr) {
		return nil, fmt.Errorf("%w for genesis header in DataComponents.Blockchain", process.ErrNilHeaderHandler)
	}

	processDebugger, err := createDisabledProcessDebugger()
	if err != nil {
		return nil, err
	}

	base := &baseProcessor{
		accountsDB:                    arguments.AccountsDB,
		blockSizeThrottler:            arguments.BlockSizeThrottler,
		forkDetector:                  arguments.ForkDetector,
		hasher:                        arguments.CoreComponents.Hasher(),
		marshalizer:                   arguments.CoreComponents.InternalMarshalizer(),
		store:                         arguments.DataComponents.StorageService(),
		shardCoordinator:              arguments.BootstrapComponents.ShardCoordinator(),
		nodesCoordinator:              arguments.NodesCoordinator,
		uint64Converter:               arguments.CoreComponents.Uint64ByteSliceConverter(),
		requestHandler:                arguments.RequestHandler,
		appStatusHandler:              arguments.StatusCoreComponents.AppStatusHandler(),
		blockChainHook:                arguments.BlockChainHook,
		txCoordinator:                 arguments.TxCoordinator,
		roundHandler:                  arguments.CoreComponents.RoundHandler(),
		epochStartTrigger:             arguments.EpochStartTrigger,
		headerValidator:               arguments.HeaderValidator,
		bootStorer:                    arguments.BootStorer,
		blockTracker:                  arguments.BlockTracker,
		dataPool:                      arguments.DataComponents.Datapool(),
		stateCheckpointModulus:        arguments.Config.StateTriesConfig.CheckpointRoundsModulus,
		blockChain:                    arguments.DataComponents.Blockchain(),
		feeHandler:                    arguments.FeeHandler,
		outportHandler:                arguments.StatusComponents.OutportHandler(),
		genesisNonce:                  genesisHdr.GetNonce(),
		versionedHeaderFactory:        arguments.BootstrapComponents.VersionedHeaderFactory(),
		headerIntegrityVerifier:       arguments.BootstrapComponents.HeaderIntegrityVerifier(),
		historyRepo:                   arguments.HistoryRepository,
		epochNotifier:                 arguments.CoreComponents.EpochNotifier(),
		enableEpochsHandler:           arguments.CoreComponents.EnableEpochsHandler(),
		roundNotifier:                 arguments.CoreComponents.RoundNotifier(),
		enableRoundsHandler:           arguments.CoreComponents.EnableRoundsHandler(),
		vmContainerFactory:            arguments.VMContainersFactory,
		vmContainer:                   arguments.VmContainer,
		processDataTriesOnCommitEpoch: arguments.Config.Debug.EpochStart.ProcessDataTrieOnCommitEpoch,
		gasConsumedProvider:           arguments.GasHandler,
		economicsData:                 arguments.CoreComponents.EconomicsData(),
		scheduledTxsExecutionHandler:  arguments.ScheduledTxsExecutionHandler,
		pruningDelay:                  pruningDelay,
		processedMiniBlocksTracker:    arguments.ProcessedMiniBlocksTracker,
		receiptsRepository:            arguments.ReceiptsRepository,
		processDebugger:               processDebugger,
		outportDataProvider:           arguments.OutportDataProvider,
		processStatusHandler:          arguments.CoreComponents.ProcessStatusHandler(),
		blockProcessingCutoffHandler:  arguments.BlockProcessingCutoffHandler,
		managedPeersHolder:            arguments.ManagedPeersHolder,
		sentSignaturesTracker:         arguments.SentSignaturesTracker,
	}

	sp := shardProcessor{
		baseProcessor: base,
	}

	argsTransactionCounter := ArgsTransactionCounter{
		AppStatusHandler: sp.appStatusHandler,
		Hasher:           sp.hasher,
		Marshalizer:      sp.marshalizer,
		ShardID:          sp.shardCoordinator.SelfId(),
	}
	sp.txCounter, err = NewTransactionCounter(argsTransactionCounter)
	if err != nil {
		return nil, err
	}

	sp.requestBlockBodyHandler = &sp
	sp.blockProcessor = &sp

	sp.chRcvAllMetaHdrs = make(chan bool)

	sp.hdrsForCurrBlock = newHdrForBlock()

	headersPool := sp.dataPool.Headers()
	headersPool.RegisterHandler(sp.receivedMetaBlock)

	sp.metaBlockFinality = process.BlockFinality

	//! -------------------- NEW CODE --------------------
	sp.txSingleSigner = arguments.TxSingleSigner
	sp.txPrivateKey = arguments.TxPrivateKey
	sp.txPublicKey = arguments.TxPublicKey
	sp.coreComponents = arguments.CoreComponents
	
	sp.problematicMiniBlocksForCurrentRound = make(map[string]*data.ProblematicMBInfo)
	sp.readyMbsInsertedInCurrentBlock = make(map[string]bool)
	//! ---------------- END OF NEW CODE -----------------	

	return &sp, nil
}

// ProcessBlock processes a block. It returns nil if all ok or the specific error
func (sp *shardProcessor) ProcessBlock(
	headerHandler data.HeaderHandler,
	bodyHandler data.BodyHandler,
	haveTime func() time.Duration,
	//! -------------------- NEW CODE --------------------
	problematicMbsForCurrRound map[string]*data.ProblematicMBInfo,
	//! ---------------- END OF NEW CODE -----------------	
) error {
	if haveTime == nil {
		return process.ErrNilHaveTimeHandler
	}

	sp.processStatusHandler.SetBusy("shardProcessor.ProcessBlock")
	defer sp.processStatusHandler.SetIdle()

	err := sp.checkBlockValidity(headerHandler, bodyHandler)
	if err != nil {
		if err == process.ErrBlockHashDoesNotMatch {
			log.Debug("requested missing shard header",
				"hash", headerHandler.GetPrevHash(),
				"for shard", headerHandler.GetShardID(),
			)

			go sp.requestHandler.RequestShardHeader(headerHandler.GetShardID(), headerHandler.GetPrevHash())
		}

		return err
	}

	sp.roundNotifier.CheckRound(headerHandler)
	sp.epochNotifier.CheckEpoch(headerHandler)
	sp.requestHandler.SetEpoch(headerHandler.GetEpoch())

	err = sp.checkScheduledRootHash(headerHandler)
	if err != nil {
		return err
	}

	log.Debug("started processing block",
		"epoch", headerHandler.GetEpoch(),
		"shard", headerHandler.GetShardID(),
		"round", headerHandler.GetRound(),
		"nonce", headerHandler.GetNonce(),
	)

	//! -------------------- NEW CODE --------------------
	if len(sp.problematicMiniBlocksForCurrentRound) == 0  && len(problematicMbsForCurrRound) > 0{
		log.Debug("*** --- SAVING non-empty received problematicMbsForCurrRound inside shardProcessor (as VALIDATOR, since leader should have them already)----***")
		sp.problematicMiniBlocksForCurrentRound = problematicMbsForCurrRound
		log.Debug("*** sp.problematicMiniBlocksForCurrentRound is now: ----***", "sp.problematicMiniBlocksForCurrentRound", sp.problematicMiniBlocksForCurrentRound)
	}
	//! ---------------- END OF NEW CODE -----------------

	header, ok := headerHandler.(data.ShardHeaderHandler)
	if !ok {
		return process.ErrWrongTypeAssertion
	}

	body, ok := bodyHandler.(*block.Body)
	if !ok {
		return process.ErrWrongTypeAssertion
	}

	go getMetricsFromBlockBody(body, sp.marshalizer, sp.appStatusHandler)

	err = sp.checkHeaderBodyCorrelation(header.GetMiniBlockHeaderHandlers(), body)
	if err != nil {
		return err
	}

	err = sp.checkScheduledMiniBlocksValidity(headerHandler)
	if err != nil {
		return err
	}

	txCounts, rewardCounts, unsignedCounts := sp.txCounter.getPoolCounts(sp.dataPool)
	log.Debug("total txs in pool", "counts", txCounts.String())
	log.Debug("total txs in rewards pool", "counts", rewardCounts.String())
	log.Debug("total txs in unsigned pool", "counts", unsignedCounts.String())

	go getMetricsFromHeader(header, uint64(txCounts.GetTotal()), sp.marshalizer, sp.appStatusHandler)

	err = sp.createBlockStarted()
	if err != nil {
		return err
	}

	sp.blockChainHook.SetCurrentHeader(header)

	sp.txCoordinator.RequestBlockTransactions(body)
	requestedMetaHdrs, requestedFinalityAttestingMetaHdrs := sp.requestMetaHeaders(header)

	if haveTime() < 0 {
		return process.ErrTimeIsOut
	}

	err = sp.txCoordinator.IsDataPreparedForProcessing(haveTime)
	if err != nil {
		return err
	}

	haveMissingMetaHeaders := requestedMetaHdrs > 0 || requestedFinalityAttestingMetaHdrs > 0
	if haveMissingMetaHeaders {
		if requestedMetaHdrs > 0 {
			log.Debug("requested missing meta headers",
				"num headers", requestedMetaHdrs,
			)
		}
		if requestedFinalityAttestingMetaHdrs > 0 {
			log.Debug("requested missing finality attesting meta headers",
				"num finality meta headers", requestedFinalityAttestingMetaHdrs,
			)
		}

		err = sp.waitForMetaHdrHashes(haveTime())

		sp.hdrsForCurrBlock.mutHdrsForBlock.RLock()
		missingMetaHdrs := sp.hdrsForCurrBlock.missingHdrs
		sp.hdrsForCurrBlock.mutHdrsForBlock.RUnlock()

		sp.hdrsForCurrBlock.resetMissingHdrs()

		if requestedMetaHdrs > 0 {
			log.Debug("received missing meta headers",
				"num headers", requestedMetaHdrs-missingMetaHdrs,
			)
		}

		if err != nil {
			return err
		}
	}

	err = sp.requestEpochStartInfo(header, haveTime)
	if err != nil {
		return err
	}

	if sp.accountsDB[state.UserAccountsState].JournalLen() != 0 {
		log.Error("shardProcessor.ProcessBlock first entry", "stack", string(sp.accountsDB[state.UserAccountsState].GetStackDebugFirstEntry()))
		return process.ErrAccountStateDirty
	}

	defer func() {
		go sp.checkAndRequestIfMetaHeadersMissing()
	}()

	err = sp.checkEpochCorrectnessCrossChain()
	if err != nil {
		return err
	}

	err = sp.checkEpochCorrectness(header)
	if err != nil {
		return err
	}

	err = sp.checkMetaHeadersValidityAndFinality()
	if err != nil {
		return err
	}

	err = sp.verifyCrossShardMiniBlockDstMe(header)
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			sp.RevertCurrentBlock()
		}
	}()

	mbIndex := sp.getIndexOfFirstMiniBlockToBeExecuted(header)
	miniBlocks := body.MiniBlocks[mbIndex:]

	startTime := time.Now()
	err = sp.txCoordinator.ProcessBlockTransaction(header, &block.Body{MiniBlocks: miniBlocks}, haveTime)
	elapsedTime := time.Since(startTime)
	log.Debug("elapsed time to process block transaction",
		"time [s]", elapsedTime,
	)
	if err != nil {
		return err
	}

	err = sp.txCoordinator.VerifyCreatedBlockTransactions(header, &block.Body{MiniBlocks: miniBlocks})
	if err != nil {
		return err
	}

	err = sp.txCoordinator.VerifyCreatedMiniBlocks(header, body)
	if err != nil {
		return err
	}

	err = sp.verifyFees(header)
	if err != nil {
		return err
	}

	if !sp.verifyStateRoot(header.GetRootHash()) {
		err = process.ErrRootStateDoesNotMatch
		return err
	}

	err = sp.blockProcessingCutoffHandler.HandleProcessErrorCutoff(header)
	if err != nil {
		return err
	}

	return nil
}

func (sp *shardProcessor) requestEpochStartInfo(header data.ShardHeaderHandler, haveTime func() time.Duration) error {
	if !header.IsStartOfEpochBlock() {
		return nil
	}
	if sp.epochStartTrigger.MetaEpoch() >= header.GetEpoch() {
		return nil
	}
	if sp.epochStartTrigger.IsEpochStart() {
		return nil
	}

	go sp.requestHandler.RequestMetaHeader(header.GetEpochStartMetaHash())

	headersPool := sp.dataPool.Headers()
	for {
		time.Sleep(timeBetweenCheckForEpochStart)
		if haveTime() < 0 {
			break
		}

		if sp.epochStartTrigger.IsEpochStart() {
			return nil
		}

		epochStartMetaHdr, err := headersPool.GetHeaderByHash(header.GetEpochStartMetaHash())
		if err != nil {
			go sp.requestHandler.RequestMetaHeader(header.GetEpochStartMetaHash())
			continue
		}

		_, _, err = headersPool.GetHeadersByNonceAndShardId(epochStartMetaHdr.GetNonce()+1, core.MetachainShardId)
		if err != nil {
			go sp.requestHandler.RequestMetaHeaderByNonce(epochStartMetaHdr.GetNonce() + 1)
			continue
		}

		return nil
	}

	return process.ErrTimeIsOut
}

// RevertStateToBlock recreates the state tries to the root hashes indicated by the provided root hash and header
func (sp *shardProcessor) RevertStateToBlock(header data.HeaderHandler, rootHash []byte) error {

	err := sp.accountsDB[state.UserAccountsState].RecreateTrie(rootHash)
	if err != nil {
		log.Debug("recreate trie with error for header",
			"nonce", header.GetNonce(),
			"header root hash", header.GetRootHash(),
			"given root hash", rootHash,
			"error", err,
		)

		return err
	}

	err = sp.epochStartTrigger.RevertStateToBlock(header)
	if err != nil {
		log.Debug("revert epoch start trigger for header",
			"nonce", header.GetNonce(),
			"error", err,
		)
		return err
	}

	return nil
}

func (sp *shardProcessor) checkEpochCorrectness(
	header data.ShardHeaderHandler,
) error {
	currentBlockHeader := sp.blockChain.GetCurrentBlockHeader()
	if check.IfNil(currentBlockHeader) {
		return nil
	}

	headerEpochBehindCurrentHeader := header.GetEpoch() < currentBlockHeader.GetEpoch()
	if headerEpochBehindCurrentHeader {
		return fmt.Errorf("%w proposed header with older epoch %d than blockchain epoch %d",
			process.ErrEpochDoesNotMatch, header.GetEpoch(), currentBlockHeader.GetEpoch())
	}

	isStartOfEpochButShouldNotBe := header.GetEpoch() == currentBlockHeader.GetEpoch() && header.IsStartOfEpochBlock()
	if isStartOfEpochButShouldNotBe {
		return fmt.Errorf("%w proposed header with same epoch %d as blockchain and it is of epoch start",
			process.ErrEpochDoesNotMatch, currentBlockHeader.GetEpoch())
	}

	incorrectStartOfEpochBlock := header.GetEpoch() != currentBlockHeader.GetEpoch() &&
		sp.epochStartTrigger.MetaEpoch() == currentBlockHeader.GetEpoch()
	if incorrectStartOfEpochBlock {
		if header.IsStartOfEpochBlock() {
			sp.dataPool.Headers().RemoveHeaderByHash(header.GetEpochStartMetaHash())
			go sp.requestHandler.RequestMetaHeader(header.GetEpochStartMetaHash())
		}
		return fmt.Errorf("%w proposed header with new epoch %d with trigger still in last epoch %d",
			process.ErrEpochDoesNotMatch, header.GetEpoch(), sp.epochStartTrigger.MetaEpoch())
	}

	isHeaderOfInvalidEpoch := header.GetEpoch() > sp.epochStartTrigger.MetaEpoch()
	if isHeaderOfInvalidEpoch {
		return fmt.Errorf("%w proposed header with epoch too high %d with trigger in epoch %d",
			process.ErrEpochDoesNotMatch, header.GetEpoch(), sp.epochStartTrigger.MetaEpoch())
	}

	isOldEpochAndShouldBeNew := sp.epochStartTrigger.IsEpochStart() &&
		header.GetRound() > sp.epochStartTrigger.EpochFinalityAttestingRound()+process.EpochChangeGracePeriod &&
		header.GetEpoch() < sp.epochStartTrigger.MetaEpoch() &&
		sp.epochStartTrigger.EpochStartRound() < sp.epochStartTrigger.EpochFinalityAttestingRound()
	if isOldEpochAndShouldBeNew {
		return fmt.Errorf("%w proposed header with epoch %d should be in epoch %d",
			process.ErrEpochDoesNotMatch, header.GetEpoch(), sp.epochStartTrigger.MetaEpoch())
	}

	isEpochStartMetaHashIncorrect := header.IsStartOfEpochBlock() &&
		!bytes.Equal(header.GetEpochStartMetaHash(), sp.epochStartTrigger.EpochStartMetaHdrHash()) &&
		header.GetEpoch() == sp.epochStartTrigger.MetaEpoch()
	if isEpochStartMetaHashIncorrect {
		go sp.requestHandler.RequestMetaHeader(header.GetEpochStartMetaHash())
		log.Warn("epoch start meta hash mismatch", "proposed", header.GetEpochStartMetaHash(), "calculated", sp.epochStartTrigger.EpochStartMetaHdrHash())
		return fmt.Errorf("%w proposed header with epoch %d has invalid epochStartMetaHash",
			process.ErrEpochDoesNotMatch, header.GetEpoch())
	}

	isNotEpochStartButShouldBe := header.GetEpoch() != currentBlockHeader.GetEpoch() &&
		!header.IsStartOfEpochBlock()
	if isNotEpochStartButShouldBe {
		return fmt.Errorf("%w proposed header with new epoch %d is not of type epoch start",
			process.ErrEpochDoesNotMatch, header.GetEpoch())
	}

	isOldEpochStart := header.IsStartOfEpochBlock() && header.GetEpoch() < sp.epochStartTrigger.MetaEpoch()
	if isOldEpochStart {
		metaBlock, err := process.GetMetaHeader(header.GetEpochStartMetaHash(), sp.dataPool.Headers(), sp.marshalizer, sp.store)
		if err != nil {
			go sp.requestHandler.RequestStartOfEpochMetaBlock(header.GetEpoch())
			return fmt.Errorf("%w could not find epoch start metablock for epoch %d",
				err, header.GetEpoch())
		}

		isMetaBlockCorrect := metaBlock.IsStartOfEpochBlock() && metaBlock.GetEpoch() == header.GetEpoch()
		if !isMetaBlockCorrect {
			return fmt.Errorf("%w proposed header with epoch %d does not include correct start of epoch metaBlock %s",
				process.ErrEpochDoesNotMatch, header.GetEpoch(), header.GetEpochStartMetaHash())
		}
	}

	return nil
}

// SetNumProcessedObj will set the num of processed transactions
func (sp *shardProcessor) SetNumProcessedObj(numObj uint64) {
	sp.txCounter.totalTxs = numObj
}

// checkMetaHeadersValidity - checks if listed metaheaders are valid as construction
func (sp *shardProcessor) checkMetaHeadersValidityAndFinality() error {
	lastCrossNotarizedHeader, _, err := sp.blockTracker.GetLastCrossNotarizedHeader(core.MetachainShardId)
	if err != nil {
		return err
	}

	log.Trace("checkMetaHeadersValidityAndFinality", "lastCrossNotarizedHeader nonce", lastCrossNotarizedHeader.GetNonce())
	usedMetaHdrs := sp.sortHeadersForCurrentBlockByNonce(true)
	if len(usedMetaHdrs[core.MetachainShardId]) == 0 {
		return nil
	}

	for _, metaHdr := range usedMetaHdrs[core.MetachainShardId] {
		log.Trace("checkMetaHeadersValidityAndFinality", "metaHeader nonce", metaHdr.GetNonce())
		err = sp.headerValidator.IsHeaderConstructionValid(metaHdr, lastCrossNotarizedHeader)
		if err != nil {
			return fmt.Errorf("%w : checkMetaHeadersValidityAndFinality -> isHdrConstructionValid", err)
		}

		lastCrossNotarizedHeader = metaHdr
	}

	err = sp.checkMetaHdrFinality(lastCrossNotarizedHeader)
	if err != nil {
		return err
	}

	return nil
}

// check if shard headers are final by checking if newer headers were constructed upon them
func (sp *shardProcessor) checkMetaHdrFinality(header data.HeaderHandler) error {
	if check.IfNil(header) {
		return process.ErrNilBlockHeader
	}

	finalityAttestingMetaHdrs := sp.sortHeadersForCurrentBlockByNonce(false)

	lastVerifiedHdr := header
	// verify if there are "K" block after current to make this one final
	nextBlocksVerified := uint32(0)
	for _, metaHdr := range finalityAttestingMetaHdrs[core.MetachainShardId] {
		if nextBlocksVerified >= sp.metaBlockFinality {
			break
		}

		// found a header with the next nonce
		if metaHdr.GetNonce() == lastVerifiedHdr.GetNonce()+1 {
			err := sp.headerValidator.IsHeaderConstructionValid(metaHdr, lastVerifiedHdr)
			if err != nil {
				log.Debug("checkMetaHdrFinality -> isHdrConstructionValid",
					"error", err.Error())
				continue
			}

			lastVerifiedHdr = metaHdr
			nextBlocksVerified += 1
		}
	}

	if nextBlocksVerified < sp.metaBlockFinality {
		go sp.requestHandler.RequestMetaHeaderByNonce(lastVerifiedHdr.GetNonce())
		go sp.requestHandler.RequestMetaHeaderByNonce(lastVerifiedHdr.GetNonce() + 1)
		return process.ErrHeaderNotFinal
	}

	return nil
}

func (sp *shardProcessor) checkAndRequestIfMetaHeadersMissing() {
	orderedMetaBlocks, _ := sp.blockTracker.GetTrackedHeaders(core.MetachainShardId)

	err := sp.requestHeadersIfMissing(orderedMetaBlocks, core.MetachainShardId)
	if err != nil {
		log.Debug("checkAndRequestIfMetaHeadersMissing", "error", err.Error())
	}
}

func (sp *shardProcessor) indexBlockIfNeeded(
	body data.BodyHandler,
	headerHash []byte,
	header data.HeaderHandler,
	lastBlockHeader data.HeaderHandler,
) {
	if !sp.outportHandler.HasDrivers() {
		return
	}

	log.Debug("preparing to index block", "hash", headerHash, "nonce", header.GetNonce(), "round", header.GetRound())
	argSaveBlock, err := sp.outportDataProvider.PrepareOutportSaveBlockData(processOutport.ArgPrepareOutportSaveBlockData{
		HeaderHash:             headerHash,
		Header:                 header,
		Body:                   body,
		PreviousHeader:         lastBlockHeader,
		HighestFinalBlockNonce: sp.forkDetector.GetHighestFinalBlockNonce(),
		HighestFinalBlockHash:  sp.forkDetector.GetHighestFinalBlockHash(),
	})
	if err != nil {
		log.Error("shardProcessor.indexBlockIfNeeded cannot prepare argSaveBlock", "error", err.Error(),
			"hash", headerHash, "nonce", header.GetNonce(), "round", header.GetRound())
		return
	}
	err = sp.outportHandler.SaveBlock(argSaveBlock)
	if err != nil {
		log.Error("shardProcessor.outportHandler.SaveBlock cannot save block", "error", err,
			"hash", headerHash, "nonce", header.GetNonce(), "round", header.GetRound())
		return
	}

	log.Debug("indexed block", "hash", headerHash, "nonce", header.GetNonce(), "round", header.GetRound())

	shardID := sp.shardCoordinator.SelfId()
	indexRoundInfo(sp.outportHandler, sp.nodesCoordinator, shardID, header, lastBlockHeader, argSaveBlock.SignersIndexes)
}

// RestoreBlockIntoPools restores the TxBlock and MetaBlock into associated pools
func (sp *shardProcessor) RestoreBlockIntoPools(headerHandler data.HeaderHandler, bodyHandler data.BodyHandler) error {
	if check.IfNil(headerHandler) {
		return process.ErrNilBlockHeader
	}

	header, ok := headerHandler.(data.ShardHeaderHandler)
	if !ok {
		return process.ErrWrongTypeAssertion
	}

	miniBlockHashes := header.MapMiniBlockHashesToShards()
	err := sp.restoreMetaBlockIntoPool(headerHandler, miniBlockHashes, header.GetMetaBlockHashes())
	if err != nil {
		return err
	}

	sp.restoreBlockBody(headerHandler, bodyHandler)

	sp.blockTracker.RemoveLastNotarizedHeaders()

	return nil
}

func (sp *shardProcessor) restoreMetaBlockIntoPool(
	headerHandler data.HeaderHandler,
	mapMiniBlockHashes map[string]uint32,
	metaBlockHashes [][]byte,
) error {
	headersPool := sp.dataPool.Headers()

	mapMetaHashMiniBlockHashes := make(map[string][][]byte)
	mapMetaHashMetaBlock := make(map[string]*block.MetaBlock)

	for _, metaBlockHash := range metaBlockHashes {
		metaBlock, errNotCritical := process.GetMetaHeaderFromStorage(metaBlockHash, sp.marshalizer, sp.store)
		if errNotCritical != nil {
			log.Debug("meta block is not fully processed yet and not committed in MetaBlockUnit",
				"hash", metaBlockHash)
			continue
		}

		mapMetaHashMetaBlock[string(metaBlockHash)] = metaBlock
		processedMiniBlocks := metaBlock.GetMiniBlockHeadersWithDst(sp.shardCoordinator.SelfId())
		for mbHash := range processedMiniBlocks {
			mapMetaHashMiniBlockHashes[string(metaBlockHash)] = append(mapMetaHashMiniBlockHashes[string(metaBlockHash)], []byte(mbHash))
		}

		headersPool.AddHeader(metaBlockHash, metaBlock)

		metablockStorer, err := sp.store.GetStorer(dataRetriever.MetaBlockUnit)
		if err != nil {
			log.Debug("unable to get storage unit",
				"unit", dataRetriever.MetaBlockUnit.String())
			return err
		}

		err = metablockStorer.Remove(metaBlockHash)
		if err != nil {
			log.Debug("unable to remove hash from MetaBlockUnit",
				"hash", metaBlockHash)
			return err
		}

		nonceToByteSlice := sp.uint64Converter.ToByteSlice(metaBlock.GetNonce())

		metaHdrNonceHashStorer, err := sp.store.GetStorer(dataRetriever.MetaHdrNonceHashDataUnit)
		if err != nil {
			log.Debug("unable to get storage unit",
				"unit", dataRetriever.MetaHdrNonceHashDataUnit.String())
			return err
		}

		errNotCritical = metaHdrNonceHashStorer.Remove(nonceToByteSlice)
		if errNotCritical != nil {
			log.Debug("error not critical",
				"error", errNotCritical.Error())
		}

		log.Trace("meta block has been restored successfully",
			"round", metaBlock.Round,
			"nonce", metaBlock.Nonce,
			"hash", metaBlockHash)
	}

	for metaBlockHash, miniBlockHashes := range mapMetaHashMiniBlockHashes {
		sp.setProcessedMiniBlocksInfo(miniBlockHashes, metaBlockHash, mapMetaHashMetaBlock[metaBlockHash])
	}

	sp.rollBackProcessedMiniBlocksInfo(headerHandler, mapMiniBlockHashes)

	return nil
}

func (sp *shardProcessor) setProcessedMiniBlocksInfo(miniBlockHashes [][]byte, metaBlockHash string, metaBlock *block.MetaBlock) {
	for _, miniBlockHash := range miniBlockHashes {
		indexOfLastTxProcessed := getIndexOfLastTxProcessedInMiniBlock(miniBlockHash, metaBlock)
		sp.processedMiniBlocksTracker.SetProcessedMiniBlockInfo([]byte(metaBlockHash), miniBlockHash, &processedMb.ProcessedMiniBlockInfo{
			FullyProcessed:         true,
			IndexOfLastTxProcessed: indexOfLastTxProcessed,
		})
	}
}

func getIndexOfLastTxProcessedInMiniBlock(miniBlockHash []byte, metaBlock *block.MetaBlock) int32 {
	for _, mbh := range metaBlock.MiniBlockHeaders {
		if bytes.Equal(mbh.Hash, miniBlockHash) {
			return int32(mbh.TxCount) - 1
		}
	}

	for _, shardData := range metaBlock.ShardInfo {
		for _, mbh := range shardData.ShardMiniBlockHeaders {
			if bytes.Equal(mbh.Hash, miniBlockHash) {
				return int32(mbh.TxCount) - 1
			}
		}
	}

	log.Warn("shardProcessor.getIndexOfLastTxProcessedInMiniBlock",
		"miniBlock hash", miniBlockHash,
		"metaBlock round", metaBlock.Round,
		"metaBlock nonce", metaBlock.Nonce,
		"error", process.ErrMissingMiniBlock)

	return common.MaxIndexOfTxInMiniBlock
}

func (sp *shardProcessor) rollBackProcessedMiniBlocksInfo(headerHandler data.HeaderHandler, mapMiniBlockHashes map[string]uint32) {
	for miniBlockHash := range mapMiniBlockHashes {
		miniBlockHeader := process.GetMiniBlockHeaderWithHash(headerHandler, []byte(miniBlockHash))
		if miniBlockHeader == nil {
			log.Warn("shardProcessor.rollBackProcessedMiniBlocksInfo: GetMiniBlockHeaderWithHash",
				"mb hash", miniBlockHash,
				"error", process.ErrMissingMiniBlockHeader)
			continue
		}

		if miniBlockHeader.GetSenderShardID() == sp.shardCoordinator.SelfId() {
			continue
		}

		sp.rollBackProcessedMiniBlockInfo(miniBlockHeader, []byte(miniBlockHash))
	}
}

func (sp *shardProcessor) rollBackProcessedMiniBlockInfo(miniBlockHeader data.MiniBlockHeaderHandler, miniBlockHash []byte) {
	indexOfFirstTxProcessed := miniBlockHeader.GetIndexOfFirstTxProcessed()
	if indexOfFirstTxProcessed == 0 {
		sp.processedMiniBlocksTracker.RemoveMiniBlockHash(miniBlockHash)
		return
	}

	_, metaBlockHash := sp.processedMiniBlocksTracker.GetProcessedMiniBlockInfo(miniBlockHash)
	if metaBlockHash == nil {
		log.Warn("shardProcessor.rollBackProcessedMiniBlockInfo: mini block was not found in ProcessedMiniBlockTracker component",
			"sender shard", miniBlockHeader.GetSenderShardID(),
			"receiver shard", miniBlockHeader.GetReceiverShardID(),
			"tx count", miniBlockHeader.GetTxCount(),
			"mb hash", miniBlockHash)
		return
	}

	sp.processedMiniBlocksTracker.SetProcessedMiniBlockInfo(metaBlockHash, miniBlockHash, &processedMb.ProcessedMiniBlockInfo{
		FullyProcessed:         false,
		IndexOfLastTxProcessed: indexOfFirstTxProcessed - 1,
	})
}

// CreateBlock creates the final block and header for the current round
func (sp *shardProcessor) CreateBlock(
	initialHdr data.HeaderHandler,
	haveTime func() bool,
) (data.HeaderHandler, data.BodyHandler, map[string]*data.ProblematicMBInfo, error) { //! MODIFIED CODE
	if check.IfNil(initialHdr) {
		return nil, nil, nil, process.ErrNilBlockHeader //! MODIFIED CODE
	}
	shardHdr, ok := initialHdr.(data.ShardHeaderHandler)
	if !ok {
		return nil, nil, nil, process.ErrWrongTypeAssertion //! MODIFIED CODE
	}

	sp.processStatusHandler.SetBusy("shardProcessor.CreateBlock")
	defer sp.processStatusHandler.SetIdle()

	err := sp.createBlockStarted()
	if err != nil {
		return nil, nil, nil, err //! MODIFIED CODE
	}

	// placeholder for shardProcessor.CreateBlock script 1

	// placeholder for shardProcessor.CreateBlock script 2

	if sp.epochStartTrigger.IsEpochStart() {
		log.Debug("CreateBlock", "IsEpochStart", sp.epochStartTrigger.IsEpochStart(),
			"epoch start meta header hash", sp.epochStartTrigger.EpochStartMetaHdrHash())
		err = shardHdr.SetEpochStartMetaHash(sp.epochStartTrigger.EpochStartMetaHdrHash())
		if err != nil {
			return nil, nil, nil, err //! MODIFIED CODE
		}

		epoch := sp.epochStartTrigger.MetaEpoch()
		if initialHdr.GetEpoch() != epoch {
			log.Debug("shardProcessor.CreateBlock: epoch from header is not the same as epoch from epoch start trigger, overwriting",
				"epoch from header", initialHdr.GetEpoch(), "epoch from epoch start trigger", epoch)
			err = shardHdr.SetEpoch(epoch)
			if err != nil {
				return nil, nil, nil, err //! MODIFIED CODE
			}
		}
	}

	sp.epochNotifier.CheckEpoch(shardHdr)
	sp.blockChainHook.SetCurrentHeader(shardHdr)
	body, processedMiniBlocksDestMeInfo, problematicMiniBlocksForCurrentRound, err := sp.createBlockBody(shardHdr, haveTime) //! MODIFIED CODE
	if err != nil {
		return nil, nil, nil, err //! MODIFIED CODE
	}

	finalBody, err := sp.applyBodyToHeader(shardHdr, body, processedMiniBlocksDestMeInfo)
	if err != nil {
		return nil, nil, nil, err //! MODIFIED CODE
	}

	for _, miniBlock := range finalBody.MiniBlocks {
		log.Trace("CreateBlock: miniblock",
			"sender shard", miniBlock.SenderShardID,
			"receiver shard", miniBlock.ReceiverShardID,
			"type", miniBlock.Type,
			"num txs", len(miniBlock.TxHashes))
	}

	return shardHdr, finalBody, problematicMiniBlocksForCurrentRound, nil //! MODIFIED CODE
}

// createBlockBody creates a list of miniblocks by filling them with transactions out of the transactions pools
// as long as the transactions limit for the block has not been reached and there is still time to add transactions
func (sp *shardProcessor) createBlockBody(shardHdr data.HeaderHandler, haveTime func() bool) (*block.Body, map[string]*processedMb.ProcessedMiniBlockInfo, map[string]*data.ProblematicMBInfo, error) { //! MODIFIED CODE
	sp.blockSizeThrottler.ComputeCurrentMaxSize()

	log.Debug("started creating block body",
		"epoch", shardHdr.GetEpoch(),
		"round", shardHdr.GetRound(),
		"nonce", shardHdr.GetNonce(),
	)

	miniBlocks, processedMiniBlocksDestMeInfo, problematicMiniBlocksForCurrentRound, err := sp.createMiniBlocks(haveTime, shardHdr.GetPrevRandSeed()) //! MODIFIED CODE
	if err != nil {
		return nil, nil, nil, err
	}

	sp.requestHandler.SetEpoch(shardHdr.GetEpoch())

	return miniBlocks, processedMiniBlocksDestMeInfo, problematicMiniBlocksForCurrentRound, nil //! MODIFIED CODE
}

// CommitBlock commits the block in the blockchain if everything was checked successfully
func (sp *shardProcessor) CommitBlock(
	headerHandler data.HeaderHandler,
	bodyHandler data.BodyHandler,
) error {
	var err error
	sp.processStatusHandler.SetBusy("shardProcessor.CommitBlock")
	defer func() {	
		if err != nil {
			sp.RevertCurrentBlock()
		//! -------------------- NEW CODE --------------------
		} else {
			//TODO: se non c'è stato nessun errore, allora mi copio le info dei problematic miniblocks dentro "waitingMbsForAATsNotarization"
			sp.updateWaitingMbsForAATsNotarization()
			sp.removeReadyMbsInsertedInCurrentRoundFromWaitingMbs()
		//! ---------------- END OF NEW CODE -----------------		
		}
		//! -------------------- NEW CODE --------------------
		//? a prescindere che il miniblocco sia stato committato o "reverted", comunque faccio il clean dei problematic mbs for current round, visto che il current round è finito
		sp.cleanProblematicMiniBlocksForCurrentRound()
		sp.cleanReadyMbsInsertedInCurrentRoundInShardProcessor()
		//! ---------------- END OF NEW CODE -----------------			
		sp.processStatusHandler.SetIdle()
	}()

	err = checkForNils(headerHandler, bodyHandler)
	if err != nil {
		return err
	}

	sp.store.SetEpochForPutOperation(headerHandler.GetEpoch())

	log.Debug("started committing block",
		"epoch", headerHandler.GetEpoch(),
		"shard", headerHandler.GetShardID(),
		"round", headerHandler.GetRound(),
		"nonce", headerHandler.GetNonce(),
	)

	err = sp.checkBlockValidity(headerHandler, bodyHandler)
	if err != nil {
		return err
	}

	header, ok := headerHandler.(data.ShardHeaderHandler)
	if !ok {
		err = process.ErrWrongTypeAssertion
		return err
	}

	err = sp.checkEpochCorrectnessCrossChain()
	if err != nil {
		return err
	}

	if header.IsStartOfEpochBlock() {
		sp.epochStartTrigger.SetProcessed(header, bodyHandler)
	}

	marshalizedHeader, err := sp.marshalizer.Marshal(header)
	if err != nil {
		return err
	}

	headerHash := sp.hasher.Compute(string(marshalizedHeader))

	sp.saveShardHeader(header, headerHash, marshalizedHeader)

	body, ok := bodyHandler.(*block.Body)
	if !ok {
		err = process.ErrWrongTypeAssertion
		return err
	}

	sp.saveBody(body, header, headerHash)

	//TODO: CONTROLLARE
	processedMetaHdrs, err := sp.getOrderedProcessedMetaBlocksFromHeader(header)
	if err != nil {
		return err
	}

	err = sp.addProcessedCrossMiniBlocksFromHeader(header)
	if err != nil {
		return err
	}

	selfNotarizedHeaders, selfNotarizedHeadersHashes, err := sp.getHighestHdrForOwnShardFromMetachain(processedMetaHdrs)
	if err != nil {
		return err
	}

	err = sp.saveLastNotarizedHeader(core.MetachainShardId, processedMetaHdrs)
	if err != nil {
		return err
	}

	err = sp.commitAll(headerHandler)
	if err != nil {
		return err
	}

	log.Info("shard block has been committed successfully",
		"epoch", header.GetEpoch(),
		"shard", header.GetShardID(),
		"round", header.GetRound(),
		"nonce", header.GetNonce(),
		"hash", headerHash,
	)
	sp.setNonceOfFirstCommittedBlock(headerHandler.GetNonce())

	sp.updateLastCommittedInDebugger(headerHandler.GetRound())

	errNotCritical := sp.checkSentSignaturesAtCommitTime(headerHandler)
	if errNotCritical != nil {
		log.Debug("checkSentSignaturesBeforeCommitting", "error", errNotCritical.Error())
	}

	errNotCritical = sp.updateCrossShardInfo(processedMetaHdrs)
	if errNotCritical != nil {
		log.Debug("updateCrossShardInfo", "error", errNotCritical.Error())
	}

	errNotCritical = sp.forkDetector.AddHeader(header, headerHash, process.BHProcessed, selfNotarizedHeaders, selfNotarizedHeadersHashes)
	if errNotCritical != nil {
		log.Debug("forkDetector.AddHeader", "error", errNotCritical.Error())
	}

	currentHeader, currentHeaderHash := getLastSelfNotarizedHeaderByItself(sp.blockChain)
	sp.blockTracker.AddSelfNotarizedHeader(sp.shardCoordinator.SelfId(), currentHeader, currentHeaderHash)

	lastSelfNotarizedHeader, lastSelfNotarizedHeaderHash := sp.getLastSelfNotarizedHeaderByMetachain()
	sp.blockTracker.AddSelfNotarizedHeader(core.MetachainShardId, lastSelfNotarizedHeader, lastSelfNotarizedHeaderHash)

	sp.notifyFinalMetaHdrs(processedMetaHdrs)

	if sp.lastRestartNonce == 0 {
		sp.lastRestartNonce = header.GetNonce()
	}

	sp.updateState(selfNotarizedHeaders, header)

	highestFinalBlockNonce := sp.forkDetector.GetHighestFinalBlockNonce()
	log.Debug("highest final shard block",
		"shard", sp.shardCoordinator.SelfId(),
		"nonce", highestFinalBlockNonce,
	)

	lastBlockHeader := sp.blockChain.GetCurrentBlockHeader()

	committedRootHash, err := sp.accountsDB[state.UserAccountsState].RootHash()
	if err != nil {
		return err
	}

	err = sp.blockChain.SetCurrentBlockHeaderAndRootHash(header, committedRootHash)
	if err != nil {
		return err
	}

	sp.blockChain.SetCurrentBlockHeaderHash(headerHash)
	sp.indexBlockIfNeeded(bodyHandler, headerHash, headerHandler, lastBlockHeader)
	sp.recordBlockInHistory(headerHash, headerHandler, bodyHandler)

	lastCrossNotarizedHeader, _, err := sp.blockTracker.GetLastCrossNotarizedHeader(core.MetachainShardId)
	if err != nil {
		return err
	}

	saveMetricsForCommittedShardBlock(
		sp.nodesCoordinator,
		sp.appStatusHandler,
		logger.DisplayByteSlice(headerHash),
		highestFinalBlockNonce,
		lastCrossNotarizedHeader,
		header,
		sp.managedPeersHolder,
	)

	headerInfo := bootstrapStorage.BootstrapHeaderInfo{
		ShardId: header.GetShardID(),
		Epoch:   header.GetEpoch(),
		Nonce:   header.GetNonce(),
		Hash:    headerHash,
	}

	nodesCoordinatorKey := sp.nodesCoordinator.GetSavedStateKey()
	epochStartKey := sp.epochStartTrigger.GetSavedStateKey()

	args := bootStorerDataArgs{
		headerInfo:                 headerInfo,
		round:                      header.GetRound(),
		lastSelfNotarizedHeaders:   sp.getBootstrapHeadersInfo(selfNotarizedHeaders, selfNotarizedHeadersHashes),
		highestFinalBlockNonce:     sp.forkDetector.GetHighestFinalBlockNonce(),
		processedMiniBlocks:        sp.processedMiniBlocksTracker.ConvertProcessedMiniBlocksMapToSlice(),
		nodesCoordinatorConfigKey:  nodesCoordinatorKey,
		epochStartTriggerConfigKey: epochStartKey,
	}

	sp.prepareDataForBootStorer(args)

	// write data to log
	go func() {
		sp.txCounter.headerExecuted(header)
		sp.txCounter.displayLogInfo(
			header,
			body,
			headerHash,
			sp.shardCoordinator.NumberOfShards(),
			sp.shardCoordinator.SelfId(),
			sp.dataPool,
			sp.blockTracker,
		)
	}()

	sp.blockSizeThrottler.Succeed(header.GetRound())

	sp.displayPoolsInfo()

	errNotCritical = sp.removeTxsFromPools(header, body)
	if errNotCritical != nil {
		log.Debug("removeTxsFromPools", "error", errNotCritical.Error())
	}

	sp.cleanupPools(headerHandler)

	sp.blockProcessingCutoffHandler.HandlePauseCutoff(header)

	return nil
}

func (sp *shardProcessor) notifyFinalMetaHdrs(processedMetaHeaders []data.HeaderHandler) {
	metaHeaders := make([]data.HeaderHandler, 0)
	metaHeadersHashes := make([][]byte, 0)

	for _, metaHeader := range processedMetaHeaders {
		metaHeaderHash, err := core.CalculateHash(sp.marshalizer, sp.hasher, metaHeader)
		if err != nil {
			log.Debug("shardProcessor.notifyFinalMetaHdrs", "error", err.Error())
			continue
		}

		metaHeaders = append(metaHeaders, metaHeader)
		metaHeadersHashes = append(metaHeadersHashes, metaHeaderHash)
	}

	if len(metaHeaders) > 0 {
		go sp.historyRepo.OnNotarizedBlocks(core.MetachainShardId, metaHeaders, metaHeadersHashes)
	}
}

func (sp *shardProcessor) displayPoolsInfo() {
	headersPool := sp.dataPool.Headers()
	miniBlocksPool := sp.dataPool.MiniBlocks()

	log.Trace("pools info",
		"shard", sp.shardCoordinator.SelfId(),
		"num headers", headersPool.GetNumHeaders(sp.shardCoordinator.SelfId()))

	log.Trace("pools info",
		"shard", core.MetachainShardId,
		"num headers", headersPool.GetNumHeaders(core.MetachainShardId))

	// numShardsToKeepHeaders represents the total number of shards for which shard node would keep tracking headers
	// (in this case this number is equal with: self shard + metachain)
	numShardsToKeepHeaders := 2
	capacity := headersPool.MaxSize() * numShardsToKeepHeaders
	log.Debug("pools info",
		"total headers", headersPool.Len(),
		"headers pool capacity", capacity,
		"total miniblocks", miniBlocksPool.Len(),
		"miniblocks pool capacity", miniBlocksPool.MaxSize(),
	)

	sp.displayMiniBlocksPool()
}

func (sp *shardProcessor) updateState(headers []data.HeaderHandler, currentHeader data.ShardHeaderHandler) {
	sp.snapShotEpochStartFromMeta(currentHeader)

	for _, header := range headers {
		if sp.forkDetector.GetHighestFinalBlockNonce() < header.GetNonce() {
			break
		}

		prevHeaderHash := header.GetPrevHash()
		prevHeader, errNotCritical := process.GetShardHeader(
			prevHeaderHash,
			sp.dataPool.Headers(),
			sp.marshalizer,
			sp.store,
		)
		if errNotCritical != nil {
			log.Debug("could not get shard header from storage")
			return
		}
		if header.IsStartOfEpochBlock() {
			sp.nodesCoordinator.ShuffleOutForEpoch(header.GetEpoch())
		}

		headerHash, err := core.CalculateHash(sp.marshalizer, sp.hasher, header)
		if err != nil {
			log.Debug("updateState.CalculateHash", "error", err.Error())
			return
		}

		headerRootHashForPruning := header.GetRootHash()
		prevHeaderRootHashForPruning := prevHeader.GetRootHash()

		scheduledHeaderRootHash, _ := sp.scheduledTxsExecutionHandler.GetScheduledRootHashForHeader(headerHash)
		headerAdditionalData := header.GetAdditionalData()
		if headerAdditionalData != nil && headerAdditionalData.GetScheduledRootHash() != nil {
			headerRootHashForPruning = headerAdditionalData.GetScheduledRootHash()
		}

		prevHeaderAdditionalData := prevHeader.GetAdditionalData()
		scheduledPrevHeaderRootHash, _ := sp.scheduledTxsExecutionHandler.GetScheduledRootHashForHeader(header.GetPrevHash())
		if prevHeaderAdditionalData != nil && prevHeaderAdditionalData.GetScheduledRootHash() != nil {
			prevHeaderRootHashForPruning = prevHeaderAdditionalData.GetScheduledRootHash()
		}

		log.Trace("updateState: prevHeader",
			"shard", prevHeader.GetShardID(),
			"epoch", prevHeader.GetEpoch(),
			"round", prevHeader.GetRound(),
			"nonce", prevHeader.GetNonce(),
			"root hash", prevHeader.GetRootHash(),
			"scheduled root hash for pruning", prevHeaderRootHashForPruning,
			"scheduled root hash after processing", scheduledPrevHeaderRootHash,
		)

		log.Trace("updateState: currHeader",
			"shard", header.GetShardID(),
			"epoch", header.GetEpoch(),
			"round", header.GetRound(),
			"nonce", header.GetNonce(),
			"root hash", header.GetRootHash(),
			"scheduled root hash for pruning", headerRootHashForPruning,
			"scheduled root hash after processing", scheduledHeaderRootHash,
		)

		sp.updateStateStorage(
			header,
			headerRootHashForPruning,
			prevHeaderRootHashForPruning,
			sp.accountsDB[state.UserAccountsState],
		)

		sp.setFinalizedHeaderHashInIndexer(header.GetPrevHash())

		finalRootHash := scheduledHeaderRootHash
		if len(finalRootHash) == 0 {
			finalRootHash = header.GetRootHash()
		}

		sp.blockChain.SetFinalBlockInfo(header.GetNonce(), headerHash, finalRootHash)
	}
}

func (sp *shardProcessor) snapShotEpochStartFromMeta(header data.ShardHeaderHandler) {
	accounts := sp.accountsDB[state.UserAccountsState]

	sp.hdrsForCurrBlock.mutHdrsForBlock.RLock()
	defer sp.hdrsForCurrBlock.mutHdrsForBlock.RUnlock()

	for _, metaHash := range header.GetMetaBlockHashes() {
		metaHdrInfo, ok := sp.hdrsForCurrBlock.hdrHashAndInfo[string(metaHash)]
		if !ok {
			continue
		}
		metaHdr, ok := metaHdrInfo.hdr.(*block.MetaBlock)
		if !ok {
			continue
		}
		if !metaHdr.IsStartOfEpochBlock() {
			continue
		}

		for _, epochStartShData := range metaHdr.EpochStart.LastFinalizedHeaders {
			if epochStartShData.ShardID != header.GetShardID() {
				continue
			}

			rootHash := epochStartShData.RootHash
			schRootHash := epochStartShData.GetScheduledRootHash()
			if schRootHash != nil {
				log.Debug("using scheduled root hash for snapshotting", "schRootHash", schRootHash)
				rootHash = schRootHash
			}
			epoch := sp.epochStartTrigger.MetaEpoch()
			log.Debug("shard trie snapshot from epoch start shard data", "rootHash", rootHash, "epoch", epoch)
			accounts.SnapshotState(rootHash, epoch)
			sp.markSnapshotDoneInPeerAccounts()
			saveEpochStartEconomicsMetrics(sp.appStatusHandler, metaHdr)
			go func() {
				err := sp.commitTrieEpochRootHashIfNeeded(metaHdr, rootHash)
				if err != nil {
					log.Warn("couldn't commit trie checkpoint", "epoch", header.GetEpoch(), "error", err)
				}
			}()
		}
	}
}

func (sp *shardProcessor) markSnapshotDoneInPeerAccounts() {
	peerAccounts := sp.accountsDB[state.PeerAccountsState]
	if check.IfNil(peerAccounts) {
		log.Warn("programming error: peerAccounts is nil while trying to take a snapshot on a shard node: this can cause OOM exceptions")
		return
	}

	peerAccountsHandler, ok := peerAccounts.(peerAccountsDBHandler)
	if !ok {
		log.Warn("programming error: peerAccounts is not of type peerAccountsDBHandler: this can cause OOM exceptions")
		return
	}

	peerAccountsHandler.MarkSnapshotDone()
	log.Debug("shardProcessor.markSnapshotDoneInPeerAccounts completed")
}

func (sp *shardProcessor) checkEpochCorrectnessCrossChain() error {
	currentHeader := sp.blockChain.GetCurrentBlockHeader()
	if check.IfNil(currentHeader) {
		return nil
	}
	if sp.epochStartTrigger.EpochStartRound() >= sp.epochStartTrigger.EpochFinalityAttestingRound() {
		return nil
	}

	lastSelfNotarizedHeader, _ := sp.getLastSelfNotarizedHeaderByMetachain()
	lastFinalizedRound := uint64(0)
	if !check.IfNil(lastSelfNotarizedHeader) {
		lastFinalizedRound = lastSelfNotarizedHeader.GetRound()
	}

	shouldRevertChain := false
	nonce := currentHeader.GetNonce()
	shouldEnterNewEpochRound := sp.epochStartTrigger.EpochFinalityAttestingRound() + process.EpochChangeGracePeriod

	for round := currentHeader.GetRound(); round > shouldEnterNewEpochRound && currentHeader.GetEpoch() < sp.epochStartTrigger.MetaEpoch(); round = currentHeader.GetRound() {
		if round <= lastFinalizedRound {
			break
		}

		shouldRevertChain = true
		prevHeader, err := process.GetShardHeader(
			currentHeader.GetPrevHash(),
			sp.dataPool.Headers(),
			sp.marshalizer,
			sp.store,
		)
		if err != nil {
			return err
		}

		nonce = currentHeader.GetNonce()
		currentHeader = prevHeader
	}

	if shouldRevertChain {
		log.Debug("blockchain is wrongly constructed",
			"reverted to nonce", nonce)

		sp.forkDetector.SetRollBackNonce(nonce)
		return process.ErrEpochDoesNotMatch
	}

	return nil
}

func (sp *shardProcessor) getLastSelfNotarizedHeaderByMetachain() (data.HeaderHandler, []byte) {
	if sp.forkDetector.GetHighestFinalBlockNonce() == sp.genesisNonce {
		return sp.blockChain.GetGenesisHeader(), sp.blockChain.GetGenesisHeaderHash()
	}

	hash := sp.forkDetector.GetHighestFinalBlockHash()
	header, err := process.GetShardHeader(hash, sp.dataPool.Headers(), sp.marshalizer, sp.store)
	if err != nil {
		log.Warn("getLastSelfNotarizedHeaderByMetachain.GetShardHeader", "error", err.Error(), "hash", hash, "nonce", sp.forkDetector.GetHighestFinalBlockNonce())
		return nil, nil
	}

	return header, hash
}

func (sp *shardProcessor) saveLastNotarizedHeader(shardId uint32, processedHdrs []data.HeaderHandler) error {
	lastCrossNotarizedHeader, lastCrossNotarizedHeaderHash, err := sp.blockTracker.GetLastCrossNotarizedHeader(shardId)
	if err != nil {
		return err
	}

	lenProcessedHdrs := len(processedHdrs)
	if lenProcessedHdrs > 0 {
		if lastCrossNotarizedHeader.GetNonce() < processedHdrs[lenProcessedHdrs-1].GetNonce() {
			lastCrossNotarizedHeader = processedHdrs[lenProcessedHdrs-1]
			lastCrossNotarizedHeaderHash, err = core.CalculateHash(sp.marshalizer, sp.hasher, lastCrossNotarizedHeader)
			if err != nil {
				return err
			}
		}
	}

	sp.blockTracker.AddCrossNotarizedHeader(shardId, lastCrossNotarizedHeader, lastCrossNotarizedHeaderHash)
	DisplayLastNotarized(sp.marshalizer, sp.hasher, lastCrossNotarizedHeader, shardId)

	return nil
}

// CreateNewHeader creates a new header
func (sp *shardProcessor) CreateNewHeader(round uint64, nonce uint64) (data.HeaderHandler, error) {
	epoch := sp.epochStartTrigger.MetaEpoch()
	header := sp.versionedHeaderFactory.Create(epoch)

	shardHeader, ok := header.(data.ShardHeaderHandler)
	if !ok {
		return nil, process.ErrWrongTypeAssertion
	}

	err := shardHeader.SetRound(round)
	if err != nil {
		return nil, err
	}

	sp.roundNotifier.CheckRound(header)

	err = shardHeader.SetNonce(nonce)
	if err != nil {
		return nil, err
	}

	err = shardHeader.SetAccumulatedFees(big.NewInt(0))
	if err != nil {
		return nil, err
	}

	err = shardHeader.SetDeveloperFees(big.NewInt(0))
	if err != nil {
		return nil, err
	}

	err = sp.setHeaderVersionData(shardHeader)
	if err != nil {
		return nil, err
	}

	return header, nil
}

func (sp *shardProcessor) setHeaderVersionData(shardHeader data.ShardHeaderHandler) error {
	if check.IfNil(shardHeader) {
		return process.ErrNilHeaderHandler
	}

	rootHash, err := sp.accountsDB[state.UserAccountsState].RootHash()
	if err != nil {
		return err
	}

	scheduledGasAndFees := sp.scheduledTxsExecutionHandler.GetScheduledGasAndFees()
	additionalVersionData := &headerVersionData.AdditionalData{
		ScheduledRootHash:        rootHash,
		ScheduledAccumulatedFees: scheduledGasAndFees.AccumulatedFees,
		ScheduledDeveloperFees:   scheduledGasAndFees.DeveloperFees,
		ScheduledGasProvided:     scheduledGasAndFees.GasProvided,
		ScheduledGasPenalized:    scheduledGasAndFees.GasPenalized,
		ScheduledGasRefunded:     scheduledGasAndFees.GasRefunded,
	}

	return shardHeader.SetAdditionalData(additionalVersionData)
}

// getHighestHdrForOwnShardFromMetachain calculates the highest shard header notarized by metachain
func (sp *shardProcessor) getHighestHdrForOwnShardFromMetachain(
	processedHdrs []data.HeaderHandler,
) ([]data.HeaderHandler, [][]byte, error) {

	ownShIdHdrs := make([]data.HeaderHandler, 0, len(processedHdrs))

	for i := 0; i < len(processedHdrs); i++ {
		hdr, ok := processedHdrs[i].(*block.MetaBlock)
		if !ok {
			return nil, nil, process.ErrWrongTypeAssertion
		}

		hdrs := sp.getHighestHdrForShardFromMetachain(sp.shardCoordinator.SelfId(), hdr)
		ownShIdHdrs = append(ownShIdHdrs, hdrs...)
	}

	process.SortHeadersByNonce(ownShIdHdrs)

	ownShIdHdrsHashes := make([][]byte, len(ownShIdHdrs))
	for i := 0; i < len(ownShIdHdrs); i++ {
		hash, _ := core.CalculateHash(sp.marshalizer, sp.hasher, ownShIdHdrs[i])
		ownShIdHdrsHashes[i] = hash
	}

	return ownShIdHdrs, ownShIdHdrsHashes, nil
}

func (sp *shardProcessor) getHighestHdrForShardFromMetachain(shardId uint32, hdr *block.MetaBlock) []data.HeaderHandler {
	ownShIdHdr := make([]data.HeaderHandler, 0, len(hdr.ShardInfo))

	for _, shardInfo := range hdr.ShardInfo {
		if shardInfo.ShardID != shardId {
			continue
		}

		ownHdr, err := process.GetShardHeader(shardInfo.HeaderHash, sp.dataPool.Headers(), sp.marshalizer, sp.store)
		if err != nil {
			go sp.requestHandler.RequestShardHeader(shardInfo.ShardID, shardInfo.HeaderHash)

			log.Debug("requested missing shard header",
				"hash", shardInfo.HeaderHash,
				"shard", shardInfo.ShardID,
			)
			continue
		}

		ownShIdHdr = append(ownShIdHdr, ownHdr)
	}

	return data.TrimHeaderHandlerSlice(ownShIdHdr)
}

// getOrderedProcessedMetaBlocksFromHeader returns all the meta blocks fully processed
func (sp *shardProcessor) getOrderedProcessedMetaBlocksFromHeader(header data.HeaderHandler) ([]data.HeaderHandler, error) {
	if check.IfNil(header) {
		return nil, process.ErrNilBlockHeader
	}

	miniBlockHeaders := header.GetMiniBlockHeaderHandlers()
	miniBlockHashes := make(map[int][]byte, len(miniBlockHeaders))
	for i := 0; i < len(miniBlockHeaders); i++ {
		miniBlockHashes[i] = miniBlockHeaders[i].GetHash()
	}

	log.Trace("cross mini blocks in body",
		"num miniblocks", len(miniBlockHashes),
	)

	processedMetaBlocks, err := sp.getOrderedProcessedMetaBlocksFromMiniBlockHashes(miniBlockHeaders, miniBlockHashes)
	if err != nil {
		return nil, err
	}

	return processedMetaBlocks, nil
}

func (sp *shardProcessor) addProcessedCrossMiniBlocksFromHeader(headerHandler data.HeaderHandler) error {
	if check.IfNil(headerHandler) {
		return process.ErrNilBlockHeader
	}

	shardHeader, ok := headerHandler.(data.ShardHeaderHandler)
	if !ok {
		return process.ErrWrongTypeAssertion
	}
	miniBlockHashes := make(map[int][]byte, len(headerHandler.GetMiniBlockHeaderHandlers()))
	for i := 0; i < len(headerHandler.GetMiniBlockHeaderHandlers()); i++ {
		miniBlockHashes[i] = headerHandler.GetMiniBlockHeaderHandlers()[i].GetHash()
	}

	sp.hdrsForCurrBlock.mutHdrsForBlock.RLock()
	for _, metaBlockHash := range shardHeader.GetMetaBlockHashes() {
		headerInfo, found := sp.hdrsForCurrBlock.hdrHashAndInfo[string(metaBlockHash)]
		if !found {
			sp.hdrsForCurrBlock.mutHdrsForBlock.RUnlock()
			return fmt.Errorf("%w : addProcessedCrossMiniBlocksFromHeader metaBlockHash = %s",
				process.ErrMissingHeader, logger.DisplayByteSlice(metaBlockHash))
		}

		metaBlock, isMetaBlock := headerInfo.hdr.(*block.MetaBlock)
		if !isMetaBlock {
			sp.hdrsForCurrBlock.mutHdrsForBlock.RUnlock()
			return process.ErrWrongTypeAssertion
		}

		crossMiniBlockHashes := metaBlock.GetMiniBlockHeadersWithDst(sp.shardCoordinator.SelfId())
		for key, miniBlockHash := range miniBlockHashes {
			_, ok = crossMiniBlockHashes[string(miniBlockHash)]
			if !ok {
				continue
			}

			miniBlockHeader := process.GetMiniBlockHeaderWithHash(headerHandler, miniBlockHash)
			if miniBlockHeader == nil {
				log.Warn("shardProcessor.addProcessedCrossMiniBlocksFromHeader: GetMiniBlockHeaderWithHash", "mb hash", miniBlockHash, "error", process.ErrMissingMiniBlockHeader)
				continue
			}

			sp.processedMiniBlocksTracker.SetProcessedMiniBlockInfo(metaBlockHash, miniBlockHash, &processedMb.ProcessedMiniBlockInfo{
				FullyProcessed:         miniBlockHeader.IsFinal(),
				IndexOfLastTxProcessed: miniBlockHeader.GetIndexOfLastTxProcessed(),
			})

			delete(miniBlockHashes, key)
		}
	}
	sp.hdrsForCurrBlock.mutHdrsForBlock.RUnlock()

	return nil
}

func (sp *shardProcessor) getOrderedProcessedMetaBlocksFromMiniBlockHashes(
	miniBlockHeaders []data.MiniBlockHeaderHandler,
	miniBlockHashes map[int][]byte,
) ([]data.HeaderHandler, error) {

	processedMetaHdrs := make([]data.HeaderHandler, 0, len(sp.hdrsForCurrBlock.hdrHashAndInfo))
	processedCrossMiniBlocksHashes := make(map[string]bool, len(sp.hdrsForCurrBlock.hdrHashAndInfo))

	sp.hdrsForCurrBlock.mutHdrsForBlock.RLock()
	for metaBlockHash, headerInfo := range sp.hdrsForCurrBlock.hdrHashAndInfo {
		if !headerInfo.usedInBlock {
			continue
		}

		metaBlock, ok := headerInfo.hdr.(*block.MetaBlock)
		if !ok {
			sp.hdrsForCurrBlock.mutHdrsForBlock.RUnlock()
			return nil, process.ErrWrongTypeAssertion
		}

		log.Trace("meta header",
			"nonce", metaBlock.Nonce,
		)

		crossMiniBlockHashes := metaBlock.GetMiniBlockHeadersWithDst(sp.shardCoordinator.SelfId())
		for hash := range crossMiniBlockHashes {
			processedCrossMiniBlocksHashes[hash] = sp.processedMiniBlocksTracker.IsMiniBlockFullyProcessed([]byte(metaBlockHash), []byte(hash))
		}

		for key, miniBlockHash := range miniBlockHashes {
			_, ok = crossMiniBlockHashes[string(miniBlockHash)]
			if !ok {
				continue
			}

			processedCrossMiniBlocksHashes[string(miniBlockHash)] = miniBlockHeaders[key].IsFinal()

			delete(miniBlockHashes, key)
		}

		log.Trace("cross mini blocks in meta header",
			"num miniblocks", len(crossMiniBlockHashes),
		)

		processedAll := true
		for hash := range crossMiniBlockHashes {
			if !processedCrossMiniBlocksHashes[hash]{
				//! -------------------- NEW CODE --------------------
				log.Debug("*** cross mini block hash is not inside processedMiniBlockHashes (while the meta header it belonged to is present). If ---PROBLEM--- is not printed right below, this is perfectly right and fine!***")
				//TODO: CONTROLLARE
				//? get hash string in hex (NOT string)
				mapStringHex := metaBlock.GetShardMiniBlockHeadersWithDstInHexString(sp.shardCoordinator.SelfId())
				hashStringHex := mapStringHex[hash]
				
				for mbHash, _ := range sp.problematicMiniBlocksForCurrentRound {
					log.Debug("*** hash == problematicMbHash ???? ***", "hash", hash, "problematicMbHash", mbHash, "equal", hash == mbHash)
					log.Debug("*** hashStringHex == problematicMbHash ???? ***", "hashStringHex", hashStringHex, "problematicMbHash", mbHash, "equal", hashStringHex == mbHash)
				}

				log.Debug("*** hashStringHex of miniblock is: ***", "hashStringHex", hashStringHex)
				//? se non sta nemmeno dentro i problematic miniblocks, allora metto processedAll = false									
				if _, ok := sp.problematicMiniBlocksForCurrentRound[hashStringHex]; !ok{
				//! ---------------- END OF NEW CODE -----------------					
					log.Debug("*** ----PROBLEM---- setting processedAll = false! due to miniblock***", "mbHash", hash)
					log.Debug("*** printing sp.problematicMiniBlocksForCurrentRound ***", "sp.problematicMBsForCurrRound", sp.problematicMiniBlocksForCurrentRound)
					processedAll = false
					break
				//! -------------------- NEW CODE --------------------
				}
				//! ---------------- END OF NEW CODE -----------------						
			}
		}

		if processedAll {
			processedMetaHdrs = append(processedMetaHdrs, metaBlock)
		}

		//TODO: in più, devo appendere i metablocchi che contenevano i problematic miniblocks!
	}
	sp.hdrsForCurrBlock.mutHdrsForBlock.RUnlock()

	process.SortHeadersByNonce(processedMetaHdrs)

	return processedMetaHdrs, nil
}

func (sp *shardProcessor) updateCrossShardInfo(processedMetaHdrs []data.HeaderHandler) error {
	lastCrossNotarizedHeader, _, err := sp.blockTracker.GetLastCrossNotarizedHeader(core.MetachainShardId)
	if err != nil {
		return err
	}

	// processedMetaHdrs is also sorted
	for i := 0; i < len(processedMetaHdrs); i++ {
		hdr := processedMetaHdrs[i]

		// remove process finished
		if hdr.GetNonce() > lastCrossNotarizedHeader.GetNonce() {
			continue
		}

		// metablock was processed and finalized
		marshalizedHeader, errMarshal := sp.marshalizer.Marshal(hdr)
		if errMarshal != nil {
			log.Debug("updateCrossShardInfo.Marshal", "error", errMarshal.Error())
			continue
		}

		headerHash := sp.hasher.Compute(string(marshalizedHeader))

		sp.saveMetaHeader(hdr, headerHash, marshalizedHeader)

		sp.processedMiniBlocksTracker.RemoveMetaBlockHash(headerHash)
	}

	return nil
}

// receivedMetaBlock is a callback function when a new metablock was received
// upon receiving, it parses the new metablock and requests miniblocks and transactions
// which destination is the current shard
func (sp *shardProcessor) receivedMetaBlock(headerHandler data.HeaderHandler, metaBlockHash []byte) {
	metaBlock, ok := headerHandler.(*block.MetaBlock)
	if !ok {
		return
	}

	log.Trace("received meta block from network",
		"round", metaBlock.Round,
		"nonce", metaBlock.Nonce,
		"hash", metaBlockHash,
	)

	sp.hdrsForCurrBlock.mutHdrsForBlock.Lock()

	haveMissingMetaHeaders := sp.hdrsForCurrBlock.missingHdrs > 0 || sp.hdrsForCurrBlock.missingFinalityAttestingHdrs > 0
	if haveMissingMetaHeaders {
		hdrInfoForHash := sp.hdrsForCurrBlock.hdrHashAndInfo[string(metaBlockHash)]
		headerInfoIsNotNil := hdrInfoForHash != nil
		headerIsMissing := headerInfoIsNotNil && check.IfNil(hdrInfoForHash.hdr)
		if headerIsMissing {
			hdrInfoForHash.hdr = metaBlock
			sp.hdrsForCurrBlock.missingHdrs--

			if metaBlock.Nonce > sp.hdrsForCurrBlock.highestHdrNonce[core.MetachainShardId] {
				sp.hdrsForCurrBlock.highestHdrNonce[core.MetachainShardId] = metaBlock.Nonce
			}
		}

		// attesting something
		if sp.hdrsForCurrBlock.missingHdrs == 0 {
			sp.hdrsForCurrBlock.missingFinalityAttestingHdrs = sp.requestMissingFinalityAttestingHeaders(
				core.MetachainShardId,
				sp.metaBlockFinality,
			)
			if sp.hdrsForCurrBlock.missingFinalityAttestingHdrs == 0 {
				log.Debug("received all missing finality attesting meta headers")
			}
		}

		missingMetaHdrs := sp.hdrsForCurrBlock.missingHdrs
		missingFinalityAttestingMetaHdrs := sp.hdrsForCurrBlock.missingFinalityAttestingHdrs
		sp.hdrsForCurrBlock.mutHdrsForBlock.Unlock()

		allMissingMetaHeadersReceived := missingMetaHdrs == 0 && missingFinalityAttestingMetaHdrs == 0
		if allMissingMetaHeadersReceived {
			sp.chRcvAllMetaHdrs <- true
		}
	} else {
		sp.hdrsForCurrBlock.mutHdrsForBlock.Unlock()
	}

	go sp.requestMiniBlocksIfNeeded(headerHandler)
}

func (sp *shardProcessor) requestMetaHeaders(shardHeader data.ShardHeaderHandler) (uint32, uint32) {
	_ = core.EmptyChannel(sp.chRcvAllMetaHdrs)

	if len(shardHeader.GetMetaBlockHashes()) == 0 {
		return 0, 0
	}

	return sp.computeExistingAndRequestMissingMetaHeaders(shardHeader)
}

func (sp *shardProcessor) computeExistingAndRequestMissingMetaHeaders(header data.ShardHeaderHandler) (uint32, uint32) {
	sp.hdrsForCurrBlock.mutHdrsForBlock.Lock()
	defer sp.hdrsForCurrBlock.mutHdrsForBlock.Unlock()

	metaBlockHashes := header.GetMetaBlockHashes()
	for i := 0; i < len(metaBlockHashes); i++ {
		hdr, err := process.GetMetaHeaderFromPool(
			metaBlockHashes[i],
			sp.dataPool.Headers())

		if err != nil {
			sp.hdrsForCurrBlock.missingHdrs++
			sp.hdrsForCurrBlock.hdrHashAndInfo[string(metaBlockHashes[i])] = &hdrInfo{
				hdr:         nil,
				usedInBlock: true,
			}
			go sp.requestHandler.RequestMetaHeader(metaBlockHashes[i])
			continue
		}

		sp.hdrsForCurrBlock.hdrHashAndInfo[string(metaBlockHashes[i])] = &hdrInfo{
			hdr:         hdr,
			usedInBlock: true,
		}

		if hdr.Nonce > sp.hdrsForCurrBlock.highestHdrNonce[core.MetachainShardId] {
			sp.hdrsForCurrBlock.highestHdrNonce[core.MetachainShardId] = hdr.Nonce
		}
	}

	if sp.hdrsForCurrBlock.missingHdrs == 0 {
		sp.hdrsForCurrBlock.missingFinalityAttestingHdrs = sp.requestMissingFinalityAttestingHeaders(
			core.MetachainShardId,
			sp.metaBlockFinality,
		)
	}

	return sp.hdrsForCurrBlock.missingHdrs, sp.hdrsForCurrBlock.missingFinalityAttestingHdrs
}

func (sp *shardProcessor) verifyCrossShardMiniBlockDstMe(header data.ShardHeaderHandler) error {
	//TODO: FARE MOLTA ATTENZIONE QUI.
	//TODO: Sembra che questa funzione si vada a prendere tutti gli hash dei miniblocchi per tutti i "metahash" presenti all'interno del blocco
	//TODO: (infatti miniBlockMetaHashes si accede come miniBlockMetaHashes[miniblockHash] = metaBlockHash)
	//TODO: Successivamente, si prende tutti i miniblocchi inseriti all'interno del blocco
	//TODO: e infine, nel for, controlla che per ogni miniblocco inserito all'interno del blocco
	//TODO: quel miniblocco sia effettivamente presente dentro miniBlockMetaHashes.
	//TODO: Allora il mio dubbio è che: ok che controlla che tutti i miniblocchi all'interno di un blocco siano parte di un metablocco
	//TODO: ma quello che non controlla è se effettivamente TUTTI i miniblocchi in quel metablocco siano stati inseriti effettivamente nel blocco
	//TODO: (come invece si "controlla" lato leader, perché se TUTTI i miniblocchi di un metablocco non sono stati inseriti, si ferma!)
	//TODO: potenziale issue da mettere sul loro GitHub/Discord?	
	miniBlockMetaHashes, err := sp.getAllMiniBlockDstMeFromMeta(header)
	if err != nil {
		//! -------------------- NEW CODE --------------------
		log.Debug("***Error inside verifyCrossShardMiniBlockDstMe***", "err", err)
		//! ---------------- END OF NEW CODE -----------------			
		return err
	}

	crossMiniBlockHashes := header.GetMiniBlockHeadersWithDstInHexString(sp.shardCoordinator.SelfId()) //! MODIFIED CODE: GetMiniBlockHeadersWithDst -> GetMiniBlockHeadersWithDstInHexString  
	for hash, hashStringHex := range crossMiniBlockHashes { //! MODIFIED CODE
		if _, ok := miniBlockMetaHashes[hash]; !ok {
			//! -------------------- NEW CODE --------------------
			//? se il miniblocco non è presente dentro miniBlockMetaHashes, devo controllare che questo non sia perché
			//? quel miniblocco è stato "processato SENZA meta header", cosa che avviene quando vado a processare ed inserire nel blocco
			//? dei miniblocchi che precedentemente erano problematici e adesso li ho trovati "ready".
			//? Quindi, se un miniblocco non è presente dentro miniBlockMetaHashes, prima controllo che non sia presente 
			//? dentro waitingMBsForAATsNotarization e, solo se non è nemmeno lì dentro, allora lancio l'errore
			if !sp.shardCoordinator.IsProblematicMBReadyToBeProcessed(hashStringHex){
				log.Debug("***Error inside verifyCrossShardMiniBlockDstMe***", "err", process.ErrCrossShardMBWithoutConfirmationFromMeta)
			//! ---------------- END OF NEW CODE -----------------						
				return process.ErrCrossShardMBWithoutConfirmationFromMeta		
			//! -------------------- NEW CODE --------------------
			}else{
				log.Debug("*** Everything good: ErrCrossShardMBWithoutConfirmationFromMeta will NOT be returned for MB, as MB 'IsProblematicMBReadyToBeProcessed'", "mbHash", hash)
				sp.readyMbsInsertedInCurrentBlock[hashStringHex] = true
			}
			//! ---------------- END OF NEW CODE -----------------
		}
	}

	return nil
}

func (sp *shardProcessor) getAllMiniBlockDstMeFromMeta(header data.ShardHeaderHandler) (map[string][]byte, error) {
	lastCrossNotarizedHeader, _, err := sp.blockTracker.GetLastCrossNotarizedHeader(core.MetachainShardId)
	if err != nil {
		return nil, err
	}

	miniBlockMetaHashes := make(map[string][]byte)

	sp.hdrsForCurrBlock.mutHdrsForBlock.RLock()
	for _, metaBlockHash := range header.GetMetaBlockHashes() {
		headerInfo, ok := sp.hdrsForCurrBlock.hdrHashAndInfo[string(metaBlockHash)]
		if !ok {
			continue
		}
		metaBlock, ok := headerInfo.hdr.(*block.MetaBlock)
		if !ok {
			continue
		}
		if metaBlock.GetRound() > header.GetRound() {
			continue
		}
		if metaBlock.GetRound() <= lastCrossNotarizedHeader.GetRound() {
			continue
		}
		if metaBlock.GetNonce() <= lastCrossNotarizedHeader.GetNonce() {
			continue
		}

		crossMiniBlockHashes := metaBlock.GetMiniBlockHeadersWithDst(sp.shardCoordinator.SelfId())
		for hash := range crossMiniBlockHashes {
			miniBlockMetaHashes[hash] = metaBlockHash
		}
	}
	sp.hdrsForCurrBlock.mutHdrsForBlock.RUnlock()

	return miniBlockMetaHashes, nil
}

// full verification through metachain header
func (sp *shardProcessor) createAndProcessMiniBlocksDstMe(haveTime func() bool) (*createAndProcessMiniBlocksDestMeInfo, error) {
	log.Debug("createAndProcessMiniBlocksDstMe has been started")

	sw := core.NewStopWatch()
	sw.Start("ComputeLongestMetaChainFromLastNotarized")
	orderedMetaBlocks, orderedMetaBlocksHashes, err := sp.blockTracker.ComputeLongestMetaChainFromLastNotarized()
	sw.Stop("ComputeLongestMetaChainFromLastNotarized")
	log.Debug("measurements", sw.GetMeasurements()...)
	if err != nil {
		return nil, err
	}

	log.Debug("metablocks ordered",
		"num metablocks", len(orderedMetaBlocks),
	)

	lastMetaHdr, _, err := sp.blockTracker.GetLastCrossNotarizedHeader(core.MetachainShardId)
	if err != nil {
		return nil, err
	}

	haveAdditionalTimeFalse := func() bool {
		return false
	}

	createAndProcessInfo := &createAndProcessMiniBlocksDestMeInfo{
		haveTime:                   haveTime,
		haveAdditionalTime:         haveAdditionalTimeFalse,
		miniBlocks:                 make(block.MiniBlockSlice, 0),
		allProcessedMiniBlocksInfo: make(map[string]*processedMb.ProcessedMiniBlockInfo),
		numTxsAdded:                uint32(0),
		numHdrsAdded:               uint32(0),
		scheduledMode:              false,
		//! -------------------- NEW CODE --------------------
		problematicMiniBlocksForCurrentRoundTxHashes: make(map[string]*data.MbInfo),
		problematicMiniBlocksForCurrentRoundFullInfo: make(map[string]*data.ProblematicMBInfo),
		//! ---------------- END OF NEW CODE -----------------		
	}


	//! -------------------- NEW CODE --------------------
	// TODO: prima ancora di processare i metablocchi, vado a vedere innanzitutto se ci sono dei miniblocchi, dentro waitingMBsForAATNotarization,
	// TODO: che possono essere processati perché hanno ricevuto la notarizzazione dest-side di TUTTE le AAT che avevano associate

	//? per il momento, me li faccio solo stampare e vedo se riesco a prelevarli dalla pool dei miniblocchi:
	readyMbsHashes := sp.shardCoordinator.GetMbsWithAllAATsNotarizedFromWaitingMBs()
	if len(readyMbsHashes) > 0 {
		log.Debug("*** ---- MBS READY TO BE PROCESSED ---- ***", "numMbs", len(readyMbsHashes), "mbsHashes", readyMbsHashes)
		
		
		//TODO: -----------------------------> TOGLIERE createAndProcessInfo da createMbsAndProcessCrossShardTransactionsDstMeForReadyMbsPreviouslyWaiting, crea problemi <-------------------------------------------------- THIS!!!!!!!!!!!!!!!!!!!!!!!
		_, errCreated := sp.createMbsAndProcessCrossShardTransactionsDstMeForReadyMbsPreviouslyWaiting(createAndProcessInfo, readyMbsHashes)
		
		if errCreated != nil {
			log.Debug("***Error: inside sp.createMbsAndProcessCrossShardTransactionsDstMeForReadyMbsPreviouslyWaiting. ---- THIS SHOULD NOT HAPPEN I THINK --- ***", "err", errCreated)
			return nil, errCreated
		}else{
			for mbHash, _ := range readyMbsHashes{
				sp.readyMbsInsertedInCurrentBlock[mbHash] = true
			}
		}
		//TODO: spostare nel "chiamato"

		/*for _, mbHash := range readyMbsHashes{
			//miniVal, _ := sp.txCoordinator.miniBlockPool.Peek(miniBlockInfo.Hash)
			mbHashBytes, err := hex.DecodeString(mbHash)
			if err != nil {
				log.Debug("***Error: cannot decode mbHash from readyMbsHashes***", "mbHash", mbHash)
			}
			miniVal, found := sp.txCoordinator.GetMiniBlockFromPool(mbHashBytes)
			if found {
				miniBlock, ok := miniVal.(*block.MiniBlock)
				if !ok {
					log.Debug("***Error: cannot cast miniVal in *block.MiniBlock type for ready miniblock ***", "mbHash", mbHash)
				}else{
					log.Debug("*** MiniBlock cast successful ***", "mbHash", mbHash, "receiverShardId", miniBlock.ReceiverShardID, "senderShardId", miniBlock.SenderShardID)
				}				
			}else{
				log.Debug("***Error: ------ READY PROBLEMATIC MINIBLOCK NOT FOUND IN MINIBLOCKS POOL ---------", "mbHash", mbHash)
			}
		}*/
	}
	

	//! ---------------- END OF NEW CODE -----------------


	// do processing in order
	sp.hdrsForCurrBlock.mutHdrsForBlock.Lock()
	for i := 0; i < len(orderedMetaBlocks); i++ {
		if !createAndProcessInfo.haveTime() && !createAndProcessInfo.haveAdditionalTime() {
			log.Debug("time is up after putting cross txs with destination to current shard",
				"scheduled mode", createAndProcessInfo.scheduledMode,
				"num txs added", createAndProcessInfo.numTxsAdded,
			)
			break
		}

		if createAndProcessInfo.numHdrsAdded >= process.MaxMetaHeadersAllowedInOneShardBlock {
			log.Debug("maximum meta headers allowed to be included in one shard block has been reached",
				"scheduled mode", createAndProcessInfo.scheduledMode,
				"meta headers added", createAndProcessInfo.numHdrsAdded,
			)
			break
		}

		createAndProcessInfo.currMetaHdr = orderedMetaBlocks[i]
		if createAndProcessInfo.currMetaHdr.GetNonce() > lastMetaHdr.GetNonce()+1 {
			log.Debug("skip searching",
				"scheduled mode", createAndProcessInfo.scheduledMode,
				"last meta hdr nonce", lastMetaHdr.GetNonce(),
				"curr meta hdr nonce", createAndProcessInfo.currMetaHdr.GetNonce())
			break
		}

		createAndProcessInfo.currMetaHdrHash = orderedMetaBlocksHashes[i]
		if len(createAndProcessInfo.currMetaHdr.GetMiniBlockHeadersWithDst(sp.shardCoordinator.SelfId())) == 0 {
			sp.hdrsForCurrBlock.hdrHashAndInfo[string(createAndProcessInfo.currMetaHdrHash)] = &hdrInfo{hdr: createAndProcessInfo.currMetaHdr, usedInBlock: true}
			createAndProcessInfo.numHdrsAdded++
			lastMetaHdr = createAndProcessInfo.currMetaHdr
			continue
		}

		createAndProcessInfo.currProcessedMiniBlocksInfo = sp.processedMiniBlocksTracker.GetProcessedMiniBlocksInfo(createAndProcessInfo.currMetaHdrHash)
		createAndProcessInfo.hdrAdded = false

		shouldContinue, errCreated := sp.createMbsAndProcessCrossShardTransactionsDstMe(createAndProcessInfo)
		if errCreated != nil {
			sp.hdrsForCurrBlock.mutHdrsForBlock.Unlock()
			return nil, errCreated
		}
		if !shouldContinue {
			break
		}

		lastMetaHdr = createAndProcessInfo.currMetaHdr
	}
	sp.hdrsForCurrBlock.mutHdrsForBlock.Unlock()

	go sp.requestMetaHeadersIfNeeded(createAndProcessInfo.numHdrsAdded, lastMetaHdr)

	for _, miniBlock := range createAndProcessInfo.miniBlocks {
		log.Debug("mini block info",
			"type", miniBlock.Type,
			"sender shard", miniBlock.SenderShardID,
			"receiver shard", miniBlock.ReceiverShardID,
			"txs added", len(miniBlock.TxHashes))
	}

	//! -------------------- NEW CODE --------------------
	for mbHash, problematicMbInfo := range createAndProcessInfo.problematicMiniBlocksForCurrentRoundTxHashes{
		problematicMbInfo := sp.generateAATsForProblematicTxs(problematicMbInfo, mbHash) //TODO: mi faccio ritornare queste info al subroundBlock (?), in modo che una volta che il blocco verrà effettivamente committato le andrò a "salvare permanentemente" 
		
		log.Debug("***ProblematicMbInfo (added inside sp.problematicMiniBlocksForCurrentRound***", 
			"mbHash", mbHash,
			"problematicTxHashes",
			problematicMbInfo.ProblematicTxHashes,
			"accountAdjustmentTxs", problematicMbInfo.AccAdjTxHashes,
			"numProblmaticTxs", len(problematicMbInfo.ProblematicTxHashes),
			"numAccountAdjTxs", len(problematicMbInfo.AccAdjTxHashes),
		)
		
		//TODO: PERCHè STO METTENDO I PROBLEMATICMBS NELLO SHARDPROCESSOR?? LO SHARD PROCESSOR NON è CHE SI RESETTA OGNI ROUND!
		//TODO: SAREBBE PIù GIUSTO RITORNARLI E BASTA!
		// ? update sp's (shardProceessor's) structure holding all problematic mbs for current round
		createAndProcessInfo.problematicMiniBlocksForCurrentRoundFullInfo[mbHash] = problematicMbInfo
	}

	sp.problematicMiniBlocksForCurrentRound = createAndProcessInfo.problematicMiniBlocksForCurrentRoundFullInfo //TODO: a questo punto forse è anche ridondante avere le problematicMiniBlocksForCurrentRoundFullInfo dentro createaAndProcessInfo, ma ormai ho fatto così, quindi pazienza, però si potrebbe togliere
	//! ---------------- END OF NEW CODE -----------------	

	log.Debug("createAndProcessMiniBlocksDstMe has been finished",
		"num txs added", createAndProcessInfo.numTxsAdded,
		"num hdrs added", createAndProcessInfo.numHdrsAdded)

	return createAndProcessInfo, nil
}

func (sp *shardProcessor) createMbsAndProcessCrossShardTransactionsDstMe(
	createAndProcessInfo *createAndProcessMiniBlocksDestMeInfo,
) (bool, error) {
	currMiniBlocksAdded, currNumTxsAdded, pendingMiniBlocksFromMetaBlock, hdrProcessFinished, hdrProcessFinishedWithProblematicMiniBlocks, errCreated := sp.txCoordinator.CreateMbsAndProcessCrossShardTransactionsDstMe( //! MODIFIED CODE
		createAndProcessInfo.currMetaHdr,
		createAndProcessInfo.currProcessedMiniBlocksInfo,
		createAndProcessInfo.haveTime,
		createAndProcessInfo.haveAdditionalTime,
		createAndProcessInfo.scheduledMode)
	if errCreated != nil {
		return false, errCreated
	}

	for miniBlockHash, processedMiniBlockInfo := range createAndProcessInfo.currProcessedMiniBlocksInfo {
		createAndProcessInfo.allProcessedMiniBlocksInfo[miniBlockHash] = &processedMb.ProcessedMiniBlockInfo{
			FullyProcessed:         processedMiniBlockInfo.FullyProcessed,
			IndexOfLastTxProcessed: processedMiniBlockInfo.IndexOfLastTxProcessed,
		}
	}

	// all txs processed, add to processed miniblocks
	createAndProcessInfo.miniBlocks = append(createAndProcessInfo.miniBlocks, currMiniBlocksAdded...)
	createAndProcessInfo.numTxsAdded += currNumTxsAdded

	if !createAndProcessInfo.hdrAdded && (currNumTxsAdded > 0 || hdrProcessFinishedWithProblematicMiniBlocks) { //! MODIFIED CODE -> SENZA QUESTO CONTROLLO IL SE IL METABLOCCO CONTIENE SOLO MINIBLOCCHI PROBLEMATICI, NON VERRA' AGGIUNTO AL ROUNDBLOCK PERCHE' currNumTxsAdded > 0 NON E' SODDISFATTO
		sp.hdrsForCurrBlock.hdrHashAndInfo[string(createAndProcessInfo.currMetaHdrHash)] = &hdrInfo{hdr: createAndProcessInfo.currMetaHdr, usedInBlock: true}
		createAndProcessInfo.numHdrsAdded++
		createAndProcessInfo.hdrAdded = true
	}
	//! -------------------- NEW CODE --------------------
	/*if len(pendingMiniBlocksFromMetaBlock) > 0 {
		sp.createAndBroadcastAATsForPendingMiniBlocksFromMetaBlock(pendingMiniBlocksFromMetaBlock)
	}*/
	for mbHash, problematicMbInfo := range pendingMiniBlocksFromMetaBlock{
		createAndProcessInfo.problematicMiniBlocksForCurrentRoundTxHashes[mbHash] = problematicMbInfo //TODO: consider adding the metablock hash too!!
	}
	//! ---------------- END OF NEW CODE -----------------	

	if !hdrProcessFinished && !hdrProcessFinishedWithProblematicMiniBlocks{ //! MODIFIED CODE
		log.Debug("meta block cannot be fully processed",
			"scheduled mode", createAndProcessInfo.scheduledMode,
			"round", createAndProcessInfo.currMetaHdr.GetRound(),
			"nonce", createAndProcessInfo.currMetaHdr.GetNonce(),
			"hash", createAndProcessInfo.currMetaHdrHash,
			"num mbs added", len(currMiniBlocksAdded),
			"num txs added", currNumTxsAdded)

		//! -------------------- NEW CODE --------------------
		//TODO: CONTROLLARE SE TOGLIERE LA POSSIBILITà DI ESEGUIRE IL MINIBLOCCO DST ME IN SCHEDULED MODE CREA PROBLEMI
		/*
		//! ---------------- END OF NEW CODE -----------------		
		if sp.enableEpochsHandler.IsScheduledMiniBlocksFlagEnabled() && !createAndProcessInfo.scheduledMode {
			createAndProcessInfo.scheduledMode = true
			createAndProcessInfo.haveAdditionalTime = process.HaveAdditionalTime()
			return sp.createMbsAndProcessCrossShardTransactionsDstMe(createAndProcessInfo)
		}
		//! -------------------- NEW CODE --------------------	
		*/
		//! ---------------- END OF NEW CODE -----------------				

		return false, nil
	}

	return true, nil
}


//! -------------------- NEW CODE --------------------
func (sp *shardProcessor) createMbsAndProcessCrossShardTransactionsDstMeForReadyMbsPreviouslyWaiting(
	createAndProcessInfo *createAndProcessMiniBlocksDestMeInfo,
	readyMbsHashes map[string]*data.AccountAjustmentTxsInfo,
) (bool, error) {
	currMiniBlocksAdded, currNumTxsAdded, _, hdrProcessFinished, _, errCreated := sp.txCoordinator.CreateMbsAndProcessCrossShardTransactionsDstMeForReadyMbsPreviouslyWaiting(
		readyMbsHashes,
		createAndProcessInfo.currProcessedMiniBlocksInfo,
		createAndProcessInfo.haveTime,
		createAndProcessInfo.haveAdditionalTime,
		createAndProcessInfo.scheduledMode)
	if errCreated != nil {
		return false, errCreated
	}

	for miniBlockHash, processedMiniBlockInfo := range createAndProcessInfo.currProcessedMiniBlocksInfo {
		createAndProcessInfo.allProcessedMiniBlocksInfo[miniBlockHash] = &processedMb.ProcessedMiniBlockInfo{
			FullyProcessed:         processedMiniBlockInfo.FullyProcessed,
			IndexOfLastTxProcessed: processedMiniBlockInfo.IndexOfLastTxProcessed,
		}
	}

	// all txs processed, add to processed miniblocks
	createAndProcessInfo.miniBlocks = append(createAndProcessInfo.miniBlocks, currMiniBlocksAdded...)
	createAndProcessInfo.numTxsAdded += currNumTxsAdded

	//if !createAndProcessInfo.hdrAdded && (currNumTxsAdded > 0 || hdrProcessFinishedWithProblematicMiniBlocks) { //! MODIFIED CODE -> SENZA QUESTO CONTROLLO IL SE IL METABLOCCO CONTIENE SOLO MINIBLOCCHI PROBLEMATICI, NON VERRA' AGGIUNTO AL ROUNDBLOCK PERCHE' currNumTxsAdded > 0 NON E' SODDISFATTO
	/*if !createAndProcessInfo.hdrAdded && currNumTxsAdded > 0 {
		sp.hdrsForCurrBlock.hdrHashAndInfo[string(createAndProcessInfo.currMetaHdrHash)] = &hdrInfo{hdr: createAndProcessInfo.currMetaHdr, usedInBlock: true}
		createAndProcessInfo.numHdrsAdded++
		createAndProcessInfo.hdrAdded = true
	}*/ //? LASCIARE COMMENTATO -> si riferisce al "currMetaHdrHash", ma qui non stiamo processando nessun meta header!!!!!!!
	//! -------------------- NEW CODE --------------------
	/*for mbHash, problematicTxHashes := range pendingMiniBlocksFromMetaBlock{
		createAndProcessInfo.problematicMiniBlocksForCurrentRoundTxHashes[mbHash] = problematicTxHashes //TODO: consider adding the metablock hash too!!
	}*/
	//! ---------------- END OF NEW CODE -----------------	

	//if !hdrProcessFinished && !hdrProcessFinishedWithProblematicMiniBlocks{ //! MODIFIED CODE
	if !hdrProcessFinished {
		log.Debug("meta block cannot be fully processed",
			"scheduled mode", createAndProcessInfo.scheduledMode,
			"round", createAndProcessInfo.currMetaHdr.GetRound(),
			"nonce", createAndProcessInfo.currMetaHdr.GetNonce(),
			"hash", createAndProcessInfo.currMetaHdrHash,
			"num mbs added", len(currMiniBlocksAdded),
			"num txs added", currNumTxsAdded)

		//! -------------------- NEW CODE --------------------
		//TODO: CONTROLLARE SE TOGLIERE LA POSSIBILITà DI ESEGUIRE IL MINIBLOCCO DST ME IN SCHEDULED MODE CREA PROBLEMI
		/*
		//! ---------------- END OF NEW CODE -----------------		
		if sp.enableEpochsHandler.IsScheduledMiniBlocksFlagEnabled() && !createAndProcessInfo.scheduledMode {
			createAndProcessInfo.scheduledMode = true
			createAndProcessInfo.haveAdditionalTime = process.HaveAdditionalTime()
			return sp.createMbsAndProcessCrossShardTransactionsDstMe(createAndProcessInfo)
		}
		//! -------------------- NEW CODE --------------------	
		*/
		//! ---------------- END OF NEW CODE -----------------				

		return false, nil
	}

	return true, nil
}
//! ---------------- END OF NEW CODE -----------------



func (sp *shardProcessor) requestMetaHeadersIfNeeded(hdrsAdded uint32, lastMetaHdr data.HeaderHandler) {
	log.Debug("meta headers added",
		"num", hdrsAdded,
		"highest nonce", lastMetaHdr.GetNonce(),
	)

	roundTooOld := sp.roundHandler.Index() > int64(lastMetaHdr.GetRound()+process.MaxRoundsWithoutNewBlockReceived)
	shouldRequestCrossHeaders := hdrsAdded == 0 && roundTooOld
	if shouldRequestCrossHeaders {
		fromNonce := lastMetaHdr.GetNonce() + 1
		toNonce := fromNonce + uint64(sp.metaBlockFinality)
		for nonce := fromNonce; nonce <= toNonce; nonce++ {
			sp.addHeaderIntoTrackerPool(nonce, core.MetachainShardId)
			sp.requestHandler.RequestMetaHeaderByNonce(nonce)
		}
	}
}

func (sp *shardProcessor) createMiniBlocks(haveTime func() bool, randomness []byte) (*block.Body, map[string]*processedMb.ProcessedMiniBlockInfo, map[string]*data.ProblematicMBInfo, error) { //! MODIFIED CODE
	var miniBlocks block.MiniBlockSlice
	processedMiniBlocksDestMeInfo := make(map[string]*processedMb.ProcessedMiniBlockInfo)

	if sp.enableEpochsHandler.IsScheduledMiniBlocksFlagEnabled() {
		miniBlocks = sp.scheduledTxsExecutionHandler.GetScheduledMiniBlocks()
		sp.txCoordinator.AddTxsFromMiniBlocks(miniBlocks)

		scheduledIntermediateTxs := sp.scheduledTxsExecutionHandler.GetScheduledIntermediateTxs()
		sp.txCoordinator.AddTransactions(scheduledIntermediateTxs[block.InvalidBlock], block.TxBlock)
	}

	// placeholder for shardProcessor.createMiniBlocks script

	if sp.accountsDB[state.UserAccountsState].JournalLen() != 0 {
		log.Error("shardProcessor.createMiniBlocks",
			"error", process.ErrAccountStateDirty,
			"stack", string(sp.accountsDB[state.UserAccountsState].GetStackDebugFirstEntry()))

		interMBs := sp.txCoordinator.CreatePostProcessMiniBlocks()
		if len(interMBs) > 0 {
			miniBlocks = append(miniBlocks, interMBs...)
		}

		log.Debug("creating mini blocks has been finished", "num miniblocks", len(miniBlocks))
		return &block.Body{MiniBlocks: miniBlocks}, processedMiniBlocksDestMeInfo, make(map[string]*data.ProblematicMBInfo), nil //! MODIFIED CODE
	}

	if !haveTime() {
		log.Debug("shardProcessor.createMiniBlocks", "error", process.ErrTimeIsOut)

		interMBs := sp.txCoordinator.CreatePostProcessMiniBlocks()
		if len(interMBs) > 0 {
			miniBlocks = append(miniBlocks, interMBs...)
		}

		log.Debug("creating mini blocks has been finished", "num miniblocks", len(miniBlocks))
		return &block.Body{MiniBlocks: miniBlocks}, processedMiniBlocksDestMeInfo, make(map[string]*data.ProblematicMBInfo), nil //! MODIFIED CODE
	}

	startTime := time.Now()
	createAndProcessMBsDestMeInfo, err := sp.createAndProcessMiniBlocksDstMe(haveTime)
	elapsedTime := time.Since(startTime)
	log.Debug("elapsed time to create mbs to me", "time", elapsedTime)
	if err != nil {
		log.Debug("createAndProcessCrossMiniBlocksDstMe", "error", err.Error())
	}
	if createAndProcessMBsDestMeInfo != nil {
		processedMiniBlocksDestMeInfo = createAndProcessMBsDestMeInfo.allProcessedMiniBlocksInfo
		if len(createAndProcessMBsDestMeInfo.miniBlocks) > 0 {
			miniBlocks = append(miniBlocks, createAndProcessMBsDestMeInfo.miniBlocks...)

			log.Debug("processed miniblocks and txs with destination in self shard",
				"num miniblocks", len(createAndProcessMBsDestMeInfo.miniBlocks),
				"num txs", createAndProcessMBsDestMeInfo.numTxsAdded,
				"num meta headers", createAndProcessMBsDestMeInfo.numHdrsAdded)
		}
	}

	if sp.blockTracker.IsShardStuck(core.MetachainShardId) {
		log.Warn("shardProcessor.createMiniBlocks",
			"error", process.ErrShardIsStuck,
			"shard", core.MetachainShardId)

		interMBs := sp.txCoordinator.CreatePostProcessMiniBlocks()
		if len(interMBs) > 0 {
			miniBlocks = append(miniBlocks, interMBs...)
		}

		log.Debug("creating mini blocks has been finished", "num miniblocks", len(miniBlocks))
		return &block.Body{MiniBlocks: miniBlocks}, processedMiniBlocksDestMeInfo, make(map[string]*data.ProblematicMBInfo), nil //! MODIFIED CODE
	}

	startTime = time.Now()
	mbsFromMe := sp.txCoordinator.CreateMbsAndProcessTransactionsFromMe(haveTime, randomness)
	elapsedTime = time.Since(startTime)
	log.Debug("elapsed time to create mbs from me", "time", elapsedTime)

	if len(mbsFromMe) > 0 {
		miniBlocks = append(miniBlocks, mbsFromMe...)

		numTxs := 0
		for _, mb := range mbsFromMe {
			numTxs += len(mb.TxHashes)
		}

		log.Debug("processed miniblocks and txs from self shard",
			"num miniblocks", len(mbsFromMe),
			"num txs", numTxs)
	}

	log.Debug("creating mini blocks has been finished", "num miniblocks", len(miniBlocks))
	return &block.Body{MiniBlocks: miniBlocks}, processedMiniBlocksDestMeInfo, createAndProcessMBsDestMeInfo.problematicMiniBlocksForCurrentRoundFullInfo, nil //! MODIFIED CODE
}

// applyBodyToHeader creates a miniblock header list given a block body
func (sp *shardProcessor) applyBodyToHeader(
	shardHeader data.ShardHeaderHandler,
	body *block.Body,
	processedMiniBlocksDestMeInfo map[string]*processedMb.ProcessedMiniBlockInfo,
) (*block.Body, error) {
	sw := core.NewStopWatch()
	sw.Start("applyBodyToHeader")
	defer func() {
		sw.Stop("applyBodyToHeader")
		log.Debug("measurements", sw.GetMeasurements()...)
	}()

	var err error
	err = shardHeader.SetMiniBlockHeaderHandlers(nil)
	if err != nil {
		return nil, err
	}

	err = shardHeader.SetRootHash(sp.getRootHash())
	if err != nil {
		return nil, err
	}

	defer func() {
		go sp.checkAndRequestIfMetaHeadersMissing()
	}()

	if check.IfNil(body) {
		return nil, process.ErrNilBlockBody
	}

	var receiptsHash []byte
	sw.Start("CreateReceiptsHash")
	receiptsHash, err = sp.txCoordinator.CreateReceiptsHash()
	sw.Stop("CreateReceiptsHash")
	if err != nil {
		return nil, err
	}

	err = shardHeader.SetReceiptsHash(receiptsHash)
	if err != nil {
		return nil, err
	}

	newBody := deleteSelfReceiptsMiniBlocks(body)

	sw.Start("createMiniBlockHeaders")
	totalTxCount, miniBlockHeaderHandlers, err := sp.createMiniBlockHeaderHandlers(newBody, processedMiniBlocksDestMeInfo)
	sw.Stop("createMiniBlockHeaders")
	if err != nil {
		return nil, err
	}

	err = shardHeader.SetMiniBlockHeaderHandlers(miniBlockHeaderHandlers)
	if err != nil {
		return nil, err
	}

	err = shardHeader.SetTxCount(uint32(totalTxCount))
	if err != nil {
		return nil, err
	}

	err = shardHeader.SetAccumulatedFees(sp.feeHandler.GetAccumulatedFees())
	if err != nil {
		return nil, err
	}

	err = shardHeader.SetDeveloperFees(sp.feeHandler.GetDeveloperFees())
	if err != nil {
		return nil, err
	}

	sw.Start("sortHeaderHashesForCurrentBlockByNonce")
	metaBlockHashes := sp.sortHeaderHashesForCurrentBlockByNonce(true)
	sw.Stop("sortHeaderHashesForCurrentBlockByNonce")
	err = shardHeader.SetMetaBlockHashes(metaBlockHashes[core.MetachainShardId])
	if err != nil {
		return nil, err
	}

	err = sp.txCoordinator.VerifyCreatedMiniBlocks(shardHeader, newBody)
	if err != nil {
		return nil, err
	}

	sp.appStatusHandler.SetUInt64Value(common.MetricNumTxInBlock, uint64(totalTxCount))
	sp.appStatusHandler.SetUInt64Value(common.MetricNumMiniBlocks, uint64(len(body.MiniBlocks)))

	marshalizedBody, err := sp.marshalizer.Marshal(newBody)
	if err != nil {
		return nil, err
	}
	sp.blockSizeThrottler.Add(shardHeader.GetRound(), uint32(len(marshalizedBody)))

	return newBody, nil
}

func (sp *shardProcessor) waitForMetaHdrHashes(waitTime time.Duration) error {
	select {
	case <-sp.chRcvAllMetaHdrs:
		return nil
	case <-time.After(waitTime):
		return process.ErrTimeIsOut
	}
}

// MarshalizedDataToBroadcast prepares underlying data into a marshalized object according to destination
func (sp *shardProcessor) MarshalizedDataToBroadcast(
	header data.HeaderHandler,
	bodyHandler data.BodyHandler,
) (map[uint32][]byte, map[string][][]byte, error) {

	if check.IfNil(bodyHandler) {
		return nil, nil, process.ErrNilMiniBlocks
	}

	body, ok := bodyHandler.(*block.Body)
	if !ok {
		return nil, nil, process.ErrWrongTypeAssertion
	}

	// Remove mini blocks which are not final from "body" to avoid sending them cross shard
	newBodyToBroadcast, err := sp.getFinalMiniBlocks(header, body)
	if err != nil {
		return nil, nil, err
	}

	mrsTxs := sp.txCoordinator.CreateMarshalizedData(newBodyToBroadcast)

	bodies := make(map[uint32]block.MiniBlockSlice)
	for _, miniBlock := range newBodyToBroadcast.MiniBlocks {
		if miniBlock.SenderShardID != sp.shardCoordinator.SelfId() ||
			miniBlock.ReceiverShardID == sp.shardCoordinator.SelfId() {
			continue
		}
		bodies[miniBlock.ReceiverShardID] = append(bodies[miniBlock.ReceiverShardID], miniBlock)
	}

	mrsData := make(map[uint32][]byte, len(bodies))
	for shardId, subsetBlockBody := range bodies {
		bodyForShard := block.Body{MiniBlocks: subsetBlockBody}
		buff, errMarshal := sp.marshalizer.Marshal(&bodyForShard)
		if errMarshal != nil {
			log.Error("shardProcessor.MarshalizedDataToBroadcast.Marshal", "error", errMarshal.Error())
			continue
		}
		mrsData[shardId] = buff
	}

	return mrsData, mrsTxs, nil
}

// IsInterfaceNil returns true if there is no value under the interface
func (sp *shardProcessor) IsInterfaceNil() bool {
	return sp == nil
}

// GetBlockBodyFromPool returns block body from pool for a given header
func (sp *shardProcessor) GetBlockBodyFromPool(headerHandler data.HeaderHandler) (data.BodyHandler, error) {
	header, ok := headerHandler.(data.ShardHeaderHandler)
	if !ok {
		return nil, process.ErrWrongTypeAssertion
	}

	miniBlocksPool := sp.dataPool.MiniBlocks()
	var miniBlocks block.MiniBlockSlice

	for _, mbHeader := range header.GetMiniBlockHeaderHandlers() {
		obj, hashInPool := miniBlocksPool.Get(mbHeader.GetHash())
		if !hashInPool {
			continue
		}

		miniBlock, typeOk := obj.(*block.MiniBlock)
		if !typeOk {
			return nil, process.ErrWrongTypeAssertion
		}

		miniBlocks = append(miniBlocks, miniBlock)
	}

	return &block.Body{MiniBlocks: miniBlocks}, nil
}

func (sp *shardProcessor) getBootstrapHeadersInfo(
	selfNotarizedHeaders []data.HeaderHandler,
	selfNotarizedHeadersHashes [][]byte,
) []bootstrapStorage.BootstrapHeaderInfo {

	numSelfNotarizedHeaders := len(selfNotarizedHeaders)

	highestNonceInSelfNotarizedHeaders := uint64(0)
	if numSelfNotarizedHeaders > 0 {
		highestNonceInSelfNotarizedHeaders = selfNotarizedHeaders[numSelfNotarizedHeaders-1].GetNonce()
	}

	isFinalNonceHigherThanSelfNotarized := sp.forkDetector.GetHighestFinalBlockNonce() > highestNonceInSelfNotarizedHeaders
	if isFinalNonceHigherThanSelfNotarized {
		numSelfNotarizedHeaders++
	}

	if numSelfNotarizedHeaders == 0 {
		return nil
	}

	lastSelfNotarizedHeaders := make([]bootstrapStorage.BootstrapHeaderInfo, 0, numSelfNotarizedHeaders)

	for index := range selfNotarizedHeaders {
		headerInfo := bootstrapStorage.BootstrapHeaderInfo{
			ShardId: selfNotarizedHeaders[index].GetShardID(),
			Nonce:   selfNotarizedHeaders[index].GetNonce(),
			Hash:    selfNotarizedHeadersHashes[index],
		}

		lastSelfNotarizedHeaders = append(lastSelfNotarizedHeaders, headerInfo)
	}

	if isFinalNonceHigherThanSelfNotarized {
		headerInfo := bootstrapStorage.BootstrapHeaderInfo{
			ShardId: sp.shardCoordinator.SelfId(),
			Nonce:   sp.forkDetector.GetHighestFinalBlockNonce(),
			Hash:    sp.forkDetector.GetHighestFinalBlockHash(),
		}

		lastSelfNotarizedHeaders = append(lastSelfNotarizedHeaders, headerInfo)
	}

	return lastSelfNotarizedHeaders
}

func (sp *shardProcessor) removeStartOfEpochBlockDataFromPools(
	_ data.HeaderHandler,
	_ data.BodyHandler,
) error {
	return nil
}

// Close - closes all underlying components
func (sp *shardProcessor) Close() error {
	return sp.baseProcessor.Close()
}

// DecodeBlockHeader method decodes block header from a given byte array
func (sp *shardProcessor) DecodeBlockHeader(dta []byte) data.HeaderHandler {
	if dta == nil {
		return nil
	}

	header, err := process.UnmarshalShardHeader(sp.marshalizer, dta)
	if err != nil {
		log.Debug("DecodeBlockHeader.UnmarshalShardHeader", "error", err.Error())
		return nil
	}

	return header
}



//! -------------------- NEW CODE --------------------
func (sp *shardProcessor) generateAATsForProblematicTxs(problematicMbInfo *data.MbInfo, mbHash string) *data.ProblematicMBInfo {
	accAdjTxs := make([]string, 0)
	for _, txHash := range problematicMbInfo.ProblematicTxHashes {
		_, aatHash, err := sp.generateAATForProblematicTxInMiniBlock(txHash, mbHash)
		if err != nil {
			log.Debug("***Error: CRITICAL. THIS SHOULDN'T HAPPEN (inside generateAATsForProblematicTxs***", "err", err.Error())
		}
		accAdjTxs = append(accAdjTxs, hex.EncodeToString(aatHash)) //TODO: CAMBIARE ASSOLUTAMENTE
	}

	if !(len(problematicMbInfo.ProblematicTxHashes) == len(accAdjTxs)) {
		log.Debug("***Error: Problem inside generateAATsForProblematicTxs: len(problematicTxs) != len(accAdjTxs). This shouldn't happen!!!!")
	}

	return &data.ProblematicMBInfo{ProblematicTxHashes: problematicMbInfo.ProblematicTxHashes, AccAdjTxHashes: accAdjTxs, SenderShardID: problematicMbInfo.SenderShardId } //allo stesso indice corrisponde la stessa tx (cioè l'hash della AAT di problematicsTxs[txIndex] è salvato dentro accAdjTxs[txIndex]
}


func (sp *shardProcessor) generateAATForProblematicTxInMiniBlock(txHash string, mbHash string) (*transaction.Transaction, []byte, error) {
	hash, err := hex.DecodeString(txHash)
	if err != nil {
		log.Debug("***Error: Cannot decode txHash string inside generateAATForTxInMiniBlock", "txHash", txHash, "err", err.Error())
	}
	tx, ok := sp.dataPool.Transactions().SearchFirstData(hash)
	if !ok{
		log.Debug("***Error: Cannot find tx inside Transaction Pool inside generateAATForTxInMiniBlock***", "txHash", txHash)
	}
	txHandler, ok := tx.(data.NormalTransactionHandler)
	if !ok {
		log.Debug("***Error: Cannot cast tx to data.NormalTransactionHandler type inside generateAATForTxInMiniBlock***", "txHash", txHash)
	}

	aat, aatHash, err := sp.createAccountAdjustmentTransaction(txHandler, txHash, mbHash) //TODO: RIMETTERE QUELLA BUONA
	if err != nil {
		log.Debug("***Error: createAccountAdjustmentTransaction returned an error", "err", err.Error())		
	}

	sp.logCreatedAAT(aat, string(aatHash))

	//Add aat inside TxPool
	cacheID := process.ShardCacherIdentifier(aat.GetSenderShard(), aat.GetReceiverShard())
	sp.dataPool.Transactions().AddData(aatHash, aat, aat.Size(), cacheID)

	return aat, aatHash, nil
}


func (sp *shardProcessor) createAccountAdjustmentTransaction(
	originalProblematicTx data.NormalTransactionHandler,
	originalProblematicTxHash string,
	originalMiniBlockHash string,
) (*transaction.Transaction, []byte, error) {	
	rcvAccOldShardId := sp.shardCoordinator.GetOldShardFromAddressBytes(originalProblematicTx.GetRcvAddr())
	rcvAccCurrentShardId := sp.shardCoordinator.GetCurrentShardFromAddressBytes(originalProblematicTx.GetRcvAddr())

	originalTxHashBytes, err := hex.DecodeString(originalProblematicTxHash)
	if err != nil {
		log.Debug("***Error: cannot decode originalProblematicTxHash string inside createAccountAdjustmentTransaction***",  "err", err.Error())
	}	
	originalMiniBlockHashBytes, err := hex.DecodeString(originalMiniBlockHash)
	if err != nil {
		log.Debug("***Error: cannot decode originalMiniBlockHash string inside createAccountAdjustmentTransaction***",  "err", err.Error())
	}		

	txPublicKeyBytes, _ := sp.txPublicKey.ToByteArray()
	
	log.Debug("***Checking that rcvAccOldShardId is SelfId inside createAccountAdjustmentTransasction (IMPORTANT!!!)***",  "rcvAccOldShardId == selfShardId", rcvAccOldShardId == sp.shardCoordinator.SelfId())

	//! NOTA: usando minTxGasPrice e minTxGasLimit mi da l'errore "insufficient gas price in tx" dentro commonTransactionValidation
	var txGasPrice = uint64(1000000000)
	var txGasLimit = uint64(70000000) //70000
	
	// Create the transaction object
	aat := &transaction.Transaction{
		Nonce: 			originalProblematicTx.GetNonce(),
		//MigrationNonce:	originalProblematicTx.GetMigrationNonce(),
		Value:    		originalProblematicTx.GetValue(),
		GasLimit: 		txGasLimit,  // Adjust as required
		GasPrice: 		txGasPrice,  // Adjust as required
		RcvAddr:  		originalProblematicTx.GetRcvAddr(), // ? RECEIVER
		SndAddr:  		originalProblematicTx.GetRcvAddr(), // ? RECEIVER (come per le AMT, in cui mi interessava solo il sender account, qui mi interessa solo il receiver account)
		Data:     		[]byte(""),
		ChainID:  		[]byte("localnet"),
		Version:  		uint32(2),
		SenderShard: 	rcvAccOldShardId, //? the shard from which the RECEIVER account has been migrated, that should be the current shard, from which we are sending this AAT)
		ReceiverShard: 	rcvAccCurrentShardId, //? the shard to which the RECEIVER account has been migrated, to which we have to send this AAT)
		OriginalTxHash: originalTxHashBytes,
		OriginalMiniBlockHash: originalMiniBlockHashBytes,
		//SignerPubKey: 	publicKeyBytes,
	}

	//! NOTA: CALCOLO PRIMA L'HASH DELLA SIGNATURE
	// Optional: Compute the transaction hash if needed
	aatHash, err := core.CalculateHash(sp.marshalizer, sp.hasher, aat) //? sp.marshalizer = InternalMarshalizer()
	if err != nil {
		return nil, nil, err
	}

	//! Soltanto DOPO aver calcolato l'hash, aggiungo la PubKey del signer, perché altrimenti,
	//! essendo diversa per ogni signer, mi andrà a rendere tutti gli hash diversi
	aat.SignerPubKey = txPublicKeyBytes

    // Generate signature data
	//? occhio: il marshalizer per GetDataForSigning deve essere il TxMarshalizer(), mentre quello per CalculateHash deve essere l'InternalMarshalizer
	aatSigningData, err := aat.GetDataForSigning(sp.shardCoordinator.AddressPubKeyConverter(), sp.coreComponents.TxMarshalizer(), sp.coreComponents.TxSignHasher())
	if err != nil {
		return nil, nil, errors.New("***Error: could not marshal transaction for signing (inside createAccountAdjustmentTransaction)***")
	}

	// Sign the transaction data
	sig, err := sp.txSingleSigner.Sign(sp.txPrivateKey, aatSigningData)
	if err != nil {
		return nil, nil, errors.New("***Error: could not sign the transaction (inside createAccountAdjustmentTransaction)***")
	}
	aat.Signature = sig
	
	return aat, aatHash, nil	
}

/*
func (sp *shardProcessor) createAccountAdjustmentTransaction2TODELETELATER(
	originalProblematicTx data.NormalTransactionHandler,
	originalProblematicTxHash string,
	originalMiniBlockHash string,
) (*transaction.Transaction, []byte, error) {	
	originalTxHashBytes, _ := hex.DecodeString("006a193be4124e22a4590e3ce52370d75731069a26000cbd5b7a50a9b584ff71")
	originalMiniBlockHashBytes, _ := hex.DecodeString("006a193be4124e22a4590e3ce52370d75731069a26000cbd5b7a50a9b584ff71")


	receiverAddressString := "erd1qyu5wthldzr8wx5c9ucg8kjagg0jfs53s8nr3zpz3hypefsdd8ssycr6th"
	receiverAddressBytes, err := sp.shardCoordinator.AddressPubKeyConverter().Decode(receiverAddressString)

	publicKeyBytes, _ := sp.txPublicKey.ToByteArray()

	var txGasPrice = uint64(1000000000)
	var txGasLimit = uint64(70000000) //70000

		// Create the transaction object
	aat := &transaction.Transaction{
		Nonce: 			uint64(1),
		//MigrationNonce:	uint64(0),
		Value:    		big.NewInt(0),
		GasLimit: 		txGasLimit,  // Adjust as required
		GasPrice: 		txGasPrice, // Adjust as required
		RcvAddr:  		receiverAddressBytes, // ? RECEIVER
		SndAddr:  		receiverAddressBytes, // ? RECEIVER (come per le AMT, in cui mi interessava solo il sender account, qui mi interessa solo il receiver account)
		Data:     		[]byte(""),
		ChainID:  		[]byte("localnet"),
		Version:  		uint32(2),
		SenderShard: 	uint32(1), //? the shard from which the RECEIVER account has been migrated, that should be the current shard, from which we are sending this AAT)
		ReceiverShard: 	uint32(0), //? the shard to which the RECEIVER account has been migrated, to which we have to send this AAT)
		OriginalTxHash: originalTxHashBytes,
		OriginalMiniBlockHash: originalMiniBlockHashBytes,
		//SignerPubKey: 	publicKeyBytes,
	}

	//! NOTA: CALCOLO PRIMA L'HASH DELLA SIGNATURE
	// Optional: Compute the transaction hash if needed
	aatHash, err := core.CalculateHash(sp.marshalizer, sp.hasher, aat)
	if err != nil {
		return nil, nil, err
	}

	//! Soltanto DOPO aver calcolato l'hash, aggiungo la PubKey del signer, perché altrimenti,
	//! essendo diversa per ogni signer, mi andrà a rendere tutti gli hash diversi
	aat.SignerPubKey = publicKeyBytes

    // Generate signature data
	//? occhio: il marshalizer per GetDataForSigning deve essere il TxMarshalizer(), mentre quello per CalculateHash deve essere l'InternalMarshalizer
	aatSigningData, err := aat.GetDataForSigning(sp.shardCoordinator.AddressPubKeyConverter(), sp.coreComponents.TxMarshalizer(), sp.coreComponents.TxSignHasher())
	if err != nil {
		return nil, nil, errors.New("***Error: could not marshal transaction for signing (inside createAccountAdjustmentTransaction)***")
	}

	// Sign the transaction data
	sig, err := sp.txSingleSigner.Sign(sp.txPrivateKey, aatSigningData)
	if err != nil {
		return nil, nil, errors.New("***Error: could not sign the transaction (inside createAccountAdjustmentTransaction)***")
	}
	aat.Signature = sig
	
	return aat, aatHash, nil	
}
*/

func (sp *shardProcessor) logCreatedAAT(aat *transaction.Transaction, aatHash string){
	log.Debug("***Succesfully Created AAT***", 
		"aatHash", aatHash,
		"originalProblematicTxHash", string(aat.GetOriginalTxHash()),
		"originalProblematicMiniiBlockHash", string(aat.GetOriginalMiniBlockHash()),
		"nonce", aat.GetNonce(),
		"value", aat.GetValue(),
		"rcvAddr", aat.GetRcvAddr(),
		"sndAddr", aat.GetSndAddr(),
		"gasPrice", aat.GetGasPrice(),
		"gasLimit", aat.GetGasLimit(),
		"data", aat.GetData(),
		"migrationNonce", aat.GetMigrationNonce(),
		"senderShard", aat.GetSenderShard(),
		"receiverShard", aat.GetReceiverShard(),
	)
}


func (sp *shardProcessor) cleanProblematicMiniBlocksForCurrentRound() {
	sp.problematicMiniBlocksForCurrentRound = make(map[string]*data.ProblematicMBInfo) //? lo ri-inizializzo ad un empy map di quel tipo, visto che il current round è finito
	log.Debug("***sp.problematicMiniBlocksForCurrentRound cleaned****", "sp.problematicMiniBlocksForCurrentRound", sp.problematicMiniBlocksForCurrentRound)
}

func (sp *shardProcessor) cleanReadyMbsInsertedInCurrentRoundInShardProcessor()() {
	sp.readyMbsInsertedInCurrentBlock = make(map[string]bool) //? lo ri-inizializzo ad un empy map di quel tipo, visto che il current round è finito
	log.Debug("***sp.readyMbsInsertedInCurrentBlock cleaned****", "sp.readyMbsInsertedInCurrentBlock", sp.readyMbsInsertedInCurrentBlock)
}

func (sp *shardProcessor) updateWaitingMbsForAATsNotarization() {
	if len(sp.problematicMiniBlocksForCurrentRound) > 0 {
		log.Debug("***problematicMiniBlocksForCurrentRound is NOT empty: updating waitingMBsForAATsNotarization***")
		waitingMbsForAATsNotarization := sp.shardCoordinator.UpdateWaitingMbsForAATsNotarization(sp.problematicMiniBlocksForCurrentRound)
		log.Debug("***Printing updated waitingMBsForAATsNotarization***", "waitingMBsForAATsNotarization", waitingMbsForAATsNotarization)
	}
}

func (sp *shardProcessor) removeReadyMbsInsertedInCurrentRoundFromWaitingMbs() {
	if len(sp.readyMbsInsertedInCurrentBlock) > 0 {
		log.Debug("***readyMbsInsertedInCurrentBlock is NOT empty: updating waitingMBsForAATsNotarization***")
		waitingMbsForAATsNotarization := sp.shardCoordinator.RemoveReadyMbsInsertedInCurrentRoundFromWaitingMbs(sp.readyMbsInsertedInCurrentBlock)
		log.Debug("***Printing updated waitingMBsForAATsNotarization***", "waitingMBsForAATsNotarization", waitingMbsForAATsNotarization)
	}
}
//! ---------------- END OF NEW CODE -----------------