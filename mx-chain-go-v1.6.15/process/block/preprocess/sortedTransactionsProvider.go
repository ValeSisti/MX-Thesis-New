package preprocess

import (
	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-go/process"
	"github.com/multiversx/mx-chain-go/sharding"
	"github.com/multiversx/mx-chain-go/storage"
	"github.com/multiversx/mx-chain-go/storage/txcache"
)

// TODO: Refactor "transactions.go" to not require the components in this file anymore
// createSortedTransactionsProvider is a "simple factory" for "SortedTransactionsProvider" objects
func createSortedTransactionsProvider(cache storage.Cacher, shardCoordinator sharding.Coordinator) SortedTransactionsProvider {
	txCache, isTxCache := cache.(TxCache)
	if isTxCache {
		return newAdapterTxCacheToSortedTransactionsProvider(txCache, shardCoordinator)
	}

	log.Error("Could not create a real [SortedTransactionsProvider], will create a disabled one")
	return &disabledSortedTransactionsProvider{}
}

// adapterTxCacheToSortedTransactionsProvider adapts a "TxCache" to the "SortedTransactionsProvider" interface
type adapterTxCacheToSortedTransactionsProvider struct {
	txCache TxCache
	shardCoordinator sharding.Coordinator
}

func newAdapterTxCacheToSortedTransactionsProvider(txCache TxCache, shardCoordinator sharding.Coordinator) *adapterTxCacheToSortedTransactionsProvider {
	adapter := &adapterTxCacheToSortedTransactionsProvider{
		txCache: txCache,
		shardCoordinator: shardCoordinator,
	}

	return adapter
}

//! -------------------- NEW CODE --------------------
/*
//! ---------------- END OF NEW CODE -----------------		
// GetSortedTransactions gets the transactions from the cache
func (adapter *adapterTxCacheToSortedTransactionsProvider) GetSortedTransactions() []*txcache.WrappedTransaction {
	txs := adapter.txCache.SelectTransactionsWithBandwidth(process.MaxNumOfTxsToSelect, process.NumTxPerSenderBatchForFillingMiniblock, process.MaxGasBandwidthPerBatchPerSender)
	return txs
}
//! -------------------- NEW CODE --------------------	
*/
//! ---------------- END OF NEW CODE -----------------	


// GetSortedTransactions gets the transactions from the cache
func (adapter *adapterTxCacheToSortedTransactionsProvider) GetSortedTransactions(migratingAccounts map[string]bool) []*txcache.WrappedTransaction { //! MODIFIED CODE
	//! -------------------- NEW CODE --------------------	
	currentEpoch := adapter.shardCoordinator.CurrentEpoch()
	selfShardId := adapter.shardCoordinator.SelfId() 
	var numTxsToSelect int
	log.Debug("*** Checking current Epoch inside GetSortedTransactions ***", "current epoch", currentEpoch )
	
	if currentEpoch >= process.EpochToStartHandlingLoadBalance && selfShardId != core.MetachainShardId{
		log.Debug("*** Setting numTxsToSelect = process.MaxNumOfTxsToSelectToHandleLoadBalance***")
		numTxsToSelect = process.MaxNumOfTxsToSelectToHandleLoadBalance
	}else{
		log.Debug("*** Setting numTxsToSelect = process.MaxNumOfTxsToSelect***")
		numTxsToSelect = process.MaxNumOfTxsToSelect
	}
	//! ---------------- END OF NEW CODE -----------------		
	
	txs := adapter.txCache.SelectTransactionsWithBandwidth(numTxsToSelect, process.NumTxPerSenderBatchForFillingMiniblock, process.MaxGasBandwidthPerBatchPerSender, migratingAccounts) //! MODIFIED CODE
	return txs
}

// NotifyAccountNonce notifies the cache about the current nonce of an account
func (adapter *adapterTxCacheToSortedTransactionsProvider) NotifyAccountNonce(accountKey []byte, nonce uint64) {
	adapter.txCache.NotifyAccountNonce(accountKey, nonce)
}

// IsInterfaceNil returns true if there is no value under the interface
func (adapter *adapterTxCacheToSortedTransactionsProvider) IsInterfaceNil() bool {
	return adapter == nil
}

// disabledSortedTransactionsProvider is a disabled "SortedTransactionsProvider" (should never be used in practice)
type disabledSortedTransactionsProvider struct {
}

//! -------------------- NEW CODE --------------------
/*
//! ---------------- END OF NEW CODE -----------------		
// GetSortedTransactions returns an empty slice
func (adapter *disabledSortedTransactionsProvider) GetSortedTransactions() []*txcache.WrappedTransaction {
	return make([]*txcache.WrappedTransaction, 0)
}
//! -------------------- NEW CODE --------------------	
*/
//! ---------------- END OF NEW CODE -----------------	

// GetSortedTransactions returns an empty slice
func (adapter *disabledSortedTransactionsProvider) GetSortedTransactions(migratingAccounts map[string]bool) []*txcache.WrappedTransaction { //! MODIFIED CODE
	return make([]*txcache.WrappedTransaction, 0)
}

// NotifyAccountNonce does nothing
func (adapter *disabledSortedTransactionsProvider) NotifyAccountNonce(_ []byte, _ uint64) {
}

// IsInterfaceNil returns true if there is no value under the interface
func (adapter *disabledSortedTransactionsProvider) IsInterfaceNil() bool {
	return adapter == nil
}
