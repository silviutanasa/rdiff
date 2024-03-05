package rdiff

import "testing"

var testsComputeDynBlSize = []struct {
	in  int64
	out int64
}{
	{in: 1, out: DefaultBlockSize},
	{in: 100, out: DefaultBlockSize},
	{in: 699, out: DefaultBlockSize},
	{in: 10000, out: DefaultBlockSize},
	{in: 100000, out: DefaultBlockSize},
	{in: 1000000, out: 1000},
	{in: 10000000, out: 3160},
	{in: 20000000, out: 4472},
	{in: 30000000, out: 5472},
	{in: 100000000, out: 10000},
	{in: 10000000000, out: 100000},
	// 160GB		  // 1MB
	{in: 20000000000, out: 131064},
	{in: 50000000000, out: MaxBlockSize},
	{in: 100000000000, out: MaxBlockSize},
	{in: 10000000000000, out: MaxBlockSize},
	{in: 1000000000000000000, out: MaxBlockSize},
}

func Test_computeDynamicBlockSize(t *testing.T) {
	for _, tt := range testsComputeDynBlSize {
		if got := computeDynamicBlockSize(tt.in); got != tt.out {
			t.Errorf("computeDynamicBlockSize() = %v, want %v", got, tt.out)
		}
	}
}
