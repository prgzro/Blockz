package types

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

type Hash [32]uint8 // Type Hash as array of uint8

func (h Hash) IsZero() bool { // this is a function IsZero that's comfirm that all bytes in the Hash are zero return true if not return false
	for i := 0; i < 32; i++ {
		if h[i] != 0 {
			return false
		}
	}
	return true
}

func (h Hash) ToSlice() []byte { //This a function to convert Hash From array of Uint8 to slices of Bytes and return it
	b := make([]byte, 32)     // create a byte slice of length 32
	for i := 0; i < 32; i++ { // looping through each byte in the Hash and assign it to the byte slice
		b[i] = h[i]
	}
	return b // return the byte slice
}

func (h Hash) String() string { // this is a function to convert Hash to string and return it
	return hex.EncodeToString(h.ToSlice()) // EncodeToString returns the hexadecimal encoding of src.
}

func HashFromBytes(b []byte) Hash {
	if len(b) != 32 {
		msg := fmt.Sprintf("Given byte slice has invalid length %d, expected 32", len(b))
		panic(msg)
	}

	var value [32]uint8
	for i := 0; i < 32; i++ {
		value[i] = b[i]
	}

	return Hash(value)

}

func RandomBytes(size int) []byte {
	token := make([]byte, size)
	rand.Read(token)
	return token

}
func RandomHash() Hash {
	return HashFromBytes(RandomBytes(32))
}
