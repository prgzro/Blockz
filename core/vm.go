package core

import (
	"crypto/sha256"
	"fmt"
	"math/big"

	"github.com/prgzro/Blockz/types"
	"golang.org/x/crypto/sha3"
)

// EVM Opcode Definitions
type OpCode byte

const (
	STOP         OpCode = 0x00
	ADD          OpCode = 0x01
	MUL          OpCode = 0x02
	SUB          OpCode = 0x03
	DIV          OpCode = 0x04
	SDIV         OpCode = 0x05
	MOD          OpCode = 0x06
	SMOD         OpCode = 0x07
	EXP          OpCode = 0x0a
	LT           OpCode = 0x10
	GT           OpCode = 0x11
	EQ           OpCode = 0x14
	ISZERO       OpCode = 0x15
	AND          OpCode = 0x16
	OR           OpCode = 0x17
	XOR          OpCode = 0x18
	NOT          OpCode = 0x19
	SHA3         OpCode = 0x20
	CALLVALUE    OpCode = 0x34
	CALLDATALOAD OpCode = 0x35
	CALLDATASIZE OpCode = 0x36
	POP          OpCode = 0x50
	MLOAD        OpCode = 0x51
	MSTORE       OpCode = 0x52
	MSTORE8      OpCode = 0x53
	SLOAD        OpCode = 0x54
	SSTORE       OpCode = 0x55
	JUMP         OpCode = 0x56
	JUMPI        OpCode = 0x57
	PC           OpCode = 0x58
	MSIZE        OpCode = 0x59
	JUMPDEST     OpCode = 0x5b
	PUSH1        OpCode = 0x60
	PUSH32       OpCode = 0x7f
	DUP1         OpCode = 0x80
	DUP16        OpCode = 0x8f
	SWAP1        OpCode = 0x90
	SWAP16       OpCode = 0x9f
	RETURN       OpCode = 0xf3
	REVERT       OpCode = 0xfd
)

// Big amounts
var (
	Big0   = big.NewInt(0)
	Big1   = big.NewInt(1)
	Big32  = big.NewInt(32)
	Max256 = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))
)

type StateDB interface {
	GetAccountState(types.Address, types.Hash) ([]byte, error)
	SetAccountState(types.Address, types.Hash, []byte) error
}

func u256(n int64) *big.Int {
	b := big.NewInt(n)
	// Ensure it is unsigned 256 bit equivalent, handle negatives if any
	return b
}

type Stack struct {
	data []*big.Int
}

func NewStack() *Stack {
	return &Stack{data: make([]*big.Int, 0, 1024)}
}

func (s *Stack) Push(v *big.Int) {
	// Ensure v is within 256 bits
	val := new(big.Int).And(v, Max256)
	s.data = append(s.data, val)
}

func (s *Stack) Pop() *big.Int {
	if len(s.data) == 0 {
		return big.NewInt(0)
	}
	v := s.data[len(s.data)-1]
	s.data = s.data[:len(s.data)-1]
	return v
}

func (s *Stack) Peek(n int) *big.Int {
	if len(s.data) <= n {
		return big.NewInt(0)
	}
	return s.data[len(s.data)-1-n]
}

func (s *Stack) Print() {
	fmt.Printf("Stack (len %d): [ ", len(s.data))
	for _, v := range s.data {
		fmt.Printf("%x ", v)
	}
	fmt.Println("]")
}

type Memory struct {
	store []byte
}

func NewMemory() *Memory {
	return &Memory{store: make([]byte, 0)}
}

func (m *Memory) expand(offset, size uint64) {
	if size == 0 {
		return
	}
	needed := offset + size
	if needed > uint64(len(m.store)) {
		// align to 32 bytes
		newSize := ((needed + 31) / 32) * 32
		newStore := make([]byte, newSize)
		copy(newStore, m.store)
		m.store = newStore
	}
}

func (m *Memory) Set(offset, size uint64, value []byte) {
	m.expand(offset, size)
	copy(m.store[offset:offset+size], value)
}

func (m *Memory) Set32(offset uint64, val *big.Int) {
	b := val.Bytes()
	padded := make([]byte, 32)
	copy(padded[32-len(b):], b) // right align
	m.Set(offset, 32, padded)
}

func (m *Memory) Get(offset, size uint64) []byte {
	m.expand(offset, size)
	res := make([]byte, size)
	copy(res, m.store[offset:offset+size])
	return res
}

