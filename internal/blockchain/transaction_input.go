package blockchain

import (
	"bytes"

	"github.com/jiuzhou-zhao/blockchain.go/pkg/utils"
)

// TXInput represents a transaction input.
type TXInput struct {
	Txid      string
	Vout      int
	Amount    int
	Signature []byte
	PubKey    []byte
}

// UsesKey checks whether the address initiated the transaction.
func (in *TXInput) UsesKey(pubKeyHash []byte) bool {
	lockingHash := utils.HashPubKey(in.PubKey)

	return bytes.Equal(lockingHash, pubKeyHash)
}
