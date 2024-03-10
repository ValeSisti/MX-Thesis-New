package sharding

import (
	"github.com/multiversx/mx-chain-go/sharding/nodesCoordinator"	
	//! -------------------- NEW CODE --------------------
	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-go/state"
	"github.com/multiversx/mx-chain-core-go/data"	
	//! ---------------- END OF NEW CODE -----------------	
)


// Coordinator defines what a shard state coordinator should hold
type Coordinator interface {
	NumberOfShards() uint32
	ComputeId(address []byte) uint32
	SelfId() uint32
	SameShard(firstAddress, secondAddress []byte) bool
	CommunicationIdentifier(destShardID uint32) string
	IsInterfaceNil() bool
	//! ------------------- NEW CODE ---------------------
	AddressPubKeyConverter() core.PubkeyConverter
	UpdateCurrentEpoch(currentEpoch uint32)
	UpdateAccountsMappingEntryFromAddressString(accountAddress string, newShardId uint32, epoch uint32) AccountsMapping
	UpdateAccountsMappingEntryFromPubKeyBytes(addressBytes []byte, newShardId uint32, epoch uint32) AccountsMapping
	//GetShardInfoFromAddressString(accountAddress string) ShardInfo //TODO: capire se mi servono. Se si, "implementarle" dentro oneShardCoordinator.go
	//GetShardInfoFromAddressBytes(pubKeyBytes []byte) ShardInfo
	GetCurrentShardFromAddressString(accountAddress string) uint32
	GetCurrentShardFromAddressBytes(pubKeyBytes []byte) uint32
	GetOldShardFromAddressString(accountAddress string) uint32
	GetOldShardFromAddressBytes(pubKeyBytes []byte) uint32
	GetEpochOfUpdateFromAddressString(accountAddress string) uint32
	GetEpochOfUpdateFromAddressBytes(pubKeyBytes []byte) uint32
	HasBeenMigratedInCurrentEpochFromAddrBytes(pubKeyBytes []byte) bool	
	HasBeenMigratedInCurrentEpochFromAddrString(accountAddress string) bool	
	AccountsMapping() AccountsMapping
	AccountsShardInfo() map[string]ShardInfo
	CurrentEpoch() uint32
	AccountsAdapter() state.AccountsAdapter
	WasPreviouslyMineAddrBytes(pubKeyBytes []byte) bool
	WasPreviouslyMineAddrString(accountAddress string) bool
	IsAddressStringInAccountsMapping(accountAddress string) bool
	UpdateWaitingMbsForAATsNotarization(problematicsMBsForCurrRound map[string]*data.ProblematicMBInfo) map[string]*data.AccountAjustmentTxsInfo
	IsMbHashStringInWaitingMbsForAATsNotarization(mbHash string) bool 	
	SetNotarizedAATInWaitingMbsForAATsNotarization(aatHash string, mbHash string) (bool, int, int)
	GetMbsWithAllAATsNotarizedFromWaitingMBs() map[string]*data.AccountAjustmentTxsInfo
	IsProblematicMBReadyToBeProcessed(mbHash string) bool 
	RemoveReadyMbsInsertedInCurrentRoundFromWaitingMbs(readyMbsIncludedInCurrentBlock map[string]bool) map[string]*data.AccountAjustmentTxsInfo
	//! ---------------- END OF NEW CODE -----------------	
}

// EpochHandler defines what a component which handles current epoch should be able to do
type EpochHandler interface {
	MetaEpoch() uint32
	IsInterfaceNil() bool
}

// PeerAccountListAndRatingHandler provides Rating Computation Capabilites for the Nodes Coordinator and ValidatorStatistics
type PeerAccountListAndRatingHandler interface {
	// GetChance returns the chances for the rating
	GetChance(uint32) uint32
	// GetStartRating gets the start rating values
	GetStartRating() uint32
	// GetSignedBlocksThreshold gets the threshold for the minimum signed blocks
	GetSignedBlocksThreshold() float32
	// ComputeIncreaseProposer computes the new rating for the increaseLeader
	ComputeIncreaseProposer(shardId uint32, currentRating uint32) uint32
	// ComputeDecreaseProposer computes the new rating for the decreaseLeader
	ComputeDecreaseProposer(shardId uint32, currentRating uint32, consecutiveMisses uint32) uint32
	// RevertIncreaseValidator computes the new rating if a revert for increaseProposer should be done
	RevertIncreaseValidator(shardId uint32, currentRating uint32, nrReverts uint32) uint32
	// ComputeIncreaseValidator computes the new rating for the increaseValidator
	ComputeIncreaseValidator(shardId uint32, currentRating uint32) uint32
	// ComputeDecreaseValidator computes the new rating for the decreaseValidator
	ComputeDecreaseValidator(shardId uint32, currentRating uint32) uint32
	// IsInterfaceNil verifies if the interface is nil
	IsInterfaceNil() bool
}

// GenesisNodesSetupHandler returns the genesis nodes info
type GenesisNodesSetupHandler interface {
	AllInitialNodes() []nodesCoordinator.GenesisNodeInfoHandler
	InitialNodesPubKeys() map[uint32][]string
	GetShardIDForPubKey(pubkey []byte) (uint32, error)
	InitialEligibleNodesPubKeysForShard(shardId uint32) ([]string, error)
	InitialNodesInfoForShard(shardId uint32) ([]nodesCoordinator.GenesisNodeInfoHandler, []nodesCoordinator.GenesisNodeInfoHandler, error)
	InitialNodesInfo() (map[uint32][]nodesCoordinator.GenesisNodeInfoHandler, map[uint32][]nodesCoordinator.GenesisNodeInfoHandler)
	GetStartTime() int64
	GetRoundDuration() uint64
	GetShardConsensusGroupSize() uint32
	GetMetaConsensusGroupSize() uint32
	NumberOfShards() uint32
	MinNumberOfNodes() uint32
	MinNumberOfShardNodes() uint32
	MinNumberOfMetaNodes() uint32
	GetHysteresis() float32
	GetAdaptivity() bool
	MinNumberOfNodesWithHysteresis() uint32
	IsInterfaceNil() bool
}
