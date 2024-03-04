package rdiff

import (
	"container/ring"
	"hash"
	"hash/adler32"
)

// M is the modulo for the Adler32 hash computation
const M = 65521

type adler32RollingHash struct {
	a              uint32      // component of Adler32 sum
	b              uint32      // component of Adler32 sum
	n              uint32      // component of Adler32 sum, the window size - it's cheaper to store it instead of computing it every time, based on the window
	window         *ring.Ring  // the window for the rolling hash computation, implemented with a circular buffer
	adler32Classic hash.Hash32 // adler32 standard hashing algorithm
}

func newAdler32RollingHash() *adler32RollingHash {
	rol := adler32RollingHash{
		a:              1,
		b:              0,
		adler32Classic: adler32.New(),
	}

	return &rol
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

// Roll adds a new element to the window, removes the oldest one, and computes the new hash components(a, b)
func (r *adler32RollingHash) Roll(b byte) byte {
	enter := uint32(b)
	// TODO: this can panic
	l := r.window.Value.(byte)
	leave := uint32(l)

	r.window.Value = b
	r.window = r.window.Next()

	r.a = (r.a + M + enter - leave) % M
	r.b = (r.b + (r.n*leave/M+1)*M + r.a - (r.n * leave) - 1) % M

	return l
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
	r.adler32Classic.Write(p)
	s := r.adler32Classic.Sum32()
	r.a, r.b = s&0xffff, s>>16
}

// Sum32 computes the Adler32 sum for the window
func (r *adler32RollingHash) Sum32() uint32 {
	return r.b<<16 | r.a&0xffff
}
