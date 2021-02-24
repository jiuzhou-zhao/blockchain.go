package cli

import (
	"fmt"
	"log"

	"github.com/jiuzhou-zhao/blockchain.go/pkg/utils"
)

func startNode(nodeID, minerAddress string) {
	fmt.Printf("Starting node %s\n", nodeID)
	if len(minerAddress) > 0 {
		if utils.IsValidAddress(minerAddress) {
			fmt.Println("Mining is on. Address to receive rewards: ", minerAddress)
		} else {
			log.Panic("Wrong miner address!")
		}
	}
	// StartServer(nodeID, minerAddress)
}
