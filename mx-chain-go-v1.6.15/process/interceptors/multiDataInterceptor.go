package interceptors

import (
	"sync"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/core/check"
	"github.com/multiversx/mx-chain-core-go/data/batch"
	"github.com/multiversx/mx-chain-core-go/marshal"
	"github.com/multiversx/mx-chain-go/common"
	"github.com/multiversx/mx-chain-go/debug/handler"
	"github.com/multiversx/mx-chain-go/p2p"
	"github.com/multiversx/mx-chain-go/process"
	"github.com/multiversx/mx-chain-go/process/interceptors/disabled"
	logger "github.com/multiversx/mx-chain-logger-go"
	//! -------------------- NEW CODE --------------------
	"github.com/multiversx/mx-chain-go/dataRetriever"
	//! ---------------- END OF NEW CODE -----------------
)

var log = logger.GetOrCreate("process/interceptors")

// ArgMultiDataInterceptor is the argument for the multi-data interceptor
type ArgMultiDataInterceptor struct {
	Topic                string
	Marshalizer          marshal.Marshalizer
	DataFactory          process.InterceptedDataFactory
	Processor            process.InterceptorProcessor
	Throttler            process.InterceptorThrottler
	AntifloodHandler     process.P2PAntifloodHandler
	WhiteListRequest     process.WhiteListHandler
	PreferredPeersHolder process.PreferredPeersHolderHandler
	CurrentPeerId        core.PeerID
	//! -------------------- NEW CODE --------------------
	DataPool 			 dataRetriever.PoolsHolder
	//! ---------------- END OF NEW CODE -----------------
}

// MultiDataInterceptor is used for intercepting packed multi data
type MultiDataInterceptor struct {
	*baseDataInterceptor
	marshalizer        marshal.Marshalizer
	factory            process.InterceptedDataFactory
	whiteListRequest   process.WhiteListHandler
	mutChunksProcessor sync.RWMutex
	chunksProcessor    process.InterceptedChunksProcessor
	//! -------------------- NEW CODE --------------------
	dataPool 			 dataRetriever.PoolsHolder
	//! ---------------- END OF NEW CODE -----------------
}

// NewMultiDataInterceptor hooks a new interceptor for packed multi data
func NewMultiDataInterceptor(arg ArgMultiDataInterceptor) (*MultiDataInterceptor, error) {
	if len(arg.Topic) == 0 {
		return nil, process.ErrEmptyTopic
	}
	if check.IfNil(arg.Marshalizer) {
		return nil, process.ErrNilMarshalizer
	}
	if check.IfNil(arg.DataFactory) {
		return nil, process.ErrNilInterceptedDataFactory
	}
	if check.IfNil(arg.Processor) {
		return nil, process.ErrNilInterceptedDataProcessor
	}
	if check.IfNil(arg.Throttler) {
		return nil, process.ErrNilInterceptorThrottler
	}
	if check.IfNil(arg.AntifloodHandler) {
		return nil, process.ErrNilAntifloodHandler
	}
	if check.IfNil(arg.WhiteListRequest) {
		return nil, process.ErrNilWhiteListHandler
	}
	if check.IfNil(arg.PreferredPeersHolder) {
		return nil, process.ErrNilPreferredPeersHolder
	}
	//! -------------------- NEW CODE --------------------
	if check.IfNil(arg.DataPool) {
		return nil, process.ErrNilDataPoolInsideMultiDataInterceptor
	}
	//! ---------------- END OF NEW CODE -----------------	

	if len(arg.CurrentPeerId) == 0 {
		return nil, process.ErrEmptyPeerID
	}

	multiDataIntercept := &MultiDataInterceptor{
		baseDataInterceptor: &baseDataInterceptor{
			throttler:            arg.Throttler,
			antifloodHandler:     arg.AntifloodHandler,
			topic:                arg.Topic,
			currentPeerId:        arg.CurrentPeerId,
			processor:            arg.Processor,
			preferredPeersHolder: arg.PreferredPeersHolder,
			debugHandler:         handler.NewDisabledInterceptorDebugHandler(),
		},
		marshalizer:      arg.Marshalizer,
		factory:          arg.DataFactory,
		whiteListRequest: arg.WhiteListRequest,
		chunksProcessor:  disabled.NewDisabledInterceptedChunksProcessor(),
		//! -------------------- NEW CODE --------------------
		dataPool:		  arg.DataPool,
		//! ---------------- END OF NEW CODE -----------------
	}

	return multiDataIntercept, nil
}

