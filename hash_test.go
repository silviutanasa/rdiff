package rdiff

import (
	"bytes"
	"hash/adler32"
	"strings"
	"testing"
)

var testsWriteAndRoll = []struct {
	out uint32
	in  string
}{
	//{0x00000001, ""}, // panics
	{0x00620062, "a"},
	{0x012600c4, "ab"},
	{0x024d0127, "abc"},
	{0x03d8018b, "abcd"},
	{0x05c801f0, "abcde"},
	{0x081e0256, "abcdef"},
	{0x0adb02bd, "abcdefg"},
	{0x0e000325, "abcdefgh"},
	{0x118e038e, "abcdefghi"},
	{0x158603f8, "abcdefghij"},
	{0x3f090f02, "Discard medicine more than two years old."},
	{0x46d81477, "He who has a shady past knows that nice guys finish last."},
	{0x40ee0ee1, "I wouldn't marry him with a ten foot pole."},
	{0x16661315, "Free! Free!/A trip/to Mars/for 900/empty jars/Burma Shave"},
	{0x5b2e1480, "The days of the digital watch are numbered.  -Tom Stoppard"},
	{0x8c3c09ea, "Nepal premier won't resign."},
	{0x45ac18fd, "For every action there is an equal and opposite government program."},
	{0x53c61462, "His money is twice tainted: 'taint yours and 'taint mine."},
	{0x7e511e63, "There is no reason for any individual to have a computer in their home. -Ken Olsen, 1977"},
	{0xe4801a6a, "It's a tiny change to the code and not completely disgusting. - Bob Manchek"},
	{0x61b507df, "size:  a.out:  bad magic"},
	{0xb8631171, "The major problem is with sendmail.  -Mark Horton"},
	{0x8b5e1904, "Give me a rock, paper and scissors and I will move the world.  CCFestoon"},
	{0x7cc6102b, "If the enemy is within range, then so are you."},
	{0x700318e7, "It's well we cannot hear the screams/That we create in others' dreams."},
	{0x1e601747, "You remind me of a TV show, but that's all right: I watch it anyway."},
	{0xb55b0b09, "C is as portable as Stonehedge!!"},
	{0x39111dd0, "Even if I could be Shakespeare, I think I should still choose to be Faraday. - A. Huxley"},
	{0x91dd304f, "The fugacity of a constituent in a mixture of gases at a given temperature is proportional to its mole fraction.  Lewis-Randall Rule"},
	{0x2e5d1316, "How can you write a big system without C++?  -Paul Glick"},
	{0xd0201df6, "'Invariant assertions' is the most elegant programming technique!  -Tom Szymanski"},
	{0x211297c8, strings.Repeat("\xff", 5548) + "8"},
	{0xbaa198c8, strings.Repeat("\xff", 5549) + "9"},
	{0x553499be, strings.Repeat("\xff", 5550) + "0"},
	{0xf0c19abe, strings.Repeat("\xff", 5551) + "1"},
	{0x8d5c9bbe, strings.Repeat("\xff", 5552) + "2"},
	{0x2af69cbe, strings.Repeat("\xff", 5553) + "3"},
	{0xc9809dbe, strings.Repeat("\xff", 5554) + "4"},
	{0x69189ebe, strings.Repeat("\xff", 5555) + "5"},
	{0x86af0001, strings.Repeat("\x00", 1e5)},
	{0x79660b4d, strings.Repeat("a", 1e5)},
	{0x110588ee, strings.Repeat("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 1e4)},
}

// sum32ByAdlerByWriteAndRoll computes the sum by prepending the input slice with
// a '\0', writing the first bytes of this slice into the sum, then
// sliding on the last byte and returning the result of Sum32
func sum32ByAdlerByWriteAndRoll(b []byte) uint32 {
	q := []byte("\x00")
	q = append(q, b...)

	rollingHasher := newAdler32RollingHash()
	rollingHasher.WriteAll(q[:len(q)-1])
	rollingHasher.Roll(q[len(q)-1])

	return rollingHasher.Sum32()
}

// TestAdler32RollingHash_WriteAndRoll uses the Adler32 classical implementation to test the 'Write and Roll' behaviour.
// This testing approach is inspired by the Adler32 package Golden technique.
// For a given string, both the classical Adler32 Sum32 and the 'Write and Roll' Sum32 are computed, and checked against
// expected output. Even if the Adler32 is already tested this way, it still brings more value/clarity to use the same approach here.
func TestAdler32RollingHash_WriteAndRoll_Golden(t *testing.T) {
	for _, g := range testsWriteAndRoll {
		in := g.in

		// We test the classic implementation
		p := []byte(g.in)
		classic := adler32.New()
		classic.Write(p)
		if got := classic.Sum32(); got != g.out {
			t.Errorf("classic implementation: for %q, expected 0x%x, got 0x%x", in, g.out, got)
			continue
		}

		if got := sum32ByAdlerByWriteAndRoll(p); got != g.out {
			t.Errorf("rolling implementation: for %q, expected 0x%x, got 0x%x", in, g.out, got)
			continue
		}
	}
}

type inWR struct {
	write []byte
	roll  []byte
}

var testGetWindowContent = []struct {
	in  inWR
	out []byte
}{
	{
		in: inWR{
			write: []byte{1, 2, 3},
			roll:  []byte{5},
		},
		out: []byte{2, 3, 5},
	},
	{
		in: inWR{
			write: []byte{1, 2, 3},
			roll:  []byte{5, 6},
		},
		out: []byte{3, 5, 6},
	},
	{
		in: inWR{
			write: []byte{1, 2, 3},
			roll:  nil,
		},
		out: []byte{1, 2, 3},
	},
	{
		in: inWR{
			write: nil,
			roll:  nil,
		},
		out: nil,
	},
	//this panics, as expected {
	//	in: inWR{
	//		write: nil,
	//		roll:  []byte{1, 2, 3},
	//	},
	//	out: nil,
	//},
}

// TestAdler32RollingHash_GetWindowContent tests both Reset and GetWindowContent.
func TestAdler32RollingHash_GetWindowContent(t *testing.T) {
	rh := newAdler32RollingHash()
	for _, g := range testGetWindowContent {
		inp := g.in

		rh.Reset()
		rh.WriteAll(inp.write)
		for _, v := range inp.roll {
			rh.Roll(v)
		}
		if got := rh.GetWindowContent(); bytes.Compare(got, g.out) != 0 {
			t.Errorf("GetWindowContent(): expected %v, got %v", g.out, got)
		}
	}
}

func BenchmarkRolling64B(b *testing.B) {
	b.SetBytes(1024)
	b.ReportAllocs()
	window := make([]byte, 64)
	for i := range window {
		window[i] = byte(i)
	}

	h := newAdler32RollingHash()
	h.WriteAll(window)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Roll(byte(i))
		h.Sum32()
	}
}
