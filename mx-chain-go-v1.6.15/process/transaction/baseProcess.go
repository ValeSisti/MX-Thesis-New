package transaction

import (
	"bytes"
	"fmt"
	"math/big"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/core/check"
	"github.com/multiversx/mx-chain-core-go/data"
	"github.com/multiversx/mx-chain-core-go/data/transaction"
	"github.com/multiversx/mx-chain-core-go/hashing"
	"github.com/multiversx/mx-chain-core-go/marshal"
	"github.com/multiversx/mx-chain-go/common"
	"github.com/multiversx/mx-chain-go/process"
	"github.com/multiversx/mx-chain-go/sharding"
	"github.com/multiversx/mx-chain-go/state"
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
)

type baseTxProcessor struct {
	accounts            state.AccountsAdapter
	shardCoordinator    sharding.Coordinator
	pubkeyConv          core.PubkeyConverter
	economicsFee        process.FeeHandler
	hasher              hashing.Hasher
	marshalizer         marshal.Marshalizer
	scProcessor         process.SmartContractProcessor
	enableEpochsHandler common.EnableEpochsHandler
	txVersionChecker    process.TxVersionCheckerHandler
	guardianChecker     process.GuardianChecker
}

func (txProc *baseTxProcessor) getAccounts(
	adrSrc, adrDst []byte,
) (state.UserAccountHandler, state.UserAccountHandler, error) {

	//! -------------------- NEW CODE --------------------
	adrSrcString, _ := txProc.shardCoordinator.AddressPubKeyConverter().Encode(adrSrc)	
	adrDstString, _ := txProc.shardCoordinator.AddressPubKeyConverter().Encode(adrDst)	
	log.Debug("***getAccounts() called***", "adrSrc", adrSrcString, "adrDst", adrDstString)
	//! ---------------- END OF NEW CODE -----------------		

	var acntSrc, acntDst state.UserAccountHandler

	shardForCurrentNode := txProc.shardCoordinator.SelfId()
	shardForSrc := txProc.shardCoordinator.ComputeId(adrSrc)
	shardForDst := txProc.shardCoordinator.ComputeId(adrDst)

	srcInShard := shardForSrc == shardForCurrentNode
	dstInShard := shardForDst == shardForCurrentNode

	if srcInShard && len(adrSrc) == 0 || dstInShard && len(adrDst) == 0 {
		return nil, nil, process.ErrNilAddressContainer
	}

	if bytes.Equal(adrSrc, adrDst) {
		acntWrp, err := txProc.accounts.LoadAccount(adrSrc)
		if err != nil {
			return nil, nil, err
		}

		account, ok := acntWrp.(state.UserAccountHandler)
		if !ok {
			return nil, nil, process.ErrWrongTypeAssertion
		}

		return account, account, nil
	}

	if srcInShard {
		//! -------------------- NEW CODE --------------------
		log.Debug("***adrSrc is in my shard inside getAccounts()***", "adrSrc", adrSrcString)
		//! ---------------- END OF NEW CODE -----------------		
		
		acntSrcWrp, err := txProc.accounts.LoadAccount(adrSrc)
		if err != nil {
			return nil, nil, err
		}

		account, ok := acntSrcWrp.(state.UserAccountHandler)
		if !ok {
			return nil, nil, process.ErrWrongTypeAssertion
		}

		acntSrc = account
	}

	if dstInShard {
		//! -------------------- NEW CODE --------------------
		log.Debug("***adrDst is in my shard inside getAccounts()***", "adrDst", adrDstString)
		//! ---------------- END OF NEW CODE -----------------
	
		acntDstWrp, err := txProc.accounts.LoadAccount(adrDst)
		if err != nil {
			return nil, nil, err
		}

		account, ok := acntDstWrp.(state.UserAccountHandler)
		if !ok {
			return nil, nil, process.ErrWrongTypeAssertion
		}

		//! -------------------- NEW CODE --------------------
		//TODO: perché lo avevo messo? A che serviva? Boh, intanto lo scommento
		/*if(txProc.shardCoordinator.IsAddressStringInAccountsMapping(string(account.AddressBytes()))){
			account.SetIsBeingMigrated(true)
		}*/
		//! ---------------- END OF NEW CODE -----------------		

		acntDst = account
	}

	return acntSrc, acntDst, nil
}



