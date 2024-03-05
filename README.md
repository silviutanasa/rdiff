# RDiff

[![GoDoc][doc-img]][doc]
[![Go Report Card][go-report-img]][go-report]

Package rdiff provides file diff between a source and a target, expressed as a collection of operations, to be applied
to the target in order to update its content to match the source.

## Installation:

```
go get github.com/silviutanasa/rdiff
```

Note that the minimum supported version is Go v1.21.

## Usage:

```Go
package main

import (
	"encoding/gob"
	"fmt"
	"log"
	"os"

	"github.com/silviutanasa/rdiff"
)

func main() {
	// test_target and test_source must exist
	// test_signature and test_delta must not exist
	app := rdiff.New(5)
	err := app.Signature("test_target", "test_signature")
	if err != nil {
		log.Fatal(err)
	}

	err = app.Delta("test_signature", "test_source", "test_delta")
	if err != nil {
		log.Fatal(err)
	}

	// inspect the delta output
	delta, err := os.Open("test_delta")
	if err != nil {
		log.Fatal(err)
	}
	defer delta.Close()

	var ops []rdiff.Operation
	gob.NewDecoder(delta).Decode(&ops)
	// the delta should be a []Operation
	// where the Operation is defined as follows:
	// type Operation struct {
	//	 Type       OpType
	//   // the index of the block from the target, for OpBlockNew -1 is used to enforce that the BlockIndex is not important in this case
	//	 BlockIndex int
	//   // additional literal data if the block was modified, or a new block if the Block was not matched (BlockIndex == 0)
	//	 Data       []byte
	// }
	// where the operations are described as follows: 
	// OpBlockKeep
	// OpBlockUpdate
	// OpBlockRemove means there is no match for a target block in the source
	// OpBlockRemove
	// OpBlockNew (as a convention BlockIndex will be -1, in this case, indicating that it has no purpose)
	fmt.Println(ops)
}

```

[doc-img]: https://pkg.go.dev/badge/silviutanasa/rdiff

[doc]: https://pkg.go.dev/github.com/silviutanasa/rdiff

[go-report-img]: https://goreportcard.com/badge/github.com/silviutanasa/rdiff

[go-report]: https://goreportcard.com/report/github.com/silviutanasa/rdiff