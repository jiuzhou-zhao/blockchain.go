package base58

import (
	"bytes"
	"crypto/sha256"
	"errors"
)

const (
	checksumLen = 4
)

var errChecksum = errors.New("checksum failed")

func checksum(payload []byte) []byte {
	firstSHA := sha256.Sum256(payload)
	secondSHA := sha256.Sum256(firstSHA[:])
	return secondSHA[:checksumLen]
}

func EncodeWithCheck(payload []byte, alphabet *Alphabet) string {
	checksum := checksum(payload)
	fullPayload := append(payload, checksum...)
	return Encode(fullPayload, alphabet)
}

func DecodeWithCheck(payload string, alphabet *Alphabet) (d []byte, err error) {
	pubKeyHash, err := Decode(payload, alphabet)
	if err != nil {
		return
	}
	actualChecksum := pubKeyHash[len(pubKeyHash)-checksumLen:]
	d = pubKeyHash[0 : len(pubKeyHash)-checksumLen]
	targetChecksum := checksum(d)
	if !bytes.Equal(actualChecksum, targetChecksum) {
		err = errChecksum
		return
	}
	return
}
