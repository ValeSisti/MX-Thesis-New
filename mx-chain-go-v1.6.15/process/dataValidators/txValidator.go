package dataValidators

import (
	"fmt"
	"math/big"
	//! -------------------- NEW CODE --------------------
	"bytes"
	"encoding/hex"

	//! ---------------- END OF NEW CODE -----------------

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/core/check"
	"github.com/multiversx/mx-chain-go/process"
	"github.com/multiversx/mx-chain-go/sharding"
	"github.com/multiversx/mx-chain-go/state"
	logger "github.com/multiversx/mx-chain-logger-go"
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"

	//! -------------------- NEW CODE --------------------
	"github.com/multiversx/mx-chain-core-go/data"
	//! ---------------- END OF NEW CODE -----------------
)

var _ process.TxValidator = (*txValidator)(nil)

var log = logger.GetOrCreate("process/dataValidators")

// txValidator represents a tx handler validator that doesn't check the validity of provided txHandler
type txValidator struct {
	accounts             state.AccountsAdapter
	shardCoordinator     sharding.Coordinator
	whiteListHandler     process.WhiteListHandler
	pubKeyConverter      core.PubkeyConverter
	txVersionChecker     process.TxVersionCheckerHandler
	maxNonceDeltaAllowed int
}

// NewTxValidator creates a new nil tx handler validator instance
func NewTxValidator(
	accounts state.AccountsAdapter,
	shardCoordinator sharding.Coordinator,
	whiteListHandler process.WhiteListHandler,
	pubKeyConverter core.PubkeyConverter,
	txVersionChecker process.TxVersionCheckerHandler,
	maxNonceDeltaAllowed int,
) (*txValidator, error) {
	if check.IfNil(accounts) {
		return nil, process.ErrNilAccountsAdapter
	}
	if check.IfNil(shardCoordinator) {
		return nil, process.ErrNilShardCoordinator
	}
	if check.IfNil(whiteListHandler) {
		return nil, process.ErrNilWhiteListHandler
	}
	if check.IfNil(pubKeyConverter) {
		return nil, fmt.Errorf("%w in NewTxValidator", process.ErrNilPubkeyConverter)
	}
	if check.IfNil(txVersionChecker) {
		return nil, process.ErrNilTransactionVersionChecker
	}

	return &txValidator{
		accounts:             accounts,
		shardCoordinator:     shardCoordinator,
		whiteListHandler:     whiteListHandler,
		maxNonceDeltaAllowed: maxNonceDeltaAllowed,
		pubKeyConverter:      pubKeyConverter,
		txVersionChecker:     txVersionChecker,
	}, nil
}

