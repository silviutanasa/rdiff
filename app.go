package rdiff

import (
	"crypto/md5" // nolint
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
)

const (
	// DefaultBlockSize is the default block size value, in bytes, used by the system
	DefaultBlockSize = 700
	// MaxBlockSize is the max block size value, in bytes, used by the system
	MaxBlockSize = 1 << 17
)

// App is the application layer of the RDiff service.
// It exposes the public API and allows for IO interactions.
type App struct {
	diffEngine *rDiff
}

// New constructs the RDiff app instance and returns a pointer to it.
// It accepts a blockSize as input, representing the size, in bytes, for splitting the target in blocks,
// in order to compute the target's signature.
// A blockSize <=0 means the size, in bytes, ii computed dynamically.
func New(blockSize int) *App {
	return &App{
		// nolint
		diffEngine: newRDiff(blockSize, newAdler32RollingHash(), md5.New()),
	}
}

// Signature computes the signature of a target file(targetFilePath) and writes it to an output file(outputFilePath)
// The target file(targetFileName) must exist, otherwise it returns an appropriate non-nil error.
// If the output file(outputFilePath) already exists, it returns an appropriate non-nil error.
// The content written to outputFilePath is serialized using gob encoding.
func (a *App) Signature(targetFilePath string, signatureFilePath string) error {
	targetFile, err := os.Open(targetFilePath)
	if err != nil {
		return err
	}
	tfInfo, err := targetFile.Stat()
	if err != nil {
		return err
	}
	targetFileSize := tfInfo.Size()
	if targetFileSize <= 0 {
		return errors.New("the target file is empty")
	}

	a.diffEngine.blockSize, err = decideBlockSize(a.diffEngine.blockSize, targetFileSize)
	if err != nil {
		return err
	}

	signatureFile, err := os.OpenFile(signatureFilePath, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
	if err != nil {
		return err
	}

	err = a.signature(targetFile, signatureFile)
	err1 := targetFile.Close()
	err2 := signatureFile.Close()

	return errors.Join(err, err1, err2)
}

// Delta computes the instruction list(operations list) in order for the target
// to be able to update its content to match the source.
// The signature file(signatureFilePath) and the source file(sourceFilePath) must exist,
// otherwise a non-nil error is returned.
// The delta file(deltaFilePath) must not exist, otherwise a non-nil error is returned.
// The content written to deltaFilePath is serialized using gob encoding.
func (a *App) Delta(signatureFilePath string, sourceFilePath string, deltaFilePath string) error {
	signatureFile, err := os.Open(signatureFilePath)
	if err != nil {
		return err
	}
	sourceFile, err := os.Open(sourceFilePath)
	if err != nil {
		return err
	}
	deltaFile, err := os.OpenFile(deltaFilePath, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
	if err != nil {
		return err
	}

	err = a.delta(signatureFile, sourceFile, deltaFile)
	err1 := signatureFile.Close()
	err2 := sourceFile.Close()
	err3 := deltaFile.Close()

	return errors.Join(err, err1, err2, err3)
}

// delta is the lower layer that performs the delta computation and data serialization.
func (a *App) delta(signature, source io.Reader, output io.Writer) error {
	var blockList []Block
	err := gob.NewDecoder(signature).Decode(&blockList)
	if err != nil {
		return err
	}
	delta, err := a.diffEngine.ComputeDelta(source, blockList)
	if err != nil {
		return err
	}

	return gob.NewEncoder(output).Encode(delta)
}

// signature is the lower layer that performs the signature computation and data serialization.
func (a *App) signature(target io.Reader, output io.Writer) error {
	signature, err := a.diffEngine.ComputeSignature(target)
	if err != nil {
		return err
	}

	return gob.NewEncoder(output).Encode(signature)
}

// computeDynamicBlockSize is the actual rsync algorithm for computing the dynamic block size, based on the file length.
// it does some computation to evenly distribute the blockSize according to fLen size.
func computeDynamicBlockSize(fLen int64) int64 {
	if fLen <= DefaultBlockSize*DefaultBlockSize {
		return DefaultBlockSize
	}

	var c, l, cnt int64
	for c, l, cnt = 1, fLen, 0; cnt < l<<2; c, l, cnt = c<<1, l>>2, cnt+1 {
	}
	if c < 0 || c >= MaxBlockSize {
		return MaxBlockSize
	}

	var blSize int64
	for c >= 8 {
		blSize |= c
		if fLen < blSize*blSize {
			blSize &= ^c
		}
		c >>= 1
	}

	return max(blSize, DefaultBlockSize)
}

// decideBlockSize make the logical decision against splitting a fileSize in blockSizes
// if blockSize <= 0 a new blocSize is dynamically computed and returned
// if blockSize > 0 it is validated against splitting the fileSize in at least 2 parts and returns a non-nil error if
// it's not able to do so (ex: round(fileSize/blockSize < 2)
func decideBlockSize(blockSize int, fileSize int64) (int, error) {
	// if provided blockSize, validate against min nr of chunks
	if blockSize > 0 {
		nrOfChunks := int(math.Ceil(float64(fileSize) / float64(blockSize)))
		// check that the diff makes sense - at least 2 blocks for the signature file
		if nrOfChunks < 2 {
			return 0, fmt.Errorf(
				"the current blockSize(%v) doesn't allow splitting target(size: %v) in at least 2 blocks/chunks",
				blockSize,
				fileSize,
			)
		}

		return blockSize, nil
	}
	// if not provided blockSize, switch to dynamic and adjust if block size is too small
	blockSize = int(computeDynamicBlockSize(fileSize))
	nrOfChunks := int(math.Ceil(float64(fileSize) / float64(blockSize)))
	// ensure a min of 2 chunks for the dynamically computed size
	if nrOfChunks < 2 {
		blockSize = int(math.Floor(float64(fileSize) / 2))
	}

	return blockSize, nil
}
