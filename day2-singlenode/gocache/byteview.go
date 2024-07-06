package gocache

// A ByteView holds an immutable view of cache bytes, it encapsulate cache Entry Value as unit of bytes
// Len() method needs to be implemented for Value interface
type ByteView struct {
	b []byte
}

// Len returns the view's length, i.e. num of bytes
func (v ByteView) Len() int {
	return len(v.b)
}

// String returns the data as a string, making a copy if necessary.
func (v ByteView) String() string {
	return string(v.b)
}

// ByteSlice returns a copy of the data as a byte slice.
func (v ByteView) ByteSlice() []byte {
	return cloneBytes(v.b)
}

// The copy built-in function copies elements from a source slice into a
// destination slice
// read only as a copy
func cloneBytes(b []byte) []byte {
	c := make([]byte, len(b))
	copy(c, b)
	return c
}
