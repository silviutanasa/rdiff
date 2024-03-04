package rdiff

import (
	"bytes"
	"crypto/md5"
	"io"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func Test_computeSignature(t *testing.T) {
	weakHasher := newAdler32RollingHash()
	weakHash := func(p []byte) uint32 {
		weakHasher.WriteAll(p)
		return weakHasher.Sum32()
	}
	strongHasher := md5.New()
	strongHash := func(p []byte) []byte {
		strongHasher.Reset()
		strongHasher.Write(p)
		return strongHasher.Sum(nil)
	}
	type args struct {
		in        io.Reader
		blockSize int
	}
	tests := []struct {
		name string
		args args
		want []Block
	}{
		{
			name: "happy",
			args: args{
				in:        bytes.NewReader([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}),
				blockSize: 3,
			},
			want: []Block{
				{StrongHash: strongHash([]byte{1, 2, 3}), WeakHash: weakHash([]byte{1, 2, 3})},
				{StrongHash: strongHash([]byte{4, 5, 6}), WeakHash: weakHash([]byte{4, 5, 6})},
				{StrongHash: strongHash([]byte{7, 8, 9}), WeakHash: weakHash([]byte{7, 8, 9})},
				{StrongHash: strongHash([]byte{10}), WeakHash: weakHash([]byte{10})},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := rdiff{
				blockSize:    tt.args.blockSize,
				weakHasher:   newAdler32RollingHash(),
				strongHasher: md5.New(),
			}
			got, _ := r.computeSignature(tt.args.in)
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("computeSignature() = %v, want %v, \nDIFF: %v", got, tt.want, diff)
			}
		})
	}
}

func Test_computeDelta(t *testing.T) {
	weakHasher := newAdler32RollingHash()
	weakHash := func(p []byte) uint32 {
		weakHasher.WriteAll(p)
		return weakHasher.Sum32()
	}
	strongHasher := md5.New()
	strongHash := func(p []byte) []byte {
		strongHasher.Reset()
		strongHasher.Write(p)
		return strongHasher.Sum(nil)
	}
	type args struct {
		blockSize int
		blockList []Block
		newData   io.Reader
	}

	tests := []struct {
		name    string
		args    args
		want    []Operation
		wantErr bool
	}{
		{
			name: "compute the diff1",
			args: args{
				blockSize: 3,
				blockList: []Block{
					{WeakHash: weakHash([]byte{1, 2, 3}), StrongHash: strongHash([]byte{1, 2, 3})},
					{WeakHash: weakHash([]byte{4, 5, 6}), StrongHash: strongHash([]byte{4, 5, 6})},
					{WeakHash: weakHash([]byte{1, 2, 3}), StrongHash: strongHash([]byte{1, 2, 3})},
					{WeakHash: weakHash([]byte{7, 8}), StrongHash: strongHash([]byte{7, 8})},
				},
				newData: bytes.NewReader([]byte{11, 5, 22, 1, 2, 3, 88, 4, 5, 6, 1, 2, 3, 7, 8, 9, 10, 11, 12, 13, 14, 15, 29}),
			},
			want: []Operation{
				{Type: OpBlockUpdate, BlockIndex: 0, Data: []byte{11, 5, 22}},
				{Type: OpBlockUpdate, BlockIndex: 1, Data: []byte{88}},
				{Type: OpBlockKeep, BlockIndex: 2, Data: nil},
				{Type: OpBlockRemove, BlockIndex: 3, Data: nil},
				{Type: OpBlockNew, BlockIndex: -1, Data: []byte{7, 8, 9, 10, 11, 12, 13, 14, 15, 29}},
			},
		},
		{
			name: "compute the diff2",
			args: args{
				blockSize: 3,
				blockList: []Block{
					{WeakHash: weakHash([]byte{1, 2, 3}), StrongHash: strongHash([]byte{1, 2, 3})},
					{WeakHash: weakHash([]byte{4, 5, 6}), StrongHash: strongHash([]byte{4, 5, 6})},
				},
				newData: bytes.NewReader([]byte{1, 2, 3, 4, 5, 6, 7, 8}),
			},
			want: []Operation{
				{Type: OpBlockKeep, BlockIndex: 0, Data: nil},
				{Type: OpBlockKeep, BlockIndex: 1, Data: nil},
				{Type: OpBlockNew, BlockIndex: -1, Data: []byte{7, 8}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := rdiff{
				blockSize:    tt.args.blockSize,
				weakHasher:   newAdler32RollingHash(),
				strongHasher: md5.New(),
			}
			got, err := r.computeDelta(tt.args.newData, tt.args.blockList)
			//fmt.Printf("\nDELTA: %v", got)
			if (err != nil) != tt.wantErr {
				t.Errorf("computeDelta() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("computeDelta() got = %v, want %v, \nDIFF: %v", got, tt.want, diff)
			}
		})
	}
}
