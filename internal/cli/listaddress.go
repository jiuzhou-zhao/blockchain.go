package cli

import (
	"fmt"
	"log"

	"github.com/jiuzhou-zhao/blockchain.go/internal/blockchain"
)

func listAddresses() {
	wallets, err := blockchain.NewWallets()
	if err != nil {
		log.Panic(err)
	}
	addresses := wallets.GetAddresses()

	for _, address := range addresses {
		fmt.Println(address)
	}
}
