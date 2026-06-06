package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"math/big"

	"github.com/prgzro/Blockz/types"
)

type PrivateKey struct { // Struct to hold ECDSA private key
	key *ecdsa.PrivateKey // ECDSA private key
}

func (k PrivateKey) Sign(data []byte) (*Signature, error) { // Sign data using the private key
	r, s, err := ecdsa.Sign(rand.Reader, k.key, data) // Sign the data
	if err != nil {                                   // Check for errors
		return nil, err // Return error if signing fails
	}

	return &Signature{ // Return the signature
		R: r,
		S: s,
	}, nil

}

func GeneratePrivateKey() PrivateKey { // Generate a new ECDSA private key
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader) // Generate the key
	if err != nil {                                             // Check for errors
		panic(err) // Panic if key generation fails
	}
	return PrivateKey{ // Return the private key
		key: key,
	}
}

func (k PrivateKey) PublicKey() PublicKey { // Get the corresponding public key
	return PublicKey{
		Key: &k.key.PublicKey, // Return the public key
	}
}

type PublicKey struct { // Struct to hold ECDSA public key
	Key *ecdsa.PublicKey // ECDSA public key
}

// /////////////////////////////////////////////////////
// GobEncode implements gob.GobEncoder for PublicKey

func (pk PublicKey) GobEncode() ([]byte, error) {
	if pk.Key == nil {
		return []byte{}, nil
	}
	return elliptic.MarshalCompressed(pk.Key.Curve, pk.Key.X, pk.Key.Y), nil
}

// GobDecode implements gob.GobDecoder for PublicKey
func (pk *PublicKey) GobDecode(data []byte) error {
	if len(data) == 0 {
		pk.Key = nil
		return nil
	}
	x, y := elliptic.UnmarshalCompressed(elliptic.P256(), data)
	if x == nil {
		return errors.New("invalid public key")
	}
	pk.Key = &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     x,
		Y:     y,
	}
	return nil
}

//////////////////////////////////////////////////////////////

func (k PublicKey) ToSlice() []byte { // Convert public key to byte slice

	return elliptic.MarshalCompressed(k.Key, k.Key.X, k.Key.Y) // Marxshal the public key

}

func (k PublicKey) Address() types.Address { // Get the address from the public key
	h := sha256.Sum256(k.ToSlice())              // Hash the public key
	return types.AddressFromBytes(h[len(h)-20:]) // Return the last 20 bytes as the address
}

type Signature struct { // Struct to hold ECDSA signature
	S, R *big.Int // Signature components
}

func (sig Signature) Verify(pubKey PublicKey, data []byte) bool { // Verify the signature using the public key
	return ecdsa.Verify(pubKey.Key, data, sig.R, sig.S) // Verify the signature
}
