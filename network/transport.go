package network

type NetAddr string // NetAddr represents a network address

type Transport interface { // Transport defines the interface for network transport mechanisms
	Consume() <-chan RPC               // Channel to receive incoming RPC messages
	Connect(Transport) error           // Connect to another Transport
	SendMessage(NetAddr, []byte) error // Send a message to a specified network address
	Broadcast([]byte) error
	Addr() NetAddr // Get the network address of this Transport
}
