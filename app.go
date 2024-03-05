package rdiff

import (
	"crypto/md5"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
)

// App is the application layer of the RDiff service.
// It exposes the public API and allows for IO interactions.
type App struct {
	rDiff
}

// New constructs the RDiff app instance and returns a pointer to it.
// It accepts a blockSize as input, representing the size, in bytes, for splitting the target in blocks,
// in order to compute the target's signature.
// A blockSize <=0 means the size, in bytes, ii computed dynamically.
func New(blockSize int) *App {
	return &App{
		rDiff{
			blockSize:    blockSize,
			weakHasher:   newAdler32RollingHash(),
			strongHasher: md5.New(),
		},
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
	// try dynamic blockSize - TODO: try to find an algorithm with better distribution
	if a.blockSize <= 0 {
		a.blockSize = int(math.Ceil(float64(targetFileSize) / 2))
	}
	nrOfChunks := int(math.Ceil(float64(targetFileSize) / float64(a.blockSize)))
	if nrOfChunks < 2 {
		return fmt.Errorf(
			"the current blockSize(%v) doesn't allow splitting target(size: %v) in at least 2 blocks/chunks",
			a.blockSize,
			targetFileSize,
		)
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
	delta, err := a.computeDelta(source, blockList)
	if err != nil {
		return err
	}

	return gob.NewEncoder(output).Encode(delta)
}

// signature is the lower layer that performs the signature computation and data serialization.
func (a *App) signature(target io.Reader, output io.Writer) error {
	signature, err := a.computeSignature(target)
	if err != nil {
		return err
	}

	return gob.NewEncoder(output).Encode(signature)
}
