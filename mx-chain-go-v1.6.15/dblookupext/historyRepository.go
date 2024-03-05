//go:generate protoc -I=. -I=$GOPATH/src -I=$GOPATH/src/github.com/multiversx/protobuf/protobuf  --gogoslick_out=. miniblockMetadata.proto

package dblookupext

import (
	"fmt"
	"sync"
	//! -------------------- NEW CODE --------------------
	"sort"	
	//! ---------------- END OF NEW CODE -----------------	

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/core/check"
	"github.com/multiversx/mx-chain-core-go/core/container"
	"github.com/multiversx/mx-chain-core-go/data"
	"github.com/multiversx/mx-chain-core-go/data/block"
	"github.com/multiversx/mx-chain-core-go/data/typeConverters"
	"github.com/multiversx/mx-chain-core-go/hashing"
	"github.com/multiversx/mx-chain-core-go/marshal"
	"github.com/multiversx/mx-chain-go/common/logging"
	"github.com/multiversx/mx-chain-go/dblookupext/esdtSupply"
	"github.com/multiversx/mx-chain-go/process"
	"github.com/multiversx/mx-chain-go/storage"
	"github.com/multiversx/mx-chain-go/storage/cache"
	logger "github.com/multiversx/mx-chain-logger-go"

	//! -------------------- NEW CODE --------------------
	"github.com/multiversx/mx-chain-core-go/data/transaction"
	"github.com/multiversx/mx-chain-go/dataRetriever"
	"github.com/multiversx/mx-chain-go/sharding"
	"github.com/multiversx/mx-chain-go/state"
	"github.com/multiversx/mx-chain-go/storage/txcache"
	//! ---------------- END OF NEW CODE -----------------	
)

var log = logger.GetOrCreate("dblookupext")

const sizeOfDeduplicationCache = 1000

// HistoryRepositoryArguments is a structure that stores all components that are needed to a history processor
type HistoryRepositoryArguments struct {
	SelfShardID                 uint32
	MiniblocksMetadataStorer    storage.Storer
	MiniblockHashByTxHashStorer storage.Storer
	BlockHashByRound            storage.Storer
	Uint64ByteSliceConverter    typeConverters.Uint64ByteSliceConverter
	EpochByHashStorer           storage.Storer
	EventsHashesByTxHashStorer  storage.Storer
	Marshalizer                 marshal.Marshalizer
	Hasher                      hashing.Hasher
	ESDTSuppliesHandler         SuppliesHandler
	//! -------------------- NEW CODE --------------------
	TransactionStorer 			storage.Storer
	MiniBlockStorer 			storage.Storer
	ShardedTxPool 				dataRetriever.ShardedTxPool
	AccountsAdapter 			state.AccountsAdapter
	ShardCoordinator			sharding.Coordinator
	//! ---------------- END OF NEW CODE -----------------	
}

type historyRepository struct {
	selfShardID                uint32
	miniblocksMetadataStorer   storage.Storer
	miniblockHashByTxHashIndex storage.Storer
	blockHashByRound           storage.Storer
	uint64ByteSliceConverter   typeConverters.Uint64ByteSliceConverter
	epochByHashIndex           *epochByHashIndex
	eventsHashesByTxHashIndex  *eventsHashesByTxHash
	marshalizer                marshal.Marshalizer
	hasher                     hashing.Hasher
	esdtSuppliesHandler        SuppliesHandler

	// These maps temporarily hold notifications of "notarized at source or destination", to deal with unwanted concurrency effects
	// The unwanted concurrency effects could be accentuated by the fast db-replay-validate mechanism.
	pendingNotarizedAtSourceNotifications      *container.MutexMap
	pendingNotarizedAtDestinationNotifications *container.MutexMap
	pendingNotarizedAtBothNotifications        *container.MutexMap

	// This cache will hold hashes of already inserted miniblock metadata records, so that we avoid repeated "put()" operations,
	// that could mistakenly override the "patch()" operations performed when consuming notarization notifications.
	deduplicationCacheForInsertMiniblockMetadata storage.Cacher

	recordBlockMutex                 sync.Mutex
	consumePendingNotificationsMutex sync.Mutex

	//! -------------------- NEW CODE --------------------
	transactionStorer          storage.Storer
	miniBlockStorer            storage.Storer
	queuingTxsSenderHandler    process.TxsSenderHandler
	shardedTxPool			   dataRetriever.ShardedTxPool
	accountsAdapter 		   state.AccountsAdapter
	shardCoordinator 		   sharding.Coordinator
	//! ---------------- END OF NEW CODE -----------------	
}

