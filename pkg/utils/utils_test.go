package utils

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/ripemd160"
)

func Test(t *testing.T) {
	t.Log(hex.EncodeToString(IntToHex(19876)))
	t.Log(strings.Repeat("0", 12) + strconv.FormatInt(19876, 16))
	t.Log(fmt.Sprintf("%016x", 19876))
}

func BenchmarkIntToHex(b *testing.B) {
	for i := 0; i < b.N; i++ {
		IntToHex(19876)
	}
}

func BenchmarkConv(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = strings.Repeat("0", 12) + strconv.FormatInt(19876, 16)
	}
}

func BenchmarkSprintf(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = fmt.Sprintf("016%x", 19876)
	}
}

func TestLoop(t *testing.T) {
	idx := 0
LP:
	for ; idx < 10; idx++ {
		if idx == 2 {
			continue LP
		}
	}
	t.Log(idx)
}

func TestNewKeyPair(t *testing.T) {
	t1 := big.NewInt(1).Bytes()
	t.Log(len(t1))
}

func Test_ripemd160(t *testing.T) {
	s := []byte{0x01, 0x06, 0xf0}
	e1 := ripemd160.New().Sum(s)

	rmd := ripemd160.New()
	_, _ = rmd.Write(s)
	e2 := rmd.Sum(nil)

	assert.False(t, bytes.Equal(e1, e2))
	assert.True(t, bytes.Equal(e1[:3], s))

	t.Log(hex.EncodeToString(e1))
	t.Log(hex.EncodeToString(e2))
}

func TestSignAndVerify(t *testing.T) {
	priKey, pubKey := NewKeyPair()

	data := []byte("abcd")
	signature, err := Sign(&priKey, data)
	assert.Nil(t, err)
	ok := VerifySign(signature, pubKey, data)
	assert.True(t, ok)
}
