package blockchain

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"

	"github.com/jiuzhou-zhao/blockchain.go/pkg/chainhash"
	"github.com/jiuzhou-zhao/blockchain.go/pkg/utils"
	"github.com/jiuzhou-zhao/bolt-client/pkg/db"
	"github.com/jiuzhou-zhao/go-fundamental/loge"
)

const (
	// blockChainDataFile = "blockchain.data" .
	genesisBlockHash = "0000924ddc0e3c989c22ec6a63bc528267d866111322537ccdddda95126445ca"
	// nolint: lll
	genesisBlockData = "61ff9703010105426c6f636b01ff98000106010950726576426c6f636b01ff9a00010954696d657374616d7001040001054e6f6e6365010600010648656967687401040001044861736801ff9a00010c5472616e73616374696f6e7301ff9c00000014ff99010101044861736801ff9a0001060140000028ff9b020101195b5d2a626c6f636b636861696e2e5472616e73616374696f6e01ff9c0001ff8e00003bff8d0301010b5472616e73616374696f6e01ff8e000104010454784944010c00010356696e01ff92000104566f757401ff9600010152010c00000023ff91020101145b5d626c6f636b636861696e2e5458496e70757401ff920001ff9000004bff8f030101075458496e70757401ff90000105010454786964010c000104566f75740104000106416d6f756e7401040001095369676e6174757265010a0001065075624b6579010a00000024ff95020101155b5d626c6f636b636861696e2e54584f757470757401ff960001ff94000039ff930301010854584f757470757401ff940001030105496e646578010400010556616c7565010400010a5075624b657948617368010a000000fe0132ff980120000000000000000000000000000000000000000000000000000000000000000001fcc083ab9e01fee179010201200000ff924dffdc0e3cff98ff9c22ffec6a63ffbc52ff8267ffd866111322537cffcdffddffdaff95126445ffca01010140326636376164326562383666356639396564336136316132633135356262646238613437613131373066306662626330366535663265313563386139393232380101020103455468652054696d65732030332f4a616e2f32303039204368616e63656c6c6f72206f6e206272696e6b206f66207365636f6e64206261696c6f757420666f722062616e6b7300010102140114963820330f0d9b371fee9d1a5b12f77f6bbf942500012432656434333834652d383337322d343361612d623636652d3932623961323866383766390000"
)

var (
	blockBucketName  = []byte("blocks")
	heightBucketName = []byte("height")
	utxoBucketName   = []byte("utxo")
	txBucketName     = []byte("tx")

	currentHeightKeyOnBucket = []byte("height")
)

type BlockChains struct {
	db                db.DB
	latestBlock       *Block
	orphanedBlocks    map[chainhash.Hash]*Block
	orphanedPreHashes map[chainhash.Hash]chainhash.Hash
	sideChains        *SideBlockChains
}

func NewBlockChains() (chains *BlockChains, err error) {
	stg, err := NewDB()
	if err != nil {
		return
	}
	chains = &BlockChains{
		db:                stg,
		orphanedBlocks:    make(map[chainhash.Hash]*Block),
		orphanedPreHashes: make(map[chainhash.Hash]chainhash.Hash),
	}
	chains.sideChains = NewSideBlockChains(chains)
	err = chains.init()
	if err != nil {
		return
	}
	return
}

func (bcs *BlockChains) Close() {
	_ = bcs.db.Close()
}

