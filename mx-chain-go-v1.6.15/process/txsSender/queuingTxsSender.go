package txsSender

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/core/accumulator"
	"github.com/multiversx/mx-chain-core-go/core/check"
	"github.com/multiversx/mx-chain-core-go/data/transaction"
	"github.com/multiversx/mx-chain-core-go/marshal"
	"github.com/multiversx/mx-chain-go/common"
	"github.com/multiversx/mx-chain-go/config"
	"github.com/multiversx/mx-chain-go/dataRetriever"
	"github.com/multiversx/mx-chain-go/process"
	"github.com/multiversx/mx-chain-go/process/factory"
	"github.com/multiversx/mx-chain-go/storage"
)



// ArgsTxsSenderWithAccumulator is a holder struct for all necessary arguments to create a NewTxsSenderWithAccumulator
type ArgsQueuingTxsSenderWithAccumulator struct {
	Marshaller        marshal.Marshalizer
	ShardCoordinator  storage.ShardCoordinator
	NetworkMessenger  NetworkMessenger
	AccumulatorConfig config.TxAccumulatorConfig
	DataPacker        process.DataPacker
}

type queuingTxsSender struct {
	marshaller       marshal.Marshalizer
	shardCoordinator storage.ShardCoordinator
	networkMessenger NetworkMessenger

	ctx           context.Context
	cancelFunc    context.CancelFunc
	txAccumulator core.Accumulator
	dataPacker    process.DataPacker
	txSentCounter uint32
}

