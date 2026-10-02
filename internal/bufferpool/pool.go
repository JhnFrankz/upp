// Package bufferpool provides a shared sync.Pool of 32KB byte buffers
// for high-throughput I/O operations (file copy, archive extraction, streaming).
package bufferpool

import "sync"

const BufferSize = 32 * 1024

var copyBufPool = sync.Pool{
	New: func() any {
		b := make([]byte, BufferSize)
		return &b
	},
}

// Get returns a pointer to a 32KB byte slice from the pool.
func Get() *[]byte {
	return copyBufPool.Get().(*[]byte)
}

// Put returns a 32KB byte slice pointer to the pool.
func Put(b *[]byte) {
	if b != nil && cap(*b) == BufferSize {
		*b = (*b)[:BufferSize]
		copyBufPool.Put(b)
	}
}