// ProcessReceivedMessage is the callback func from the p2p.Messenger and will be called each time a new message was received
// (for the topic this validator was registered to)
func (mdi *MultiDataInterceptor) ProcessReceivedMessage(message p2p.MessageP2P, fromConnectedPeer core.PeerID, _ p2p.MessageHandler) error {
	//! ------------------- NEW CODE ---------------------
	//log.Debug("***Calling MultiDataInterceptor.ProcessReceivedMessage***", "topic", message.Topic(), "key", string(message.Key()))
	//! ---------------- END OF NEW CODE -----------------		
	err := mdi.preProcessMesage(message, fromConnectedPeer)
	if err != nil {
 		//! ------------------- NEW CODE ---------------------
		 log.Debug("***mdi.preProcessMessage(message, fromConnectedPeer) returned an error inside MultiDataInterceptor.ProcessReceivedMessage***", "err", err.Error())
		 //! ---------------- END OF NEW CODE -----------------			
		return err
	}

	b := batch.Batch{}
	err = mdi.marshalizer.Unmarshal(&b, message.Data())
	if err != nil {
		//! ------------------- NEW CODE ---------------------
		log.Debug("***mdi.marshalizer.Unmarshal(&b, message.Data()) returned an error inside MultiDataInterceptor.ProcessReceivedMessage. Calling mdi.throttler.EndProcessing() and blacklisting peer***", "err", err.Error())
		//! ---------------- END OF NEW CODE -----------------			
		mdi.throttler.EndProcessing()

		// this situation is so severe that we need to black list de peers
		reason := "unmarshalable data got on topic " + mdi.topic
		mdi.antifloodHandler.BlacklistPeer(message.Peer(), reason, common.InvalidMessageBlacklistDuration)
		mdi.antifloodHandler.BlacklistPeer(fromConnectedPeer, reason, common.InvalidMessageBlacklistDuration)

		return err
	}
	multiDataBuff := b.Data
	lenMultiData := len(multiDataBuff)
	if lenMultiData == 0 {
		//! -------------------- NEW CODE --------------------
		log.Debug("***lenMultiData == 0 inside ProcessReceivedMessage. Ending processing***")
		//! ---------------- END OF NEW CODE -----------------		
		mdi.throttler.EndProcessing()
		return process.ErrNoDataInMessage
	}

	err = mdi.antifloodHandler.CanProcessMessagesOnTopic(
		fromConnectedPeer,
		mdi.topic,
		uint32(lenMultiData),
		uint64(len(message.Data())),
		message.SeqNo(),
	)
	if err != nil {
		//! -------------------- NEW CODE --------------------
		log.Debug("***mdi.antifloodHandler.CanProcessMessagesOnTopic returned an error inside ProcessReceivedMessage. Ending processing***", "err", err.Error(), "topic", mdi.topic)
		//! ---------------- END OF NEW CODE -----------------			
		mdi.throttler.EndProcessing()
		return err
	}

	mdi.mutChunksProcessor.RLock()
	checkChunksRes, err := mdi.chunksProcessor.CheckBatch(&b, mdi.whiteListRequest)
	mdi.mutChunksProcessor.RUnlock()
	if err != nil {
		//! -------------------- NEW CODE --------------------
		log.Debug("***mdi.chunksProcessor.CheckBatch returned an error inside ProcessReceivedMessage. Ending processing***", "err", err.Error())
		//! ---------------- END OF NEW CODE -----------------			
		mdi.throttler.EndProcessing()
		return err
	}

	isIncompleteChunk := checkChunksRes.IsChunk && !checkChunksRes.HaveAllChunks
	if isIncompleteChunk {
		mdi.throttler.EndProcessing()
		return nil
	}
	isCompleteChunk := checkChunksRes.IsChunk && checkChunksRes.HaveAllChunks
	if isCompleteChunk {
		multiDataBuff = [][]byte{checkChunksRes.CompleteBuffer}
	}

	listInterceptedData := make([]process.InterceptedData, len(multiDataBuff))
	errOriginator := mdi.antifloodHandler.IsOriginatorEligibleForTopic(message.Peer(), mdi.topic)

	for index, dataBuff := range multiDataBuff {
		//! -------------------- NEW CODE --------------------
		//? NON scommentare, dava problemi
		//log.Debug("***inside for index, dataBuff := range multiDataBuff of ProcessReceivedMessage. Ending processing***", "err", err.Error())
		//! ---------------- END OF NEW CODE -----------------			
		var interceptedData process.InterceptedData
		interceptedData, err = mdi.interceptedData(dataBuff, message.Peer(), fromConnectedPeer)
		listInterceptedData[index] = interceptedData
		if err != nil {
			//! -------------------- NEW CODE --------------------
			log.Debug("***Error: mdi.interceptedData. Ending processing***", "err", err.Error())
			//! ---------------- END OF NEW CODE -----------------					
			mdi.throttler.EndProcessing()
			return err
		}

		isWhiteListed := mdi.whiteListRequest.IsWhiteListed(interceptedData)
		if !isWhiteListed && errOriginator != nil {
			//! -------------------- NEW CODE --------------------
			log.Debug("***!isWhiteListed && errOriginator != nil is true inside ProcessReceivedMessage. Ending processing***")
			//! ---------------- END OF NEW CODE -----------------			
			mdi.throttler.EndProcessing()
			log.Trace("got message from peer on topic only for validators", "originator",
				p2p.PeerIdToShortString(message.Peer()),
				"topic", mdi.topic,
				"err", errOriginator)
			return errOriginator
		}

		isForCurrentShard := interceptedData.IsForCurrentShard()
		shouldProcess := isForCurrentShard || isWhiteListed

		//! -------------------- NEW CODE --------------------
		//? SERVE PER QUANDO VENGONO INVIATE LE TRANSAZIONI RIMASTE IN CODA DI UN ACCOUNT CHE è STATO MIGRATO!!!!
		//TODO: SCOMMENTARE ASSOLUTAMENTE
		//_, txPresentInShardedTxPool := mdi.dataPool.Transactions().SearchFirstData(interceptedData.Hash())
		
		//if (txPresentInShardedTxPool){
			//! -------------------- NEW CODE --------------------
			//log.Debug("*** ----- Setting shouldProcess = true ----- ***")
			//! ---------------- END OF NEW CODE -----------------				
			shouldProcess = true
		//}
		//! ---------------- END OF NEW CODE -----------------		

		if !shouldProcess {
			//! -------------------- NEW CODE --------------------
			log.Debug("***!shouldProcess is true inside ProcessReceivedMessage. Ending processing***")
			//! ---------------- END OF NEW CODE -----------------			
			log.Debug("intercepted data should not be processed",
				"pid", p2p.MessageOriginatorPid(message),
				"seq no", p2p.MessageOriginatorSeq(message),
				"topic", message.Topic(),
				"hash", interceptedData.Hash(),
				"is for this shard", isForCurrentShard,
				"is white listed", isWhiteListed,
			)
			mdi.throttler.EndProcessing()
			return process.ErrInterceptedDataNotForCurrentShard
		}
	}

	go func() {
		/*
		//! -------------------- NEW CODE --------------------
		log.Debug("***inside go func() of ProcessReceivedMessage***")
		//! ---------------- END OF NEW CODE -----------------
		*/			
		for _, interceptedData := range listInterceptedData {
			//! -------------------- NEW CODE --------------------
			//log.Debug("***processInterceptedData called inside MultiDataInterceptor.ProcessReceivedMessage***", "interceptedData", interceptedData.Hash())
			//! ---------------- END OF NEW CODE -----------------			
			mdi.processInterceptedData(interceptedData, message)
		}
		mdi.throttler.EndProcessing()
	}()

	return nil
}

