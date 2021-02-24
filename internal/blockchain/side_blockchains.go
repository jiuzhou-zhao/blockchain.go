package blockchain

import (
	"errors"
	"fmt"

	"github.com/jiuzhou-zhao/blockchain.go/pkg/chainhash"
	"github.com/jiuzhou-zhao/go-fundamental/loge"
)

type ChainLooker interface {
	GetTXOChangeUtil(height int64) (deletedTx map[string]interface{}, uTx map[string][]TXOutput)
	GetUTXO(txID string, outIndex int) *TXOutput
}

type UTXO struct {
	VOut   int
	Amount int
}

type sideBlockChain struct {
	baseBucket int64 // 0: main chain
	mainHeight int64 // PreBLock Height on main chain
	blocks     []*Block
	baseSTXO   map[string][]int
	baseUTXO   map[string][]TXOutput
	sTXO       map[string][]int
	uTXO       map[string][]TXOutput
}

func newSideBlockChain(baseID, mainHeight int64, block *Block, sTXO map[string][]int,
	uTXO map[string][]TXOutput) *sideBlockChain {
	sb := &sideBlockChain{
		baseBucket: baseID,
		mainHeight: mainHeight,
		blocks:     []*Block{block},
		baseSTXO:   sTXO,
		baseUTXO:   uTXO,
		sTXO:       make(map[string][]int),
		uTXO:       make(map[string][]TXOutput),
	}
	for key, ints := range sTXO {
		sb.sTXO[key] = append(sb.sTXO[key], ints...)
	}
	for key, outputs := range uTXO {
		sb.uTXO[key] = append(sb.uTXO[key], outputs...)
	}
	sb.adjustUXTO(block)
	return sb
}

func (sb *sideBlockChain) adjustUXTO(block *Block) {
	adjustUXTOOut([]*Block{block}, sb.sTXO, sb.uTXO)
}

func adjustUXTOOut(blocks []*Block, sTXO map[string][]int, uTXO map[string][]TXOutput) {
	for _, block := range blocks {
		for _, transaction := range block.Transactions {
			if !transaction.IsCoinbase() {
				for _, input := range transaction.Vin {
					if vouts, ok := uTXO[input.Txid]; ok {
						for idx := 0; idx < len(vouts); idx++ {
							if vouts[idx].Index == input.Vout {
								vouts = append(vouts[:idx], vouts[idx+1:]...)
								break
							}
						}
						uTXO[input.Txid] = vouts
					} else {
						sTXO[input.Txid] = append(sTXO[input.Txid], input.Vout)
					}
				}
			}
			uTXO[transaction.TxID] = transaction.Vout
		}
	}
}

func (sb *sideBlockChain) GetLatestBlock() *Block {
	return sb.blocks[len(sb.blocks)-1]
}

func (sb *sideBlockChain) AddBlock(block *Block) int64 {
	block.Height = sb.blocks[len(sb.blocks)-1].Height + 1
	sb.blocks = append(sb.blocks, block)
	sb.adjustUXTO(block)
	return block.Height
}

func (sb *sideBlockChain) GetBlockByHash(h *chainhash.Hash) (*Block, int) {
	for idx, block := range sb.blocks {
		if block.Hash.IsEqual(h) {
			return block, idx
		}
	}
	return nil, 0
}

func (sb *sideBlockChain) GetTXO4Split(idx int) (sTXO map[string][]int, uTXO map[string][]TXOutput) {
	if idx < 0 {
		idx = len(sb.blocks) - 1
	}

	sTXO = make(map[string][]int)
	uTXO = make(map[string][]TXOutput)

	for key, ints := range sb.baseSTXO {
		sTXO[key] = append(sTXO[key], ints...)
	}
	for key, outputs := range sb.baseUTXO {
		uTXO[key] = append(uTXO[key], outputs...)
	}
	adjustUXTOOut(sb.blocks[:idx+1], sTXO, uTXO)
	return
}

type SideBlockChains struct {
	cl          ChainLooker
	idBase      int64
	blockChains map[int64]*sideBlockChain
	blockHashes map[chainhash.Hash]interface{}
}

