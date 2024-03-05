package rdiff_test

import (
	"encoding/gob"
	"fmt"
	"log"
	"os"

	"rdiff"
)

func ExampleNew() {
	// first create both target and source files
	err := os.WriteFile("test_source.bin", []byte{12, 32, 1, 2, 3, 4, 5, 6, 7, 8}, 0666)
	defer os.Remove("test_source.bin")
	if err != nil {
		log.Fatal(err)
	}
	err = os.WriteFile("test_target.bin", []byte{1, 2, 3, 4, 5, 6, 7}, 0666)
	defer os.Remove("test_target.bin")
	if err != nil {
		log.Fatal(err)
	}
	app := rdiff.New(3)

	// second process the Signature and then the Delta
	err = app.Signature("test_target.bin", "test_signature")
	defer os.Remove("test_signature")
	if err != nil {
		log.Fatal(err)
	}

	err = app.Delta("test_signature", "test_source.bin", "test_delta")
	if err != nil {
		log.Fatal(err)
	}

	// third inspect the Delta for results
	delta, err := os.Open("test_delta")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove("test_delta")
	defer delta.Close()

	var ops []rdiff.Operation
	gob.NewDecoder(delta).Decode(&ops)
	fmt.Println(ops)

	// Output:
	// [{1 0 [12 32]} {0 1 []} {2 2 []} {3 -1 [7 8]}]
}
