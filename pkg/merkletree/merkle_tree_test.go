package merkletree

import (
	"testing"

	"github.com/jiuzhou-zhao/blockchain.go/pkg/chainhash"
	"github.com/stretchr/testify/assert"
)

func TestNewMerkleTree(t *testing.T) {
	hs := BuildMerkleTreeStore([]*chainhash.Hash{
		{0x01},
		{0x02},
		{0x03},
		{0x04},
		{0x05},
	})
	assert.True(t, len(hs) > 0)

	hs2 := NewMerkleTree([]*chainhash.Hash{
		{0x01},
		{0x02},
		{0x03},
		{0x04},
		{0x05},
	})

	assert.NotNil(t, hs2)

	assert.True(t, hs[len(hs)-1].IsEqual(&hs2.RootNode.Data))
}

func Test_nextPowerOfTwo(t *testing.T) {
	for idx := 0; idx < 10; idx++ {
		t.Logf("%d -> %d\n", idx, nextPowerOfTwo(idx))
	}
}
