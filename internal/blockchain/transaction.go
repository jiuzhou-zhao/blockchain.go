package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/gob"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math/big"
	"strings"
	"time"

	"github.com/jiuzhou-zhao/blockchain.go/pkg/chainhash"
	"github.com/jiuzhou-zhao/blockchain.go/pkg/utils"
	uuid "github.com/satori/go.uuid"
)

const subsidy = 10

// Transaction represents a Bitcoin transaction.
type Transaction struct {
	TxID string
	Vin  []TXInput
	Vout []TXOutput

	R string
}

// IsCoinbase checks whether the transaction is coinbase.
func (tx Transaction) IsCoinbase() bool {
	return len(tx.Vin) == 1 && len(tx.Vin[0].Txid) == 0 && tx.Vin[0].Vout == -1
}

// Serialize returns a serialized Transaction.
func (tx Transaction) Serialize() []byte {
	var encoded bytes.Buffer

	enc := gob.NewEncoder(&encoded)
	err := enc.Encode(tx)
	if err != nil {
		log.Panic(err)
	}

	return encoded.Bytes()
}

func (tx *Transaction) simpleVerify() error {
	if tx.R == "" {
		return errors.New("no R")
	}
	if tx.TxID == "" {
		return errors.New("no tx id")
	}
	if len(tx.Vin) <= 0 {
		return errors.New("no inputs")
	}

	if !tx.IsCoinbase() {
		for _, input := range tx.Vin {
			if len(input.PubKey) == 0 || len(input.Signature) == 0 {
				return errors.New("no pubkey or signature")
			}
		}
	}

	for _, output := range tx.Vout {
		if output.Value <= 0 {
			return fmt.Errorf("utxo %d no value", output.Index)
		}
		if len(output.PubKeyHash) == 0 {
			return fmt.Errorf("no pubkey hash on output %d", output.Index)
		}
	}

	return nil
}

// Hash returns the hash of the Transaction.
func (tx *Transaction) Hash() *chainhash.Hash {
	if tx.R == "" {
		tx.R = uuid.NewV4().String()
	}
	txCopy := *tx
	txCopy.TxID = ""

	h := chainhash.HashH(txCopy.Serialize())
	return &h
}

func (tx *Transaction) DefSign(bcs *BlockChains, priKey ecdsa.PrivateKey) error {
	if bcs == nil {
		return errors.New("invalid input")
	}
	cond, err := bcs.GetCond4TransactionVerify(tx)
	if err != nil {
		return err
	}
	if cond == nil {
		return errors.New("invalid inputs")
	}
	return tx.Sign(priKey, cond)
}

// Sign signs each input of a Transaction.
func (tx *Transaction) Sign(privKey ecdsa.PrivateKey, vc *TransactionVerifyCond) error {
	if tx.IsCoinbase() {
		return nil
	}

	if vc == nil {
		return errors.New("no vc")
	}

	for idx := 0; idx < len(tx.Vin); idx++ {
		utxo := vc.Get(tx.Vin[idx].Txid, tx.Vin[idx].Vout)
		if utxo == nil {
			return fmt.Errorf("utxo not exists: %s,%d", tx.Vin[idx].Txid, tx.Vin[idx].Vout)
		}
		tx.Vin[idx].Amount = utxo.Value
	}

	txCopy := tx.TrimmedCopy()

	for inID, vin := range txCopy.Vin {
		txCopy.Vin[inID].Signature = nil
		txCopy.Vin[inID].PubKey = vc.Get(vin.Txid, vin.Vout).PubKeyHash

		dataToSign := fmt.Sprintf("%x\n", txCopy)

		r, s, err := ecdsa.Sign(rand.Reader, &privKey, []byte(dataToSign))
		if err != nil {
			return fmt.Errorf("%w", err)
		}
		signature := append(r.Bytes(), s.Bytes()...)

		tx.Vin[inID].Signature = signature
		txCopy.Vin[inID].PubKey = nil
	}

	return nil
}

// String returns a human-readable representation of a transaction.
func (tx Transaction) String() string {
	lines := make([]string, 0, len(tx.Vin)+len(tx.Vout)+1)
	lines = append(lines, fmt.Sprintf("--- Transaction %x:", tx.TxID))

	for i, input := range tx.Vin {
		lines = append(lines, fmt.Sprintf("     Input %d:", i))
		lines = append(lines, fmt.Sprintf("       TxID:      %x", input.Txid))
		lines = append(lines, fmt.Sprintf("       Out:       %d", input.Vout))
		lines = append(lines, fmt.Sprintf("       Signature: %x", input.Signature))
		lines = append(lines, fmt.Sprintf("       PubKey:    %x", input.PubKey))
	}

	for i, output := range tx.Vout {
		lines = append(lines, fmt.Sprintf("     Output %d:", i))
		lines = append(lines, fmt.Sprintf("       Value:  %d", output.Value))
		lines = append(lines, fmt.Sprintf("       Script: %x", output.PubKeyHash))
	}

	return strings.Join(lines, "\n")
}

// TrimmedCopy creates a trimmed copy of Transaction to be used in signing.
func (tx *Transaction) TrimmedCopy() Transaction {
	inputs := make([]TXInput, 0, len(tx.Vin))
	for _, vin := range tx.Vin {
		inputs = append(inputs, TXInput{vin.Txid, vin.Vout, 0, nil, nil})
	}

	outputs := make([]TXOutput, 0, len(tx.Vout))
	outputs = append(outputs, tx.Vout...)

	txCopy := Transaction{tx.TxID, inputs, outputs, tx.R}

	return txCopy
}