func NewSideBlockChains(cl ChainLooker) *SideBlockChains {
	return &SideBlockChains{
		cl:          cl,
		idBase:      0,
		blockChains: make(map[int64]*sideBlockChain),
		blockHashes: make(map[chainhash.Hash]interface{}),
	}
}

func (sbs *SideBlockChains) BlockExists(h chainhash.Hash) bool {
	if _, ok := sbs.blockHashes[h]; ok {
		return true
	}
	return false
}

func (sbs *SideBlockChains) NewSortedBlocks(blocks []*Block, preBlockOnMain *Block) (int64, error) {
	if len(blocks) == 0 {
		return 0, nil
	}
	h := sbs.NewBlock(blocks[0], preBlockOnMain)
	if h <= 0 {
		return h, nil
	}
	lastHash := blocks[0].Hash
	for idx := 1; idx < len(blocks); idx++ {
		if !blocks[idx].PrevBlockHash.IsEqual(&lastHash) {
			return 0, errors.New("unsorted blocks")
		}
		lastHash = blocks[idx].Hash
		hCur := sbs.NewBlock(blocks[idx], nil)
		if hCur != h+1 {
			loge.Errorf(nil, "height check failed: %v, %v", h, hCur)
			break
		}
		h = hCur
	}
	return h, nil
}

func (sbs *SideBlockChains) verifyTxInput(input TXInput, deletedTxOnM map[string]interface{}, uTxOnM map[string][]TXOutput,
	sTXOOnS map[string][]int, uTXOOnS map[string][]TXOutput) (*TXOutput, error) {
	utxo := sbs.cl.GetUTXO(input.Txid, input.Vout)
	if utxo != nil {
		if v, ok := deletedTxOnM[input.Txid]; ok {
			return nil, fmt.Errorf("utxo txid int the future of main chain: %v", v)
		}
		if outputs, ok := sTXOOnS[input.Txid]; ok {
			for _, output := range outputs {
				if output == input.Vout {
					return nil, fmt.Errorf("utxo %s,%d has been consumed", input.Txid, input.Vout)
				}
			}
		}
		return utxo, nil
	}

	if outputs, ok := uTxOnM[input.Txid]; ok {
		for _, output := range outputs {
			if output.Index == input.Vout {
				return &output, nil
			}
		}
		return nil, fmt.Errorf("no utxo output %d for %s", input.Vout, input.Txid)
	}
	if outputs, ok := uTXOOnS[input.Txid]; ok {
		for _, output := range outputs {
			if output.Index == input.Vout {
				return &output, nil
			}
		}
		return nil, fmt.Errorf("no utxo output %d for %s", input.Vout, input.Txid)
	}
	return nil, fmt.Errorf("no utxo: %s, %d", input.Txid, input.Vout)
}

func (sbs *SideBlockChains) verifyBlock(block *Block, heightOnMC int64, sTXOOnS map[string][]int,
	uTXOOnS map[string][]TXOutput) error {
	deletedTxOnM, uTxOnM := sbs.cl.GetTXOChangeUtil(heightOnMC)

	for _, transaction := range block.Transactions {
		err := transaction.simpleVerify()
		if err != nil {
			return err
		}
		if transaction.IsCoinbase() {
			continue
		}
		inputAmount := 0
		outAmount := 0
		if !transaction.IsCoinbase() {
			for _, input := range transaction.Vin {
				utxo, err := sbs.verifyTxInput(input, deletedTxOnM, uTxOnM, sTXOOnS, uTXOOnS)
				if err != nil {
					return err
				}
				inputAmount += utxo.Value
			}
		}
		for _, output := range transaction.Vout {
			outAmount += output.Value
		}
		if inputAmount < outAmount {
			return fmt.Errorf("invalid amount: %v, %v", inputAmount, outAmount)
		}
	}
	return nil
}

