//go:generate protoc -I=. -I=$GOPATH/src -I=$GOPATH/src/github.com/multiversx/protobuf/protobuf  --gogoslick_out=$GOPATH/src transaction.proto
package transaction

import (
	"math/big"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/core/check"
	"github.com/multiversx/mx-chain-core-go/data"
)

var _ = data.TransactionHandler(&Transaction{})

// IsInterfaceNil verifies if underlying object is nil
func (tx *Transaction) IsInterfaceNil() bool {
	return tx == nil
}

// SetValue sets the value of the transaction
func (tx *Transaction) SetValue(value *big.Int) {
	tx.Value = value
}

// SetData sets the data of the transaction
func (tx *Transaction) SetData(data []byte) {
	tx.Data = data
}

// SetRcvAddr sets the receiver address of the transaction
func (tx *Transaction) SetRcvAddr(addr []byte) {
	tx.RcvAddr = addr
}

// SetSndAddr sets the sender address of the transaction
func (tx *Transaction) SetSndAddr(addr []byte) {
	tx.SndAddr = addr
}

// TrimSlicePtr creates a copy of the provided slice without the excess capacity
func TrimSlicePtr(in []*Transaction) []*Transaction {
	if len(in) == 0 {
		return []*Transaction{}
	}
	ret := make([]*Transaction, len(in))
	copy(ret, in)
	return ret
}

// TrimSliceHandler creates a copy of the provided slice without the excess capacity
func TrimSliceHandler(in []data.TransactionHandler) []data.TransactionHandler {
	if len(in) == 0 {
		return []data.TransactionHandler{}
	}
	ret := make([]data.TransactionHandler, len(in))
	copy(ret, in)
	return ret
}

