package sharding

import (
	"bytes"
	"math"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-go/sharding/nodesCoordinator"
	//! -------------------- NEW CODE --------------------
	"github.com/multiversx/mx-chain-go/state"
	"github.com/multiversx/mx-chain-core-go/data"
	"github.com/multiversx/mx-chain-core-go/data/transaction"
	//! ---------------- END OF NEW CODE -----------------
)

var _ Coordinator = (*multiShardCoordinator)(nil)

//! ------------------- NEW CODE ---------------------
type ShardInfo struct {
	oldShardId uint32
	currentShardId uint32
	updatedInEpoch uint32
}

type AccountsMapping struct {
	currentEpoch uint32
	accountsShardInfo map[string]ShardInfo //TODO: should I put *ShardInfo instead???
}

/*type SingleAATInfo struct{
	aatxHash string //the txHash that has been generated for the AAT
	//processed bool //if the AAT has been inserted in a block source side //TODO: non serve a nulla, visto che una AAT qui dentro ce la metto se effettivamente è stata inserita in un blocco e dunque processata source side
	notarizedOnDest bool //if the AAT has been notarized destination side
}

//TODO: rename in "ProblematicMBInfo"
type AccountAjustmentTxsInfo struct { //single AAT accessed as AccountAdjustmentTxsInfo.AATsInfo[originalProblematicTxHash]
	numAATs int
	numNotarizedAATs int
	originalProblematicTxHashes []string
	AATsInfo map[string]*SingleAATInfo
	senderShardId uint32
}*/

//! ---------------- END OF NEW CODE -----------------	


// multiShardCoordinator struct defines the functionality for handling transaction dispatching to
// the corresponding shards. The number of shards is currently passed as a constructor
// parameter, and later it should be calculated by this structure
type multiShardCoordinator struct {
	maskHigh       uint32
	maskLow        uint32
	selfId         uint32
	numberOfShards uint32
	//! ------------------- NEW CODE ---------------------
	accountsMapping AccountsMapping
	addressPubKeyConverter core.PubkeyConverter
	accountsAdapter state.AccountsAdapter
	waitingMbsForAATsNotarization map[string]*data.AccountAjustmentTxsInfo
	pendingTxsForMigratingAccounts map[string][]*transaction.Transaction
	rootHashBeforeNewAccounts []byte
	//! ---------------- END OF NEW CODE -----------------		
}

// NewMultiShardCoordinator returns a new multiShardCoordinator and initializes the masks
func NewMultiShardCoordinator(numberOfShards, selfId uint32, addressPubKeyConverter core.PubkeyConverter) (*multiShardCoordinator, error) {
	if numberOfShards < 1 {
		return nil, nodesCoordinator.ErrInvalidNumberOfShards
	}
	if selfId >= numberOfShards && selfId != core.MetachainShardId {
		return nil, nodesCoordinator.ErrInvalidShardId
	}

	sr := &multiShardCoordinator{}
	sr.selfId = selfId
	sr.numberOfShards = numberOfShards
	sr.maskHigh, sr.maskLow = sr.calculateMasks()

	//! -------------------- NEW CODE --------------------
	sr.accountsMapping = AccountsMapping{
        currentEpoch:      0,                    // Assign a value to currentEpoch
        accountsShardInfo: make(map[string]ShardInfo), // Initialize accountsShardInfo map
    }
	sr.addressPubKeyConverter = addressPubKeyConverter
	sr.waitingMbsForAATsNotarization = make(map[string]*data.AccountAjustmentTxsInfo)
	sr.pendingTxsForMigratingAccounts = make(map[string][]*transaction.Transaction)
	sr.rootHashBeforeNewAccounts = make([]byte, 0)
	//! ---------------- END OF NEW CODE -----------------	

	return sr, nil
}

// calculateMasks will create two numbers whose binary form is composed of as many
// ones needed to be taken into consideration for the shard assignment. The result
// of a bitwise AND operation of an address with this mask will result in the
// shard id where a transaction from that address will be dispatched
func (msc *multiShardCoordinator) calculateMasks() (uint32, uint32) {
	n := math.Ceil(math.Log2(float64(msc.numberOfShards)))
	return (1 << uint(n)) - 1, (1 << uint(n-1)) - 1
}

