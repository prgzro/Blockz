package types

import (
	"encoding/hex"
	"fmt"
)

type Address [20]uint8

func (a Address) ToSlice() []byte { // Convert Address to a byte slice
	b := make([]byte, 20) // make slice wih length 20
	for i := 0; i < 20; i++ {
		b[i] = a[i]
	}
	return b
}

func (a Address) String() string { // Convert Address to a hex string
	return hex.EncodeToString(a.ToSlice())
}
func AddressFromBytes(b []byte) Address { // Create Address from a byte slice
	if len(b) != 20 { // Check if length is 20
		msg := fmt.Sprintf("Given Bytes with length %d should be 20", len(b)) // Format error message
		panic(msg)                                                            // Panic if length is not 20
	}

	var value [20]uint8       // Create an array of 20 uint8
	for i := 0; i < 20; i++ { // Copy bytes into the array
		value[i] = b[i] // Assign byte to array
	}
	return Address(value) // Return Address
}

func (a Address) Equals(other Address) bool {
	return a == other
}

func RandomAddress() Address {
	b := make([]byte, 20)
	for i := 0; i < 20; i++ {
		b[i] = byte(i + 1) // Deterministic random for testing
	}
	return AddressFromBytes(b)
}
