package crypto

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGeneratePrivateKet(t *testing.T) {
	privKey := GeneratePrivateKey()
	pubKey := privKey.PublicKey()
	address := pubKey.Address()

	msg := []byte("Hello, World!")
	sig, err := privKey.Sign(msg)
	assert.Nil(t, err)

	assert.True(t, sig.Verify(pubKey, msg))

	fmt.Println(address)
	fmt.Println(*sig)
	// fmt.Println(address.String())

}

func TestKeypairSignVerifySuccess(t *testing.T) {
	privKey := GeneratePrivateKey()
	pubKey := privKey.PublicKey()

	msg := []byte("Hello , World !!")

	sign, err := privKey.Sign(msg)
	assert.Nil(t, err)

	assert.True(t, sign.Verify(pubKey, msg))
}

func TestKeypairSignVerifyFail(t *testing.T) {
	privKey := GeneratePrivateKey()
	pubKey := privKey.PublicKey()

	msg := []byte("Hello , World !!")

	sign, err := privKey.Sign(msg)
	assert.Nil(t, err)

	otherKey := GeneratePrivateKey()
	otherPubKey := otherKey.PublicKey()

	assert.False(t, sign.Verify(otherPubKey, msg))
	assert.False(t, sign.Verify(pubKey, []byte("FUCK YOU!!")))
}

func TestNewPrgZrO(t *testing.T) {
	privKey := GeneratePrivateKey()

	fmt.Println(privKey.PublicKey().Address())

}
