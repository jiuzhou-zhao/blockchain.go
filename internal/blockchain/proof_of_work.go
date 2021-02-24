package blockchain

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"math"
	"math/big"

	"github.com/jiuzhou-zhao/blockchain.go/pkg/utils"
)

var maxNonce = math.MaxUint32

const targetBits = 16

// ProofOfWork represents a proof-of-work.
type ProofOfWork struct {
	block  *Block
	target *big.Int
}

// NewProofOfWork builds and returns a ProofOfWork.
func NewProofOfWork(b *Block) *ProofOfWork {
	target := big.NewInt(1)
	target.Lsh(target, uint(256-targetBits))

	pow := &ProofOfWork{b, target}

	return pow
}

func (pow *ProofOfWork) prepareData(nonce uint32) []byte {
	data := bytes.Join(
		[][]byte{
			pow.block.PrevBlockHash[:],
			pow.block.HashTransactions()[:],
			utils.IntToHex(pow.block.Timestamp),
			utils.IntToHex(int64(targetBits)),
			utils.IntToHex(int64(nonce)),
		},
		[]byte{},
	)

	return data
}

// Run performs a proof-of-work.
func (pow *ProofOfWork) Run() (uint32, []byte) {
	var hashInt big.Int
	var hash [32]byte
	var nonce uint32

	fmt.Printf("Mining a new block")
	for nonce < uint32(maxNonce) {
		data := pow.prepareData(nonce)

		hash = sha256.Sum256(data)
		if math.Remainder(float64(nonce), 100000) == 0 {
			fmt.Printf("\r%x", hash)
		}
		hashInt.SetBytes(hash[:])

		if hashInt.Cmp(pow.target) == -1 {
			break
		} else {
			nonce++
		}
	}
	fmt.Print("\n\n")

	return nonce, hash[:]
}

// Validate validates block's PoW.
func (pow *ProofOfWork) Validate() bool {
	var hashInt big.Int

	data := pow.prepareData(pow.block.Nonce)
	hash := sha256.Sum256(data)
	hashInt.SetBytes(hash[:])

	isValid := hashInt.Cmp(pow.target) == -1

	return isValid
}
