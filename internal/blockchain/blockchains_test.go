package blockchain

import (
	"crypto/ecdsa"
	"fmt"
	"testing"

	"github.com/jiuzhou-zhao/blockchain.go/pkg/utils"
	"github.com/stretchr/testify/assert"
)

type testWallet struct {
	priKey ecdsa.PrivateKey
	pubKey []byte
}

func (tw *testWallet) Address() string {
	return utils.Pubkey2Address(tw.pubKey, version)
}

func newTestWallet() *testWallet {
	priKey, pubKey := utils.NewKeyPair()
	return &testWallet{
		priKey: priKey,
		pubKey: pubKey,
	}
}

func reInitBlockWithNewWallet(t *testing.T) (*BlockChains, *testWallet) {
	_ = DBRebuild4Debug()

	bcs, err := NewBlockChains()
	assert.Nil(t, err)
	assert.NotNil(t, bcs)

	wallet := newTestWallet()

	latestBlock := bcs.GetLatestBlock()
	assert.NotNil(t, latestBlock)

	txCoinbase := NewCoinbaseTX(wallet.Address(), "block1*")
	err = txCoinbase.DefSign(bcs, wallet.priKey)
	assert.Nil(t, err)

	block1 := MineBlock([]*Transaction{txCoinbase}, latestBlock.Hash)
	err = bcs.AddBlock(block1)
	assert.Nil(t, err)

	latestBlock = bcs.GetLatestBlock()
	assert.NotNil(t, latestBlock)
	assert.True(t, latestBlock.Hash.IsEqual(&block1.Hash))
	assert.True(t, bcs.GetBestHeight() == 2)
	assert.True(t, bcs.GetBalance(wallet.Address()) == 10)

	return bcs, wallet
}

func giveHeACoinbaseMoney(t *testing.T, bcs *BlockChains, address string) {
	txCoinbase := NewCoinbaseTX(address, "4coinbase*")

	block2 := MineBlock([]*Transaction{txCoinbase}, bcs.GetLatestBlock().Hash)
	err := bcs.AddBlock(block2)
	assert.Nil(t, err)
}

func TestBlockChains_AddBlock(t *testing.T) {
	bcs, wallet := reInitBlockWithNewWallet(t)
	defer bcs.Close()

	wallet2 := newTestWallet()

	txCoinbase := NewCoinbaseTX(wallet.Address(), "block2*")
	err := txCoinbase.DefSign(bcs, wallet.priKey)
	assert.Nil(t, err)

	tx, err := NewTransaction(wallet.pubKey, wallet.Address(), 4, wallet2.Address(), nil, bcs)
	assert.Nil(t, err)
	err = tx.DefSign(bcs, wallet.priKey)
	assert.Nil(t, err)

	block2 := MineBlock([]*Transaction{txCoinbase, tx}, bcs.GetLatestBlock().Hash)
	err = bcs.AddBlock(block2)
	assert.Nil(t, err)
	assert.True(t, bcs.GetBalance(wallet.Address()) == 16)
	assert.True(t, bcs.GetBalance(wallet2.Address()) == 4)

	latestBlock := bcs.GetLatestBlock()
	assert.NotNil(t, latestBlock)
	assert.True(t, latestBlock.Hash.IsEqual(&block2.Hash))
	assert.True(t, bcs.GetBestHeight() == 3)

	err = bcs.AddBlock(block2)
	assert.NotNil(t, err)

	txCoinbase = NewCoinbaseTX(wallet.Address(), "block1*")
	tx, err = NewTransaction(wallet.pubKey, wallet.Address(), 1, wallet2.Address(), nil, bcs)
	assert.Nil(t, err)

	block3 := MineBlock([]*Transaction{txCoinbase, tx}, latestBlock.Hash)
	err = bcs.AddBlock(block3)
	assert.NotNil(t, err)
}

