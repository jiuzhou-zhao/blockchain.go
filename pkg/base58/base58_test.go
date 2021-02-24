package base58

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBase58(t *testing.T) {
	s := []byte("00010966776006953D5567439E5E39F86A0D273BEED61967F6")
	e := Encode(s, BitcoinAlphabet)

	d, err := Decode(e, BitcoinAlphabet)
	assert.Nil(t, err)

	assert.True(t, bytes.Equal(s, d))

	t.Log(s, e, d)
}

func TestBase58WithCheck(t *testing.T) {
	s := []byte("123")
	e := EncodeWithCheck(s, BitcoinAlphabet)

	d, err := DecodeWithCheck(e, BitcoinAlphabet)
	assert.Nil(t, err)

	assert.True(t, bytes.Equal(s, d))

	t.Log(s, e, d)
}