//! -------------------- NEW CODE --------------------
func (txProc *baseTxProcessor) getReceiverAccount(
	adrDst []byte,
) (state.UserAccountHandler, state.UserAccountHandler, error) {

	//! -------------------- NEW CODE --------------------
	adrDstString, _ := txProc.shardCoordinator.AddressPubKeyConverter().Encode(adrDst)
	log.Debug("***getReceiverAccount() called***", "adrDst", adrDstString)
	//! ---------------- END OF NEW CODE -----------------		

	var acntSrc, acntDst state.UserAccountHandler

	shardForCurrentNode := txProc.shardCoordinator.SelfId()
	//shardForSrc := txProc.shardCoordinator.ComputeId(adrSrc) //! MODIFIED CODE
	shardForDst := txProc.shardCoordinator.ComputeId(adrDst)

	//srcInShard := shardForSrc == shardForCurrentNode //! MODIFIED CODE
	dstInShard := shardForDst == shardForCurrentNode

	//if srcInShard && len(adrSrc) == 0 || dstInShard && len(adrDst) == 0 { //! MODIFIED CODE
	if dstInShard && len(adrDst) == 0 {
		return nil, nil, process.ErrNilAddressContainer
	}

	//! -------------------- NEW CODE --------------------
	/*
	//! ---------------- END OF NEW CODE -----------------		
	if bytes.Equal(adrSrc, adrDst) {
		acntWrp, err := txProc.accounts.LoadAccount(adrSrc)
		if err != nil {
			return nil, nil, err
		}

		account, ok := acntWrp.(state.UserAccountHandler)
		if !ok {
			return nil, nil, process.ErrWrongTypeAssertion
		}

		return account, account, nil
	}

	if srcInShard {
		acntSrcWrp, err := txProc.accounts.LoadAccount(adrSrc)
		if err != nil {
			return nil, nil, err
		}

		account, ok := acntSrcWrp.(state.UserAccountHandler)
		if !ok {
			return nil, nil, process.ErrWrongTypeAssertion
		}

		acntSrc = account
	}
	//! -------------------- NEW CODE --------------------	
	*/
	//! ---------------- END OF NEW CODE -----------------	

	if dstInShard {
		log.Debug("***adrDst is in my shard inside getReceiverAccount(), everything good!***", "adrDst", adrDstString)		

		acntDstWrp, err := txProc.accounts.LoadAccount(adrDst)
		if err != nil {
			return nil, nil, err
		}

		account, ok := acntDstWrp.(state.UserAccountHandler)
		if !ok {
			return nil, nil, process.ErrWrongTypeAssertion
		}

		//! -------------------- NEW CODE --------------------
		//TODO: perché lo avevo messo? A che serviva? Boh, intanto lo scommento
		/*if(txProc.shardCoordinator.IsAddressStringInAccountsMapping(string(account.AddressBytes()))){
			account.SetIsBeingMigrated(true)
		}*/
		//! ---------------- END OF NEW CODE -----------------		

		acntDst = account
	//! -------------------- NEW CODE --------------------
	}else{
		//? Check if the receiver account cannot be found because it has been "recently" migrated from my shard
		hasBeenMigrated := txProc.shardCoordinator.WasPreviouslyMineAddrBytes(adrDst)
		if hasBeenMigrated{
			log.Debug("***Account is not present because it has been migrated: SPECIAL CASE OF AATs! Returning process.ErrAccountNotFoundBecauseMigrated*** ---------------------PROBLEM--------------------- (THIS SHOULD NEVER BE PRINTED, AS PROBLEMATICMBS (AND TXS) ARE CHECKED BEFORE TRYING TO PROCESS TRANSACTIONS DST SIDE!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!)",  "adrDst", adrDstString)
			return nil, nil, process.ErrAccountNotFoundBecauseMigrated
		}
		log.Debug("***Error: adrDst is not present is current shard (getReceiverAccount) while it should be (?)***", "adrDst", adrDstString)
	//! ---------------- END OF NEW CODE -----------------
	}

	return acntSrc, acntDst, nil
}
//! ---------------- END OF NEW CODE -----------------


