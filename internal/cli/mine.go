package cli

import (
	"fmt"
	"log"

	"github.com/jiuzhou-zhao/blockchain.go/internal/blockchain"
	"github.com/jiuzhou-zhao/blockchain.go/pkg/utils"
)

func mine(address string) {
	if !utils.IsValidAddress(address) {
		log.Panic("ERROR: Address is not valid")
	}
	bcs, _ := blockchain.NewBlockChains()
	defer bcs.Close()

	latestBlock := bcs.GetLatestBlock()
	if latestBlock == nil {
		panic("no block")
	}

	//
	//
	//
	block1 := blockchain.MineBlock([]*blockchain.Transaction{
		blockchain.NewCoinbaseTX(address, "onlyMine"),
	}, latestBlock.Hash)
	err := bcs.AddBlock(block1)
	if err != nil {
		panic(err)
	}
	fmt.Println("money on ", address)
}
