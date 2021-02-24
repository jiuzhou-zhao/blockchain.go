package cli

import (
	"fmt"

	"github.com/jiuzhou-zhao/blockchain.go/internal/blockchain"
)

func reindexUTXO() {
	bcs, _ := blockchain.NewBlockChains()
	defer bcs.Close()

	err := bcs.ReindexUTXO()
	if err != nil {
		panic(err)
	}

	txIDs := make(map[string]interface{})
	bcs.ScanUTXO(nil, func(txID string, output blockchain.TXOutput) bool {
		txIDs[txID] = true
		return true
	})
	fmt.Printf("Done! There are %d transactions in the UTXO set.\n", len(txIDs))
}
