package network

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"

	"github.com/prgzro/Blockz/core"
	"github.com/sirupsen/logrus"
)

type MessageType byte

const (
	MessageTypeTx         MessageType = 0x01 // MessageTypeTx represents a transaction message
	MessageTypeBlock      MessageType = 0x02 // MessageTypeBlock represents a block message
	MessageTypeGetBlocks  MessageType = 0x03 // MessageTypeGetBlocks represents a request for blocks
	MessageTypeBlocks     MessageType = 0x06 // MessageTypeBlocks represents a response with multiple blocks
	MessageTypeStatus     MessageType = 0x04 // MessageTypeStatus represents node chain status
	MessageTypeGetStatus  MessageType = 0x05 // MessageTypeGetStatus requests node chain status
	MessageTypeGetPeers   MessageType = 0x07 // MessageTypeGetPeers requests list of peers
	MessageTypeSharePeers MessageType = 0x08 // MessageTypeSharePeers shares list of peers
)

type RPC struct { // RPC represents a remote procedure call message
	From    NetAddr
	Payload io.Reader
}

type Message struct {
	Header MessageType
	Data   []byte
}

func NewMessage(t MessageType, data []byte) *Message {
	return &Message{
		Header: t,
		Data:   data,
	}
}

func (msg *Message) Bytes() []byte {
	buf := &bytes.Buffer{}
	gob.NewEncoder(buf).Encode(msg)
	return buf.Bytes()
}

type DecodedMessage struct {
	From NetAddr
	Data any
}

type StatusMessage struct {
	ID      string
	Version uint32
	Height  uint32
}

type RPCDecodeFunc func(RPC) (*DecodedMessage, error)

func DefaultRPCDecodeFunc(rpc RPC) (*DecodedMessage, error) {
	msg := Message{}
	if err := gob.NewDecoder(rpc.Payload).Decode(&msg); err != nil {
		return nil, fmt.Errorf("failed to decode message from %s: %s", rpc.From, err)
	}

	logrus.WithFields(logrus.Fields{
		"from": rpc.From,
		"type": msg.Header,
	}).Debug("new incoming message")

	switch msg.Header {
	case MessageTypeTx:
		tx := new(core.Transaction)
		if err := tx.Decode(core.NewGobTxDecoder(bytes.NewReader(msg.Data))); err != nil {
			return nil, err
		}
		return &DecodedMessage{
			From: rpc.From,
			Data: tx,
		}, nil

	case MessageTypeBlock:
		block := new(core.Block)
		if err := block.Decode(core.NewGobBlockDecoder(bytes.NewReader(msg.Data))); err != nil {
			return nil, err
		}
		return &DecodedMessage{
			From: rpc.From,
			Data: block,
		}, nil
	case MessageTypeGetBlocks:
		getBlocks := new(GetBlocksMessage)
		if err := gob.NewDecoder(bytes.NewReader(msg.Data)).Decode(getBlocks); err != nil {
			return nil, err
		}
		return &DecodedMessage{
			From: rpc.From,
			Data: getBlocks,
		}, nil
	case MessageTypeBlocks:
		blocks := new(BlocksMessage)
		if err := gob.NewDecoder(bytes.NewReader(msg.Data)).Decode(blocks); err != nil {
			return nil, err
		}
		return &DecodedMessage{
			From: rpc.From,
			Data: blocks,
		}, nil
	case MessageTypeStatus:
		status := new(StatusMessage)
		if err := gob.NewDecoder(bytes.NewReader(msg.Data)).Decode(status); err != nil {
			return nil, err
		}
		return &DecodedMessage{
			From: rpc.From,
			Data: status,
		}, nil
	case MessageTypeGetStatus:
		return &DecodedMessage{
			From: rpc.From,
			Data: &GetStatusMessage{},
		}, nil
	case MessageTypeGetPeers:
		return &DecodedMessage{
			From: rpc.From,
			Data: &GetPeersMessage{},
		}, nil
	case MessageTypeSharePeers:
		sharePeers := new(SharePeersMessage)
		if err := gob.NewDecoder(bytes.NewReader(msg.Data)).Decode(sharePeers); err != nil {
			return nil, err
		}
		return &DecodedMessage{
			From: rpc.From,
			Data: sharePeers,
		}, nil
	default:
		return nil, fmt.Errorf("invalid message type %x", msg.Header)
	}
}

type GetStatusMessage struct{}

type GetBlocksMessage struct {
	From uint32
	To   uint32 // To=0 means fetch until latest
}

type BlocksMessage struct {
	Blocks []*core.Block
}

type GetPeersMessage struct{}

type SharePeersMessage struct {
	Peers []NetAddr
}

type RPCProcessor interface {
	ProcessMessage(*DecodedMessage) error
}
