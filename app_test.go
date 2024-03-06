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

type blIn struct {
	blSize int
	fSize  int64
}

var testsDecideBlSize = []struct {
	in      blIn
	out     int
	wantErr bool
}{
	{in: blIn{blSize: -100, fSize: 100}, out: 50},
	{in: blIn{blSize: 0, fSize: 100}, out: 50},
	{in: blIn{blSize: 0, fSize: 699}, out: 349},
	{in: blIn{blSize: 0, fSize: 700}, out: 350},
	{in: blIn{blSize: 0, fSize: 701}, out: DefaultBlockSize},
	{in: blIn{blSize: 0, fSize: 750}, out: DefaultBlockSize},
	{in: blIn{blSize: 0, fSize: 1000000}, out: 1000},
	{in: blIn{blSize: 60, fSize: 100}, out: 60},
	{in: blIn{blSize: 59, fSize: 100}, out: 59},
	{in: blIn{blSize: 61, fSize: 60}, wantErr: true},
	{in: blIn{blSize: 100, fSize: 60}, wantErr: true},
	{in: blIn{blSize: 0, fSize: 10000}, out: DefaultBlockSize},
	{in: blIn{blSize: 0, fSize: 1000000}, out: 1000},
	{in: blIn{blSize: 0, fSize: 10000000}, out: 3160},
	{in: blIn{blSize: 0, fSize: 20000000}, out: 4472},
	{in: blIn{blSize: 0, fSize: 30000000}, out: 5472},
	{in: blIn{blSize: 0, fSize: 100000000}, out: 10000},
	{in: blIn{blSize: 0, fSize: 10000000000}, out: 100000},
	{in: blIn{blSize: 0, fSize: 20000000000}, out: 131064},
	{in: blIn{blSize: 0, fSize: 100000000000}, out: MaxBlockSize},
	{in: blIn{blSize: 0, fSize: 1000000000000}, out: MaxBlockSize},
}

func Test_decideBlockSize(t *testing.T) {
	for _, tt := range testsDecideBlSize {
		in := tt.in
		got, err := decideBlockSize(in.blSize, in.fSize)
		if (err != nil) != tt.wantErr {
			t.Errorf("decideBlockSize() error = %v, wantErr %v", err, tt.wantErr)
			continue
		}
		if got != tt.out {
			t.Errorf("decideBlockSize() = %v, want %v", got, tt.out)
		}
	}
}
