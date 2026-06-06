package core

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/prgzro/Blockz/types"
	"github.com/syndtr/goleveldb/leveldb"
)

var (
	blockPrefix     = []byte("block_")
	blockHashPrefix = []byte("blockhash_")
	txPrefix        = []byte("tx_")
)

type LevelDBStore struct {
	db *leveldb.DB
}

func NewLevelDBStore(path string) (*LevelDBStore, error) {
	db, err := leveldb.OpenFile(path, nil)
	if err != nil {
		return nil, err
	}
	return &LevelDBStore{db: db}, nil
}

func (s *LevelDBStore) Put(b *Block) error {
	buf := new(bytes.Buffer)
	if err := b.Encode(NewGobBlockEncoder(buf)); err != nil {
		return err
	}

	// Index by height
	heightKey := append(blockPrefix, heightToBytes(b.Header.Height)...)
	if err := s.db.Put(heightKey, buf.Bytes(), nil); err != nil {
		return err
	}

	// Index by hash
	blockHash := b.Hash(BlockHasher{})
	hashKey := append(blockHashPrefix, blockHash.ToSlice()...)
	if err := s.db.Put(hashKey, buf.Bytes(), nil); err != nil {
		return err
	}

	// Index transactions
	for _, tx := range b.Transactions {
		txBuf := new(bytes.Buffer)
		if err := tx.Encode(NewGobTxEncoder(txBuf)); err != nil {
			return err
		}
		txHash := tx.Hash(TxHasher{})
		txKey := append(txPrefix, txHash.ToSlice()...)
		if err := s.db.Put(txKey, txBuf.Bytes(), nil); err != nil {
			return err
		}
	}

	return nil
}

func (s *LevelDBStore) Get(height uint32) (*Block, error) {
	key := append(blockPrefix, heightToBytes(height)...)
	data, err := s.db.Get(key, nil)
	if err != nil {
		return nil, fmt.Errorf("block at height %d not found: %s", height, err)
	}

	block := new(Block)
	if err := block.Decode(NewGobBlockDecoder(bytes.NewReader(data))); err != nil {
		return nil, err
	}
	return block, nil
}

func (s *LevelDBStore) GetByHash(hash types.Hash) (*Block, error) {
	key := append(blockHashPrefix, hash.ToSlice()...)
	data, err := s.db.Get(key, nil)
	if err != nil {
		return nil, fmt.Errorf("block with hash %s not found: %s", hash, err)
	}

	block := new(Block)
	if err := block.Decode(NewGobBlockDecoder(bytes.NewReader(data))); err != nil {
		return nil, err
	}
	return block, nil
}

func (s *LevelDBStore) GetTxByHash(hash types.Hash) (*Transaction, error) {
	key := append(txPrefix, hash.ToSlice()...)
	data, err := s.db.Get(key, nil)
	if err != nil {
		return nil, fmt.Errorf("transaction with hash %s not found: %s", hash, err)
	}

	tx := new(Transaction)
	if err := tx.Decode(NewGobTxDecoder(bytes.NewReader(data))); err != nil {
		return nil, err
	}
	return tx, nil
}

func (s *LevelDBStore) Close() error {
	return s.db.Close()
}

func heightToBytes(h uint32) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, h)
	return b
}
