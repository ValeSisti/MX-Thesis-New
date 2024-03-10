package processor

import (
	"github.com/multiversx/mx-chain-go/dataRetriever"
	"github.com/multiversx/mx-chain-go/process"
	//! -------------------- NEW CODE --------------------
	"github.com/multiversx/mx-chain-go/sharding"
	//! ---------------- END OF NEW CODE -----------------	
)

// ArgTxInterceptorProcessor is the argument for the interceptor processor used for transactions
// (balance txs, smart contract results, reward and so on)
type ArgTxInterceptorProcessor struct {
	ShardedDataCache dataRetriever.ShardedDataCacherNotifier
	TxValidator      process.TxValidator
	//! -------------------- NEW CODE --------------------
	ShardCoordinator sharding.Coordinator	
	//! ---------------- END OF NEW CODE -----------------	
}