// nolint: gocognit
func (bcs *BlockChains) init() error {
	return bcs.db.Update(func(tx db.Tx) error {
		blockBucket := tx.Bucket(blockBucketName)
		heightBucket := tx.Bucket(heightBucketName)
		txBucket := tx.Bucket(txBucketName)

		if blockBucket == nil || heightBucket == nil || txBucket == nil {
			_ = tx.DeleteBucket(blockBucketName)
			_ = tx.DeleteBucket(heightBucketName)
			_ = tx.DeleteBucket(txBucketName)

			blockBucket = nil
			heightBucket = nil
			txBucket = nil
		}
		if blockBucket == nil {
			_, err := tx.CreateBucket(blockBucketName)
			if err != nil {
				return err
			}
		}
		if heightBucket == nil {
			_, err := tx.CreateBucket(heightBucketName)
			if err != nil {
				return err
			}
		}
		if txBucket == nil {
			_, err := tx.CreateBucket(txBucketName)
			if err != nil {
				return err
			}
		}

		bcs.latestBlock = bcs.getLatestBlockOnTx(tx)
		if bcs.latestBlock == nil {
			err := bcs.createGenesisBlockOnTx(tx)
			if err != nil {
				return err
			}
			bcs.latestBlock = bcs.getLatestBlockOnTx(tx)
			if bcs.latestBlock == nil {
				return errors.New("no block on chain")
			}
			err = bcs.ReindexUTXOOnTx(tx)
			if err != nil {
				return err
			}
		}

		if bcs.latestBlock == nil {
			return errors.New("no genesis block")
		}

		return nil
	})
}

func (bcs *BlockChains) getLatestBlockOnTx(tx db.Tx) *Block {
	_, key := bcs.getLastHeightOnTx(tx)
	if key == nil {
		return nil
	}
	blockBucket := tx.Bucket(blockBucketName)
	return DeserializeBlock(blockBucket.Get(key))
}

func (bcs *BlockChains) getLastHeightOnTx(tx db.Tx) (heightKey, hashKey []byte) {
	heightBucket := tx.Bucket(heightBucketName)
	heightKey = heightBucket.Get(currentHeightKeyOnBucket)
	if heightKey == nil {
		return
	}
	hashKey = heightBucket.Get(heightKey)
	return
}

func (bcs *BlockChains) createGenesisBlockOnTx(tx db.Tx) error {
	blockBucket := tx.Bucket(blockBucketName)
	heightBucket := tx.Bucket(heightBucketName)

	plaintGenesisHash, err := hex.DecodeString(genesisBlockHash)
	if err != nil {
		return fmt.Errorf("hex genesisBlockHash failed: %w", err)
	}
	plaintGenesisData, err := hex.DecodeString(genesisBlockData)
	if err != nil {
		return fmt.Errorf("hex genesisBlockData failed: %w", err)
	}
	h, err := chainhash.NewHash(plaintGenesisHash)
	if err != nil {
		return fmt.Errorf("invalid genesisBlockHash: %w", err)
	}
	err = blockBucket.Put([]byte(h.String()), plaintGenesisData)
	if err != nil {
		return fmt.Errorf("put key failed: %w", err)
	}

	block := DeserializeBlock(plaintGenesisData)
	if block == nil {
		return errors.New("decode genesis block failed")
	}
	err = heightBucket.Put(block.HeightS(), []byte(h.String()))
	if err != nil {
		return fmt.Errorf("put key failed: %w", err)
	}

	err = heightBucket.Put(currentHeightKeyOnBucket, block.HeightS())
	if err != nil {
		return fmt.Errorf("put key failed: %w", err)
	}

	return nil
}

func (bcs *BlockChains) GetLatestBlock() *Block {
	return bcs.latestBlock
}

func (bcs *BlockChains) AddBlock(block *Block) error {
	if bcs.blockExists(block.Hash) {
		return errors.New("block exists")
	}
	err := block.Check()
	if err != nil {
		return err
	}
	return bcs.processNewBlock(block)
}

func (bcs *BlockChains) addSortedBlocks(blocks []*Block) error {
	if blocks[0].PrevBlockHash.IsEqual(&bcs.latestBlock.Hash) {
		return bcs.add2MainBlocks(blocks)
	}

	h, _ := bcs.sideChains.NewSortedBlocks(blocks, bcs.getBlockOnMainChain(&blocks[0].PrevBlockHash))
	if h <= bcs.latestBlock.Height {
		return nil
	}

	return bcs.switchChain(blocks[len(blocks)-1])
}

