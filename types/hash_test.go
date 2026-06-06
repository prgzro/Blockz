package types

import (
	"fmt"
	"testing"
)

func TestHexString(t *testing.T) {
	h := RandomHash()
	fmt.Println(h)

	HashFromBytes(RandomBytes(32))
}
