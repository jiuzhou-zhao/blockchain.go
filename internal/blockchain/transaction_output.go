package blockchain

import (
	"bytes"
	"encoding/gob"
	"log"

	"github.com/jiuzhou-zhao/blockchain.go/pkg/utils"
)

// TXOutput represents a transaction output.
type TXOutput struct {
	Index      int
	Value      int
	PubKeyHash []byte
}

// Lock signs the output.
func (out *TXOutput) Lock(address string) {
	pubKeyHash, err := utils.Address2PubkeyHash(address)
	if err != nil {
		log.Fatal("ERROR: invalid address")
	}
	out.PubKeyHash = pubKeyHash
}

// IsLockedWithKey checks if the output can be used by the owner of the pubkey.
func (out *TXOutput) IsLockedWithKey(pubKeyHash []byte) bool {
	return bytes.Equal(out.PubKeyHash, pubKeyHash)
}

// NewTXOutput create a new TXOutput.
func NewTXOutput(index, value int, address string) *TXOutput {
	txo := &TXOutput{index, value, nil}
	txo.Lock(address)

	return txo
}

// TXOutputs collects TXOutput.
type TXOutputs struct {
	Outputs []TXOutput
}

// Serialize serializes TXOutputs.
func (outs TXOutputs) Serialize() []byte {
	var buff bytes.Buffer

	enc := gob.NewEncoder(&buff)
	err := enc.Encode(outs)
	if err != nil {
		log.Panic(err)
	}

	return buff.Bytes()
}

// DeserializeOutputs deserializes TXOutputs.
func DeserializeOutputs(data []byte) (*TXOutputs, error) {
	var outputs TXOutputs

	dec := gob.NewDecoder(bytes.NewReader(data))
	err := dec.Decode(&outputs)
	if err != nil {
		return nil, err
	}

	return &outputs, nil
}