type notarizedNotification struct {
	metaNonce uint64
	metaHash  []byte
}

// NewHistoryRepository will create a new instance of HistoryRepository
func NewHistoryRepository(arguments HistoryRepositoryArguments) (*historyRepository, error) {
	if check.IfNil(arguments.MiniblocksMetadataStorer) {
		return nil, core.ErrNilStore
	}
	if check.IfNil(arguments.MiniblockHashByTxHashStorer) {
		return nil, core.ErrNilStore
	}
	if check.IfNil(arguments.EpochByHashStorer) {
		return nil, core.ErrNilStore
	}
	if check.IfNil(arguments.Marshalizer) {
		return nil, core.ErrNilMarshalizer
	}
	if check.IfNil(arguments.Hasher) {
		return nil, core.ErrNilHasher
	}
	if check.IfNil(arguments.EventsHashesByTxHashStorer) {
		return nil, core.ErrNilStore
	}
	if check.IfNil(arguments.ESDTSuppliesHandler) {
		return nil, errNilESDTSuppliesHandler
	}
	if check.IfNil(arguments.Uint64ByteSliceConverter) {
		return nil, process.ErrNilUint64Converter
	}
	//! -------------------- NEW CODE --------------------
	if check.IfNil(arguments.TransactionStorer) {
		return nil, process.ErrNilTransactionStorer
	}	
	if check.IfNil(arguments.MiniBlockStorer) {
		return nil, process.ErrNilMiniBlockStorer
	}		
	if check.IfNil(arguments.ShardedTxPool) {
		return nil, process.ErrNilShardedTxPool
	}		
	if check.IfNil(arguments.AccountsAdapter) {
		return nil, process.ErrNilAccountsAdapter
	}
	if check.IfNil(arguments.MiniBlockStorer) {
		return nil, process.ErrNilShardCoordinator
	}	
	//! ---------------- END OF NEW CODE -----------------	
					

	hashToEpochIndex := newHashToEpochIndex(arguments.EpochByHashStorer, arguments.Marshalizer)
	deduplicationCacheForInsertMiniblockMetadata, _ := cache.NewLRUCache(sizeOfDeduplicationCache)

	eventsHashesToTxHashIndex := newEventsHashesByTxHash(arguments.EventsHashesByTxHashStorer, arguments.Marshalizer)

	return &historyRepository{
		selfShardID:                           arguments.SelfShardID,
		miniblocksMetadataStorer:              arguments.MiniblocksMetadataStorer,
		blockHashByRound:                      arguments.BlockHashByRound,
		marshalizer:                           arguments.Marshalizer,
		hasher:                                arguments.Hasher,
		epochByHashIndex:                      hashToEpochIndex,
		miniblockHashByTxHashIndex:            arguments.MiniblockHashByTxHashStorer,
		pendingNotarizedAtSourceNotifications: container.NewMutexMap(),
		pendingNotarizedAtDestinationNotifications:   container.NewMutexMap(),
		pendingNotarizedAtBothNotifications:          container.NewMutexMap(),
		deduplicationCacheForInsertMiniblockMetadata: deduplicationCacheForInsertMiniblockMetadata,
		eventsHashesByTxHashIndex:                    eventsHashesToTxHashIndex,
		esdtSuppliesHandler:                          arguments.ESDTSuppliesHandler,
		uint64ByteSliceConverter:                     arguments.Uint64ByteSliceConverter,
		//! -------------------- NEW CODE --------------------
		transactionStorer:         			   arguments.TransactionStorer,
		miniBlockStorer:          			   arguments.MiniBlockStorer,
		shardedTxPool: 						   arguments.ShardedTxPool,
		accountsAdapter: 					   arguments.AccountsAdapter,
		shardCoordinator: 					   arguments.ShardCoordinator,
		//! ---------------- END OF NEW CODE -----------------		
	}, nil
}