// ComputeId calculates the shard for a given address container
func (msc *multiShardCoordinator) ComputeId(address []byte) uint32 {
	//! ------------------- NEW CODE ---------------------
	if (msc.IsAddressBytesInAccountsMapping(address)){
		return msc.GetCurrentShardFromAddressBytes(address)
	}else{
		return msc.ComputeIdFromBytes(address)
	}
	//! ---------------- END OF NEW CODE -----------------
}


// ComputeIdFromBytes calculates the shard for a given address
func (msc *multiShardCoordinator) ComputeIdFromBytes(address []byte) uint32 {
	if core.IsEmptyAddress(address) {
		return msc.selfId
	}

	var bytesNeed int
	if msc.numberOfShards <= 256 {
		bytesNeed = 1
	} else if msc.numberOfShards <= 65536 {
		bytesNeed = 2
	} else if msc.numberOfShards <= 16777216 {
		bytesNeed = 3
	} else {
		bytesNeed = 4
	}

	startingIndex := 0
	if len(address) > bytesNeed {
		startingIndex = len(address) - bytesNeed
	}

	buffNeeded := address[startingIndex:]
	if core.IsSmartContractOnMetachain(buffNeeded, address) {
		return core.MetachainShardId
	}

	addr := uint32(0)
	for i := 0; i < len(buffNeeded); i++ {
		addr = addr<<8 + uint32(buffNeeded[i])
	}

	shard := addr & msc.maskHigh
	if shard > msc.numberOfShards-1 {
		shard = addr & msc.maskLow
	}

	return shard
}

// NumberOfShards returns the number of shards
func (msc *multiShardCoordinator) NumberOfShards() uint32 {
	return msc.numberOfShards
}

// SelfId gets the shard id of the current node
func (msc *multiShardCoordinator) SelfId() uint32 {
	return msc.selfId
}

// SameShard returns weather two addresses belong to the same shard
func (msc *multiShardCoordinator) SameShard(firstAddress, secondAddress []byte) bool {
	if core.IsEmptyAddress(firstAddress) || core.IsEmptyAddress(secondAddress) {
		return true
	}

	if bytes.Equal(firstAddress, secondAddress) {
		return true
	}

	return msc.ComputeId(firstAddress) == msc.ComputeId(secondAddress)
}

// CommunicationIdentifier returns the identifier between current shard ID and destination shard ID
// identifier is generated such as the first shard from identifier is always smaller or equal than the last
func (msc *multiShardCoordinator) CommunicationIdentifier(destShardID uint32) string {
	return core.CommunicationIdentifierBetweenShards(msc.selfId, destShardID)
}

//! ------------------- NEW CODE ---------------------
func (msc *multiShardCoordinator) AddressPubKeyConverter() core.PubkeyConverter {
	return msc.addressPubKeyConverter
}


func (msc *multiShardCoordinator) UpdateCurrentEpoch(currentEpoch uint32) {
	msc.accountsMapping.currentEpoch = currentEpoch
}


func (msc *multiShardCoordinator) UpdateAccountsMappingEntryFromAddressString(accountAddress string, newShardId uint32, epoch uint32) AccountsMapping {
	if !msc.IsAddressStringInAccountsMapping(accountAddress){ //if it's the first time we add the account to accountsMapping because it wasn't already present
		//intialize both oldShardId and currentShardId to the currentShardId
		//notice that in general situations, i.e. when the account has been migrated multiple times, it will never happen that the oldShardId and the currentShardId will be the same anymore
		acntAddrBytes, _ := msc.addressPubKeyConverter.Decode(accountAddress)
		oldShardIdFromAdrBytes := msc.ComputeIdFromBytes(acntAddrBytes)
		msc.accountsMapping.accountsShardInfo[accountAddress] = ShardInfo{oldShardId: oldShardIdFromAdrBytes, currentShardId: newShardId, updatedInEpoch: epoch}
	}else{
		oldCurrentShardId := msc.accountsMapping.accountsShardInfo[accountAddress].currentShardId
		msc.accountsMapping.accountsShardInfo[accountAddress] = ShardInfo{oldShardId: oldCurrentShardId, currentShardId: newShardId, updatedInEpoch: epoch}
	}
	return msc.accountsMapping
}