func (txProc *baseTxProcessor) getAccountFromAddress(adrSrc []byte) (state.UserAccountHandler, error) {
	shardForCurrentNode := txProc.shardCoordinator.SelfId()
	shardForSrc := txProc.shardCoordinator.ComputeId(adrSrc)
	if shardForCurrentNode != shardForSrc {
		return nil, nil
	}

	acnt, err := txProc.accounts.LoadAccount(adrSrc)
	if err != nil {
		return nil, err
	}

	userAcc, ok := acnt.(state.UserAccountHandler)
	if !ok {
		return nil, process.ErrWrongTypeAssertion
	}

	return userAcc, nil
}

func (txProc *baseTxProcessor) checkTxValues(
	tx *transaction.Transaction,
	acntSnd, acntDst state.UserAccountHandler,
	isUserTxOfRelayed bool,
) error {
	err := txProc.verifyGuardian(tx, acntSnd)
	if err != nil {
		return err
	}
	err = txProc.checkUserNames(tx, acntSnd, acntDst)
	if err != nil {
		return err
	}
	if check.IfNil(acntSnd) {
		return nil
	}
	if acntSnd.GetNonce() < tx.Nonce {
		//! ------------------- NEW CODE ---------------------
		log.Debug("***acntSnd.GetNonce() < tx.Nonce", "acntSnd.GetNonce()", acntSnd.GetNonce(), "tx.Nonce", tx.Nonce)
		//! ---------------- END OF NEW CODE -----------------			
		return process.ErrHigherNonceInTransaction
	}
	if acntSnd.GetNonce() > tx.Nonce {
		//! ------------------- NEW CODE ---------------------
		log.Debug("***acntSnd.GetNonce() > tx.Nonce", "acntSnd.GetNonce()", acntSnd.GetNonce(), "tx.Nonce", tx.Nonce)
		//! ---------------- END OF NEW CODE -----------------			
		return process.ErrLowerNonceInTransaction
	}
	err = txProc.economicsFee.CheckValidityTxValues(tx)
	if err != nil {
		//! ------------------- NEW CODE ---------------------
		log.Debug("***Error during CheckValidityTxValues")
		//! ---------------- END OF NEW CODE -----------------			
		return err
	}

	var txFee *big.Int
	if isUserTxOfRelayed {
		if tx.GasLimit < txProc.economicsFee.ComputeGasLimit(tx) {
			return process.ErrNotEnoughGasInUserTx
		}
		txFee = txProc.economicsFee.ComputeFeeForProcessing(tx, tx.GasLimit)
	} else {
		txFee = txProc.economicsFee.ComputeTxFee(tx)
	}

	if acntSnd.GetBalance().Cmp(txFee) < 0 {
		return fmt.Errorf("%w, has: %s, wanted: %s",
			process.ErrInsufficientFee,
			acntSnd.GetBalance().String(),
			txFee.String(),
		)
	}

	if !txProc.enableEpochsHandler.IsPenalizedTooMuchGasFlagEnabled() {
		// backwards compatibility issue when provided gas limit and gas price exceeds the available balance before the
		// activation of the "penalize too much gas" flag
		txFee = core.SafeMul(tx.GasLimit, tx.GasPrice)
	}

	cost := big.NewInt(0).Add(txFee, tx.Value)
	if acntSnd.GetBalance().Cmp(cost) < 0 {
		return process.ErrInsufficientFunds
	}

	return nil
}

//! ------------------- NEW CODE ---------------------