// RecordBlock records a block
// This function is not called on a goroutine, but synchronously instead, right after committing a block
func (hr *historyRepository) RecordBlock(blockHeaderHash []byte,
	blockHeader data.HeaderHandler,
	blockBody data.BodyHandler,
	scrResultsFromPool map[string]data.TransactionHandler,
	receiptsFromPool map[string]data.TransactionHandler,
	createdIntraShardMiniBlocks []*block.MiniBlock,
	logs []*data.LogData) error {
	hr.recordBlockMutex.Lock()
	defer hr.recordBlockMutex.Unlock()

	log.Debug("RecordBlock()", "nonce", blockHeader.GetNonce(), "blockHeaderHash", blockHeaderHash, "header type", fmt.Sprintf("%T", blockHeader))

	body, ok := blockBody.(*block.Body)
	if !ok {
		return errCannotCastToBlockBody
	}

	epoch := blockHeader.GetEpoch()

	err := hr.epochByHashIndex.saveEpochByHash(blockHeaderHash, epoch)
	if err != nil {
		return newErrCannotSaveEpochByHash("block header", blockHeaderHash, err)
	}

	for _, miniblock := range body.MiniBlocks {
		if miniblock.Type == block.PeerBlock {
			continue
		}

		err = hr.recordMiniblock(blockHeaderHash, blockHeader, miniblock, epoch)
		if err != nil {
			logging.LogErrAsErrorExceptAsDebugIfClosingError(log, err, "cannot record miniblock",
				"type", miniblock.Type, "error", err)
			continue
		}
	}

	for _, miniBlock := range createdIntraShardMiniBlocks {
		err = hr.recordMiniblock(blockHeaderHash, blockHeader, miniBlock, epoch)
		if err != nil {
			logging.LogErrAsErrorExceptAsDebugIfClosingError(log, err, "cannot record in shard miniblock",
				"type", miniBlock.Type, "error", err)
		}
	}

	err = hr.eventsHashesByTxHashIndex.saveResultsHashes(epoch, scrResultsFromPool, receiptsFromPool)
	if err != nil {
		return err
	}

	err = hr.esdtSuppliesHandler.ProcessLogs(blockHeader.GetNonce(), logs)
	if err != nil {
		return err
	}

	err = hr.putHashByRound(blockHeaderHash, blockHeader)
	if err != nil {
		return err
	}

	return nil
}

func (hr *historyRepository) putHashByRound(blockHeaderHash []byte, header data.HeaderHandler) error {
	roundToByteSlice := hr.uint64ByteSliceConverter.ToByteSlice(header.GetRound())
	return hr.blockHashByRound.Put(roundToByteSlice, blockHeaderHash)
}

func (hr *historyRepository) recordMiniblock(blockHeaderHash []byte, blockHeader data.HeaderHandler, miniblock *block.MiniBlock, epoch uint32) error {
	miniblockHash, err := hr.computeMiniblockHash(miniblock)
	if err != nil {
		return err
	}

	if hr.hasRecentlyInsertedMiniblockMetadata(miniblockHash, epoch) {
		return nil
	}

	err = hr.epochByHashIndex.saveEpochByHash(miniblockHash, epoch)
	if err != nil {
		return newErrCannotSaveEpochByHash("miniblock", miniblockHash, err)
	}

	miniblockMetadata := &MiniblockMetadata{
		Type:               int32(miniblock.Type),
		Epoch:              epoch,
		HeaderHash:         blockHeaderHash,
		MiniblockHash:      miniblockHash,
		Round:              blockHeader.GetRound(),
		HeaderNonce:        blockHeader.GetNonce(),
		SourceShardID:      miniblock.GetSenderShardID(),
		DestinationShardID: miniblock.GetReceiverShardID(),
	}

	err = hr.putMiniblockMetadata(miniblockHash, miniblockMetadata)
	if err != nil {
		return err
	}

	hr.markMiniblockMetadataAsRecentlyInserted(miniblockHash, epoch)

	for _, txHash := range miniblock.TxHashes {
		errPut := hr.miniblockHashByTxHashIndex.Put(txHash, miniblockHash)
		if errPut != nil {
			logging.LogErrAsWarnExceptAsDebugIfClosingError(log, errPut, "miniblockHashByTxHashIndex.Put()",
				"txHash", txHash, "err", errPut)
			continue
		}
	}

	return nil
}

func (hr *historyRepository) computeMiniblockHash(miniblock *block.MiniBlock) ([]byte, error) {
	return core.CalculateHash(hr.marshalizer, hr.hasher, miniblock)
}

