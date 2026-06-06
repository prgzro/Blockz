package core

import (
	"bytes"
	"testing"

	"github.com/prgzro/Blockz/crypto"
	"github.com/stretchr/testify/assert"
)

func TestSignTransaction(t *testing.T) {

	privKey := crypto.GeneratePrivateKey()
	tx := &Transaction{
		Data: []byte("Foo"),
	}

	assert.Nil(t, tx.Sign(privKey))
	assert.NotNil(t, tx.Signature)

}

func TestVerifyTransaction(t *testing.T) {

	privKey := crypto.GeneratePrivateKey()
	tx := &Transaction{
		Data: []byte("Foo"),
	}

	assert.Nil(t, tx.Sign(privKey))
	assert.Nil(t, tx.Verify())

	otherPrivKey := crypto.GeneratePrivateKey()
	tx.From = otherPrivKey.PublicKey()
	assert.NotNil(t, tx.Verify())
}

func TestPrgZroTransaction(t *testing.T) {

	privKey := crypto.GeneratePrivateKey()

	tx := &Transaction{
		Data:     []byte("PrgZro"),
		GasLimit: 1000,
	}

	tx1 := &Transaction{
		Data:     []byte("Mohamedd"),
		From:     crypto.GeneratePrivateKey().PublicKey(),
		GasLimit: 1000,
	}

	assert.Nil(t, tx.Sign(privKey))
	assert.Nil(t, tx.Verify())
	assert.Nil(t, tx1.Sign(privKey))
	assert.Nil(t, tx1.Verify())
}

func TestTxEncodeDecode(t *testing.T) {
	tx := randomTxWithSignature(t)
	buf := &bytes.Buffer{}
	// There's an unsolved Error from EP8 please solve it
	// https://youtu.be/5Xb9gJn_Ffo?list=PL0xRBLFXXsP6-hxQmCDcl_BHJMm0mhxx7&t=1700
	assert.Nil(t, tx.Encode(NewGobTxEncoder(buf)))

	txDecoded := new(Transaction)

	assert.Nil(t, txDecoded.Decode(NewGobTxDecoder(buf)))
	assert.Equal(t, tx, txDecoded)
}

func randomTxWithSignature(t *testing.T) *Transaction {
	privKey := crypto.GeneratePrivateKey()
	tx := Transaction{
		Data:     nil,
		GasLimit: 1000,
	}

	assert.Nil(t, tx.Sign(privKey))

	return &tx
}
