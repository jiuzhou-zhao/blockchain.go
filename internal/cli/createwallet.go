package cli

import (
	"fmt"

	"github.com/jiuzhou-zhao/blockchain.go/internal/blockchain"
)

func createWallet() {
	wallets, _ := blockchain.NewWallets()
	address := wallets.CreateWallet()
	wallets.SaveToFile()

	fmt.Printf("Your new address: %s\n", address)
}