//TODO: vedere se è giusto
func (msc *multiShardCoordinator) UpdateAccountsMappingEntryFromPubKeyBytes(pubKeyBytes []byte, newShardId uint32, epoch uint32) AccountsMapping {
	accountAddress, _ := msc.addressPubKeyConverter.Encode(pubKeyBytes)
	if !msc.IsAddressStringInAccountsMapping(accountAddress){ //if it's the first time we add the account to accountsMapping because it wasn't already present
		//intialize both oldShardId and currentShardId to the currentShardId
		//notice that in general situations, i.e. when the account has been migrated multiple times, it will never happen that the oldShardId and the currentShardId will be the same anymore
		oldShardIdFromAdrBytes := msc.ComputeIdFromBytes(pubKeyBytes)
		msc.accountsMapping.accountsShardInfo[accountAddress] = ShardInfo{oldShardId: oldShardIdFromAdrBytes, currentShardId: newShardId, updatedInEpoch: epoch}
	}else{
		oldCurrentShardId := msc.accountsMapping.accountsShardInfo[accountAddress].currentShardId
		msc.accountsMapping.accountsShardInfo[accountAddress] = ShardInfo{oldShardId: oldCurrentShardId, currentShardId: newShardId, updatedInEpoch: epoch}
	}
	return msc.accountsMapping
}


func (msc *multiShardCoordinator) IsAddressBytesInAccountsMapping(pubKeyBytes []byte) bool {
	accountAddress, _ := msc.addressPubKeyConverter.Encode(pubKeyBytes)
	_, exists := msc.accountsMapping.accountsShardInfo[accountAddress]
	return exists
}

func (msc *multiShardCoordinator) IsAddressStringInAccountsMapping(accountAddress string) bool {
	_, exists := msc.accountsMapping.accountsShardInfo[accountAddress]
	return exists
}

func (msc *multiShardCoordinator) AccountsMapping() AccountsMapping {
	return msc.accountsMapping
}

func (msc *multiShardCoordinator) AccountsShardInfo() map[string]ShardInfo {
	return msc.accountsMapping.accountsShardInfo
}

func (msc *multiShardCoordinator) CurrentEpoch() uint32 {
	return msc.accountsMapping.currentEpoch
}

func (msc *multiShardCoordinator) GetCurrentShardFromAddressString(accountAddress string) uint32 {
	return msc.accountsMapping.accountsShardInfo[accountAddress].currentShardId
}

func (msc *multiShardCoordinator) GetCurrentShardFromAddressBytes(pubKeyBytes []byte) uint32 {
	accountAddress, _ := msc.addressPubKeyConverter.Encode(pubKeyBytes)
	return msc.accountsMapping.accountsShardInfo[accountAddress].currentShardId
}

func (msc *multiShardCoordinator) GetOldShardFromAddressString(accountAddress string) uint32 {
	return msc.accountsMapping.accountsShardInfo[accountAddress].oldShardId
}

func (msc *multiShardCoordinator) GetOldShardFromAddressBytes(pubKeyBytes []byte) uint32 {
	accountAddress, _ := msc.addressPubKeyConverter.Encode(pubKeyBytes)
	return msc.accountsMapping.accountsShardInfo[accountAddress].oldShardId
}

func (msc *multiShardCoordinator) GetEpochOfUpdateFromAddressString(accountAddress string) uint32 {
	return msc.accountsMapping.accountsShardInfo[accountAddress].updatedInEpoch
}

func (msc *multiShardCoordinator) GetEpochOfUpdateFromAddressBytes(pubKeyBytes []byte) uint32 {
	accountAddress, _ := msc.addressPubKeyConverter.Encode(pubKeyBytes)
	return msc.accountsMapping.accountsShardInfo[accountAddress].updatedInEpoch
}

