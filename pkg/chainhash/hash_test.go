package chainhash

import "testing"

func TestHashCopy(t *testing.T) {
	h1 := Hash{0x01}
	t.Log(h1)
	h2 := h1
	t.Log(h2)
	h2[3] = 0x03
	t.Log(h1)
	t.Log(h2)
}
