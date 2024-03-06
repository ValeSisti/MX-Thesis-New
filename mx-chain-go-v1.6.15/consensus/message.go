//go:generate protoc -I=. -I=$GOPATH/src -I=$GOPATH/src/github.com/multiversx/protobuf/protobuf  --gogoslick_out=. message.proto
package consensus

import "github.com/multiversx/mx-chain-core-go/core"

// MessageType specifies what type of message was received
type MessageType int

// NewConsensusMessage creates a new Message object
func NewConsensusMessage(
	blHeaderHash []byte,
	signatureShare []byte,
	body []byte,
	header []byte,
	pubKey []byte,
	sig []byte,
	msg int,
	roundIndex int64,
	chainID []byte,
	pubKeysBitmap []byte,
	aggregateSignature []byte,
	leaderSignature []byte,
	currentPid core.PeerID,
	invalidSigners []byte,
	//! -------------------- NEW CODE --------------------
	problematicMBsInfo []byte,
	//! ---------------- END OF NEW CODE -----------------	
) *Message {
	return &Message{
		BlockHeaderHash:    blHeaderHash,
		SignatureShare:     signatureShare,
		Body:               body,
		Header:             header,
		PubKey:             pubKey,
		Signature:          sig,
		MsgType:            int64(msg),
		RoundIndex:         roundIndex,
		ChainID:            chainID,
		PubKeysBitmap:      pubKeysBitmap,
		AggregateSignature: aggregateSignature,
		LeaderSignature:    leaderSignature,
		OriginatorPid:      currentPid.Bytes(),
		InvalidSigners:     invalidSigners,
		//! -------------------- NEW CODE --------------------
		ProblematicMBsForCurrentRound: problematicMBsInfo,
		//! ---------------- END OF NEW CODE -----------------		
	}
}


//TODO: VEDI SE IL PROBLEMA è IL CONSENSU MESSAGE, PERCHé IL PROBLEMA STA A RIGA 178 DI consensus/spos/worker.go, che referenzia consensus.Message
//TODO: non vorrei che ci fosse qualche componente da qualche altra parte che lavora con i consensus.Message e che crea dei problemi
//TODO: però perché muore solo il validator03??? 


//TODO: il problema c'è quando Processo il blocco? Perché per validator02 che è il leader va tutto bene, stessa cosa per gli altri due nodi,
//TODO: che non hanno delle txs "from me"(?)(?)(?)


//TODO: CONFERMO, MUORE SOLO CHI NON è IL LEADER
//TODO: ANCHE SE NON MANDO TXS??? -> si, muore nel momento in cui viene inserita nel blocco la AMT