// nolint: gocognit
func (bcs *BlockChains) switchChain(block *Block) error {
	blocks, err := bcs.sideChains.SwitchMainChain(block)
	if err != nil {
		return err
	}
	if len(blocks) == 0 {
		return errors.New("switch no blocks")
	}

	switchedBlocks := make([]*Block, 0)

	var preBlock *Block
	err = bcs.db.Update(func(tx db.Tx) error {
		blockBucket := tx.Bucket(blockBucketName)
		heightBucket := tx.Bucket(heightBucketName)
		txBucket := tx.Bucket(txBucketName)

		var errDB error

		preBlock = DeserializeBlock(blockBucket.Get([]byte(blocks[0].PrevBlockHash.String())))
		if preBlock == nil {
			return errors.New("get block on storage failed")
		}

		h := preBlock.Height + 1
		for ; h <= bcs.latestBlock.Height; h++ {
			hKey := []byte(strconv.FormatInt(h, 10))
			key := heightBucket.Get(hKey)
			block := DeserializeBlock(blockBucket.Get(key))
			if block == nil {
				return errors.New("switch failed")
			}
			switchedBlocks = append(switchedBlocks, block)
			errDB = blockBucket.Delete(key)
			if errDB != nil {
				return errDB
			}
			errDB = heightBucket.Delete(hKey)
			if errDB != nil {
				return errDB
			}
			for _, transaction := range block.Transactions {
				errDB = txBucket.Delete([]byte(transaction.TxID))
				if errDB != nil {
					return errDB
				}
			}
		}

		uTXOBucket := tx.Bucket(utxoBucketName)
		for idx := len(switchedBlocks) - 1; idx >= 0; idx-- {
			for _, transaction := range switchedBlocks[idx].Transactions {
				errDB = uTXOBucket.Delete([]byte(transaction.TxID))
				if errDB != nil {
					return errDB
				}
				if transaction.IsCoinbase() {
					continue
				}
				for _, input := range transaction.Vin {
					uTXOs, _ := DeserializeOutputs(uTXOBucket.Get([]byte(input.Txid)))
					if uTXOs == nil {
						uTXOs = &TXOutputs{}
					}
					uTXOs.Outputs = append(uTXOs.Outputs, TXOutput{
						Index:      input.Vout,
						Value:      input.Amount,
						PubKeyHash: utils.HashPubKey(input.PubKey),
					})

					errDB = uTXOBucket.Put([]byte(input.Txid), uTXOs.Serialize())
					if errDB != nil {
						return errDB
					}
				}
			}
		}

		bcs.latestBlock = preBlock
		errDB = heightBucket.Put(currentHeightKeyOnBucket, preBlock.HeightS())
		if errDB != nil {
			return fmt.Errorf("put latest height info failed: %w", errDB)
		}
		return bcs.add2MainBlocksOnTx(tx, blocks, preBlock)
	})
	if err != nil {
		return err
	}
	// TODO 安全性 - 写db？
	_, _ = bcs.sideChains.NewSortedBlocks(switchedBlocks, preBlock)
	return nil
}

func (bcs *BlockChains) blockExists(h chainhash.Hash) bool {
	if _, ok := bcs.orphanedBlocks[h]; ok {
		return true
	}
	if bcs.getBlockOnMainChain(&h) != nil {
		return true
	}
	if bcs.sideChains.BlockExists(h) {
		return true
	}
	return false
}

func (bcs *BlockChains) getBlockOnMainChain(hash *chainhash.Hash) (block *Block) {
	_ = bcs.db.View(func(tx db.Tx) error {
		bucket := tx.Bucket(blockBucketName)
		block = DeserializeBlock(bucket.Get([]byte(hash.String())))
		return nil
	})
	return
}