func TestBlockChains_AddBlock_Orphaned(t *testing.T) {
	bcs, wallet := reInitBlockWithNewWallet(t)
	defer bcs.Close()

	giveHeACoinbaseMoney(t, bcs, wallet.Address())
	assert.True(t, bcs.GetBestHeight() == 3)
	assert.True(t, bcs.GetBalance(wallet.Address()) == 20)

	giveHeACoinbaseMoney(t, bcs, wallet.Address())
	assert.True(t, bcs.GetBestHeight() == 4)
	assert.True(t, bcs.GetBalance(wallet.Address()) == 30)

	wallet2 := newTestWallet()

	//
	//
	//

	tx, err := NewTransaction(wallet.pubKey, wallet.Address(), 4, wallet2.Address(), nil, bcs)
	assert.Nil(t, err)
	err = tx.DefSign(bcs, wallet.priKey)
	assert.Nil(t, err)

	block := MineBlock([]*Transaction{NewCoinbaseTX(wallet.Address(), "block2*"), tx}, bcs.GetLatestBlock().Hash)

	//
	//
	//
	tx2, err := NewTransaction(wallet.pubKey, wallet.Address(), 1, wallet2.Address(),
		func(txID string, output TXOutput) bool {
			for _, input := range tx.Vin {
				if input.Txid == txID {
					return true
				}
			}
			return false
		}, bcs)
	assert.Nil(t, err)
	err = tx2.DefSign(bcs, wallet.priKey)
	assert.Nil(t, err)

	block2 := MineBlock([]*Transaction{NewCoinbaseTX(wallet.Address(), "block2*"), tx2}, block.Hash)

	//
	//
	//
	tx3, err := NewTransaction(wallet.pubKey, wallet.Address(), 1, wallet2.Address(),
		func(txID string, output TXOutput) bool {
			for _, input := range tx.Vin {
				if input.Txid == txID {
					return true
				}
			}
			for _, input := range tx2.Vin {
				if input.Txid == txID {
					return true
				}
			}
			return false
		}, bcs)
	assert.Nil(t, err)
	err = tx3.DefSign(bcs, wallet.priKey)
	assert.Nil(t, err)

	block3 := MineBlock([]*Transaction{NewCoinbaseTX(wallet.Address(), "block2*"), tx3}, block2.Hash)

	//
	//
	//
	err = bcs.AddBlock(block3)
	assert.Nil(t, err)
	err = bcs.AddBlock(block2)
	assert.Nil(t, err)
	err = bcs.AddBlock(block)
	assert.Nil(t, err)
	assert.True(t, bcs.GetBestHeight() == 7)
	assert.True(t, bcs.GetBalance(wallet.Address()) == 54)
	t.Log(bcs.GetBalance(wallet.Address()))
}

// nolint: funlen
func TestBlockChains_AddBlock_SideChain(t *testing.T) {
	/*
		main
			G	[01]10		[02]10		[03]10-4
		side
										[11]10-3	[12]10
	*/
	_ = DBRebuild4Debug()
	bcs, err := NewBlockChains()
	assert.Nil(t, err)
	assert.NotNil(t, bcs)
	defer bcs.Close()

	wallet := newTestWallet()
	wallet2 := newTestWallet()

	//
	//
	//
	block01 := MineBlock([]*Transaction{NewCoinbaseTX(wallet.Address(), "block01*")},
		bcs.GetLatestBlock().Hash)
	err = bcs.AddBlock(block01)
	assert.Nil(t, err)
	h01 := block01.Hash
	t.Log(h01)

	block02 := MineBlock([]*Transaction{NewCoinbaseTX(wallet.Address(), "block02*")},
		bcs.GetLatestBlock().Hash)
	err = bcs.AddBlock(block02)
	assert.Nil(t, err)
	assert.True(t, bcs.GetBestHeight() == 3)
	assert.True(t, bcs.GetBalance(wallet.Address()) == 20)

	//
	//
	//
	tx03, err := NewTransaction(wallet.pubKey, wallet.Address(), 4, wallet2.Address(), nil, bcs)
	assert.Nil(t, err)
	err = tx03.DefSign(bcs, wallet.priKey)
	assert.Nil(t, err)

	block03 := MineBlock([]*Transaction{NewCoinbaseTX(wallet.Address(), "block03*"), tx03},
		bcs.GetLatestBlock().Hash)
	err = bcs.AddBlock(block03)
	assert.Nil(t, err)
	assert.True(t, bcs.GetBestHeight() == 4)
	assert.True(t, bcs.GetBalance(wallet.Address()) == 26)
	t.Log(bcs.GetBalance(wallet.Address()))

	//
	//
	//

	inputs := make(map[string][]TXOutput)
	outputs := make([]TXOutput, 0)
	inputs[block02.Transactions[0].TxID] = append(inputs[block02.Transactions[0].TxID], TXOutput{
		Index:      0,
		Value:      block02.Transactions[0].Vout[0].Value,
		PubKeyHash: block02.Transactions[0].Vout[0].PubKeyHash,
	})
	outputs = append(outputs, TXOutput{
		Index:      0,
		Value:      3,
		PubKeyHash: utils.HashPubKey(wallet2.pubKey),
	})
	tx04, err := NewUTXOTransactionEx(wallet.pubKey, wallet.Address(), inputs, outputs)
	assert.Nil(t, err)
	condOutputs := make(map[string][]TXOutput)
	condOutputs[block02.Transactions[0].TxID] = append(condOutputs[block02.Transactions[0].TxID], TXOutput{
		Index:      0,
		Value:      block02.Transactions[0].Vout[0].Value,
		PubKeyHash: block02.Transactions[0].Vout[0].PubKeyHash,
	})
	cond := &TransactionVerifyCond{
		Outputs: condOutputs,
	}
	err = tx04.Sign(wallet.priKey, cond)
	assert.Nil(t, err)

	block11 := MineBlock([]*Transaction{NewCoinbaseTX(wallet.Address(), "block04*"), tx04}, block02.Hash)
	err = bcs.AddBlock(block11)
	assert.Nil(t, err)
	assert.True(t, bcs.GetBestHeight() == 4)

	block12 := MineBlock([]*Transaction{NewCoinbaseTX(wallet.Address(), "block12*")}, block11.Hash)
	err = bcs.AddBlock(block12)
	assert.Nil(t, err)
	assert.True(t, bcs.GetBestHeight() == 5)
	assert.True(t, bcs.GetLatestBlock().Hash.IsEqual(&block12.Hash))
	t.Log(bcs.GetBestHeight())
	assert.True(t, bcs.GetBalance(wallet.Address()) == 37)
	t.Log(bcs.GetBalance(wallet.Address()))

	fnPrintUTXO := func(b *Block, id string) {
		fmt.Println(id + "--------------------")
		for _, transaction := range b.Transactions {
			fmt.Printf("txID: %v\n", transaction.TxID)
			for _, output := range transaction.Vout {
				fmt.Printf("%d %v %v\n", output.Index, output.Value, output.PubKeyHash)
			}
		}
	}
	fnPrintUTXO(block01, "01")
	fnPrintUTXO(block02, "02")
	fnPrintUTXO(block03, "03")
	fnPrintUTXO(block11, "11")
	fnPrintUTXO(block12, "12")
	fmt.Println("UTXO===========")
	bcs.ScanUTXO(utils.HashPubKey(wallet.pubKey), func(txID string, output TXOutput) bool {
		fmt.Printf("%v %v %v\n", txID, output.Index, output.Value)
		return true
	})
}

