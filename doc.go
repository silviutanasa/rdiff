// Copyright 2024 Silviu Tanasă. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

/*
Package rdiff provides file diff between a source and a target, expressed as a collection of operations to be applied
to the target in order to update its content to match the source.

The public API exposes 3 operations: New, Signature and Delta

		// usage example:
		//
		// creates a new instance with a block size of 3 bytes.
		rd := rdiff.New(3)
	    // target_file_path must exist prior to this call
		// signature_file_path must not exist prior to this call
		// signature_file_path content will be serialized using gob encoding
		err := rd.Signature("target_file_path", "signature_file_path")
		if err != nil {
			return err
		}
		// signature_file_path must exist prior to this call
		// source_file_path must exist prior to this call
		// delta_file_path must not exist prior to this call
		// delta_file_path content will be serialized using gob encoding
		err = rd.Delta("signature_file_path", "source_file_path", "delta_file_path")
		if err != nil {
			...
		}
		...
*/
package rdiff
