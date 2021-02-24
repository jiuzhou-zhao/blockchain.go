package main

import (
	"bytes"
	"crypto/elliptic"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"log"

	"github.com/jiuzhou-zhao/blockchain.go/internal/blockchain"
	"github.com/jiuzhou-zhao/blockchain.go/pkg/chainhash"
	"github.com/jiuzhou-zhao/blockchain.go/pkg/utils"
)

const (
	version             = byte(0x00)
	genesisCoinbaseData = "The Times 03/Jan/2009 Chancellor on brink of second bailout for banks"
)

func main() {
	gob.Register(elliptic.P256())

	priKey, pubKey := utils.NewKeyPair()

	var priKeyBuf bytes.Buffer
	encoder := gob.NewEncoder(&priKeyBuf)
	err := encoder.Encode(priKey)
	if err != nil {
		log.Panic(err)
	}
	var pubKeyBuf bytes.Buffer
	encoder = gob.NewEncoder(&pubKeyBuf)
	err = encoder.Encode(pubKey)
	if err != nil {
		log.Panic(err)
	}

	address := utils.Pubkey2Address(pubKeyBuf.Bytes(), version)

	genesis := blockchain.MineBlock([]*blockchain.Transaction{blockchain.NewCoinbaseTX(address, genesisCoinbaseData)},
		chainhash.ZeroHash)
	genesis.Height = 1

	fmt.Println("-------------------------")
	fmt.Printf("wallet private key: %s\n", hex.EncodeToString(priKeyBuf.Bytes()))
	fmt.Printf("wallet public key: %s\n", hex.EncodeToString(pubKeyBuf.Bytes()))
	fmt.Printf("wallet address: %s\n", address)
	fmt.Printf("genesis block hash: %s\n", hex.EncodeToString(genesis.Hash[:]))
	fmt.Printf("genesis block data: %s\n", hex.EncodeToString(genesis.Serialize()))
}
