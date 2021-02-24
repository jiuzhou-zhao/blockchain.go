package blockchain

import (
	"fmt"
	"log"
	"strconv"

	"github.com/boltdb/bolt"
	"github.com/jiuzhou-zhao/blockchain.go/pkg/utils"
	"github.com/jiuzhou-zhao/bolt-client/pkg/db"
	"github.com/jiuzhou-zhao/go-fundamental/loge"
)

func (bcs *BlockChains) FindUTXOByTxVinOnTX(tx *bolt.Tx, txInput TXInput) (txOutput *TXOutput) {
	b := tx.Bucket(utxoBucketName)
	outpus, err := DeserializeOutputs(b.Get([]byte(txInput.Txid)))
	if err != nil {
		return nil
	}
	for _, output := range outpus.Outputs {
		if output.Index == txInput.Vout {
			return &output
		}
	}
	return nil
}

func (bcs *BlockChains) ScanUTXO(pubkeyHash []byte, cb func(txID string, output TXOutput) bool) {
	if cb == nil {
		return
	}
	err := bcs.db.View(func(tx db.Tx) error {
		b := tx.Bucket(utxoBucketName)
		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			outs, err := DeserializeOutputs(v)
			if err != nil {
				continue
			}

			for _, out := range outs.Outputs {
				if len(pubkeyHash) == 0 || out.IsLockedWithKey(pubkeyHash) {
					if !cb(string(k), out) {
						return nil
					}
				}
			}
		}

		return nil
	})
	if err != nil {
		loge.Errorf(nil, "db failed: %v", err)
	}
}

type UTXOFilter func(txID string, output TXOutput) bool

// FindSpendableOutputs finds and returns unspent outputs to reference in inputs.
func (bcs *BlockChains) FindSpendableOutputs(pubkeyHash []byte, amount int,
	blockUTXO UTXOFilter) (int, map[string][]TXOutput) {
	unspentOutputs := make(map[string][]TXOutput)
	accumulated := 0

	bcs.ScanUTXO(pubkeyHash, func(txID string, output TXOutput) bool {
		if blockUTXO != nil && blockUTXO(txID, output) {
			return true
		}
		accumulated += output.Value
		unspentOutputs[txID] = append(unspentOutputs[txID], output)
		return accumulated < amount
	})

	return accumulated, unspentOutputs
}

// FindUTXOByPubKeyHash finds UTXO for a public key hash.
func (bcs *BlockChains) FindUTXOByPubKeyHash(pubKeyHash []byte) []TXOutput {
	var uTXOs []TXOutput

	bcs.ScanUTXO(pubKeyHash, func(txID string, output TXOutput) bool {
		uTXOs = append(uTXOs, output)
		return true
	})

	return uTXOs
}

// Update updates the UTXO set with transactions from the Block
// The Block is considered to be the tip of a blockchain.
func (bcs *BlockChains) UpdateUTXOInTx(block *Block, tx db.Tx) error {
	b := tx.Bucket(utxoBucketName)

	for _, tx := range block.Transactions {
		newOutputs := TXOutputs{}
		newOutputs.Outputs = append(newOutputs.Outputs, tx.Vout...)

		err := b.Put([]byte(tx.TxID), newOutputs.Serialize())
		if err != nil {
			return err
		}

		if tx.IsCoinbase() {
			continue
		}
		for _, vin := range tx.Vin {
			updatedOuts := TXOutputs{}
			outsBytes := b.Get([]byte(vin.Txid))
			outs, err := DeserializeOutputs(outsBytes)
			if err != nil {
				continue
			}

			for _, out := range outs.Outputs {
				if out.Index != vin.Vout {
					updatedOuts.Outputs = append(updatedOuts.Outputs, out)
				}
			}

			if len(updatedOuts.Outputs) == 0 {
				err := b.Delete([]byte(vin.Txid))
				if err != nil {
					return err
				}
			} else {
				err := b.Put([]byte(vin.Txid), updatedOuts.Serialize())
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

//  ReindexUTXOOnTx rebuilds the UTXO set
func (bcs *BlockChains) ReindexUTXO() error {
	return bcs.db.Update(func(tx db.Tx) error {
		return bcs.ReindexUTXOOnTx(tx)
	})
}

//  ReindexUTXOOnTx rebuilds the UTXO set
func (bcs *BlockChains) ReindexUTXOOnTx(tx db.Tx) error {
	_ = tx.DeleteBucket(utxoBucketName)

	_, err := tx.CreateBucket(utxoBucketName)
	if err != nil {
		return err
	}

	uTXOs := bcs.FindUTXOOnTX(tx)

	b := tx.Bucket(utxoBucketName)

	for txID, outs := range uTXOs {
		err = b.Put([]byte(txID), outs.Serialize())
		if err != nil {
			return err
		}
	}

	return nil
}

func (bcs *BlockChains) getKeyByHeightOnTX(tx db.Tx, height int64) string {
	b := tx.Bucket(heightBucketName)
	d := b.Get([]byte(strconv.FormatInt(height, 10)))
	if d == nil {
		return ""
	}
	return string(d)
}

func (bcs *BlockChains) getBlockByHeightOnTX(tx db.Tx, height int64) *Block {
	bk := bcs.getKeyByHeightOnTX(tx, height)
	if bk == "" {
		return nil
	}
	b := tx.Bucket(blockBucketName)
	return DeserializeBlock(b.Get([]byte(bk)))
}

func (bcs *BlockChains) GetTXOChangeUtil(height int64) (deletedTx map[string]interface{}, uTx map[string][]TXOutput) {
	deletedTx = make(map[string]interface{})
	uTx = make(map[string][]TXOutput)
	_ = bcs.db.View(func(tx db.Tx) error {
		heightK, _ := bcs.getLastHeightOnTx(tx)
		maxHeight, err := strconv.ParseInt(string(heightK), 10, 64)
		if err != nil {
			return err
		}
		for idx := height + 1; idx <= maxHeight; idx++ {
			block := bcs.getBlockByHeightOnTX(tx, idx)
			for _, transaction := range block.Transactions {
				deletedTx[transaction.TxID] = height
				if !transaction.IsCoinbase() {
					for _, input := range transaction.Vin {
						if _, ok := deletedTx[input.Txid]; ok {
							continue
						}
						uTx[input.Txid] = append(uTx[input.Txid], TXOutput{
							Index:      input.Vout,
							Value:      input.Amount,
							PubKeyHash: input.PubKey,
						})
					}
				}
			}
		}
		return nil
	})
	return
}

func (bcs *BlockChains) GetUTXO(txID string, outIndex int) (output *TXOutput) {
	_ = bcs.db.View(func(tx db.Tx) error {
		b := tx.Bucket(utxoBucketName)
		d := b.Get([]byte(txID))
		if d == nil {
			c := b.Cursor()

			for k, _ := c.First(); k != nil; k, _ = c.Next() {
				fmt.Println(string(k))
			}
		}
		ots, err := DeserializeOutputs(d)
		if err != nil {
			return err
		}
		for oi, ot := range ots.Outputs {
			if ot.Index == outIndex {
				output = &ots.Outputs[oi]
				break
			}
		}
		return nil
	})

	return output
}

func (bcs *BlockChains) GetBalance(address string) int {
	balance := 0

	pubKeyHash, err := utils.Address2PubkeyHash(address)
	if err != nil {
		log.Panic(err)
	}
	bcs.ScanUTXO(pubKeyHash, func(txID string, output TXOutput) bool {
		balance += output.Value
		return true
	})

	return balance
}