type TransactionVerifyCond struct {
	Outputs map[string][]TXOutput
}

func (tv *TransactionVerifyCond) Get(txID string, idx int) *TXOutput {
	if outputs, ok := tv.Outputs[txID]; ok {
		for _, output := range outputs {
			if output.Index == idx {
				return &output
			}
		}
	}
	return nil
}

// Verify verifies signatures of Transaction inputs.
func (tx *Transaction) Verify(vc *TransactionVerifyCond) error {
	if tx.IsCoinbase() {
		return nil
	}
	if vc == nil {
		return errors.New("no condition transactions")
	}

	txCopy := tx.TrimmedCopy()
	curve := elliptic.P256()

	for inID, vin := range tx.Vin {
		utxo := vc.Get(vin.Txid, vin.Vout)
		if utxo == nil {
			return fmt.Errorf("utxo %s,%d not exists", vin.Txid, vin.Vout)
		}
		if utxo.Value != vin.Amount {
			return fmt.Errorf("amount mismatch: %v - %v", utxo.Value, vin.Amount)
		}
		txCopy.Vin[inID].Signature = nil
		txCopy.Vin[inID].PubKey = utxo.PubKeyHash

		r := big.Int{}
		s := big.Int{}
		sigLen := len(vin.Signature)
		r.SetBytes(vin.Signature[:(sigLen / 2)])
		s.SetBytes(vin.Signature[(sigLen / 2):])

		x := big.Int{}
		y := big.Int{}
		keyLen := len(vin.PubKey)
		x.SetBytes(vin.PubKey[:(keyLen / 2)])
		y.SetBytes(vin.PubKey[(keyLen / 2):])

		dataToVerify := fmt.Sprintf("%x\n", txCopy)

		rawPubKey := ecdsa.PublicKey{Curve: curve, X: &x, Y: &y}
		if !ecdsa.Verify(&rawPubKey, []byte(dataToVerify), &r, &s) {
			return errors.New("hash verify failed")
		}
		txCopy.Vin[inID].PubKey = nil
	}

	return nil
}

// NewCoinbaseTX creates a new coinbase transaction.
func NewCoinbaseTX(to, data string) *Transaction {
	if data == "" {
		data = uuid.NewV4().String() + "-" + time.Now().String()
	}

	txin := TXInput{"", -1, 0, nil, []byte(data)}
	txout := NewTXOutput(0, subsidy, to)
	tx := Transaction{
		TxID: "",
		Vin:  []TXInput{txin},
		Vout: []TXOutput{*txout},
	}
	tx.TxID = hex.EncodeToString(tx.Hash()[:])

	return &tx
}

// NewUTXOTransaction creates a new transaction.
func NewUTXOTransaction(wallet *Wallet, to string, amount int, blockUTXO UTXOFilter,
	blockChains *BlockChains) (*Transaction, error) {
	if wallet == nil || to == "" || amount <= 0 || blockChains == nil {
		return nil, errors.New("invalid input")
	}
	pubKeyHash := utils.HashPubKey(wallet.PublicKey)
	acc, uTXOs := blockChains.FindSpendableOutputs(pubKeyHash, amount, blockUTXO)

	if acc < amount {
		return nil, errors.New("no enough amount")
	}

	return NewUTXOTransactionEx(wallet.PublicKey, wallet.GetAddress(),
		uTXOs, []TXOutput{*NewTXOutput(0, amount, to)})
}

func NewTransaction(pubKey []byte, address string, amount int, to string, blockUTXO UTXOFilter,
	blockChains *BlockChains) (*Transaction, error) {
	if len(pubKey) == 0 || address == "" || amount <= 0 || blockChains == nil {
		return nil, errors.New("invalid input")
	}
	acc, uTXOs := blockChains.FindSpendableOutputs(utils.HashPubKey(pubKey), amount, blockUTXO)
	if acc < amount {
		return nil, errors.New("no enough amount")
	}
	return NewUTXOTransactionEx(pubKey, address, uTXOs, []TXOutput{*NewTXOutput(0, amount, to)})
}

// NewUTXOTransaction creates a new transaction.
func NewUTXOTransactionEx(pubKey []byte, address string, inputs map[string][]TXOutput,
	outputs []TXOutput) (*Transaction, error) {
	amount := 0
	txInputs := make([]TXInput, 0, len(inputs))
	for txID, input := range inputs {
		for _, i := range input {
			txInputs = append(txInputs, TXInput{
				Txid:      txID,
				Vout:      i.Index,
				Signature: nil,
				PubKey:    pubKey,
			})
			amount += i.Value
		}
	}

	idx := 0
	var output TXOutput
	for idx, output = range outputs {
		output.Index = idx
		amount -= output.Value
	}
	if amount < 0 {
		return nil, errors.New("no enough amount")
	}
	if amount > 0 {
		outputs = append(outputs, *NewTXOutput(idx+1, amount, address))
	}

	tx := Transaction{"", txInputs, outputs, ""}
	tx.TxID = hex.EncodeToString(tx.Hash()[:])
	return &tx, nil
}

// DeserializeTransaction deserializes a transaction.
func DeserializeTransaction(data []byte) Transaction {
	var transaction Transaction

	decoder := gob.NewDecoder(bytes.NewReader(data))
	err := decoder.Decode(&transaction)
	if err != nil {
		log.Panic(err)
	}

	return transaction
}
