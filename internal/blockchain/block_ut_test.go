package blockchain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBLockCheck(t *testing.T) {
	var block *Block
	assert.NotNil(t, block.Check())

	block = &Block{}
	assert.NotNil(t, block.Check())

	block.Transactions = append(block.Transactions, &Transaction{})
	assert.NotNil(t, block.Check())

	block.Transactions = []*Transaction{NewCoinbaseTX("1EhHbToNa5vkBZrGoD97ThNTffqVQNS9cd", "")}
	assert.NotNil(t, block.Check())
}
