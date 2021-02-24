package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/jiuzhou-zhao/blockchain.go/pkg/utils"
)

const (
	version    = byte(0x00)
	walletFile = "wallet.dat"
)

// Wallet stores private and public keys.
type Wallet struct {
	PrivateKey ecdsa.PrivateKey
	PublicKey  []byte
}

//
type Wallets struct {
	Wallets map[string]*Wallet
}

// creates Wallets and fills it form a file f it exists.
func NewWallets() (*Wallets, error) {
	wallets := Wallets{}
	wallets.Wallets = make(map[string]*Wallet)
	err := wallets.LoadFromFile()
	return &wallets, err
}

// adds a Wallet to Wallets.
func (ws *Wallets) CreateWallet() string {
	wallet := NewWallet()
	address := wallet.GetAddress()
	ws.Wallets[address] = wallet
	return address
}

// return s an array of addresses stored in the wallet file.
func (ws *Wallets) GetAddresses() []string {
	addresses := make([]string, 0, len(ws.Wallets))
	for address := range ws.Wallets {
		addresses = append(addresses, address)
	}

	return addresses
}

// returns a Wallet by its address.
func (ws Wallets) GetWallet(address string) Wallet {
	return *ws.Wallets[address]
}

// loads wallets from the file.
func (ws *Wallets) LoadFromFile() error {
	if _, err := os.Stat(walletFile); os.IsNotExist(err) {
		return fmt.Errorf("%w", err)
	}
	fileContent, err := ioutil.ReadFile(walletFile)
	if err != nil {
		log.Panic(err)
	}
	var wallets Wallets
	gob.Register(elliptic.P256())
	decoder := gob.NewDecoder(bytes.NewReader(fileContent))
	err = decoder.Decode(&wallets)
	if err != nil {
		log.Panic(err)
	}
	ws.Wallets = wallets.Wallets
	return nil
}

// saves wallets to a file.
func (ws Wallets) SaveToFile() {
	var content bytes.Buffer
	gob.Register(elliptic.P256())
	encoder := gob.NewEncoder(&content)
	err := encoder.Encode(ws)
	if err != nil {
		log.Panic(err)
	}

	err = ioutil.WriteFile(walletFile, content.Bytes(), 0600)
	if err != nil {
		log.Panic(err)
	}
}

func NewWallet() *Wallet {
	private, public := utils.NewKeyPair()
	wallet := Wallet{private, public}
	return &wallet
}

func (wallet Wallet) GetAddress() string {
	return utils.Pubkey2Address(wallet.PublicKey, version)
}