func (msc *multiShardCoordinator) HasBeenMigratedInCurrentEpochFromAddrBytes(pubKeyBytes []byte) bool {
	accountAddress, _ := msc.addressPubKeyConverter.Encode(pubKeyBytes)
	log.Debug("*** HasBeenMigratedInCurrentEpochFromAddrBytes ***", "addr", accountAddress, "updatedInEpoch", msc.accountsMapping.accountsShardInfo[accountAddress].updatedInEpoch, "currentEpoch", msc.accountsMapping.currentEpoch)
	return msc.accountsMapping.accountsShardInfo[accountAddress].updatedInEpoch == msc.accountsMapping.currentEpoch
}

func (msc *multiShardCoordinator) HasBeenMigratedInCurrentEpochFromAddrString(accountAddress string) bool {
	return msc.accountsMapping.accountsShardInfo[accountAddress].updatedInEpoch == msc.accountsMapping.currentEpoch
}

func (msc *multiShardCoordinator) GetShardInfoFromAddressString(accountAddress string) ShardInfo {
	return msc.accountsMapping.accountsShardInfo[accountAddress]
}

func (msc *multiShardCoordinator) GetShardInfoFromAddressBytes(pubKeyBytes []byte) ShardInfo {
	accountAddress, _ := msc.addressPubKeyConverter.Encode(pubKeyBytes)
	return msc.accountsMapping.accountsShardInfo[accountAddress]
}

func (msc *multiShardCoordinator) SetAccountsAdapter(accountsAdapter state.AccountsAdapter) {
	msc.accountsAdapter = accountsAdapter
}

func (msc *multiShardCoordinator) AccountsAdapter() state.AccountsAdapter {
	return msc.accountsAdapter
}



func (msc *multiShardCoordinator) WasPreviouslyMineAddrBytes(pubKeyBytes []byte) bool {
	accountAddress, _ := msc.addressPubKeyConverter.Encode(pubKeyBytes)
	selfShardId := msc.SelfId()

	if _, ok := msc.accountsMapping.accountsShardInfo[accountAddress]; !ok{
		// receiver account is not event present inside accountsMapping, therefore it has never been migrated
		// which also means that is not possible that a tx inside a miniblock with as receiver account the account with this address,
		// that I'm processing dst me (because theoretically it is mine) is problematic
		log.Debug("***WasPreviouslyMineAddrBytes: account is not present inside AccountsMapping, returning false!***", "account", accountAddress)
		return false
	}
	
	oldShardId := msc.accountsMapping.accountsShardInfo[accountAddress].oldShardId
	currentShardId := msc.accountsMapping.accountsShardInfo[accountAddress].currentShardId
	updatedInEpoch := msc.accountsMapping.accountsShardInfo[accountAddress].updatedInEpoch
	currentEpoch := msc.accountsMapping.currentEpoch

	log.Debug("***WasPreviouslyMineAddrBytes***",
		"account", accountAddress,
		"oldShardId", oldShardId,
		"currentShardId", currentShardId,
		"updatedInEpoch", updatedInEpoch,
		"currentEpoch", currentEpoch,
		"selfShardId", selfShardId,
		"oldShardId == selfShardId", oldShardId == selfShardId,
		"currentShardId != selfShardId", currentShardId != selfShardId,
		"updatedInEpoch == currentEpoch", updatedInEpoch == currentEpoch,
	)
	
	if (oldShardId == selfShardId && currentShardId != selfShardId && updatedInEpoch == currentEpoch){
		return true
	}

	return false
}