func (bcs *BlockChains) verifyBlockTransactionsOnMainChain(blocks []*Block) error {
	for _, block := range blocks {
		for _, transaction := range block.Transactions {
			vc, err := bcs.GetCond4TransactionVerify(transaction)
			if err != nil {
				return err
			}
			err = transaction.Verify(vc)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (bcs *BlockChains) add2MainBlocks(blocks []*Block) error {
	err := bcs.verifyBlockTransactionsOnMainChain(blocks)
	if err != nil {
		loge.Errorf(nil, "verify blocks failed: %v", err)
		return err
	}
	return bcs.db.Update(func(tx db.Tx) error {
		return bcs.add2MainBlocksOnTx(tx, blocks, bcs.latestBlock)
	})
}

func (bcs *BlockChains) add2MainBlocksOnTx(tx db.Tx, blocks []*Block, preBlock *Block) error {
	if len(blocks) == 0 {
		return nil
	}
	if preBlock == nil {
		return errors.New("add to main chain: invalid pre block param")
	}

	if !blocks[0].PrevBlockHash.IsEqual(&preBlock.Hash) {
		return errors.New("add to main chain: unknown block")
	}

	lastHash := preBlock.Hash
	lastHeight := preBlock.Height

	blockBucket := tx.Bucket(blockBucketName)
	heightBucket := tx.Bucket(heightBucketName)
	txBucket := tx.Bucket(txBucketName)

	var errDB error
	for _, block := range blocks {
		if !block.PrevBlockHash.IsEqual(&lastHash) {
			return errors.New("add to main chain: invalid block")
		}
		lastHeight++
		lastHash = block.Hash
		block.Height = lastHeight

		for _, transaction := range block.Transactions {
			if transaction.TxID == "" {
				loge.Fatal(nil, "no tx id")
			}
		}
		for _, transaction := range block.Transactions {
			if txBucket.Get([]byte(transaction.TxID)) != nil {
				loge.Fatalf(nil, "exists tx: %s", transaction.TxID)
			}
			errDB = txBucket.Put([]byte(transaction.TxID), block.HeightS())
			if errDB != nil {
				return fmt.Errorf("%w", errDB)
			}
		}
		errDB = blockBucket.Put([]byte(block.Hash.String()), block.Serialize())
		if errDB != nil {
			return fmt.Errorf("%w", errDB)
		}
		errDB = heightBucket.Put(block.HeightS(), []byte(block.Hash.String()))
		if errDB != nil {
			return fmt.Errorf("%w", errDB)
		}
		errDB = bcs.UpdateUTXOInTx(block, tx)
		if errDB != nil {
			return fmt.Errorf("%w", errDB)
		}
	}

	latestBlock := blocks[len(blocks)-1]
	errDB = heightBucket.Put(currentHeightKeyOnBucket, latestBlock.HeightS())
	if errDB != nil {
		return errDB
	}
	bcs.latestBlock = latestBlock
	return nil
}

func (bcs *BlockChains) processNewBlock(block *Block) error {
	blocks := []*Block{
		block,
	}

	preHash := block.PrevBlockHash
	for {
		preBlock, ok := bcs.orphanedBlocks[preHash]
		if ok {
			blocks = append([]*Block{preBlock}, blocks...)
			preHash = preBlock.PrevBlockHash
			continue
		}

		if bcs.getBlockOnMainChain(&preHash) != nil {
			break
		}
		if bcs.sideChains.BlockExists(preHash) {
			break
		}

		bcs.orphanedBlocks[block.Hash] = block
		bcs.orphanedPreHashes[block.PrevBlockHash] = block.Hash
		return nil
	}

	for {
		var ub *Block
		if h, ok := bcs.orphanedPreHashes[blocks[len(blocks)-1].Hash]; ok {
			ub = bcs.orphanedBlocks[h]
		}
		if ub == nil {
			break
		}
		blocks = append(blocks, ub)
	}

	//

	for _, b := range blocks {
		delete(bcs.orphanedBlocks, b.Hash)
		delete(bcs.orphanedPreHashes, b.PrevBlockHash)
	}

	return bcs.addSortedBlocks(blocks)
}

func (bcs *BlockChains) ScanBlocks(fnOb func(*Block) error) (err error) {
	if fnOb == nil {
		return errors.New("no ob")
	}
	err = bcs.db.View(func(tx db.Tx) error {
		iter := bcs.IteratorOnTx(tx)
		for {
			block := iter.Next()
			if block == nil {
				break
			}
			err = fnOb(block)
			if err != nil {
				break
			}
		}
		return err
	})
	return
}

func (bcs *BlockChains) IteratorOnTx(tx db.Tx) *Iterator {
	return newIterator(tx, bcs.latestBlock.Hash)
}

// FindTransaction finds a transaction by its ID.
func (bcs *BlockChains) FindTransaction(txID string) (tx *Transaction, err error) {
	err = bcs.db.View(func(txDb db.Tx) error {
		iter := bcs.IteratorOnTx(txDb)
		for {
			block := iter.Next()
			if block == nil {
				break
			}

			for idx, txInBlock := range block.Transactions {
				if txInBlock.TxID == txID {
					tx = block.Transactions[idx]
					break
				}
			}
		}
		return nil
	})

	if tx == nil && err == nil {
		err = errors.New("no transaction")
	}

	return
}

// FindUTXOOnTX finds all unspent transaction outputs and returns transactions with spent outputs removed.
// nolint: gocognit
func (bcs *BlockChains) FindUTXOOnTX(tx db.Tx) map[string]TXOutputs {
	uTXOs := make(map[string]TXOutputs)
	sTXOs := make(map[string][]int)
	bci := bcs.IteratorOnTx(tx)

	for {
		block := bci.Next()
		if block == nil {
			break
		}

		for _, tx := range block.Transactions {
			txID := tx.TxID

		Outputs:
			for outIdx, out := range tx.Vout {
				if sTXOs[txID] != nil {
					for _, spentOutIdx := range sTXOs[txID] {
						if spentOutIdx == outIdx {
							continue Outputs
						}
					}
				}

				outs := uTXOs[txID]
				outs.Outputs = append(outs.Outputs, out)
				uTXOs[txID] = outs
			}

			if !tx.IsCoinbase() {
				for _, in := range tx.Vin {
					sTXOs[in.Txid] = append(sTXOs[in.Txid], in.Vout)
				}
			}
		}
	}

	return uTXOs
}

func (bcs *BlockChains) GetBestHeight() int64 {
	return bcs.latestBlock.Height
}

func (bcs *BlockChains) GetBlock(hash *chainhash.Hash) *Block {
	return bcs.getBlockOnMainChain(hash)
}

func (bcs *BlockChains) GetBlockHashes() []chainhash.Hash {
	var blocks []chainhash.Hash

	_ = bcs.db.View(func(tx db.Tx) error {
		bci := bcs.IteratorOnTx(tx)

		for {
			block := bci.Next()
			if block == nil {
				break
			}

			blocks = append(blocks, block.Hash)
		}
		return nil
	})

	return blocks
}

func (bcs *BlockChains) FindTransactions(txIDs []string) (map[string]Transaction, error) {
	prevTXs := make(map[string]Transaction)
	txIDMap := make(map[string]interface{})
	for _, txID := range txIDs {
		txIDMap[txID] = true
	}

	_ = bcs.db.View(func(tx db.Tx) error {
		iter := bcs.IteratorOnTx(tx)
		for {
			block := iter.Next()
			if block == nil {
				break
			}

			for _, tx := range block.Transactions {
				if _, ok := txIDMap[tx.TxID]; ok {
					prevTXs[tx.TxID] = *tx
					break
				}
			}
		}
		return nil
	})

	if len(prevTXs) != len(txIDs) {
		return nil, errors.New("transaction is not found")
	}

	return prevTXs, nil
}

func (bcs *BlockChains) GetCond4TransactionVerify(transaction *Transaction) (*TransactionVerifyCond, error) {
	if transaction == nil {
		return nil, nil
	}
	outputs := make(map[string][]TXOutput)
	if !transaction.IsCoinbase() {
		for _, input := range transaction.Vin {
			op := bcs.GetUTXO(input.Txid, input.Vout)
			if op == nil {
				return nil, fmt.Errorf("no input: %s,%d", input.Txid, input.Vout)
			}
			outputs[input.Txid] = append(outputs[input.Txid], *op)
		}
	}
	return &TransactionVerifyCond{Outputs: outputs}, nil
}
