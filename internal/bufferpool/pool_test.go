package bufferpool_test

import (
	"testing"

	"github.com/JhnFrankz/upp/internal/bufferpool"
)

func TestBufferPool_GetAndPut(t *testing.T) {
	bufPtr := bufferpool.Get()
	if bufPtr == nil {
		t.Fatal("expected non-nil buffer pointer")
	}
	if len(*bufPtr) != bufferpool.BufferSize {
		t.Fatalf("expected buffer size %d, got %d", bufferpool.BufferSize, len(*bufPtr))
	}
	if cap(*bufPtr) != bufferpool.BufferSize {
		t.Fatalf("expected buffer capacity %d, got %d", bufferpool.BufferSize, cap(*bufPtr))
	}

	// Verify we can write to it
	(*bufPtr)[0] = 0x42
	(*bufPtr)[bufferpool.BufferSize-1] = 0x24

	// Put it back
	bufferpool.Put(bufPtr)

	// Get another (or same) buffer
	bufPtr2 := bufferpool.Get()
	if bufPtr2 == nil {
		t.Fatal("expected non-nil buffer pointer on second Get")
	}
	if len(*bufPtr2) != bufferpool.BufferSize {
		t.Fatalf("expected buffer size %d, got %d", bufferpool.BufferSize, len(*bufPtr2))
	}
	bufferpool.Put(bufPtr2)
}

func TestBufferPool_PutNilOrWrongSize(t *testing.T) {
	// Should not panic on nil
	bufferpool.Put(nil)

	// Should ignore slices with wrong size
	wrong := make([]byte, 10)
	bufferpool.Put(&wrong)

	// Normal Get should still yield correct BufferSize
	bufPtr := bufferpool.Get()
	if len(*bufPtr) != bufferpool.BufferSize {
		t.Fatalf("expected buffer size %d, got %d", bufferpool.BufferSize, len(*bufPtr))
	}
	bufferpool.Put(bufPtr)
}

func TestBufferPool_PutReslicedBuffer(t *testing.T) {
	bufPtr := bufferpool.Get()
	*bufPtr = (*bufPtr)[:100]
	bufferpool.Put(bufPtr)

	bufPtr2 := bufferpool.Get()
	if len(*bufPtr2) != bufferpool.BufferSize {
		t.Fatalf("expected buffer size %d after putting resliced buffer, got %d", bufferpool.BufferSize, len(*bufPtr2))
	}
	if cap(*bufPtr2) != bufferpool.BufferSize {
		t.Fatalf("expected buffer capacity %d after putting resliced buffer, got %d", bufferpool.BufferSize, cap(*bufPtr2))
	}
	bufferpool.Put(bufPtr2)
}