func (hr *historyRepository) hasRecentlyInsertedMiniblockMetadata(miniblockHash []byte, epoch uint32) bool {
	key := hr.buildKeyOfDeduplicationCacheForInsertMiniblockMetadata(miniblockHash, epoch)
	return hr.deduplicationCacheForInsertMiniblockMetadata.Has(key)
}

// When building the key for the deduplication cache, we must take into account the epoch as well, in order to handle this case:
// - miniblock M added in a fork at the end of epoch E,
// - miniblock M re-added, on the canonical chain this time, in the next epoch E + 1.
// This way we do not mistakenly ignore to update the "epochByHashIndex".
func (hr *historyRepository) buildKeyOfDeduplicationCacheForInsertMiniblockMetadata(miniblockHash []byte, epoch uint32) []byte {
	return []byte(fmt.Sprintf("%d_%x", epoch, miniblockHash))
}

func (hr *historyRepository) markMiniblockMetadataAsRecentlyInserted(miniblockHash []byte, epoch uint32) {
	key := hr.buildKeyOfDeduplicationCacheForInsertMiniblockMetadata(miniblockHash, epoch)
	_ = hr.deduplicationCacheForInsertMiniblockMetadata.Put(key, nil, 0)
}

// GetMiniblockMetadataByTxHash will return a history transaction for the given hash from storage
func (hr *historyRepository) GetMiniblockMetadataByTxHash(hash []byte) (*MiniblockMetadata, error) {
	miniblockHash, err := hr.miniblockHashByTxHashIndex.Get(hash)
	if err != nil {
		return nil, err
	}

	return hr.getMiniblockMetadataByMiniblockHash(miniblockHash)
}

func (hr *historyRepository) putMiniblockMetadata(hash []byte, metadata *MiniblockMetadata) error {
	metadataBytes, err := hr.marshalizer.Marshal(metadata)
	if err != nil {
		return err
	}

	err = hr.miniblocksMetadataStorer.PutInEpoch(hash, metadataBytes, metadata.Epoch)
	if err != nil {
		return newErrCannotSaveMiniblockMetadata(hash, err)
	}

	return nil
}

func (hr *historyRepository) getMiniblockMetadataByMiniblockHash(hash []byte) (*MiniblockMetadata, error) {
	epoch, err := hr.epochByHashIndex.getEpochByHash(hash)
	if err != nil {
		return nil, err
	}

	metadataBytes, err := hr.miniblocksMetadataStorer.GetFromEpoch(hash, epoch)
	if err != nil {
		return nil, err
	}

	metadata := &MiniblockMetadata{}
	err = hr.marshalizer.Unmarshal(metadata, metadataBytes)
	if err != nil {
		return nil, err
	}

	return metadata, nil
}

// GetEpochByHash will return epoch for a given hash
// This works for Blocks, Miniblocks
// It doesn't work for transactions (not needed, there we have a static storer for "miniblockHashByTxHashIndex" as well)!
func (hr *historyRepository) GetEpochByHash(hash []byte) (uint32, error) {
	return hr.epochByHashIndex.getEpochByHash(hash)
}

// OnNotarizedBlocks notifies the history repository about notarized blocks
func (hr *historyRepository) OnNotarizedBlocks(shardID uint32, headers []data.HeaderHandler, headersHashes [][]byte) {
	for i, headerHandler := range headers {
		headerHash := headersHashes[i]

		log.Trace("onNotarizedBlocks():", "shardID", shardID, "nonce", headerHandler.GetNonce(), "headerHash", headerHash, "type", fmt.Sprintf("%T", headerHandler))

		metaBlock, isMetaBlock := headerHandler.(*block.MetaBlock)
		if isMetaBlock {
			for _, miniBlock := range metaBlock.MiniBlockHeaders {
				hr.onNotarizedMiniblock(headerHandler.GetNonce(), headerHash, headerHandler.GetShardID(), miniBlock)
			}

			for _, shardData := range metaBlock.ShardInfo {
				shardDataCopy := shardData
				hr.onNotarizedInMetaBlock(headerHandler.GetNonce(), headerHash, &shardDataCopy)
			}
		} else {
			log.Error("onNotarizedBlocks(): unexpected type of header", "type", fmt.Sprintf("%T", headerHandler))
		}
	}

	hr.consumePendingNotificationsWithLock()
}

