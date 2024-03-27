package processor

import (
	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/core/check"
	"github.com/multiversx/mx-chain-go/process"
	logger "github.com/multiversx/mx-chain-logger-go"
	//! -------------------- NEW CODE --------------------
	"github.com/multiversx/mx-chain-go/sharding"
	"github.com/multiversx/mx-chain-core-go/data"
	//! ---------------- END OF NEW CODE -----------------		
)

var _ process.InterceptorProcessor = (*TxInterceptorProcessor)(nil)
var txLog = logger.GetOrCreate("process/interceptors/processor/txlog")

// TxInterceptorProcessor is the processor used when intercepting transactions
// (smart contract results, receipts, transaction) structs which satisfy TransactionHandler interface.
type TxInterceptorProcessor struct {
	shardedPool process.ShardedPool
	txValidator process.TxValidator
	//! -------------------- NEW CODE --------------------
	shardCoordinator sharding.Coordinator
	//! ---------------- END OF NEW CODE -----------------
}

// NewTxInterceptorProcessor creates a new TxInterceptorProcessor instance
func NewTxInterceptorProcessor(argument *ArgTxInterceptorProcessor) (*TxInterceptorProcessor, error) {
	if argument == nil {
		return nil, process.ErrNilArgumentStruct
	}
	if check.IfNil(argument.ShardedDataCache) {
		return nil, process.ErrNilDataPoolHolder
	}
	if check.IfNil(argument.TxValidator) {
		return nil, process.ErrNilTxValidator
	}
	//! -------------------- NEW CODE --------------------
	if check.IfNil(argument.ShardCoordinator) {
		return nil, process.ErrNilShardCoordinator
	}
	//! ---------------- END OF NEW CODE -----------------

	return &TxInterceptorProcessor{
		shardedPool: argument.ShardedDataCache,
		txValidator: argument.TxValidator,
		//! -------------------- NEW CODE --------------------
		shardCoordinator: argument.ShardCoordinator,
		//! ---------------- END OF NEW CODE -----------------		

	}, nil
}

// Validate checks if the intercepted data can be processed
func (txip *TxInterceptorProcessor) Validate(data process.InterceptedData, _ core.PeerID) error {
	interceptedTx, ok := data.(process.InterceptedTransactionHandler)
	if !ok {
		return process.ErrWrongTypeAssertion
	}

	return txip.txValidator.CheckTxValidity(interceptedTx)
}