func (sbs *SideBlockChains) NewBlock(block *Block, preBlockOnMain *Block) int64 {
	if preBlockOnMain != nil {
		block.Height = preBlockOnMain.Height + 1
		sbs.blockHashes[block.Hash] = true
		err := sbs.verifyBlock(block, preBlockOnMain.Height, nil, nil)
		if err != nil {
			loge.Errorf(nil, "verify block #%v failed: %v", block.Height, err)
			return 0
		}
		return sbs.newChain(0, preBlockOnMain.Height, block, nil, nil)
	}

	for id, chain := range sbs.blockChains {
		var chainTop bool
		var idx int
		var preBlock *Block

		if block.PrevBlockHash.IsEqual(&chain.GetLatestBlock().Hash) {
			chainTop = true
			idx = -1
			block.Height = chain.GetLatestBlock().Height + 1
		} else {
			preBlock, idx = chain.GetBlockByHash(&block.PrevBlockHash)
			if preBlock == nil {
				continue
			}
			block.Height = preBlock.Height + 1
		}

		sTXO, uTXO := chain.GetTXO4Split(idx)
		err := sbs.verifyBlock(block, chain.mainHeight, sTXO, uTXO)
		if err != nil {
			loge.Errorf(nil, "verify block #%v failed: %v", block.Height, err)
			return 0
		}
		sbs.blockHashes[block.Hash] = true
		if chainTop {
			return chain.AddBlock(block)
		}
		return sbs.newChain(id, chain.mainHeight, block, sTXO, uTXO)
	}

	return 0
}

func (sbs *SideBlockChains) newChain(baseID, mainHeight int64, block *Block, sTXO map[string][]int,
	uTXO map[string][]TXOutput) int64 {
	sbs.idBase++
	sbs.blockChains[sbs.idBase] = newSideBlockChain(baseID, mainHeight, block, sTXO, uTXO)
	return block.Height
}

func (sbs *SideBlockChains) getChainIDByTop(h *chainhash.Hash) int64 {
	for id, chain := range sbs.blockChains {
		if chain.GetLatestBlock().Hash.IsEqual(h) {
			return id
		}
	}
	return 0
}

func (sbs *SideBlockChains) SwitchMainChain(block *Block) ([]*Block, error) {
	bestID := sbs.getChainIDByTop(&block.Hash)
	if bestID <= 0 {
		return nil, errors.New("no chain to switch")
	}

	var blocks []*Block
	bucketID := bestID
	var preBlock *Block
	for {
		chain := sbs.blockChains[bucketID]
		saveFlag := 0

		currentBlocks := make([]*Block, 0)
		for idx := len(chain.blocks) - 1; idx >= 0; idx-- {
			if preBlock == nil || preBlock.PrevBlockHash.IsEqual(&chain.blocks[idx].Hash) {
				saveFlag = 1
			}
			if saveFlag == 0 {
				currentBlocks = append([]*Block{chain.blocks[idx]}, currentBlocks...)
			} else {
				delete(sbs.blockHashes, chain.blocks[idx].Hash)
				blocks = append([]*Block{chain.blocks[idx]}, blocks...)
			}
		}

		if len(currentBlocks) > 0 {
			chain.mainHeight = preBlock.Height - 1
			chain.blocks = currentBlocks
			chain.baseSTXO = nil
			chain.baseUTXO = nil
			chain.sTXO = make(map[string][]int)
			chain.uTXO = make(map[string][]TXOutput)
			adjustUXTOOut(chain.blocks, chain.sTXO, chain.uTXO)
		} else {
			delete(sbs.blockChains, bucketID)
		}

		bucketID = chain.baseBucket
		chain.baseBucket = 0
		preBlock = chain.blocks[0]
		if bucketID == 0 {
			break
		}
	}

	blocksByHash := make(map[chainhash.Hash]*Block)
	for _, b := range blocks {
		blocksByHash[b.Hash] = b
	}
	for id, chain := range sbs.blockChains {
		if chain.baseBucket == 0 {
			continue
		}
		if b, ok := blocksByHash[chain.blocks[0].PrevBlockHash]; ok {
			chain.baseBucket = 0
			chain.mainHeight = b.Height
			chain.baseSTXO = nil
			chain.baseUTXO = nil
			chain.sTXO = make(map[string][]int)
			chain.uTXO = make(map[string][]TXOutput)
			adjustUXTOOut(chain.blocks, chain.sTXO, chain.uTXO)
			sbs.blockChains[id] = chain
		}
	}
	return blocks, nil
}
