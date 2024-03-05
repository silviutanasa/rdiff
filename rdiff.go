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

// Operation represents the set of instructions given by the source to the target, in order to allow the target
// to update its content(target = source)
type Operation struct {
	Type       OpType
	BlockIndex int    // the index of the block from the target, for OpBlockNew -1 is used to enforce that the BlockIndex is not important in this case
	Data       []byte // additional literal data if the block was modified, or a new block if the Block was not matched (BlockIndex == 0)
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

func (r *rDiff) computeSignature(in io.Reader) ([]Block, error) {
	var output []Block
	block := make([]byte, r.blockSize)
	// it's enough a single Reset call, as the WriteAll method acts like a Reset and Write.
	r.weakHasher.Reset()
	for {
		n, err := in.Read(block)
		if n == 0 && err == io.EOF {
			break
		}
		if err != nil && err != io.EOF {
			return output, err
		}

		block = block[:n]
		r.strongHasher.Reset()
		r.strongHasher.Write(block)
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

func (r *rDiff) computeDelta(newData io.Reader, blockList []Block) ([]Operation, error) {
	output := make(map[int]Operation, len(blockList))
	searchList := computeSearchList(blockList)
	rolling := false
	block := make([]byte, r.blockSize)
	var lit []byte
	strongFound := false
	// it's enough a single Reset call, as the WriteAll method acts like a Reset and Write.
	r.weakHasher.Reset()
	for {
		// adjusting reading size to block of bytes or single byte
		// after a found match in the target, we need to read up to a full block
		// if rolling is in place, then we read up to a single byte
		if !rolling {
			block = block[:r.blockSize]
		} else {
			block = block[:1]
		}

		n, err := newData.Read(block)
		if n == 0 && err == io.EOF {
			// the last read block will not be added to the delta if it was not matched in the target,
			// so we need to add it to the literal collection
			if !strongFound {
				l := r.weakHasher.GetWindowContent()
				lit = append(lit, l...)
			}
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
			lit = append(lit, oldest)
		}

		if bl, found := searchList[r.weakHasher.Sum32()]; found {
			for i, element := range bl {
				r.strongHasher.Reset()
				r.strongHasher.Write(r.weakHasher.GetWindowContent())
				if bytes.Compare(element.strongHash, r.strongHasher.Sum(nil)) == 0 {
					opType := OpBlockKeep
					if len(lit) > 0 {
						opType = OpBlockUpdate
					}
					op := Operation{
						Type:       opType,
						BlockIndex: element.blockIndex,
					}
					op.Data = append(op.Data, lit...)
					output[element.blockIndex] = op
					rolling = false
					lit = lit[:0]

					strongFound = true

					//remove the strong hash from the list, because if we have identical blocks in the target,
					//then we'll always match the same block
					searchList[r.weakHasher.Sum32()] = slices.Delete(bl, i, i+1)

					break
				}
			}
			if strongFound {
				continue
			}
		}
		strongFound = false
		rolling = true
	}

	if len(lit) > 0 {
		op := Operation{
			Type:       OpBlockNew,
			BlockIndex: -1,
		}
		op.Data = append(op.Data, lit...)
		output[-1] = op
	}

	return updateDelta(blockList, output), nil
}

func computeSearchList(blockList []Block) map[uint32][]blockData {
	sl := make(map[uint32][]blockData, len(blockList))
	for i, block := range blockList {
		sl[block.WeakHash] = append(sl[block.WeakHash], blockData{strongHash: block.StrongHash, blockIndex: i})
	}

	return sl
}

func updateDelta(target []Block, delta map[int]Operation) []Operation {
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