// GetDataForSigning returns the serialized transaction having an empty signature field
func (tx *Transaction) GetDataForSigning(encoder data.Encoder, marshaller data.Marshaller, hasher data.Hasher) ([]byte, error) {
	if check.IfNil(encoder) {
		return nil, ErrNilEncoder
	}
	if check.IfNil(marshaller) {
		return nil, ErrNilMarshalizer
	}
	if check.IfNil(hasher) {
		return nil, ErrNilHasher
	}

	receiverAddr, err := encoder.Encode(tx.RcvAddr)
	if err != nil {
		return nil, err
	}

	senderAddr, err := encoder.Encode(tx.SndAddr)
	if err != nil {
		return nil, err
	}

	//! ------------------- NEW CODE ---------------------
	/*
	//! ---------------- END OF NEW CODE -----------------		
	

	ftx := &FrontendTransaction{
		Nonce:            tx.Nonce,
		Value:            tx.Value.String(),
		Receiver:         receiverAddr,
		Sender:           senderAddr,
		GasPrice:         tx.GasPrice,
		GasLimit:         tx.GasLimit,
		SenderUsername:   tx.SndUserName,
		ReceiverUsername: tx.RcvUserName,
		Data:             tx.Data,
		ChainID:          string(tx.ChainID),
		Version:          tx.Version,
		Options:          tx.Options,
	}

	//! ------------------- NEW CODE ---------------------
	*/
	var ftx *FrontendTransaction

	//!NOTA: l'unico campo che posso controllare per vedere se è una AMT o no è
	//! la SignerPubKey, perché "for non-pointer fields like uint64, there's no definite way to know if the field was explicitly set to its zero value or if it simply retains the default value due to not being set"
	//! Invece, "For slices (like []byte fields in your struct), you can check if they are nil or have a length of 0"
	
	isAccountMigration := len(tx.SignerPubKey) > 0 && !(len(tx.OriginalTxHash) > 0 && len(tx.OriginalMiniBlockHash) > 0)
	isAccountAdjustment := len(tx.SignerPubKey) > 0 && (len(tx.OriginalTxHash) > 0 && len(tx.OriginalMiniBlockHash) > 0)
	
	if (isAccountAdjustment){
		ftx = &FrontendTransaction{
			Nonce:            tx.Nonce, //TODO: controllare
			//MigrationNonce:   tx.MigrationNonce,
			Value:            tx.Value.String(),
			Receiver:         receiverAddr,
			Sender:           senderAddr,
			GasPrice:         tx.GasPrice,
			GasLimit:         tx.GasLimit,
			SenderUsername:   tx.SndUserName,
			ReceiverUsername: tx.RcvUserName,
			Data:             tx.Data,
			ChainID:          string(tx.ChainID),
			Version:          tx.Version,
			Options:          tx.Options,
			SenderShard:      tx.SenderShard,
			ReceiverShard:    tx.ReceiverShard,
			SignerPubKey:     tx.SignerPubKey,
			OriginalTxHash:   tx.OriginalTxHash,
			OriginalMiniBlockHash: tx.OriginalMiniBlockHash,
		}	
	}else if (isAccountMigration) {
		ftx = &FrontendTransaction{
			//Nonce:            tx.Nonce, //TODO: controllare
			MigrationNonce:   tx.MigrationNonce,
			Value:            tx.Value.String(),
			Receiver:         receiverAddr,
			Sender:           senderAddr,
			GasPrice:         tx.GasPrice,
			GasLimit:         tx.GasLimit,
			SenderUsername:   tx.SndUserName,
			ReceiverUsername: tx.RcvUserName,
			Data:             tx.Data,
			ChainID:          string(tx.ChainID),
			Version:          tx.Version,
			Options:          tx.Options,
			SenderShard:      tx.SenderShard,
			ReceiverShard:    tx.ReceiverShard,
			SignerPubKey:     tx.SignerPubKey,
		}
	}else{ //? è una transazione normale
		ftx = &FrontendTransaction{
			Nonce:            tx.Nonce,
			Value:            tx.Value.String(),
			Receiver:         receiverAddr,
			Sender:           senderAddr,
			GasPrice:         tx.GasPrice,
			GasLimit:         tx.GasLimit,
			SenderUsername:   tx.SndUserName,
			ReceiverUsername: tx.RcvUserName,
			Data:             tx.Data,
			ChainID:          string(tx.ChainID),
			Version:          tx.Version,
			Options:          tx.Options,
		}
	}

	//! ---------------- END OF NEW CODE -----------------	
	if len(tx.GuardianAddr) > 0 {
		guardianAddr, errGuardian := encoder.Encode(tx.GuardianAddr)
		if errGuardian != nil {
			return nil, errGuardian
		}

		ftx.GuardianAddr = guardianAddr
	}

	ftxBytes, err := marshaller.Marshal(ftx)
	if err != nil {
		return nil, err
	}

	shouldSignOnTxHash := tx.Version > core.InitialVersionOfTransaction && tx.HasOptionHashSignSet()
	if !shouldSignOnTxHash {
		return ftxBytes, nil
	}

	ftxHash := hasher.Compute(string(ftxBytes))

	return ftxHash, nil
}

// HasOptionGuardianSet returns true if the guarded transaction option is set
func (tx *Transaction) HasOptionGuardianSet() bool {
	return tx.Options&MaskGuardedTransaction > 0
}

// HasOptionHashSignSet returns true if the signed with hash option is set
func (tx *Transaction) HasOptionHashSignSet() bool {
	return tx.Options&MaskSignedWithHash > 0
}

// CheckIntegrity checks for not nil fields and negative value
func (tx *Transaction) CheckIntegrity() error {
	if tx.Signature == nil {
		return data.ErrNilSignature
	}
	if tx.Value == nil {
		return data.ErrNilValue
	}
	if tx.Value.Sign() < 0 {
		return data.ErrNegativeValue
	}
	if len(tx.RcvUserName) > core.MaxUserNameLength {
		return data.ErrInvalidUserNameLength
	}
	if len(tx.SndUserName) > core.MaxUserNameLength {
		return data.ErrInvalidUserNameLength
	}

	return nil
}
