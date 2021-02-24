package blockchain

import (
	"github.com/jiuzhou-zhao/blockchain.go/pkg/chainhash"
	"github.com/jiuzhou-zhao/bolt-client/pkg/db"
)

// Iterator is used to iterate over blockchain blocks.
type Iterator struct {
	tx          db.Tx
	currentHash chainhash.Hash
}

func newIterator(tx db.Tx, currentHash chainhash.Hash) *Iterator {
	return &Iterator{
		tx:          tx,
		currentHash: currentHash,
	}
}

// Next returns next block starting from the tip.
func (i *Iterator) Next() *Block {
	var block *Block

	if i.currentHash.IsEqual(&chainhash.ZeroHash) {
		return nil
	}

	b := i.tx.Bucket(blockBucketName)
	encodedBlock := b.Get([]byte(i.currentHash.String()))
	block = DeserializeBlock(encodedBlock)
	i.currentHash = block.PrevBlockHash

	return block
}
