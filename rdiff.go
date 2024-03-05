package rdiff

import (
	"bytes"
	"hash"
	"io"
	"slices"
)

// OpType represents a block operation/instruction, useful to recompute the target, based on source.
type OpType byte

const (
	// OpBlockKeep means a block is unchanged and should be kept as it is.
	OpBlockKeep OpType = iota
	// OpBlockUpdate means there is a match in the target, and it contains extra data(literal) compared to the initial block
	OpBlockUpdate
	// OpBlockRemove means there is no match for a target block in the source
	OpBlockRemove
	// OpBlockNew means there is a literal block in the source that doesn't have any match in the target - new data
	OpBlockNew
)

// Block represents a chunk of data(bytes) used by the target to split its data.
type Block struct {
	StrongHash []byte
	WeakHash   uint32
}

// Operation represents an instruction given by the source to the target, in order to allow the target to update its content.
type Operation struct {
	Type OpType
	// the index of the block from the target, for OpBlockNew -1 is used to enforce that the BlockIndex
	// is not important in this case
	BlockIndex int
	// additional literal data, if the block was modified, or a new block if the Block was not matched (BlockIndex == 0)
	Data []byte
}

// blockData is used to compute the block search list(map[uint32][]blockData)
type blockData struct {
	strongHash []byte
	blockIndex int
}

type rDiff struct {
	blockSize    int
	weakHasher   *adler32RollingHash
	strongHasher hash.Hash
}

func newRDiff(blockSize int, weakHasher *adler32RollingHash, strongHasher hash.Hash) *rDiff {
	return &rDiff{
		blockSize:    blockSize,
		weakHasher:   weakHasher,
		strongHasher: strongHasher,
	}
}

// ComputeSignature computes the signature of a target and returns a []Block, based on the blockSize.
// Every Block contains the weak hash and strong hash.
// It returns a non-nil error in case target encounters a reading error, other than io.EOF.
func (r *rDiff) ComputeSignature(target io.Reader) ([]Block, error) {
	var output []Block
	block := make([]byte, r.blockSize)
	// it's enough a single Reset call, as the WriteAll method acts like a Reset and Write.
	r.weakHasher.Reset()
	for {
		n, err := target.Read(block)
		if n == 0 && err == io.EOF {
			break
		}
		if err != nil && err != io.EOF {
			return output, err
		}

		block = block[:n]
		r.strongHasher.Reset()
		_, _ = r.strongHasher.Write(block)
		// it doesn't need reset, as it's always rewriting the digest
		r.weakHasher.WriteAll(block)
		bl := Block{
			StrongHash: r.strongHasher.Sum(nil),
			WeakHash:   r.weakHasher.Sum32(),
		}
		output = append(output, bl)
	}

	return output, nil
}

// ComputeDelta computes the instruction list(operations list) based on the target's blockList
// to be able to update its content to match the source.
func (r *rDiff) ComputeDelta(source io.Reader, blockList []Block) ([]Operation, error) {
	tempDelta := make(map[int]Operation, len(blockList))
	searchList := computeSearchList(blockList)
	block := make([]byte, r.blockSize)
	var literal []byte
	// it's enough a single Reset call, as the WriteAll method acts like a Reset and Write.
	r.weakHasher.Reset()
	rolling := false
	for {
		n, err := r.read(source, block, rolling)
		if n == 0 && err == io.EOF {
			break
		}
		if err != nil && err != io.EOF {
			return nil, err
		}

		block = block[:n]
		if !rolling {
			r.weakHasher.WriteAll(block)
		} else {
			oldest := r.weakHasher.Roll(block[0])
			literal = append(literal, oldest)
		}

		if blIdx := r.searchBlock(searchList, r.weakHasher.Sum32()); blIdx != -1 {
			rolling = false

			tempDelta[blIdx] = createOperation(blIdx, literal)
			literal = literal[:0]

			continue
		}

		rolling = true
	}

	r.updateDeltaWithLiteralBlockOperation(tempDelta, rolling, literal)

	return computeFinalDelta(blockList, tempDelta), nil
}
func (r *rDiff) updateDeltaWithLiteralBlockOperation(delta map[int]Operation, rolling bool, literal []byte) {
	// the last read block will not be added to the delta if it was not matched in the target,
	// so we need to add it to the literal collection
	if rolling {
		l := r.weakHasher.GetWindowContent()
		literal = append(literal, l...)
	}
	// collecting leftovers literals into a single new block
	if len(literal) > 0 {
		op := Operation{
			Type:       OpBlockNew,
			BlockIndex: -1,
		}
		op.Data = append(op.Data, literal...)
		delta[-1] = op
	}
}
func (r *rDiff) read(reader io.Reader, block []byte, rolling bool) (int, error) {
	// adjusting reading size to block of bytes or single byte
	// after a found match in the target, we need to read up to a full block
	// if rolling is in place, then we read up to a single byte
	if !rolling {
		block = block[:r.blockSize]
	} else {
		block = block[:1]
	}

	return reader.Read(block)
}
func (r *rDiff) searchBlock(searchList map[uint32][]blockData, weakHash uint32) int {
	if bl, found := searchList[weakHash]; found {
		r.strongHasher.Reset()
		currBlockContent := r.weakHasher.GetWindowContent()
		// nolint
		r.strongHasher.Write(currBlockContent)
		strongHash := r.strongHasher.Sum(nil)
		blFoundIdx := slices.IndexFunc(bl, func(el blockData) bool { return bytes.Equal(el.strongHash, strongHash) })
		if blFoundIdx != -1 {
			blockIndex := bl[blFoundIdx].blockIndex
			//remove the strong hash from the list, because if we have identical blocks in the target,
			//then we'll always match the same block
			searchList[weakHash] = slices.Delete(bl, blFoundIdx, blFoundIdx+1)

			return blockIndex
		}
	}

	return -1
}

func createOperation(index int, lit []byte) Operation {
	opType := OpBlockKeep
	if len(lit) > 0 {
		opType = OpBlockUpdate
	}
	op := Operation{
		Type:       opType,
		BlockIndex: index,
	}
	op.Data = append(op.Data, lit...)

	return op
}

func computeSearchList(blockList []Block) map[uint32][]blockData {
	sl := make(map[uint32][]blockData, len(blockList))
	for i, block := range blockList {
		sl[block.WeakHash] = append(sl[block.WeakHash], blockData{strongHash: block.StrongHash, blockIndex: i})
	}

	return sl
}

func computeFinalDelta(target []Block, delta map[int]Operation) []Operation {
	// len(target)+1 is used to cover the max possible size: all target blocks + 1 extra literal block(if any)
	output := make([]Operation, 0, len(target)+1)
	for i := range target {
		op, ok := delta[i]
		if !ok {
			removed := Operation{
				Type:       OpBlockRemove,
				BlockIndex: i,
			}
			output = append(output, removed)

			continue
		}
		output = append(output, op)
	}
	if extra, ok := delta[-1]; ok {
		output = append(output, extra)
	}

	return output
}