// CheckTxValidity will filter transactions that needs to be added in pools
func (txv *txValidator) CheckTxValidity(interceptedTx process.InterceptedTransactionHandler) error {
	//! -------------------- NEW CODE --------------------
	//log.Debug("***CheckTxValidity called***")	


	normalTransactionHandler, ok := interceptedTx.Transaction().(data.NormalTransactionHandler)
	isAccountMigrationTransaction := ok && len(normalTransactionHandler.GetSignerPubKey()) > 0  && !(len(normalTransactionHandler.GetOriginalMiniBlockHash()) > 0 && len(normalTransactionHandler.GetOriginalTxHash()) > 0)
	isAccountAdjustmentTransaction := ok && len(normalTransactionHandler.GetSignerPubKey()) > 0  && (len(normalTransactionHandler.GetOriginalMiniBlockHash()) > 0 && len(normalTransactionHandler.GetOriginalTxHash()) > 0)


	//! ---------------- END OF NEW CODE -----------------			
	interceptedData, ok := interceptedTx.(process.InterceptedData)
	if ok {
		//! -------------------- NEW CODE --------------------
		//log.Debug("***interceptedData, ok := interceptedTx.(process.InterceptedData) cast is ok***")	
		//! ---------------- END OF NEW CODE -----------------				
		if txv.whiteListHandler.IsWhiteListed(interceptedData) && !(isAccountAdjustmentTransaction){ //! MODIFIED CODE -> risolve l'errore tale per cui quando una AAT ri-arriva ad un validator, 
																									//! questo isWhiteListed ritorna true e quindi la AAT viene riaggiunta alla txPool, causandone il re-inserimento in un nuovo blocco, visto che skippa il controllo sotto (checkIfAATAlreadyInsertedInABlock)
			//! -------------------- NEW CODE --------------------
			log.Debug("***interceptedData is not whitelisted inside CheckTxValidity. Returning nil***", "interceptedTx", hex.EncodeToString(interceptedData.Hash()))	
			//! ---------------- END OF NEW CODE -----------------					
			return nil
		}
	}

	//! -------------------- NEW CODE --------------------
	wasPreviouslyMine := txv.shardCoordinator.WasPreviouslyMineAddrBytes(interceptedTx.SenderAddress())
	//! ---------------- END OF NEW CODE -----------------	
	if txv.isSenderInDifferentShard(interceptedTx) && !wasPreviouslyMine{ //! MODIFIED
		//! -------------------- NEW CODE --------------------
		//log.Debug("***sender is in different shard for interceptedTx. Returning nil***", "hash", hex.EncodeToString(interceptedData.Hash()))	
		//! ---------------- END OF NEW CODE -----------------				
		return nil
	}

	accountHandler, err := txv.getSenderAccount(interceptedTx)
	if err != nil && !wasPreviouslyMine{ //! MODIFIED CODE
		//! -------------------- NEW CODE --------------------
		//log.Debug("***could not get sender account inside CheckTxValidity***", "err", err.Error())	
		//! ---------------- END OF NEW CODE -----------------			
		return err
	}
	//! -------------------- NEW CODE --------------------
	

	
	if (isAccountAdjustmentTransaction){
		return txv.checkIfAATAlreadyInsertedInABlock(interceptedTx, accountHandler) //TODO: implement
		//TODO: ADD AAT LOGIC
	}else if (isAccountMigrationTransaction) {
		//! -------------------- NEW CODE --------------------
		//log.Debug("***isAccountMigrationTransaction = true inside CheckTxValidity. Calling txv.checkAccountForAMT***")	
		//! ---------------- END OF NEW CODE -----------------	
		return txv.checkAccountForAMT(interceptedTx, accountHandler)
	}else{
		//! -------------------- NEW CODE --------------------
		//log.Debug("***isAccountMigrationTransaction = false inside CheckTxValidity. Calling txv.checkAccount***")	
		//! ---------------- END OF NEW CODE -----------------
		return txv.checkAccount(interceptedTx, accountHandler)
	}
	//! ---------------- END OF NEW CODE -----------------	
}

func (txv *txValidator) checkAccount(
	interceptedTx process.InterceptedTransactionHandler,
	accountHandler vmcommon.AccountHandler,
) error {
	
	//! -------------------- NEW CODE --------------------
	if accountHandler == nil {
		return fmt.Errorf("*** account is no more present inside this shard, as it has probably removed on the way, once its corresponding AMT has been notarized dest-side ***")
	}
	//! ---------------- END OF NEW CODE -----------------
	
	err := txv.checkNonce(interceptedTx, accountHandler)
	if err != nil {
		//! -------------------- NEW CODE --------------------
		log.Debug("***checkNonce returned an error inside CheckTxValidity***", "err", err.Error())	
		//! ---------------- END OF NEW CODE -----------------			
		return err
	}

	account, err := txv.getSenderUserAccount(interceptedTx, accountHandler)
	if err != nil {
		//! -------------------- NEW CODE --------------------
		log.Debug("***getSenderUserAccount returned an error inside CheckTxValidity***", "err", err.Error())	
		//! ---------------- END OF NEW CODE -----------------			
		return err
	}

	return txv.checkBalance(interceptedTx, account)
}

func (txv *txValidator) getSenderUserAccount(
	interceptedTx process.InterceptedTransactionHandler,
	accountHandler vmcommon.AccountHandler,
) (state.UserAccountHandler, error) {
	senderAddress := interceptedTx.SenderAddress()
	account, ok := accountHandler.(state.UserAccountHandler)
	if !ok {
		return nil, fmt.Errorf("%w, account is not of type *state.Account, address: %s",
			process.ErrWrongTypeAssertion,
			txv.pubKeyConverter.SilentEncode(senderAddress, log),
		)
	}
	return account, nil
}

