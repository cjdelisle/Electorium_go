// SPDX-License-Identifier: MIT OR ISC
package electorium_testable_test

import (
	"fmt"
	"os"
	"testing"

	electorium_testable "github.com/cjdelisle/Electorium_go/test"
)

func FuzzVsRust(f *testing.F) {
	verbose := false
	for _, a := range os.Args {
		if a == "--manual" {
			fmt.Println("Manual mode enabled")
			verbose = true
		}
	}
	fuzz := electorium_testable.MkFuzz(verbose)
	f.Fuzz(func(t *testing.T, input []byte) {
		fuzz.FuzzCompare(input, verbose)
	})
}