func (hr *historyRepository) onNotarizedInMetaBlock(metaBlockNonce uint64, metaBlockHash []byte, shardData *block.ShardData) {
	if metaBlockNonce < 1 {
		return
	}

	for _, miniblockHeader := range shardData.GetShardMiniBlockHeaders() {
		hr.onNotarizedMiniblock(metaBlockNonce, metaBlockHash, shardData.GetShardID(), miniblockHeader)
	}
}

func (hr *historyRepository) onNotarizedMiniblock(metaBlockNonce uint64, metaBlockHash []byte, shardOfContainingBlock uint32, miniblockHeader block.MiniBlockHeader) {
	miniblockHash := miniblockHeader.Hash
	isIntra := miniblockHeader.SenderShardID == miniblockHeader.ReceiverShardID
	isToMeta := miniblockHeader.ReceiverShardID == core.MetachainShardId
	isNotarizedAtSource := miniblockHeader.SenderShardID == shardOfContainingBlock
	isNotarizedAtDestination := miniblockHeader.ReceiverShardID == shardOfContainingBlock
	isNotarizedAtBoth := isIntra || isToMeta

	notFromMe := miniblockHeader.SenderShardID != hr.selfShardID
	notToMe := miniblockHeader.ReceiverShardID != hr.selfShardID
	isPeerMiniblock := miniblockHeader.Type == block.PeerBlock
	iDontCare := (notFromMe && notToMe) || isPeerMiniblock
	if iDontCare {
		return
	}

	log.Trace("onNotarizedMiniblock()",
		"metaBlockNonce", metaBlockNonce,
		"metaBlockHash", metaBlockHash,
		"shardOfContainingBlock", shardOfContainingBlock,
		"miniblock", miniblockHash,
		"direction", fmt.Sprintf("[%d -> %d]", miniblockHeader.SenderShardID, miniblockHeader.ReceiverShardID),
	)
	//! -------------------- NEW CODE --------------------
	log.Debug("***onNotarizedMiniblock()***",
		"metaBlockNonce", metaBlockNonce,
		"metaBlockHash", metaBlockHash,
		"shardOfContainingBlock", shardOfContainingBlock,
		"miniblock", miniblockHash,
		"isNotarizedAtBoth", isNotarizedAtBoth,
		"isNotarizedAtSource", isNotarizedAtSource,
		"isNotarizedAtDestination", isNotarizedAtDestination,
		"direction", fmt.Sprintf("[%d -> %d]", miniblockHeader.SenderShardID, miniblockHeader.ReceiverShardID),
	)
	//! ---------------- END OF NEW CODE -----------------	

	if isNotarizedAtBoth {
		hr.pendingNotarizedAtBothNotifications.Set(string(miniblockHash), &notarizedNotification{
			metaNonce: metaBlockNonce,
			metaHash:  metaBlockHash,
		})
	} else if isNotarizedAtSource {
		hr.pendingNotarizedAtSourceNotifications.Set(string(miniblockHash), &notarizedNotification{
			metaNonce: metaBlockNonce,
			metaHash:  metaBlockHash,
		})
	} else if isNotarizedAtDestination {
		hr.pendingNotarizedAtDestinationNotifications.Set(string(miniblockHash), &notarizedNotification{
			metaNonce: metaBlockNonce,
			metaHash:  metaBlockHash,
		})
		//! -------------------- NEW CODE --------------------
		//TODO: il nome va bene? Il comportamento in base allo shard sorgente/destinazione è diverso!!
		sendTransactionsAndRemoveAccountForFinalAMTsInsideMiniBlock(hr, miniblockHash)
		//! ---------------- END OF NEW CODE -----------------		
	} else {
		log.Error("onNotarizedMiniblock(): unexpected condition, notification not understood")
	}
}


