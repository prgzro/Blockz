package network

import (
	"sync"

	"github.com/prgzro/Blockz/core"
	"github.com/prgzro/Blockz/types"
)

type TxPool struct {
	all     *TxSortedMap
	pending *TxSortedMap
	// The maxLength of the total pool of transactions.
	// When the pool is full we will prune the oldest transaction.
	maxLength int
}

func NewTxPool(maxLength int) *TxPool {
	return &TxPool{
		all:       NewTxSortedMap(),
		pending:   NewTxSortedMap(),
		maxLength: maxLength,
	}
}

func (p *TxPool) Add(tx *core.Transaction) {
	// prune the oldest transaction that is sitting in the all pool
	if p.all.Count() == p.maxLength {
		oldest := p.all.First()
		p.all.Remove(oldest.Hash(core.TxHasher{}))
	}

	if !p.all.Contains(tx.Hash(core.TxHasher{})) {
		p.all.Add(tx)
		p.pending.Add(tx)
	}
}

func (p *TxPool) Contains(hash types.Hash) bool {
	return p.all.Contains(hash)
}

// Pending returns a slice of transactions that are in the pending pool
func (p *TxPool) Pending() []*core.Transaction {
	return p.pending.txx.Data
}

func (p *TxPool) ClearPending() {
	p.pending.Clear()
}

func (p *TxPool) PendingCount() int {
	return p.pending.Count()
}

type TxSortedMap struct {
	lock   sync.RWMutex
	lookup map[types.Hash]*core.Transaction
	txx    *types.List[*core.Transaction]
}

func NewTxSortedMap() *TxSortedMap {
	return &TxSortedMap{
		lookup: make(map[types.Hash]*core.Transaction),
		txx:    types.NewList[*core.Transaction](),
	}
}

func (t *TxSortedMap) First() *core.Transaction {
	t.lock.RLock()
	defer t.lock.RUnlock()

	first := t.txx.Get(0)
	return t.lookup[first.Hash(core.TxHasher{})]
}

func (t *TxSortedMap) Get(h types.Hash) *core.Transaction {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return t.lookup[h]
}

func (t *TxSortedMap) Add(tx *core.Transaction) {
	hash := tx.Hash(core.TxHasher{})

	t.lock.Lock()
	defer t.lock.Unlock()

	if _, ok := t.lookup[hash]; !ok {
		t.lookup[hash] = tx
		t.txx.Insert(tx)
	}
}

func (t *TxSortedMap) Remove(h types.Hash) {
	t.lock.Lock()
	defer t.lock.Unlock()

	t.txx.Remove(t.lookup[h])
	delete(t.lookup, h)
}

func (t *TxSortedMap) Count() int {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return len(t.lookup)
}

func (t *TxSortedMap) Contains(h types.Hash) bool {
	t.lock.RLock()
	defer t.lock.RUnlock()

	_, ok := t.lookup[h]
	return ok
}

func (t *TxSortedMap) Clear() {
	t.lock.Lock()
	defer t.lock.Unlock()

	t.lookup = make(map[types.Hash]*core.Transaction)
	t.txx.Clear()
}

//// THE OLD CODE/////

// package network

// import (
// 	"sort"

// 	"github.com/prgzro/Blockz/core"
// 	"github.com/prgzro/Blockz/types"
// )

// type TxMpaSorter struct {
// 	transactions []*core.Transaction
// }

// func NewTxMapSorter(txMap map[types.Hash]*core.Transaction) *TxMpaSorter {
// 	txx := make([]*core.Transaction, len(txMap))
// 	i := 0
// 	for _, val := range txMap {
// 		txx[i] = val
// 		i++
// 	}
// 	s := &TxMpaSorter{txx}
// 	sort.Sort(s)
// 	return s
// }

// func (s *TxMpaSorter) Len() int {
// 	return len(s.transactions)
// }

// func (s *TxMpaSorter) Swap(i, j int) {
// 	s.transactions[i], s.transactions[j] = s.transactions[j], s.transactions[i]
// }

// func (s *TxMpaSorter) Less(i, j int) bool {
// 	return s.transactions[i].FirstSeen() < s.transactions[j].FirstSeen()
// }

// type TxPool struct {
// 	transactions map[types.Hash]*core.Transaction
// }

// func NewTxPool() *TxPool {
// 	return &TxPool{
// 		transactions: make(map[types.Hash]*core.Transaction),
// 	}
// }

// func (p *TxPool) Transactions() []*core.Transaction {
// 	s := NewTxMapSorter(p.transactions)
// 	return s.transactions
// }

// // Add adds an transaction to the pool , the caller is responsible checking if the
// // tx already exist
// func (p *TxPool) Add(tx *core.Transaction) error {
// 	hash := tx.Hash(core.TxHasher{})
// 	p.transactions[hash] = tx

// 	return nil
// }

// func (p *TxPool) Has(hash types.Hash) bool {
// 	_, ok := p.transactions[hash]
// 	return ok
// }

// func (p *TxPool) Len() int {
// 	return len(p.transactions)
// }

// func (p *TxPool) Flush() {
// 	p.transactions = make(map[types.Hash]*core.Transaction)
// }