// Save will save the received data into the cacher
func (txip *TxInterceptorProcessor) Save(dataToProcess process.InterceptedData, peerOriginator core.PeerID, _ string) error { //! MODIFIED CODE -> data -> dataToProcess (altrimenti mi dava problemi quando facevo data.NormalTransactionHandler)
	interceptedTx, ok := dataToProcess.(process.InterceptedTransactionHandler)
	if !ok {
		return process.ErrWrongTypeAssertion
	}

	err := txip.txValidator.CheckTxWhiteList(dataToProcess)
	if err != nil {
		log.Debug(
			"TxInterceptorProcessor.Save: not whitelisted cross transactions will not be added in pool",
			"nonce", interceptedTx.Nonce(),
			"sender address", interceptedTx.SenderAddress(),
			"sender shard", interceptedTx.SenderShardId(),
			"receiver shard", interceptedTx.ReceiverShardId(),
		)
		return nil
	}

	txLog.Debug("received transaction", "pid", peerOriginator.Pretty(), "hash", dataToProcess.Hash())
	cacherIdentifier := process.ShardCacherIdentifier(interceptedTx.SenderShardId(), interceptedTx.ReceiverShardId())
	txip.shardedPool.AddData(
		dataToProcess.Hash(),
		interceptedTx.Transaction(),
		interceptedTx.Transaction().Size(),
		cacherIdentifier,
	)

	
	//! -------------------- NEW CODE --------------------
	//? se il receiver è stato "recentemente" migrato, metto la tx anche nella cache del vecchio shard, perché potrebbero arrivare miniblocchi "vecchi" (ma anche nemmeno tanto vecchi) 
	//? che contengono transazioni VERSO il il nostro account che è stato migratio, e che, trovandosi all'interno di un miniblocco con "ReceiverShardID" il vecchio shard
	//? ma la tx potrebbe essere arrivata in un momento successivo alla modifica dell'AccountsMapping, e quando si andrà a vedere se quella transazioni è presente nella
	//? shardedTxPool con cache id ad esempio 2_1 (quando però il nostro account è stato spostato da 1 -> 0 e la tx, una volta ricevuta, è stata inserita nella cache con cache id 2_0),
	//? quella tx NON verrà trovata è verrà richiesta in loop, senza mai essere "ricevuta" dove la stiamo andando a cercare (cioè nella cache con id 2_1), visto che una volta
	//? ricevuta verrà sempre messa nella cache con cache id 2_0! -> VEDI IPAD PAGINA 185

	//? SOLUZIONE: se il receiverAccount è stato "recentemente" migrato, 
	//? 		   metto la tx ANCHE nella "veccahia" cache, ovvero nella cache con id {senderShardId}_{receiverOldShardID()}
	
	
	normalTransactionHandler, ok := interceptedTx.Transaction().(data.NormalTransactionHandler)
	isSpecialTransaction := ok && len(normalTransactionHandler.GetSignerPubKey()) > 0
	
	//? metto anche !isSpecialTransaction perché altrimenti
	if !isSpecialTransaction && txip.shardCoordinator.HasBeenMigratedInCurrentEpochFromAddrBytes(interceptedTx.Transaction().GetRcvAddr()) { // if receiver address recently migrated 
		receiverAddrOldShardId := txip.shardCoordinator.GetOldShardFromAddressBytes(interceptedTx.Transaction().GetRcvAddr())
		oldCacherIdentifier := process.ShardCacherIdentifier(interceptedTx.SenderShardId(), receiverAddrOldShardId)

		log.Debug("*** Receiver account has been 'recently' migrated: adding the interceptedTx also in the old cache!! --- THIS AVOIDS PROBLEM ON IPAD PAGE 185 ---")
		txip.shardedPool.AddData(
			dataToProcess.Hash(),
			interceptedTx.Transaction(),
			interceptedTx.Transaction().Size(),
			oldCacherIdentifier,
		)
	}else if !isSpecialTransaction && txip.shardCoordinator.HasBeenMigratedInCurrentEpochFromAddrBytes(interceptedTx.Transaction().GetSndAddr()) { // if receiver address recently migrated 
		senderAddrOldShardId := txip.shardCoordinator.GetOldShardFromAddressBytes(interceptedTx.Transaction().GetSndAddr())
		oldCacherIdentifier := process.ShardCacherIdentifier(senderAddrOldShardId, interceptedTx.ReceiverShardId())

		log.Debug("*** Sender account has been 'recently' migrated: adding the interceptedTx also in the old cache!! --- THIS AVOIDS PROBLEM ON IPAD PAGE 185 but for sender addresses ---")
		txip.shardedPool.AddData(
			dataToProcess.Hash(),
			interceptedTx.Transaction(),
			interceptedTx.Transaction().Size(),
			oldCacherIdentifier,
		)
	}
	//! ---------------- END OF NEW CODE -----------------	

	return nil
}

// RegisterHandler registers a callback function to be notified of incoming transactions
func (txip *TxInterceptorProcessor) RegisterHandler(_ func(topic string, hash []byte, data interface{})) {
	log.Error("txInterceptorProcessor.RegisterHandler", "error", "not implemented")
}

// IsInterfaceNil returns true if there is no value under the interface
func (txip *TxInterceptorProcessor) IsInterfaceNil() bool {
	return txip == nil
}