func (msc *multiShardCoordinator) WasPreviouslyMineAddrString(accountAddress string) bool {
	selfShardId := msc.SelfId()

	if _, ok := msc.accountsMapping.accountsShardInfo[accountAddress]; !ok{
		// receiver account is not event present inside accountsMapping, therefore it has never been migrated
		// which also means that is not possible that a tx inside a miniblock with as receiver account the account with this address,
		// that I'm processing dst me (because theoretically it is mine) is problematic
		return false
	}

	oldShardId := msc.accountsMapping.accountsShardInfo[accountAddress].oldShardId
	currentShardId := msc.accountsMapping.accountsShardInfo[accountAddress].currentShardId
	updatedInEpoch := msc.accountsMapping.accountsShardInfo[accountAddress].updatedInEpoch
	currentEpoch := msc.accountsMapping.currentEpoch

	if (oldShardId == selfShardId && currentShardId != selfShardId && updatedInEpoch == currentEpoch){
		return true
	}
	
	return false
}


func (msc *multiShardCoordinator) UpdateWaitingMbsForAATsNotarization(problematicsMBsForCurrRound map[string]*data.ProblematicMBInfo) map[string]*data.AccountAjustmentTxsInfo{
	for mbHash, mbInfo := range problematicsMBsForCurrRound{
		aatsInfo := make(map[string]*data.SingleAATInfo)
		for _, aatHash := range mbInfo.AccAdjTxHashes{
			aatInfo := &data.SingleAATInfo{
				AatxHash: aatHash,
				NotarizedOnDest: false,
			}
			aatsInfo[aatHash] = aatInfo
		}
		msc.waitingMbsForAATsNotarization[mbHash] = &data.AccountAjustmentTxsInfo{
			NumAATs: len(mbInfo.AccAdjTxHashes), 
			NumNotarizedAATs: 0, 
			OriginalProblematicTxHashes: mbInfo.ProblematicTxHashes, 
			AATsInfo: aatsInfo, 
			SenderShardId: mbInfo.SenderShardID,
			//IncludedInBlock: false,
		}
	}
	return msc.waitingMbsForAATsNotarization
}


func (msc *multiShardCoordinator) IsMbHashStringInWaitingMbsForAATsNotarization(mbHash string) bool {
	log.Debug("*** Checking if mbHash is in waitingMbsForAATsNotarization ***", "mbHash", mbHash)
	log.Debug("*** Printing current msc.waitingMbsForAATNotarization *** ", "msc.waitingMbsForAATsNotarization", msc.waitingMbsForAATsNotarization)
	
	_, exists := msc.waitingMbsForAATsNotarization[mbHash]
	log.Debug("*** returning if mbHash exists in msc.waitingMbsForAATNotarization*** ", "exists", exists)
	return exists
}

func (msc *multiShardCoordinator) SetNotarizedAATInWaitingMbsForAATsNotarization(aatHash string, mbHash string) (bool, int, int) {
	aatsInfo, ok := msc.waitingMbsForAATsNotarization[mbHash]
	if !ok {
		log.Debug("*** Error: cannot find mbHash inside waitingMbsForAATsNotarization ***")
		return false, 0, 0
	}
	aatInfo, ok := aatsInfo.AATsInfo[aatHash]
	if !ok {
		log.Debug("*** Error: cannot find aatHash inside waitingMbsForAATsNotarization[mbHash].AATsInfo ***")
		return false, 0, 0
	}
	if !aatInfo.NotarizedOnDest{
		aatInfo.NotarizedOnDest = true
		aatsInfo.NumNotarizedAATs += 1
		log.Debug("*** aatInfo.notarizedOnDest set true for AAT inside waitingMbs... ***", "aatHash", aatHash)
		return true, aatsInfo.NumAATs, aatsInfo.NumNotarizedAATs
	}
	//msc.waitingMbsForAATsNotarization[mbHash].AATsInfo[aatHash].notarizedOnDest = true //? -> fatto direttamente così non worka
	log.Debug("*** Error: (?) aatInfo.notarizedOnDest was already true inside waitingForAATsNtoarization[mbHash] ***")
	return false, 0, 0
}