func (mdi *MultiDataInterceptor) interceptedData(dataBuff []byte, originator core.PeerID, fromConnectedPeer core.PeerID) (process.InterceptedData, error) {
	interceptedData, err := mdi.factory.Create(dataBuff)
	if err != nil {
		//! ------------------- NEW CODE ---------------------
		log.Debug("***Cannot create object from received bytes. Blacklisting peer***", "err", err.Error())
		//! ---------------- END OF NEW CODE -----------------		
		// this situation is so severe that we need to black list de peers
		reason := "can not create object from received bytes, topic " + mdi.topic + ", error " + err.Error()
		mdi.antifloodHandler.BlacklistPeer(originator, reason, common.InvalidMessageBlacklistDuration)
		mdi.antifloodHandler.BlacklistPeer(fromConnectedPeer, reason, common.InvalidMessageBlacklistDuration)

		return nil, err
	}

	//! ------------------- NEW CODE ---------------------
	//log.Debug("***Calling receivedDebugInterceptedData() inside interceptedData***")
	//! ---------------- END OF NEW CODE -----------------
	mdi.receivedDebugInterceptedData(interceptedData)

	//! ------------------- NEW CODE ---------------------
	log.Debug("***Calling CheckValidity() inside interceptedData***")
	//! ---------------- END OF NEW CODE -----------------	
	err = interceptedData.CheckValidity()
	if err != nil {
		//! ------------------- NEW CODE ---------------------
		log.Debug("***interceptedData.CheckValidity() of the MultiDataInterceptor returned an error. Calling processDebugInterceptedData()***", "err", err.Error())
		//! ---------------- END OF NEW CODE -----------------		
		mdi.processDebugInterceptedData(interceptedData, err)

		isWrongVersion := err == process.ErrInvalidTransactionVersion || err == process.ErrInvalidChainID
		if isWrongVersion {
			// this situation is so severe that we need to black list de peers
			reason := "wrong version of received intercepted data, topic " + mdi.topic + ", error " + err.Error()
			mdi.antifloodHandler.BlacklistPeer(originator, reason, common.InvalidMessageBlacklistDuration)
			mdi.antifloodHandler.BlacklistPeer(fromConnectedPeer, reason, common.InvalidMessageBlacklistDuration)
		}

		return nil, err
	}

	return interceptedData, nil
}

// RegisterHandler registers a callback function to be notified on received data
func (mdi *MultiDataInterceptor) RegisterHandler(handler func(topic string, hash []byte, data interface{})) {
	mdi.processor.RegisterHandler(handler)
}

// SetChunkProcessor sets the intercepted chunks processor
func (mdi *MultiDataInterceptor) SetChunkProcessor(processor process.InterceptedChunksProcessor) error {
	if check.IfNil(processor) {
		return process.ErrNilChunksProcessor
	}

	mdi.mutChunksProcessor.Lock()
	mdi.chunksProcessor = processor
	mdi.mutChunksProcessor.Unlock()

	return nil
}

// Close will call the chunk processor's close method
func (mdi *MultiDataInterceptor) Close() error {
	mdi.mutChunksProcessor.RLock()
	defer mdi.mutChunksProcessor.RUnlock()

	return mdi.chunksProcessor.Close()
}

// IsInterfaceNil returns true if there is no value under the interface
func (mdi *MultiDataInterceptor) IsInterfaceNil() bool {
	return mdi == nil
}
