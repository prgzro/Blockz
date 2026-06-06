package network

import (
	"bytes"
	"fmt"
	"sync"
)

type LocalTransport struct { // LocalTransport is an in-memory transport for testing purposes
	addr      NetAddr                     // Network address of this transport
	consumeCh chan RPC                    // Channel to receive incoming RPC messages
	lock      sync.RWMutex                // Mutex to protect access to peers map
	peers     map[NetAddr]*LocalTransport // Map of connected peers
}

func NewLocalTransport(addr NetAddr) *LocalTransport {
	return &LocalTransport{
		addr:      addr,
		consumeCh: make(chan RPC, 1024),              // Buffered channel for incoming messages channel RPC , size is 1024
		peers:     make(map[NetAddr]*LocalTransport), // Initialize the peers map
	}
}

func (t *LocalTransport) Consume() <-chan RPC { // Consume returns the channel to receive incoming RPC messages
	return t.consumeCh
}

func (t *LocalTransport) Connect(tr Transport) error { // Connect connects this transport to another transport
	t.lock.Lock()                             // Lock the mutex for writing
	defer t.lock.Unlock()                     // Ensure the mutex is unlocked after the function returns
	t.peers[tr.Addr()] = tr.(*LocalTransport) // Add the peer to the peers map
	return nil                                // Return nil to indicate success
}

func (t *LocalTransport) SendMessage(to NetAddr, payload []byte) error { // SendMessage sends a message to the specified network address
	t.lock.RLock()         // Lock the mutex for reading
	defer t.lock.RUnlock() // Ensure the mutex is unlocked after the function returns

	peer, ok := t.peers[to] // Look up the peer in the peers map
	if !ok {                // If the peer is not found
		return fmt.Errorf("%s: could not send message to unknown peer %s", t.addr, to) // Return an error indicating the peer was not found
	}

	peer.consumeCh <- RPC{ // Send the RPC message to the peer's consume channel
		From:    t.addr,                   // Set the From field to this transport's address
		Payload: bytes.NewReader(payload), // Set the Payload field to the provided payload
	}
	return nil

}

func (t *LocalTransport) Broadcast(payload []byte) error {
	for _, peer := range t.peers {
		if err := t.SendMessage(peer.Addr(), payload); err != nil {
			return err
		}
	}
	return nil
}
func (t *LocalTransport) Addr() NetAddr { // Addr returns the network address of this transport
	return t.addr
}