func (msc *multiShardCoordinator) GetMbsWithAllAATsNotarizedFromWaitingMBs() map[string]*data.AccountAjustmentTxsInfo {
	log.Debug("*** GetMbsWithAllAATsNotarizedFromWaitingMBs called ***")
	mbsWithAllAAtsNotarized := make(map[string]*data.AccountAjustmentTxsInfo)

	for mbHash, aatsInfo := range msc.waitingMbsForAATsNotarization{
		log.Debug("*** checking if mb is ready to be processed inside waitingMbsForAATsNotarization", "numAAts", aatsInfo.NumAATs, "numNotarizedAAts", aatsInfo.NumNotarizedAATs)
		if aatsInfo.NumNotarizedAATs == aatsInfo.NumAATs{
			log.Debug("*** Found mb ready to be processed inside waitingMbsForAATsNotarization ***", "mbHash", mbHash)
			mbsWithAllAAtsNotarized[mbHash]= aatsInfo
		}
	}

	log.Debug("*** Returning mbs with ALL aats notarized from waiting mbs: ***", "mbsWithAllAATsNotarized", mbsWithAllAAtsNotarized)

	return mbsWithAllAAtsNotarized
}

func (msc *multiShardCoordinator) IsProblematicMBReadyToBeProcessed(mbHash string) bool {
	log.Debug("*** IsProblematicMBReadyToBeProcessed called ***", "mbHash", mbHash)
	log.Debug("*** Current msc.waitingMbsForAATsNotarization ***", "msc.waitingMbsForAATsNotarization", msc.waitingMbsForAATsNotarization)

	mbInfo, found := msc.waitingMbsForAATsNotarization[mbHash]
	if found {
		if mbInfo.NumAATs == mbInfo.NumNotarizedAATs{
			return true
		}
	}
	return false
}

func (msc *multiShardCoordinator) RemoveReadyMbsInsertedInCurrentRoundFromWaitingMbs(readyMbsIncludedInCurrentBlock map[string]bool) map[string]*data.AccountAjustmentTxsInfo{
	for mbHash, _ := range readyMbsIncludedInCurrentBlock{
		delete(msc.waitingMbsForAATsNotarization, mbHash)
		log.Debug("*** ready mb inserted in current round removed from waitingMbsForAATsNotarization ***", "mbHash", mbHash)
	}
	return msc.waitingMbsForAATsNotarization
}


func (msc *multiShardCoordinator) PutTransactionInPendingForMigratingAccount(txSender string, tx *transaction.Transaction){
	if _, ok := msc.pendingTxsForMigratingAccounts[txSender]; !ok{
		msc.pendingTxsForMigratingAccounts[txSender] = make([]*transaction.Transaction, 0)
	}
	msc.pendingTxsForMigratingAccounts[txSender] = append(msc.pendingTxsForMigratingAccounts[txSender], tx)

	log.Debug("*** Transaction put inside PendingTxsForMigratingAccounts ***")
}


func (msc *multiShardCoordinator) GetTransactionsReceivedForMigratingAccount(txSender string) []*transaction.Transaction{
	log.Debug("*** GetTransactionsReceivedForMigratingAccount called ***")
	if _, ok := msc.pendingTxsForMigratingAccounts[txSender]; !ok{
		log.Debug("*** Sender is not present inside PendingTxsForMigratingAccounts ***")
		return make([]*transaction.Transaction, 0)
	}
	return msc.pendingTxsForMigratingAccounts[txSender]
}

func (msc *multiShardCoordinator) RemoveAccountFromPendingTxsForMigratingAccounts(accountAddr string){
	if _, ok := msc.pendingTxsForMigratingAccounts[accountAddr]; !ok{
		delete(msc.pendingTxsForMigratingAccounts, accountAddr)
	}
}

func (msc *multiShardCoordinator) SetRootHashBeforeNewAccounts(rootHash []byte){
	msc.rootHashBeforeNewAccounts = rootHash
}

func (msc *multiShardCoordinator) GetRootHashBeforeNewAccounts() []byte {
	return msc.rootHashBeforeNewAccounts
}
//! ---------------- END OF NEW CODE -----------------	

// IsInterfaceNil returns true if there is no value under the interface
func (msc *multiShardCoordinator) IsInterfaceNil() bool {
	return msc == nil
}