func (txProc *baseTxProcessor) checkTxValuesForAMT(
	tx *transaction.Transaction,
	acntSnd, acntDst state.UserAccountHandler,
	isUserTxOfRelayed bool,
) error {
	err := txProc.verifyGuardian(tx, acntSnd)
	if err != nil {
		return err
	}
	err = txProc.checkUserNames(tx, acntSnd, acntDst)
	if err != nil {
		return err
	}
	if check.IfNil(acntSnd) {
		return nil
	}
	if acntSnd.GetMigrationNonce() < tx.MigrationNonce {
		//! ------------------- NEW CODE ---------------------
		log.Debug("***acntSnd.GetMigrationNonce() < tx.MigrationNonce", "acntSnd.GetMigrationNonce()", acntSnd.GetMigrationNonce(), "tx.MigrationNonce", tx.MigrationNonce)
		//! ---------------- END OF NEW CODE -----------------			
		return process.ErrHigherMigrationNonceInTransaction
	}
	if acntSnd.GetMigrationNonce() > tx.MigrationNonce {
		//! ------------------- NEW CODE ---------------------
		log.Debug("***acntSnd.GetMigrationNonce() > tx.MigrationNonce", "acntSnd.GetMigrationNonce()", acntSnd.GetMigrationNonce(), "tx.MigrationNonce", tx.MigrationNonce)
		//! ---------------- END OF NEW CODE -----------------			
		return process.ErrLowerMigrationNonceInTransaction
	}
	err = txProc.economicsFee.CheckValidityTxValues(tx) //? Credo che non faccia nulla, perché economicsFee è di tipo process.FeeHandler, ma se vedo il metodo CheckValidityTxValues di FeeHandler è disabilitato (?)
	if err != nil {
		//! ------------------- NEW CODE ---------------------
		log.Debug("***Error during CheckValidityTxValues")
		//! ---------------- END OF NEW CODE -----------------	
		return err
	}

	/*var txFee *big.Int
	if isUserTxOfRelayed {
		if tx.GasLimit < txProc.economicsFee.ComputeGasLimit(tx) {
			return process.ErrNotEnoughGasInUserTx
		}
		txFee = txProc.economicsFee.ComputeFeeForProcessing(tx, tx.GasLimit)
	} else {
		txFee = txProc.economicsFee.ComputeTxFee(tx)
	}*/

	/*txFee := big.NewInt(0)

	if acntSnd.GetBalance().Cmp(txFee) < 0 {
		return fmt.Errorf("%w, has: %s, wanted: %s",
			process.ErrInsufficientFee,
			acntSnd.GetBalance().String(),
			txFee.String(),
		)
	}

	if !txProc.enableEpochsHandler.IsPenalizedTooMuchGasFlagEnabled() {
		// backwards compatibility issue when provided gas limit and gas price exceeds the available balance before the
		// activation of the "penalize too much gas" flag
		txFee = core.SafeMul(tx.GasLimit, tx.GasPrice)
	}

	cost := big.NewInt(0).Add(txFee, tx.Value)
	if acntSnd.GetBalance().Cmp(cost) < 0 {
		return process.ErrInsufficientFunds
	}*/

	return nil
}

func (txProc *baseTxProcessor) checkTxValuesForAAT(
	tx *transaction.Transaction,
	acntSnd, acntDst state.UserAccountHandler,
	isUserTxOfRelayed bool,
) error {
	//TODO: implementare??? Serve????
	return nil
}

//! ---------------- END OF NEW CODE -----------------	

func (txProc *baseTxProcessor) checkUserNames(tx *transaction.Transaction, acntSnd, acntDst state.UserAccountHandler) error {
	isUserNameWrong := len(tx.SndUserName) > 0 &&
		!check.IfNil(acntSnd) && !bytes.Equal(tx.SndUserName, acntSnd.GetUserName())
	if isUserNameWrong {
		return process.ErrUserNameDoesNotMatch
	}

	isUserNameWrong = len(tx.RcvUserName) > 0 &&
		!check.IfNil(acntDst) && !bytes.Equal(tx.RcvUserName, acntDst.GetUserName())
	if isUserNameWrong {
		if check.IfNil(acntSnd) {
			return process.ErrUserNameDoesNotMatchInCrossShardTx
		}
		return process.ErrUserNameDoesNotMatch
	}

	return nil
}

func (txProc *baseTxProcessor) processIfTxErrorCrossShard(tx *transaction.Transaction, errorString string) error {
	txHash, err := core.CalculateHash(txProc.marshalizer, txProc.hasher, tx)
	if err != nil {
		return err
	}

	snapshot := txProc.accounts.JournalLen()
	err = txProc.scProcessor.ProcessIfError(nil, txHash, tx, errorString, nil, snapshot, 0)
	if err != nil {
		return err
	}

	return nil
}

