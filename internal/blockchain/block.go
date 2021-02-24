package blockchain

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/jiuzhou-zhao/blockchain.go/pkg/chainhash"
	"github.com/jiuzhou-zhao/blockchain.go/pkg/merkletree"
)

// Block represents a block in the blockchain.
type Block struct {
	PrevBlockHash chainhash.Hash
	Timestamp     int64
	Nonce         uint32
	Height        int64

	Hash         chainhash.Hash
	Transactions []*Transaction
}

func NewBlock(transactions []*Transaction, prevBlockHash chainhash.Hash) *Block {
	return &Block{
		PrevBlockHash: prevBlockHash,
		Timestamp:     time.Now().Unix(),
		Transactions:  transactions,
	}
}

// MineBlock creates and returns Block.
func MineBlock(transactions []*Transaction, prevBlockHash chainhash.Hash) *Block {
	block := NewBlock(transactions, prevBlockHash)
	block.Mine()
	return block
}

func (b *Block) Mine() {
	pow := NewProofOfWork(b)
	nonce, hash := pow.Run()

	h, _ := chainhash.NewHash(hash)
	b.Hash = *h
	b.Nonce = nonce
}

func (b *Block) HeightS() []byte {
	return []byte(strconv.FormatInt(b.Height, 10))
}

// HashTransactions returns a hash of the transactions in the block.
func (b *Block) HashTransactions() *chainhash.Hash {
	transactionHashes := make([]*chainhash.Hash, 0, len(b.Transactions))

	for _, tx := range b.Transactions {
		transactionHashes = append(transactionHashes, tx.Hash())
	}

	return merkletree.CalcMerkleTreeRootHash(transactionHashes)
}

func (b *Block) Check() error {
	if b == nil {
		return errors.New("empty block")
	}
	if len(b.Transactions) == 0 {
		return errors.New("no transactions")
	}
	if !b.Transactions[0].IsCoinbase() {
		return errors.New("not start with coin base")
	}

	for _, transaction := range b.Transactions {
		err := transaction.simpleVerify()
		if err != nil {
			return fmt.Errorf("%w", err)
		}
	}

	if !NewProofOfWork(b).Validate() {
		return errors.New("pow error")
	}

	return nil
}

// Serialize serializes the block.
func (b *Block) Serialize() []byte {
	var result bytes.Buffer

	_ = gob.NewEncoder(&result).Encode(b)

	return result.Bytes()
}

// DeserializeBlock deserializes a block.
func DeserializeBlock(d []byte) *Block {
	var block Block

	err := gob.NewDecoder(bytes.NewReader(d)).Decode(&block)
	if err != nil {
		return nil
	}

	return &block
}
