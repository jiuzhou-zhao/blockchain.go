package cli

import (
	"fmt"
	"strconv"

	"github.com/jiuzhou-zhao/blockchain.go/internal/blockchain"
)

func printChain() {
	bcs, _ := blockchain.NewBlockChains()
	defer bcs.Close()

	err := bcs.ScanBlocks(func(block *blockchain.Block) error {
		fmt.Printf("============ Block %x ============\n", block.Hash)
		fmt.Printf("Height: %d\n", block.Height)
		fmt.Printf("Prev. block: %x\n", block.PrevBlockHash)
		pow := blockchain.NewProofOfWork(block)
		fmt.Printf("PoW: %s\n\n", strconv.FormatBool(pow.Validate()))
		for _, tx := range block.Transactions {
			fmt.Println(tx)
		}
		fmt.Printf("\n\n")
		return nil
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
