package cli

import (
	"fmt"
	"log"

	"github.com/jiuzhou-zhao/blockchain.go/internal/blockchain"
	"github.com/jiuzhou-zhao/blockchain.go/pkg/utils"
)

func getBalance(address string) {
	if !utils.IsValidAddress(address) {
		log.Panic("ERROR: Address is not valid")
	}
	bcs, _ := blockchain.NewBlockChains()
	defer bcs.Close()

	balance := 0

	pubKeyHash, err := utils.Address2PubkeyHash(address)
	if err != nil {
		log.Panic(err)
	}
	bcs.ScanUTXO(pubKeyHash, func(txID string, output blockchain.TXOutput) bool {
		balance += output.Value
		return true
	})

	fmt.Printf("Balance of '%s': %d\n", address, balance)
}