//! -------------------- NEW CODE --------------------
func sendTransactionsAndRemoveAccountForFinalAMTsInsideMiniBlock(hr *historyRepository, miniblockHash []byte){
	mbStorer := hr.miniBlockStorer
	buff, err := mbStorer.Get(miniblockHash) //TODO: posso usare anche GetFromEpoch, che magari è più veloce (credo che l'epoca dovrei saperla dall'header del blocco, VEDI!)
	if err != nil {
		log.Debug("***Cannot get miniblock from storer inside historyRepository***", "mbHash", string(miniblockHash))
	}

	mb := &block.MiniBlock{}
	err = hr.marshalizer.Unmarshal(mb, buff)
	if err != nil {
		log.Debug("***Cannot unmarshal miniblock inside historyRepository***")
	}

	log.Debug("***Miniblock type inside historyRepository***", "mb.GetType()", mb.GetType(), "isTxBlockType", mb.GetType() == block.TxBlock)

	mbTxHashes := mb.TxHashes
	for _, txHash := range mbTxHashes {
		log.Debug("***Tx inside miniblock inside historyRepository***", "txHash", string(txHash))
		txStorer := hr.transactionStorer
		if err != nil {
			log.Debug("***Error while retrieving the storer for transactions inside historyRepository***")
		}

		buff, err := txStorer.Get(txHash) //TODO: posso usare anche GetFromEpoch, che magari è più veloce (credo che l'epoca dovrei saperla dall'header del blocco, VEDI!)
		if err != nil {
			log.Debug("***Cannot get transaction from storer inside historyRepository***", "txHash", string(txHash))
		}

		tx := &transaction.Transaction{}
		err = hr.marshalizer.Unmarshal(tx, buff)
		if err != nil {
			log.Debug("***Cannot unmarshal transaction inside historyRepository***")
		}

		// ! AAT CHECK OK 
		isAccountMigrationTransaction := len(tx.SignerPubKey) > 0 && !(len(tx.OriginalMiniBlockHash) > 0 && len(tx.OriginalMiniBlockHash) > 0)
		isAccountAdjustmentTransaction := len(tx.SignerPubKey) > 0 && (len(tx.OriginalMiniBlockHash) > 0 && len(tx.OriginalMiniBlockHash) > 0)
		if isAccountMigrationTransaction {
			log.Debug("***---------ACCOUNT MIGRATION TRANSACTION notarized dest-side inside historyRepository------------***", "txHash", txHash)			
		}else if isAccountAdjustmentTransaction {
			log.Debug("***---------ACCOUNT ADJUSTMENT TRANSACTION notarized dest-side inside historyRepository------------***", "txHash", txHash)			
		}

	
		if(isAccountMigrationTransaction){
			//? NOTA: ci sono solo i due casi in cui o io faccio parte del sender shard, oppure del destination shard,
			//? perché altrimenti non sarei arrivata fino a qui, dal momento che già la funzione OnNotarizedMiniBlock
			//? si occupa di controllare se il miniblocco mi interessa (e questo è il caso solo se il miniblocco è
			//? "from me" oppure "to me", quindi se faccio parte o del sender o del destination shard)

			if(hr.selfShardID == tx.SenderShard){ //Se faccio parte del sender shard

				log.Debug("*** ----- ACCOUNT MIGRATION TRANSACTION FROM ME NOTARIZED ON DESTINATION ----- ****", "txHash", string(txHash), "migratedAccount", string(tx.SndAddr), "migrationNonce", tx.MigrationNonce)


				//prendo tutte le txs di questo sender dalla cache
				txsForSender := fetchTxsForSender(hr, string(tx.SndAddr), tx.SenderShard)

				if len(txsForSender) > 0 {
					log.Debug("*** Sending queuing transaction of migrated account ...***", "migratedAccount", string(tx.SndAddr), "amtxHash", string(txHash))
					logTxsToBeSent(txsForSender)
					
					numTxs, err := hr.queuingTxsSenderHandler.SendBulkTransactions(txsForSender)
					if err != nil{
						log.Debug("*** Failed to send transactions through SendBulkTransactions ***", "err", err.Error())
					}
					log.Debug("*** Transactions sent through SendBulkTransactions ***", "numTxs", numTxs)					
				}else{
					log.Debug("*** No queuing transaction for migrated account. NOT calling SendBulkTransactions(queuingTxs)***", "migratedAccount", string(tx.SndAddr))
				}
				

				
				//TODO: capire se SendBulkTranasction è bloccante e, se sì, eliminare l'account tramite hr.accountsAdapter.RemoveAccount o quello che sia
				log.Debug("*** Removing migrated account from source shard ***")
				//hr.accountsAdapter.RemoveAccount(tx.SndAddr) //TODO: RIMETTERE
			
				
			}else if (hr.selfShardID == tx.ReceiverShard){ //Altrimenti, se faccio parte del destination shard (andava bene anche solo else, visto che ci sono solo due casi possibili, ma per sicurezza ho messo else if)
				hr.shardedTxPool.RemoveAccountFromMigratingAccounts(string(tx.SndAddr))
				//TODO: set IsBeingMigrated to false (?)

				log.Debug("*** Migrating Accounts map is now: ***", "migratingAccounts", hr.shardedTxPool.GetMigratingAccounts())


				//is migrated account actually in my shard??? It should be. Here is the check:
				accountHandler, err := hr.accountsAdapter.LoadAccount(tx.SndAddr)
				if err != nil {
					log.Debug("*** Migrated account has not been created in the destination shard r***")
				}
	
				//faccio il cast a UserAccount così ho disponibili i metodi per il flag IsBeingMigrated
				userAccountHandler, ok := accountHandler.(state.UserAccountHandler)
				if !ok{
					log.Debug("***Cannot cast accountHandler inside historyRepository")
				}

				log.Debug("*** Migrated account state inside destination shard: ***",
					 		"nonce", userAccountHandler.GetNonce(), 
							"balance", userAccountHandler.GetBalance().String(),
							"migrationNonce", userAccountHandler.GetMigrationNonce(),
							"username", userAccountHandler.GetUserName(),
							"rootHash", string(userAccountHandler.GetRootHash()),
				)

				log.Debug("*** AccountsMapping is now: ***", "accountsMapping", hr.shardCoordinator.AccountsMapping(), "len", len(hr.shardCoordinator.AccountsShardInfo()))

			}
		}
	}
}