// NewTxsSenderWithAccumulator creates a new instance of TxsSenderHandler, which initializes internally an accumulator.NewTimeAccumulator
func NewQueuingTxsSenderWithAccumulator(args ArgsQueuingTxsSenderWithAccumulator) (*queuingTxsSender, error) {
	if check.IfNil(args.Marshaller) {
		return nil, process.ErrNilMarshalizer
	}
	if check.IfNil(args.ShardCoordinator) {
		return nil, process.ErrNilShardCoordinator
	}
	if check.IfNil(args.NetworkMessenger) {
		return nil, process.ErrNilMessenger
	}
	if check.IfNil(args.DataPacker) {
		return nil, dataRetriever.ErrNilDataPacker
	}

	txAccumulator, err := accumulator.NewTimeAccumulator(
		time.Duration(args.AccumulatorConfig.MaxAllowedTimeInMilliseconds)*time.Millisecond,
		time.Duration(args.AccumulatorConfig.MaxDeviationTimeInMilliseconds)*time.Millisecond,
		log,
	)
	if err != nil {
		return nil, err
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	ret := &queuingTxsSender{
		marshaller:       args.Marshaller,
		shardCoordinator: args.ShardCoordinator,
		networkMessenger: args.NetworkMessenger,
		dataPacker:       args.DataPacker,
		ctx:              ctx,
		cancelFunc:       cancelFunc,
		txAccumulator:    txAccumulator,
		txSentCounter:    0,
	}
	go ret.sendFromTxAccumulator(ret.ctx)
	go ret.printTxSentCounter(ret.ctx)

	return ret, nil
}

// SendBulkTransactions sends the provided transactions as a bulk, optimizing transfer between nodes
func (ts *queuingTxsSender) SendBulkTransactions(txs []*transaction.Transaction) (uint64, error) {
	if len(txs) == 0 {
		return 0, process.ErrNoTxToProcess
	}

	ts.addTransactionsToSendPipe(txs)

	return uint64(len(txs)), nil
}

func (ts *queuingTxsSender) addTransactionsToSendPipe(txs []*transaction.Transaction) {
	for _, tx := range txs {
		ts.txAccumulator.AddData(tx)
	}
}

func (ts *queuingTxsSender) sendFromTxAccumulator(ctx context.Context) {
	outputChannel := ts.txAccumulator.OutputChannel()

	for {
		select {
		case objs := <-outputChannel:
			{
				ts.sendTxObjsFromChannel(objs)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (ts *queuingTxsSender) sendTxObjsFromChannel(objs []interface{}) {
	if len(objs) == 0 {
		return
	}

	txs := make([]*transaction.Transaction, 0, len(objs))
	for _, obj := range objs {
		tx, ok := obj.(*transaction.Transaction)
		if !ok {
			continue
		}

		txs = append(txs, tx)
	}

	atomic.AddUint32(&ts.txSentCounter, uint32(len(txs)))
	ts.sendBulkTransactions(txs)
}

func (ts *queuingTxsSender) sendBulkTransactions(txs []*transaction.Transaction) {
	transactionsByShards := make(map[uint32][][]byte)
	log.Trace("txsSender.sendBulkTransactions sending txs",
		"num", len(txs),
	)

	for _, tx := range txs {
		marshalledTx, err := ts.marshaller.Marshal(tx)
		if err != nil {
			log.Warn("txsSender.sendBulkTransactions",
				"marshaller error", err,
			)
			continue
		}

		senderShardId := ts.shardCoordinator.ComputeId(tx.SndAddr)
		transactionsByShards[senderShardId] = append(transactionsByShards[senderShardId], marshalledTx)
	}

	for shardId, txsForShard := range transactionsByShards {
		//! -------------------- NEW CODE --------------------
		log.Debug("***txsForShard inside queuingTxsSender***", "len(txsForShard)", len(txsForShard), "shardId", string(shardId))
		//! ---------------- END OF NEW CODE -----------------			
		err := ts.sendBulkTransactionsFromShard(txsForShard, shardId)
		log.LogIfError(err)
	}
}

func (ts *queuingTxsSender) sendBulkTransactionsFromShard(transactions [][]byte, senderShardId uint32) error {
	// the topic identifier is made of the current shard id and sender's shard id
	identifier := factory.TransactionTopic + ts.shardCoordinator.CommunicationIdentifier(senderShardId)
	//! -------------------- NEW CODE --------------------
	log.Debug("***topic identifier inside sendBulkTransactionsFromShard***", "topic", identifier)
	
	//TODO: TOGLIERE ASSOLUTAMENTE, E' MOMENTANEO PER CAPIRE SE DALLO SHARD 1 HO DIRITTO A SCRIVERE SUL TOPIC "transactions_0"
	//identifier = "transactions_0"
	
	//! ---------------- END OF NEW CODE -----------------

	packets, err := ts.dataPacker.PackDataInChunks(transactions, common.MaxBulkTransactionSize)
	if err != nil {
		return err
	}

	for _, buff := range packets {
		log.Trace("txsSender.sendBulkTransactionsFromShard",
			"topic", identifier,
			"size", len(buff),
		)

		//! -------------------- NEW CODE --------------------
		log.Debug("***queuingTxsSender.sendBulkTransactionsFromShard***",
			"topic", identifier,
			"size", len(buff),
		)
		//! ---------------- END OF NEW CODE -----------------		

		ts.networkMessenger.BroadcastOnChannel(
			SendTransactionsPipe,
			identifier,
			buff)
	}

	return nil
}

// printTxSentCounter prints the peak transaction counter from a time frame of about 'numSecondsBetweenPrints' seconds
// if this peak value is 0 (no transaction was sent through the REST API interface), the print will not be done
// the peak counter resets after each print. There is also a total number of transactions sent to p2p
// TODO make this function testable. Refactor if necessary.
func (ts *queuingTxsSender) printTxSentCounter(ctx context.Context) {
	maxTxCounter := uint32(0)
	totalTxCounter := uint64(0)
	counterSeconds := 0

	for {
		select {
		case <-time.After(time.Second):
			txSent := atomic.SwapUint32(&ts.txSentCounter, 0)
			if txSent > maxTxCounter {
				maxTxCounter = txSent
			}
			totalTxCounter += uint64(txSent)

			counterSeconds++
			if counterSeconds > numSecondsBetweenPrints {
				counterSeconds = 0

				if maxTxCounter > 0 {
					log.Info("sent transactions on network",
						"max/sec", maxTxCounter,
						"total", totalTxCounter,
					)
				}
				maxTxCounter = 0
			}
		case <-ctx.Done():
			return
		}
	}
}

// IsInterfaceNil checks if the underlying pointer is nil
func (ts *queuingTxsSender) IsInterfaceNil() bool {
	return ts == nil
}

// Close calls the cancel function of the background context and closes the network messenger
func (ts *queuingTxsSender) Close() error {
	ts.cancelFunc()
	err := ts.txAccumulator.Close()
	log.LogIfError(err)
	return ts.networkMessenger.Close()
}
