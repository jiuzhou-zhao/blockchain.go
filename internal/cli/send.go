package cli

import (
	"fmt"
	"log"

	"github.com/jiuzhou-zhao/blockchain.go/internal/blockchain"
	"github.com/jiuzhou-zhao/blockchain.go/pkg/utils"
)

func send(from, to string, amount int, mineNow bool) {
	if !utils.IsValidAddress(from) {
		log.Panic("ERROR: Sender address is not valid")
	}
	if !utils.IsValidAddress(to) {
		log.Panic("ERROR: Recipient address is not valid")
	}

	bcs, _ := blockchain.NewBlockChains()
	defer bcs.Close()

	wallets, err := blockchain.NewWallets()
	if err != nil {
		log.Panic(err)
	}
	wallet := wallets.GetWallet(from)

	tx, err := blockchain.NewUTXOTransaction(&wallet, to, amount, nil, bcs)
	if err != nil {
		log.Panic(err)
	}

	if mineNow {
		cbTx := blockchain.NewCoinbaseTX(from, "")
		txs := []*blockchain.Transaction{cbTx, tx}

		err = bcs.AddBlock(blockchain.MineBlock(txs, bcs.GetLatestBlock().Hash))
		if err != nil {
			log.Panic(err)
		}
	} else {
		// sendTx(knownNodes[0], tx)
	}

	fmt.Println("Success!")
}