func fetchTxsForSender(hr *historyRepository, sender string, senderShard uint32) []*transaction.Transaction {
	//? NOTA: codice preso e ispirato da "fetchTxsForSender" dell'apiTransactionProcessor nel file apiTransactionProcessor.go
	cacheId := process.ShardCacherIdentifier(senderShard, senderShard)
	cache := hr.shardedTxPool.ShardDataStore(cacheId)
	txCache, ok := cache.(*txcache.TxCache)
	if !ok {
		log.Warn("fetchTxsForSender could not cast to TxCache")
		return nil
	}

	wrappedTxsForSender := txCache.GetTransactionsPoolForSender(sender)

	sort.Slice(wrappedTxsForSender, func(i, j int) bool {
		return wrappedTxsForSender[i].Tx.GetNonce() < wrappedTxsForSender[j].Tx.GetNonce()
	})
	
	txsForSender := convertWrappedTxsListToTransactionList(wrappedTxsForSender)

	return txsForSender
}



func convertWrappedTxsListToTransactionList(wrappedTxsList []*txcache.WrappedTransaction) []*transaction.Transaction {
	var txsList []*transaction.Transaction

	for _, wtx := range wrappedTxsList {
		if tx, ok := wtx.Tx.(*transaction.Transaction); ok {
			txsList = append(txsList, tx)
		}else{
			log.Debug("***Cannot cast transaction inside convertWrappedTxsListToTransactionList of historyRepository***", "txHash", string(wtx.TxHash))
		}
	}

	return txsList
}

func logTxsToBeSent(txsList []*transaction.Transaction) {
	var txsStringList []string

	for _, tx := range txsList {
		txsStringList = append(txsStringList, tx.String())
	}

	log.Debug("***Transactions to be sent***", "txsStringList", txsStringList)
}


//! ---------------- END OF NEW CODE -----------------


