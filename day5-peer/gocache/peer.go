package gocache

// PeerPicker is the interface that must be implemented by gocahe to locate
// the peer that owns a specific key.
type PeerPicker interface {
	PickPeer(key string) (peer PeerClient, ok bool)
}

// PeerGetter is the interface that must be implemented by a peer to find cache from group
type PeerClient interface {
	Request(group string, key string) ([]byte, error)
}