func (txv *txValidator) checkBalance(interceptedTx process.InterceptedTransactionHandler, account state.UserAccountHandler) error {	
	
	//! -------------------- NEW CODE --------------------
	if account == nil {
		return fmt.Errorf("*** account is no more present inside this shard, as it has probably removed on the way, once its corresponding AMT has been notarized dest-side ***")
	}
	//! ---------------- END OF NEW CODE -----------------
	
	accountBalance := account.GetBalance()
	//! -------------------- NEW CODE --------------------
	//log.Debug("*** Account Balance inside checkBalance ***", "accountBalance", accountBalance, "address", txv.pubKeyConverter.SilentEncode(interceptedTx.SenderAddress(), log))
	senderIsMigrating := accountBalance.Cmp(big.NewInt(0)) == 0 && account.GetNonce() == uint64(0)
	if senderIsMigrating{
		//log.Debug("*** Balance can't be checked for migrating account, as its state is not yet update. Returning nil ***", "accountBalance", accountBalance, "address", txv.pubKeyConverter.SilentEncode(interceptedTx.SenderAddress(), log))	
		return nil
	}
	//! ---------------- END OF NEW CODE -----------------
	txFee := interceptedTx.Fee()
	if accountBalance.Cmp(txFee) < 0 {
		senderAddress := interceptedTx.SenderAddress()
		//! -------------------- NEW CODE --------------------
		log.Debug("***ERROR INSIDE checkBalance()***")
		//! ---------------- END OF NEW CODE -----------------			
		return fmt.Errorf("%w, for address: %s, wanted %v, have %v",
			process.ErrInsufficientFunds,
			txv.pubKeyConverter.SilentEncode(senderAddress, log),
			txFee,
			accountBalance,
		)
	}

	return nil
}

func (txv *txValidator) checkNonce(interceptedTx process.InterceptedTransactionHandler, accountHandler vmcommon.AccountHandler) error {
	
	//! -------------------- NEW CODE --------------------
	if accountHandler == nil {
		return fmt.Errorf("*** account is no more present inside this shard, as it has probably removed on the way, once its corresponding AMT has been notarized dest-side ***")
	}
	//! ---------------- END OF NEW CODE -----------------
	
	accountNonce := accountHandler.GetNonce()
	txNonce := interceptedTx.Nonce()
	lowerNonceInTx := txNonce < accountNonce
	veryHighNonceInTx := txNonce > accountNonce+uint64(txv.maxNonceDeltaAllowed)
	if lowerNonceInTx || veryHighNonceInTx {
		return fmt.Errorf("%w lowerNonceInTx: %v, veryHighNonceInTx: %v",
			process.ErrWrongTransaction,
			lowerNonceInTx,
			veryHighNonceInTx,
		)
	}
	return nil
}

func (txv *txValidator) isSenderInDifferentShard(interceptedTx process.InterceptedTransactionHandler) bool {
	shardID := txv.shardCoordinator.SelfId()
	txShardID := interceptedTx.SenderShardId()
	return shardID != txShardID
}

func (txv *txValidator) getSenderAccount(interceptedTx process.InterceptedTransactionHandler) (vmcommon.AccountHandler, error) {
	senderAddress := interceptedTx.SenderAddress()
	accountHandler, err := txv.accounts.GetExistingAccount(senderAddress)
	if err != nil {
		return nil, fmt.Errorf("%w for address %s and shard %d, err: %s",
			process.ErrAccountNotFound,
			txv.pubKeyConverter.SilentEncode(senderAddress, log),
			txv.shardCoordinator.SelfId(),
			err.Error(),
		)
	}

	return accountHandler, nil
}

func getTxData(interceptedTx process.InterceptedTransactionHandler) ([]byte, error) {
	tx := interceptedTx.Transaction()
	if tx == nil {
		return nil, process.ErrNilTransaction
	}

	return tx.GetData(), nil
}

// CheckTxWhiteList will check if the cross shard transactions are whitelisted and could be added in pools
func (txv *txValidator) CheckTxWhiteList(data process.InterceptedData) error {
	interceptedTx, ok := data.(process.InterceptedTransactionHandler)
	if !ok {
		return process.ErrWrongTypeAssertion
	}

	isTxCrossShardDestMe := interceptedTx.SenderShardId() != txv.shardCoordinator.SelfId() &&
		interceptedTx.ReceiverShardId() == txv.shardCoordinator.SelfId()
	if !isTxCrossShardDestMe {
		return nil
	}

	if txv.whiteListHandler.IsWhiteListed(data) {
		return nil
	}

	return process.ErrTransactionIsNotWhitelisted
}

// IsInterfaceNil returns true if there is no value under the interface
func (txv *txValidator) IsInterfaceNil() bool {
	return txv == nil
}

//! -------------------- NEW CODE --------------------