// VerifyTransaction verifies the account states in respect with the transaction data
func (txProc *baseTxProcessor) VerifyTransaction(tx *transaction.Transaction) error {
	if check.IfNil(tx) {
		return process.ErrNilTransaction
	}
	//! -------------------- NEW CODE --------------------
	log.Debug("***baseTxProcessor.VerifyTransaction called*** -------POTENTIAL PROBLEM--------- (getAccounts -> devo capire come separare getAccounts e getReceiverAccount anche qui????????)")
	//! ---------------- END OF NEW CODE -----------------
	senderAccount, receiverAccount, err := txProc.getAccounts(tx.SndAddr, tx.RcvAddr)
	if err != nil {
		return err
	}

	return txProc.checkTxValues(tx, senderAccount, receiverAccount, false)
}

// Setting a guardian is allowed with regular transactions on a guarded account
// but in this case is set with the default epochs delay
func (txProc *baseTxProcessor) checkOperationAllowedToBypassGuardian(tx *transaction.Transaction) error {
	if !process.IsSetGuardianCall(tx.GetData()) {
		return fmt.Errorf("%w, not allowed to bypass guardian", process.ErrTransactionNotExecutable)
	}

	err := txProc.CheckSetGuardianExecutable(tx)
	if err != nil {
		return err
	}
	if len(tx.GetRcvUserName()) > 0 || len(tx.GetSndUserName()) > 0 {
		return fmt.Errorf("%w, SetGuardian does not support usernames", process.ErrTransactionNotExecutable)
	}

	return nil
}

// CheckSetGuardianExecutable checks if the setGuardian builtin function is executable
func (txProc *baseTxProcessor) CheckSetGuardianExecutable(tx data.TransactionHandler) error {
	err := txProc.scProcessor.CheckBuiltinFunctionIsExecutable(core.BuiltInFunctionSetGuardian, tx)
	if err != nil {
		return fmt.Errorf("%w, CheckBuiltinFunctionIsExecutable %s", process.ErrTransactionNotExecutable, err.Error())
	}

	return nil
}

func (txProc *baseTxProcessor) checkGuardedAccountUnguardedTxPermission(tx *transaction.Transaction, account state.UserAccountHandler) error {
	err := txProc.checkOperationAllowedToBypassGuardian(tx)
	if err != nil {
		return err
	}

	// block non-guarded setGuardian Txs if there is a pending guardian
	hasPendingGuardian := txProc.guardianChecker.HasPendingGuardian(account)
	if process.IsSetGuardianCall(tx.GetData()) && hasPendingGuardian {
		return fmt.Errorf("%w, %s", process.ErrTransactionNotExecutable, process.ErrCannotReplaceGuardedAccountPendingGuardian.Error())
	}

	return nil
}

func (txProc *baseTxProcessor) verifyGuardian(tx *transaction.Transaction, account state.UserAccountHandler) error {
	if check.IfNil(account) {
		return nil
	}
	isTransactionGuarded := txProc.txVersionChecker.IsGuardedTransaction(tx)
	if !account.IsGuarded() {
		if isTransactionGuarded {
			return fmt.Errorf("%w, %s", process.ErrTransactionNotExecutable, process.ErrGuardedTransactionNotExpected.Error())
		}

		return nil
	}
	if !isTransactionGuarded {
		return txProc.checkGuardedAccountUnguardedTxPermission(tx, account)
	}

	acc, ok := account.(vmcommon.UserAccountHandler)
	if !ok {
		return fmt.Errorf("%w, %s", process.ErrTransactionNotExecutable, process.ErrWrongTypeAssertion.Error())
	}

	guardian, err := txProc.guardianChecker.GetActiveGuardian(acc)
	if err != nil {
		return fmt.Errorf("%w, %s", process.ErrTransactionNotExecutable, err.Error())
	}

	if !bytes.Equal(guardian, tx.GuardianAddr) {
		return fmt.Errorf("%w, %s", process.ErrTransactionNotExecutable, process.ErrTransactionAndAccountGuardianMismatch.Error())
	}

	return nil
}
