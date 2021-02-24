package utils

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"log"
	"math/big"

	"github.com/jiuzhou-zhao/blockchain.go/pkg/base58"
	"golang.org/x/crypto/ripemd160"
)

func IntToHex(num int64) []byte {
	buff := new(bytes.Buffer)
	err := binary.Write(buff, binary.BigEndian, num)
	if err != nil {
		log.Panic(err)
	}
	return buff.Bytes()
}

// ReverseBytes reverses a byte array.
func ReverseBytes(data []byte) {
	for i, j := 0, len(data)-1; i < j; i, j = i+1, j-1 {
		data[i], data[j] = data[j], data[i]
	}
}

func NewKeyPair() (ecdsa.PrivateKey, []byte) {
	curve := elliptic.P256()
	private, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		log.Panic(err)
	}
	pubKey := append(private.PublicKey.X.Bytes(), private.PublicKey.Y.Bytes()...)
	return *private, pubKey
}

func HashPubKey(pubKey []byte) []byte {
	publicSHA256 := sha256.Sum256(pubKey)
	ripemd := ripemd160.New()
	_, err := ripemd.Write(publicSHA256[:])
	if err != nil {
		log.Panic(err)
	}
	return ripemd.Sum(nil)
}

func Base58Encode(payload []byte) string {
	return base58.Encode(payload, base58.BitcoinAlphabet)
}

func Base58Decode(payload string) ([]byte, error) {
	return base58.Decode(payload, base58.BitcoinAlphabet)
}

func Base58EncodeWithCheck(payload []byte) string {
	return base58.EncodeWithCheck(payload, base58.BitcoinAlphabet)
}

func Base58DecodeWithCheck(payload string) ([]byte, error) {
	return base58.DecodeWithCheck(payload, base58.BitcoinAlphabet)
}

func Pubkey2Address(key []byte, version byte) string {
	payload := HashPubKey(key)
	versionedPayload := append([]byte{version}, payload...)
	return Base58EncodeWithCheck(versionedPayload)
}

func Address2PubkeyHash(address string) (pubkeyHash []byte, err error) {
	payload, err := Base58DecodeWithCheck(address)
	if err != nil {
		return
	}
	pubkeyHash = payload[1:]
	return
}

func IsValidAddress(address string) bool {
	_, err := Address2PubkeyHash(address)
	return err == nil
}

func Sign(priKey *ecdsa.PrivateKey, d []byte) ([]byte, error) {
	r, s, err := ecdsa.Sign(rand.Reader, priKey, d)
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}
	return append(r.Bytes(), s.Bytes()...), nil
}

func VerifySign(signature []byte, pubKey []byte, d []byte) bool {
	r := big.Int{}
	s := big.Int{}
	sigLen := len(signature)
	r.SetBytes(signature[:(sigLen / 2)])
	s.SetBytes(signature[(sigLen / 2):])

	x := big.Int{}
	y := big.Int{}
	keyLen := len(pubKey)
	x.SetBytes(pubKey[:(keyLen / 2)])
	y.SetBytes(pubKey[(keyLen / 2):])

	rawPubKey := ecdsa.PublicKey{Curve: elliptic.P256(), X: &x, Y: &y}
	return ecdsa.Verify(&rawPubKey, d, &r, &s)
}