// nolint: funlen
func TestBlockChains_AddBlock_SideChain2(t *testing.T) {
	/*
		main	G	[01]10	[02]10	[03]10	[04]10
		side				[11]10-1[12]10-3[13]10
									[21]10
											[31]10	[32]10-4
									[41]10	[42]10-7
		---
		main	G	[1]10	[11]10-1[21]10	[31]10	[32]10-4
		side
							[02]10	[03]10	[04]10
									[12]10-3[13]10
							[41]10	[42]10-7
		---
		side									[14]10-1[15]10-2
	*/
	_ = DBRebuild4Debug()
	bcs, err := NewBlockChains()
	assert.Nil(t, err)
	assert.NotNil(t, bcs)
	defer bcs.Close()

	wallet := newTestWallet()
	wallet2 := newTestWallet()

	//
	//
	//
	block01 := MineBlock([]*Transaction{NewCoinbaseTX(wallet.Address(), "block01*")},
		bcs.GetLatestBlock().Hash)
	err = bcs.AddBlock(block01)
	assert.Nil(t, err)
	h01 := block01.Hash

	block02 := MineBlock([]*Transaction{NewCoinbaseTX(wallet.Address(), "block02*")},
		bcs.GetLatestBlock().Hash)
	err = bcs.AddBlock(block02)
	assert.Nil(t, err)
	h02 := block02.Hash

	block03 := MineBlock([]*Transaction{NewCoinbaseTX(wallet.Address(), "block03*")},
		bcs.GetLatestBlock().Hash)
	err = bcs.AddBlock(block03)
	assert.Nil(t, err)
	h03 := block03.Hash

	block04 := MineBlock([]*Transaction{NewCoinbaseTX(wallet.Address(), "block04*")},
		bcs.GetLatestBlock().Hash)
	err = bcs.AddBlock(block04)
	assert.Nil(t, err)
	h04 := block04.Hash

	//
	//
	//
	fnNewPayTransaction := func(payTransaction *Transaction, payAmount int) *Transaction {
		inputs := make(map[string][]TXOutput)
		outputs := make([]TXOutput, 0)
		inputs[payTransaction.TxID] = append(inputs[payTransaction.TxID], TXOutput{
			Index:      0,
			Value:      payTransaction.Vout[0].Value,
			PubKeyHash: payTransaction.Vout[0].PubKeyHash,
		})
		outputs = append(outputs, TXOutput{
			Index:      0,
			Value:      payAmount,
			PubKeyHash: utils.HashPubKey(wallet2.pubKey),
		})
		newTx, errI := NewUTXOTransactionEx(wallet.pubKey, wallet.Address(), inputs, outputs)
		assert.Nil(t, errI)
		condOutputs := make(map[string][]TXOutput)
		condOutputs[payTransaction.TxID] = append(condOutputs[payTransaction.TxID], TXOutput{
			Index:      0,
			Value:      payTransaction.Vout[0].Value,
			PubKeyHash: payTransaction.Vout[0].PubKeyHash,
		})
		cond := &TransactionVerifyCond{
			Outputs: condOutputs,
		}
		errI = newTx.Sign(wallet.priKey, cond)
		assert.Nil(t, errI)
		return newTx
	}

	block11 := MineBlock([]*Transaction{
		NewCoinbaseTX(wallet.Address(), "block11*"),
		fnNewPayTransaction(block01.Transactions[0], 1),
	},
		h01)
	err = bcs.AddBlock(block11)
	assert.Nil(t, err)
	h11 := block11.Hash

	block12 := MineBlock([]*Transaction{
		NewCoinbaseTX(wallet.Address(), "block12*"),
		fnNewPayTransaction(block11.Transactions[0], 3),
	}, h11)
	err = bcs.AddBlock(block12)
	assert.Nil(t, err)
	h12 := block12.Hash

	block13 := MineBlock([]*Transaction{NewCoinbaseTX(wallet.Address(), "block13*")}, h12)
	err = bcs.AddBlock(block13)
	assert.Nil(t, err)
	h13 := block13.Hash

	block21 := MineBlock([]*Transaction{NewCoinbaseTX(wallet.Address(), "block21*")}, h11)
	err = bcs.AddBlock(block21)
	assert.Nil(t, err)
	h21 := block21.Hash

	block41 := MineBlock([]*Transaction{NewCoinbaseTX(wallet.Address(), "block41*")}, h11)
	err = bcs.AddBlock(block41)
	assert.Nil(t, err)
	h41 := block41.Hash

	block42 := MineBlock([]*Transaction{
		NewCoinbaseTX(wallet.Address(), "block42*"),
		fnNewPayTransaction(block41.Transactions[0], 7),
	}, h41)
	err = bcs.AddBlock(block42)
	assert.Nil(t, err)
	h42 := block42.Hash

	block31 := MineBlock([]*Transaction{NewCoinbaseTX(wallet.Address(), "block31*")}, h21)
	err = bcs.AddBlock(block31)
	assert.Nil(t, err)
	h31 := block31.Hash

	assert.True(t, bcs.GetBestHeight() == 5)
	assert.True(t, bcs.GetBalance(wallet.Address()) == 40)
	t.Log(bcs.GetBestHeight())
	t.Log(bcs.GetBalance(wallet.Address()))

	block32 := MineBlock([]*Transaction{
		NewCoinbaseTX(wallet.Address(), "block32*"),
		fnNewPayTransaction(block21.Transactions[0], 4),
	}, h31)
	err = bcs.AddBlock(block32)
	assert.Nil(t, err)
	h32 := block32.Hash

	assert.True(t, bcs.GetBestHeight() == 6)
	assert.True(t, bcs.GetBalance(wallet.Address()) == 45)
	t.Log(bcs.GetBestHeight())
	t.Log(bcs.GetBalance(wallet.Address()))

	block14 := MineBlock([]*Transaction{
		NewCoinbaseTX(wallet.Address(), "block14*"),
		fnNewPayTransaction(block13.Transactions[0], 1),
	}, h13)
	err = bcs.AddBlock(block14)
	assert.Nil(t, err)
	h14 := block14.Hash

	block15 := MineBlock([]*Transaction{
		NewCoinbaseTX(wallet.Address(), "block15*"),
		fnNewPayTransaction(block14.Transactions[0], 2),
	}, h14)
	err = bcs.AddBlock(block15)
	assert.Nil(t, err)
	h15 := block15.Hash

	assert.True(t, bcs.GetBestHeight() == 7)
	assert.True(t, bcs.GetBalance(wallet.Address()) == 53)
	t.Log(bcs.GetBestHeight())
	t.Log(bcs.GetBalance(wallet.Address()))

	t.Log(h01, h02, h03, h04, h11, h12, h13, h21, h41, h42, h31, h32, h14, h15)
}
