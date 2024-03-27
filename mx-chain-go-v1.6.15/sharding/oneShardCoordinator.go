package sharding

import (
	"github.com/multiversx/mx-chain-core-go/core"
	//! -------------------- NEW CODE --------------------
	"github.com/multiversx/mx-chain-go/state"
	"github.com/multiversx/mx-chain-core-go/data"
	"github.com/multiversx/mx-chain-core-go/data/transaction"
	//! ---------------- END OF NEW CODE -----------------
)

var _ Coordinator = (*OneShardCoordinator)(nil)

// OneShardCoordinator creates a shard coordinator object
type OneShardCoordinator struct{}

// NumberOfShards gets number of shards
func (osc *OneShardCoordinator) NumberOfShards() uint32 {
	return 1
}

// ComputeId gets shard for the given address
func (osc *OneShardCoordinator) ComputeId(_ []byte) uint32 {
	return 0
}

// SelfId gets shard of the current node
func (osc *OneShardCoordinator) SelfId() uint32 {
	return 0
}

// SameShard returns weather two addresses belong to the same shard
func (osc *OneShardCoordinator) SameShard(_, _ []byte) bool {
	return true
}

// CommunicationIdentifier returns the identifier between current shard ID and destination shard ID
// for this implementation, it will always return "_0" as there is a single shard
func (osc *OneShardCoordinator) CommunicationIdentifier(destShardID uint32) string {
	return core.CommunicationIdentifierBetweenShards(destShardID, 0)
}

// IsInterfaceNil returns true if there is no value under the interface
func (osc *OneShardCoordinator) IsInterfaceNil() bool {
	return osc == nil
}

//! -------------------- NEW CODE --------------------
func (osc *OneShardCoordinator) AddressPubKeyConverter() core.PubkeyConverter {
	return nil
}

func (osc *OneShardCoordinator) UpdateCurrentEpoch(currentEpoch uint32) {}


func (osc *OneShardCoordinator) UpdateAccountsMappingEntryFromAddressString(accountAddress string, newShardId uint32, epoch uint32) AccountsMapping {
	return AccountsMapping{}
}

func (osc *OneShardCoordinator) UpdateAccountsMappingEntryFromPubKeyBytes(pubKeyBytes []byte, newShardId uint32, epoch uint32) AccountsMapping {
	return AccountsMapping{}
}

func (osc *OneShardCoordinator) GetCurrentShardFromAddressString(accountAddress string) uint32 {
	return 999
}

func (osc *OneShardCoordinator) GetCurrentShardFromAddressBytes(pubKeyBytes []byte) uint32 {
	return 999
}

func (osc *OneShardCoordinator) GetOldShardFromAddressString(accountAddress string) uint32 {
	return 999
}

func (osc *OneShardCoordinator) GetOldShardFromAddressBytes(pubKeyBytes []byte) uint32 {
	return 999
}

func (osc *OneShardCoordinator) GetEpochOfUpdateFromAddressString(accountAddress string) uint32 {
	return 999
}

func (osc *OneShardCoordinator) GetEpochOfUpdateFromAddressBytes(pubKeyBytes []byte) uint32 {
	return 999
}

func (osc *OneShardCoordinator)	HasBeenMigratedInCurrentEpochFromAddrBytes(pubKeyBytes []byte) bool {
	return false
}

func (osc *OneShardCoordinator)	HasBeenMigratedInCurrentEpochFromAddrString(accountAddress string) bool {
	return false
}

/*func (osc *OneShardCoordinator) GetShardInfoFromAddressString(accountAddress string) ShardInfo {
	return ShardInfo{}
}

func (osc *OneShardCoordinator) GetShardInfoFromAddressBytes(pubKeyBytes []byte) ShardInfo{
	return ShardInfo{}
}*/

func (osc *OneShardCoordinator) AccountsMapping() AccountsMapping {
	return AccountsMapping{}
}

func (osc *OneShardCoordinator) AccountsShardInfo() map[string]ShardInfo {
	return nil
}

func (osc *OneShardCoordinator) CurrentEpoch() uint32 {
	return 0
}

func (osc *OneShardCoordinator) AccountsAdapter() state.AccountsAdapter {
	return nil
}

func (osc *OneShardCoordinator) WasPreviouslyMineAddrBytes(pubKeyBytes []byte) bool {
	return false
}

func (osc *OneShardCoordinator) WasPreviouslyMineAddrString(accountAddress string) bool {
	return false
}

func (osc *OneShardCoordinator) IsAddressStringInAccountsMapping(accountAddress string) bool {
	return false
}

func (osc *OneShardCoordinator) UpdateWaitingMbsForAATsNotarization(problematicsMBsForCurrRound map[string]*data.ProblematicMBInfo) map[string]*data.AccountAjustmentTxsInfo{
	return nil
}

func (osc *OneShardCoordinator) IsMbHashStringInWaitingMbsForAATsNotarization(mbHash string) bool {
	return false
}

func (osc *OneShardCoordinator) SetNotarizedAATInWaitingMbsForAATsNotarization(aatHash string, mbHash string) (bool, int, int) {
	return false, 0, 0
}

func (osc *OneShardCoordinator) GetMbsWithAllAATsNotarizedFromWaitingMBs() map[string]*data.AccountAjustmentTxsInfo {
	return nil
}

func (osc *OneShardCoordinator) IsProblematicMBReadyToBeProcessed(mbHash string) bool {
	return false
}

func (osc *OneShardCoordinator) RemoveReadyMbsInsertedInCurrentRoundFromWaitingMbs(readyMbsIncludedInCurrentBlock map[string]bool) map[string]*data.AccountAjustmentTxsInfo{
	return nil
}

func (osc *OneShardCoordinator) PutTransactionInPendingForMigratingAccount(txSender string, tx *transaction.Transaction){}


func (osc *OneShardCoordinator) GetTransactionsReceivedForMigratingAccount(txSender string) []*transaction.Transaction{
	return make([]*transaction.Transaction, 0)
}

func (osc *OneShardCoordinator) RemoveAccountFromPendingTxsForMigratingAccounts(txSender string){}


func (osc *OneShardCoordinator) SetRootHashBeforeNewAccounts(rootHash []byte){}


func (osc *OneShardCoordinator) GetRootHashBeforeNewAccounts() []byte {
	return make([]byte, 0)
}
//! ---------------- END OF NEW CODE -----------------	