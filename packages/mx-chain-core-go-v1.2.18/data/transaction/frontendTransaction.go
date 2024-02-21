package transaction

// FrontendTransaction represents the DTO used in transaction signing/validation.
type FrontendTransaction struct {
	Nonce             uint64 `json:"nonce,omitempty"`
	Value             string `json:"value"`
	Receiver          string `json:"receiver"`
	Sender            string `json:"sender"`
	SenderUsername    []byte `json:"senderUsername,omitempty"`
	ReceiverUsername  []byte `json:"receiverUsername,omitempty"`
	GasPrice          uint64 `json:"gasPrice"`
	GasLimit          uint64 `json:"gasLimit"`
	Data              []byte `json:"data,omitempty"`
	Signature         string `json:"signature,omitempty"`
	ChainID           string `json:"chainID"`
	Version           uint32 `json:"version"`
	Options           uint32 `json:"options,omitempty"`
	GuardianAddr      string `json:"guardian,omitempty"`
	GuardianSignature string `json:"guardianSignature,omitempty"`
//! -------------------- NEW CODE --------------------
	MigrationNonce    uint64 `json:"migrationNonce,omitempty"`
	SenderShard       uint32 `json:"senderShard,omitempty"`
	ReceiverShard     uint32 `json:"receiverShard,omitempty"`
	SignerPubKey      []byte `json:"signerPubKey,omitempty"`
	OriginalTxHash        []byte `json:"originalTxHash,omitempty"`
	OriginalMiniBlockHash []byte `json:"originalMiniBlockHash,omitempty"`
//! ---------------- END OF NEW CODE -----------------
}