package rdiff

import (
	"container/ring"
	"hash"
	"hash/adler32"
)

// M is the modulo for the Adler32 hash computation
const M = 65521

type adler32RollingHash struct {
	// component of Adler32 sum
	a uint32
	// component of Adler32 sum
	b uint32
	// component of Adler32 sum, the window size
	// it's cheaper to store it instead of computing it every time, based on the window
	n uint32
	// the window for the rolling hash computation, implemented with a circular buffer
	window *ring.Ring
	// adler32 standard hashing algorithm
	adler32Classic hash.Hash32
}

func newAdler32RollingHash() *adler32RollingHash {
	rol := adler32RollingHash{
		a:              1,
		b:              0,
		adler32Classic: adler32.New(),
	}

	return &rol
}

// WriteAll writes p []byte to the window.
// As the window is a circular buffer of fixed size, successive calls will overwrite each other's data.
func (r *adler32RollingHash) WriteAll(p []byte) {
	bufSize := len(p)
	if bufSize == 0 {
		return
	}
	if bufSize != int(r.n) {
		r.window = ring.New(bufSize)
		r.n = uint32(bufSize)
	}

	for _, b := range p {
		r.window.Value = b
		r.window = r.window.Next()
	}

	r.adler32Classic.Reset()
	_, _ = r.adler32Classic.Write(p)
	s := r.adler32Classic.Sum32()
	r.a, r.b = s&0xffff, s>>16
}

// Roll adds a new byte to the window, removes the oldest one, and computes the new hash components(a, b)
// Roll returns the removed/'popped out' byte.
// It panics if the window is not initialized, so before any Roll call, there should be at least one WriteAll call.
func (r *adler32RollingHash) Roll(b byte) byte {
	enter := uint32(b)
	l := r.window.Value.(byte)
	leave := uint32(l)

	r.window.Value = b
	r.window = r.window.Next()

	r.a = (r.a + M + enter - leave) % M
	r.b = (r.b + (r.n*leave/M+1)*M + r.a - (r.n * leave) - 1) % M

	return l
}

// Sum32 computes the Adler32 sum for the window
func (r *adler32RollingHash) Sum32() uint32 {
	return r.b<<16 | r.a&0xffff
}

// Reset resets the internal state
func (r *adler32RollingHash) Reset() {
	r.a = 1
	r.b = 0
	r.n = 0
	r.window = nil
	r.adler32Classic.Reset()
}

// GetWindowContent returns the data from the internal rolling window.
func (r *adler32RollingHash) GetWindowContent() []byte {
	if r.window == nil {
		return nil
	}

	wc := make([]byte, 0, r.n)
	for i := 0; i < int(r.n); i++ {
		if el, ok := r.window.Value.(byte); ok {
			wc = append(wc, el)
		}

		r.window = r.window.Next()
	}

	return wc
}
