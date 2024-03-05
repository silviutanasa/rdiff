package rdiff

import (
	"bytes"
	"crypto/md5"
	"testing"

	"github.com/google/go-cmp/cmp"
)

var rDiffE2ETests = []struct {
	in      inE2E
	out     []Operation
	wantErr bool
}{
	{
		in: inE2E{
			blockSize: 3,
			target:    []byte{1, 2, 3, 4, 5, 6, 1, 2, 3, 7, 8},
			source:    []byte{11, 5, 22, 1, 2, 3, 88, 4, 5, 6, 1, 2, 3, 7, 8, 9, 10, 11, 12, 13, 14, 15, 29},
		},
		out: []Operation{
			{Type: OpBlockUpdate, BlockIndex: 0, Data: []byte{11, 5, 22}},
			{Type: OpBlockUpdate, BlockIndex: 1, Data: []byte{88}},
			{Type: OpBlockKeep, BlockIndex: 2},
			{Type: OpBlockRemove, BlockIndex: 3},
			{Type: OpBlockNew, BlockIndex: -1, Data: []byte{7, 8, 9, 10, 11, 12, 13, 14, 15, 29}},
		},
	},
	{
		in: inE2E{
			blockSize: 3,
			target:    []byte{1, 2, 3, 4, 5, 6},
			source:    []byte{1, 2, 3, 4, 5, 6, 1, 2, 3, 7, 8},
		},
		out: []Operation{
			{Type: OpBlockKeep, BlockIndex: 0},
			{Type: OpBlockKeep, BlockIndex: 1},
			{Type: OpBlockNew, BlockIndex: -1, Data: []byte{1, 2, 3, 7, 8}},
		},
	},
	{
		in: inE2E{
			blockSize: 2,
			target:    []byte{1, 2, 3, 4, 5, 6, 7},
			source:    []byte{3, 4, 5, 6, 7, 8},
		},
		out: []Operation{
			{Type: OpBlockRemove, BlockIndex: 0},
			{Type: OpBlockKeep, BlockIndex: 1},
			{Type: OpBlockKeep, BlockIndex: 2},
			{Type: OpBlockRemove, BlockIndex: 3},
			{Type: OpBlockNew, BlockIndex: -1, Data: []byte{7, 8}},
		},
	},
	{
		in: inE2E{
			blockSize: 2,
			target:    []byte{1, 2, 3, 4, 5, 6, 7},
			source:    []byte{1, 2, 3, 4, 5, 6, 7},
		},
		out: []Operation{
			{Type: OpBlockKeep, BlockIndex: 0},
			{Type: OpBlockKeep, BlockIndex: 1},
			{Type: OpBlockKeep, BlockIndex: 2},
			{Type: OpBlockKeep, BlockIndex: 3},
		},
	},
	{
		in: inE2E{
			blockSize: 3,
			target:    nil,
			source:    []byte{3, 4, 5, 6, 7, 8},
		},
		out: []Operation{
			{Type: OpBlockNew, BlockIndex: -1, Data: []byte{3, 4, 5, 6, 7, 8}},
		},
	},
	{
		in: inE2E{
			blockSize: 3,
			target:    []byte{1, 2, 3, 4, 5, 6},
			source:    nil,
		},
		out: []Operation{
			{Type: OpBlockRemove, BlockIndex: 0},
			{Type: OpBlockRemove, BlockIndex: 1},
		},
	},
	{
		in: inE2E{
			blockSize: 0,
			target:    nil,
			source:    nil,
		},
		out: []Operation{},
	},
}

type inE2E struct {
	blockSize int
	target    []byte
	source    []byte
}

// TestRDiffE2E performs an "E2E" cycle for the rDiff flow.
// Actors:
//   - target: an existing 'file';
//   - source: a new 'file'
//   - signature: a 'file' containing the target split in 'block'
//   - delta: a 'file' containing the operations set needed to update the target content to match the source
//
// Expected results: a set of operations(description) based on which we can update the target content to match source's content(target==source)
// Flow:
// 1. starting from an input(target and block size), compute the target signature
// 2. starting from a computed target signature(computed at step 1), compute the delta/diff
//
// The decision to create a single suite to cover both signature and delta, was taken in order to make testing more accurate,
// more readable and easier to support/extend in the future. Also, another reason is that ComputeSignature and ComputeDelta were meant
// to 'be together', joke aside ComputeDelta has no purpose if the ComputeSignature is not previously called.
// The approach also well covers both methods (ex: if the ComputeSignature has bugs, it will cause ComputeDelta
// to return bad results also), so there is no downside, only upsides.
func TestRDiffE2E(t *testing.T) {
	for _, tt := range rDiffE2ETests {
		inp := tt.in
		r := rDiff{
			blockSize:    inp.blockSize,
			weakHasher:   newAdler32RollingHash(),
			strongHasher: md5.New(),
		}
		sig, err := r.ComputeSignature(bytes.NewReader(inp.target))
		var got []Operation
		if err == nil {
			got, err = r.ComputeDelta(bytes.NewReader(inp.source), sig)
		}
		if (err != nil) != tt.wantErr {
			t.Errorf("rDiff E2E error = %v, wantErr %v", err, tt.wantErr)
			return
		}
		if diff := cmp.Diff(got, tt.out); diff != "" {
			t.Errorf("rDiff E2E got = %v, want %v, \nDIFF: %v", got, tt.out, diff)
		}
	}
}