// Notifications are consumed within a critical section so that we don't have competing put() operations for the same miniblock metadata,
// which could have resulted in mistakenly overriding the "notarization (hyperblock) coordinates".
func (hr *historyRepository) consumePendingNotificationsWithLock() {
	hr.consumePendingNotificationsMutex.Lock()
	defer hr.consumePendingNotificationsMutex.Unlock()

	if hr.pendingNotarizedAtSourceNotifications.Len() == 0 &&
		hr.pendingNotarizedAtDestinationNotifications.Len() == 0 &&
		hr.pendingNotarizedAtBothNotifications.Len() == 0 {
		return
	}

	log.Trace("consumePendingNotificationsWithLock() begin",
		"len(source)", hr.pendingNotarizedAtSourceNotifications.Len(),
		"len(destination)", hr.pendingNotarizedAtDestinationNotifications.Len(),
		"len(both)", hr.pendingNotarizedAtBothNotifications.Len(),
	)

	hr.consumePendingNotificationsNoLock(hr.pendingNotarizedAtSourceNotifications, func(metadata *MiniblockMetadata, notification *notarizedNotification) {
		metadata.NotarizedAtSourceInMetaNonce = notification.metaNonce
		metadata.NotarizedAtSourceInMetaHash = notification.metaHash
	})

	hr.consumePendingNotificationsNoLock(hr.pendingNotarizedAtDestinationNotifications, func(metadata *MiniblockMetadata, notification *notarizedNotification) {
		metadata.NotarizedAtDestinationInMetaNonce = notification.metaNonce
		metadata.NotarizedAtDestinationInMetaHash = notification.metaHash
	})

	hr.consumePendingNotificationsNoLock(hr.pendingNotarizedAtBothNotifications, func(metadata *MiniblockMetadata, notification *notarizedNotification) {
		metadata.NotarizedAtSourceInMetaNonce = notification.metaNonce
		metadata.NotarizedAtSourceInMetaHash = notification.metaHash
		metadata.NotarizedAtDestinationInMetaNonce = notification.metaNonce
		metadata.NotarizedAtDestinationInMetaHash = notification.metaHash
	})

	log.Trace("consumePendingNotificationsWithLock() end",
		"len(source)", hr.pendingNotarizedAtSourceNotifications.Len(),
		"len(destination)", hr.pendingNotarizedAtDestinationNotifications.Len(),
		"len(both)", hr.pendingNotarizedAtBothNotifications.Len(),
	)
}

func (hr *historyRepository) consumePendingNotificationsNoLock(pendingMap *container.MutexMap, patchMetadataFunc func(*MiniblockMetadata, *notarizedNotification)) {
	for _, key := range pendingMap.Keys() {
		notification, ok := pendingMap.Get(key)
		if !ok {
			continue
		}

		keyTyped, ok := key.(string)
		if !ok {
			log.Error("consumePendingNotificationsNoLock(): bad key", "key", key)
			continue
		}

		notificationTyped, ok := notification.(*notarizedNotification)
		if !ok {
			log.Error("consumePendingNotificationsNoLock(): bad value", "value", fmt.Sprintf("%T", notification))
			continue
		}

		miniblockHash := []byte(keyTyped)
		metadata, err := hr.getMiniblockMetadataByMiniblockHash(miniblockHash)
		if err != nil {
			// Maybe not yet committed / saved in storer
			continue
		}

		patchMetadataFunc(metadata, notificationTyped)
		err = hr.putMiniblockMetadata(miniblockHash, metadata)
		if err != nil {
			logging.LogErrAsErrorExceptAsDebugIfClosingError(log, err, "consumePendingNotificationsNoLock(): cannot put miniblock metadata",
				"miniblockHash", miniblockHash, "err", err)
			continue
		}

		pendingMap.Remove(key)
	}
}

// GetResultsHashesByTxHash will return results hashes by transaction hash
func (hr *historyRepository) GetResultsHashesByTxHash(txHash []byte, epoch uint32) (*ResultsHashesByTxHash, error) {
	return hr.eventsHashesByTxHashIndex.getEventsHashesByTxHash(txHash, epoch)
}

// IsEnabled will always return true
func (hr *historyRepository) IsEnabled() bool {
	return true
}

// RevertBlock will return the modification for the current block header
func (hr *historyRepository) RevertBlock(blockHeader data.HeaderHandler, blockBody data.BodyHandler) error {
	return hr.esdtSuppliesHandler.RevertChanges(blockHeader, blockBody)
}

// GetESDTSupply will return the supply from the storage for the given token
func (hr *historyRepository) GetESDTSupply(token string) (*esdtSupply.SupplyESDT, error) {
	return hr.esdtSuppliesHandler.GetESDTSupply(token)
}

// IsInterfaceNil returns true if there is no value under the interface
func (hr *historyRepository) IsInterfaceNil() bool {
	return hr == nil
}

//! -------------------- NEW CODE --------------------
func (hr *historyRepository) SetQueuingTxsSenderHandler(queuingTxsSenderHandler process.TxsSenderHandler) {
	hr.queuingTxsSenderHandler = queuingTxsSenderHandler
}
//! ---------------- END OF NEW CODE -----------------