func (txv *txValidator) checkAccountForAMT(
	interceptedTx process.InterceptedTransactionHandler,
	accountHandler vmcommon.AccountHandler,
) error {

	log.Debug("***checkAccountForAMT called***")			

	//TODO: per una AMT, oltre a controllare il migration nonce (invece del nonce), devo controllare anche
	//TODO: che il balance della tx sia uguale al balance del sender, che il sender e il receiver siano lo stesso account,
	//TODO: e che il sender shard e il receiver shard siano quelli del risultato della predizione -> questo verrà dopo //TODO: implementare


	err := txv.checkMigrationNonce(interceptedTx, accountHandler)
	if err != nil {
		log.Debug("***checkMigrationNonce returned an error***", "err", err.Error())	
		return err
	}

	//check that sender and receiver accounts are the same
	if !bytes.Equal(interceptedTx.Transaction().GetSndAddr(), interceptedTx.Transaction().GetRcvAddr()) {
		return process.ErrSenderAndReceiverNotTheSameInAMT
	}


	account, err := txv.getSenderUserAccount(interceptedTx, accountHandler)
	if err != nil {
		return err
	}

	return txv.checkBalanceForAMT(interceptedTx, account)
}



func (txv *txValidator) checkBalanceForAMT(interceptedTx process.InterceptedTransactionHandler, account state.UserAccountHandler) error {
	log.Debug("***checkBalanceForAMT called***")	
	accountBalance := account.GetBalance()
	txValue := interceptedTx.Transaction().GetValue()
	
	if !(accountBalance.Cmp(txValue) == 0) { //? if the account balance is NOT equal to the transaction value
		log.Debug("***Account Balance and TxValue: ", "accountBalance", accountBalance.String(), "txValue", txValue.String())
		return process.ValueForAMTIsNotTheSameAsAccountBalance
	}

	return nil
}

func (txv *txValidator) checkMigrationNonce(interceptedTx process.InterceptedTransactionHandler, accountHandler vmcommon.AccountHandler) error {
	userAccountHandler, ok := accountHandler.(state.UserAccountHandler)
	if !ok{
		return fmt.Errorf("cannot cast to user account (insideMigrationNonce)")
	}
	accountMigrationNonce := userAccountHandler.GetMigrationNonce()
	
	
	txMigrationNonce := interceptedTx.Transaction().(data.NormalTransactionHandler).GetMigrationNonce()
	log.Debug("***txMigrationNonce inside checkMigrationNonce***", "txMigrationNonce", string(txMigrationNonce))		
	
	lowerMigrationNonceInTx := txMigrationNonce < accountMigrationNonce
	if lowerMigrationNonceInTx {
		log.Debug("***lowerMigrationNonceInTx := txMigrationNonce < accountMigrationNonce***", "lowerMigrationNonceInTx", lowerMigrationNonceInTx, "txMigrationNonce", string(txMigrationNonce), "accountMigrationNonce", string(accountMigrationNonce))		
	}
	
	veryHighMigrationNonceInTx := txMigrationNonce > accountMigrationNonce+uint64(txv.maxNonceDeltaAllowed)
	if veryHighMigrationNonceInTx{
		log.Debug("***veryHighMigrationNonceInTx := txMigrationNonce > accountMigrationNonce+uint64(txv.maxNonceDeltaAllowed)***", "veryHighMigrationNonceInTx", veryHighMigrationNonceInTx)		
	}
	
	if lowerMigrationNonceInTx || veryHighMigrationNonceInTx {
		return fmt.Errorf("%w lowerMigrationNonceInTx: %v, veryHighMigrationNonceInTx: %v",
			process.ErrWrongTransaction,
			lowerMigrationNonceInTx,
			veryHighMigrationNonceInTx,
		)
	}
	return nil
}


func (txv *txValidator) checkIfAATAlreadyInsertedInABlock(interceptedTx process.InterceptedTransactionHandler, accountHandler vmcommon.AccountHandler) error{
	//? check if AAT was removed before -> (implies) has been already inserted in a block
	originalMbHashBytes := interceptedTx.Transaction().(data.NormalTransactionHandler).GetOriginalMiniBlockHash()
	originalMbHashString := hex.EncodeToString(originalMbHashBytes)

	if txv.shardCoordinator.IsMbHashStringInWaitingMbsForAATsNotarization(originalMbHashString) || txv.shardCoordinator.IsMbHashStringInWaitingMbsForAATsNotarization(string(originalMbHashBytes)){
		log.Debug("***---------------AAT has already been inserted in a block and won't be processed again source side--------------- THIS SHOULD SOLVE THE PROBLEM OF AATs INSERTED MULTIPLE TIMES IN SUBSEQUENT BLOCKS***")
		return process.ErrAATAlreadyProcessed
	}
	return nil
}
//! ---------------- END OF NEW CODE -----------------	