type EVM struct {
	code     []byte
	pc       uint64
	stack    *Stack
	memory   *Memory
	state    StateDB
	contract types.Address
	caller   types.Address
	gasUsed  uint64
	gasLimit uint64
	retData  []byte
	callData []byte
	value    *big.Int

	// Tracing enables instruction-level logging
	Tracing bool
}

func NewEVM(code []byte, callData []byte, state StateDB, contract types.Address, caller types.Address, value *big.Int, gasLimit uint64) *EVM {
	if value == nil {
		value = big.NewInt(0)
	}
	return &EVM{
		code:     code,
		callData: callData,
		pc:       0,
		stack:    NewStack(),
		memory:   NewMemory(),
		state:    state,
		contract: contract,
		caller:   caller,
		value:    value,
		gasLimit: gasLimit,
		Tracing:  false,
	}
}

func (evm *EVM) Run() ([]byte, error) {
	for evm.pc < uint64(len(evm.code)) {
		op := OpCode(evm.code[evm.pc])

		if evm.Tracing {
			fmt.Printf("PC: %04d | OP: %02x | StackLen: %d\n", evm.pc, op, len(evm.stack.data))
		}

		if err := evm.consumeGas(1); err != nil { // basic constant gas cost for prototype
			return nil, err
		}

		evm.pc++

		switch {
		case op >= PUSH1 && op <= PUSH32:
			size := uint64(op - PUSH1 + 1)
			if evm.pc+size > uint64(len(evm.code)) {
				return nil, fmt.Errorf("EVM Error: push out of bounds")
			}
			val := new(big.Int).SetBytes(evm.code[evm.pc : evm.pc+size])
			evm.stack.Push(val)
			evm.pc += size

		case op >= DUP1 && op <= DUP16:
			n := int(op - DUP1 + 1)
			if len(evm.stack.data) < n {
				return nil, fmt.Errorf("EVM Error: dup out of bounds")
			}
			val := new(big.Int).Set(evm.stack.data[len(evm.stack.data)-n])
			evm.stack.Push(val)

		case op >= SWAP1 && op <= SWAP16:
			n := int(op - SWAP1 + 1)
			if len(evm.stack.data) <= n {
				return nil, fmt.Errorf("EVM Error: swap out of bounds")
			}
			top := len(evm.stack.data) - 1
			evm.stack.data[top], evm.stack.data[top-n] = evm.stack.data[top-n], evm.stack.data[top]

		case op == POP:
			evm.stack.Pop()

		case op == ADD:
			x, y := evm.stack.Pop(), evm.stack.Pop()
			res := new(big.Int).Add(x, y)
			evm.stack.Push(res)

		case op == SUB:
			x, y := evm.stack.Pop(), evm.stack.Pop()
			res := new(big.Int).Sub(x, y)
			evm.stack.Push(res)

		case op == MUL:
			x, y := evm.stack.Pop(), evm.stack.Pop()
			res := new(big.Int).Mul(x, y)
			evm.stack.Push(res)

		case op == DIV:
			x, y := evm.stack.Pop(), evm.stack.Pop()
			if y.Cmp(Big0) == 0 {
				evm.stack.Push(new(big.Int))
			} else {
				res := new(big.Int).Div(x, y)
				evm.stack.Push(res)
			}

		case op == EQ:
			x, y := evm.stack.Pop(), evm.stack.Pop()
			if x.Cmp(y) == 0 {
				evm.stack.Push(big.NewInt(1))
			} else {
				evm.stack.Push(big.NewInt(0))
			}

		case op == LT:
			x, y := evm.stack.Pop(), evm.stack.Pop()
			if x.Cmp(y) < 0 {
				evm.stack.Push(big.NewInt(1))
			} else {
				evm.stack.Push(big.NewInt(0))
			}

		case op == GT:
			x, y := evm.stack.Pop(), evm.stack.Pop()
			if x.Cmp(y) > 0 {
				evm.stack.Push(big.NewInt(1))
			} else {
				evm.stack.Push(big.NewInt(0))
			}

		case op == ISZERO:
			x := evm.stack.Pop()
			if x.Cmp(Big0) == 0 {
				evm.stack.Push(big.NewInt(1))
			} else {
				evm.stack.Push(big.NewInt(0))
			}

		case op == AND:
			x, y := evm.stack.Pop(), evm.stack.Pop()
			res := new(big.Int).And(x, y)
			evm.stack.Push(res)

		case op == OR:
			x, y := evm.stack.Pop(), evm.stack.Pop()
			res := new(big.Int).Or(x, y)
			evm.stack.Push(res)

		case op == SHA3:
			offset := evm.stack.Pop().Uint64()
			size := evm.stack.Pop().Uint64()
			data := evm.memory.Get(offset, size)
			hash := sha3.NewLegacyKeccak256()
			hash.Write(data)
			evm.stack.Push(new(big.Int).SetBytes(hash.Sum(nil)))

		case op == MLOAD:
			offset := evm.stack.Pop().Uint64()
			val := new(big.Int).SetBytes(evm.memory.Get(offset, 32))
			evm.stack.Push(val)

		case op == MSTORE:
			offset := evm.stack.Pop().Uint64()
			val := evm.stack.Pop()
			evm.memory.Set32(offset, val)

		case op == MSTORE8:
			offset := evm.stack.Pop().Uint64()
			val := evm.stack.Pop()
			evm.memory.Set(offset, 1, []byte{byte(val.Uint64() & 0xff)})

		case op == SLOAD:
			key := evm.stack.Pop()

			keyBytes := key.Bytes()
			padded := make([]byte, 32)
			copy(padded[32-len(keyBytes):], keyBytes)
			keyHash := sha256.Sum256(padded)

			value, err := evm.state.GetAccountState(evm.contract, types.Hash(keyHash))
			if err != nil || len(value) == 0 {
				evm.stack.Push(big.NewInt(0))
			} else {
				evm.stack.Push(new(big.Int).SetBytes(value))
			}

		case op == SSTORE:
			key := evm.stack.Pop()
			val := evm.stack.Pop()

			keyBytes := key.Bytes()
			padded := make([]byte, 32)
			copy(padded[32-len(keyBytes):], keyBytes)
			keyHash := sha256.Sum256(padded)

			valBytes := val.Bytes()
			if len(valBytes) == 0 {
				valBytes = []byte{0}
			}

			// Save to persistent storage
			evm.state.SetAccountState(evm.contract, types.Hash(keyHash), valBytes)

		case op == CALLVALUE:
			evm.stack.Push(evm.value)

		case op == CALLDATALOAD:
			offset := evm.stack.Pop().Uint64()
			res := make([]byte, 32)
			if offset < uint64(len(evm.callData)) {
				copy(res, evm.callData[offset:])
			}
			evm.stack.Push(new(big.Int).SetBytes(res))

		case op == CALLDATASIZE:
			evm.stack.Push(big.NewInt(int64(len(evm.callData))))

		case op == JUMP:
			dest := evm.stack.Pop().Uint64()
			if !evm.isValidJumpDest(dest) {
				return nil, fmt.Errorf("EVM Error: invalid jump destination %d", dest)
			}
			evm.pc = dest

		case op == JUMPI:
			dest := evm.stack.Pop().Uint64()
			cond := evm.stack.Pop()
			if cond.Cmp(Big0) != 0 {
				if !evm.isValidJumpDest(dest) {
					return nil, fmt.Errorf("EVM Error: invalid jump destination %d", dest)
				}
				evm.pc = dest
			}

		case op == JUMPDEST:
			// No-op

		case op == RETURN:
			offset := evm.stack.Pop().Uint64()
			size := evm.stack.Pop().Uint64()
			evm.retData = evm.memory.Get(offset, size)
			return evm.retData, nil

		case op == REVERT:
			offset := evm.stack.Pop().Uint64()
			size := evm.stack.Pop().Uint64()
			evm.retData = evm.memory.Get(offset, size)
			return nil, fmt.Errorf("EVM REVERT: %x", evm.retData)

		case op == STOP:
			return nil, nil

		default:
			// Implement a fallback mapping for unsupported opcodes to avoid crash but log it
			fmt.Printf("EVM WARNING: unsupported opcode %02x at pc %d\n", op, evm.pc-1)
			// return nil, fmt.Errorf("EVM Error: unsupported opcode %02x", op)
		}
	}

	return evm.retData, nil
}

func (evm *EVM) isValidJumpDest(dest uint64) bool {
	if dest >= uint64(len(evm.code)) {
		return false
	}
	return OpCode(evm.code[dest]) == JUMPDEST
}

func (vm *EVM) consumeGas(amount uint64) error {
	if vm.gasUsed+amount > vm.gasLimit {
		return fmt.Errorf("out of gas: used %d, limit %d", vm.gasUsed+amount, vm.gasLimit)
	}
	vm.gasUsed += amount
	return nil
}

func (vm *EVM) GasUsed() uint64 {
	return vm.gasUsed
}
