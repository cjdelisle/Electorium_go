// SPDX-License-Identifier: MIT OR ISC
package electorium_testable

/*
#cgo LDFLAGS: -L../../Electorium/fuzzable/target/debug -lfuzzable
#include <stdbool.h>
#include "../../Electorium/fuzzable/electorium_fuzzable.h"
*/
import "C"

import (
	"fmt"
	"unsafe"

	electorium "github.com/cjdelisle/Electorium_go"
	"golang.org/x/exp/slices"
)

// [ Flags ][ VoterId ][ VoteFor ][ NumVotes ]
const VOTE_WIDTH = 4

var NAMES = getNames()

func mkId(input uint8) string {
	return NAMES[input]
}

func idToNum(vote *electorium.Vote) int16 {
	if vote == nil {
		return -1
	}
	return int16(slices.Index(NAMES, vote.VoterId))
}

func parseVote(data []byte) electorium.Vote {
	return electorium.Vote{
		WillingCandidate: data[0]&1 == 1,
		VoterId:          mkId(data[1]),
		VoteFor:          mkId(data[2]),
		NumberOfVotes:    uint64(data[3]),
	}
}

func mkVotes(data []byte) []electorium.Vote {
	okLen := len(data) / VOTE_WIDTH * VOTE_WIDTH
	out := make([]electorium.Vote, 0, okLen/VOTE_WIDTH)
	for i := 0; i < okLen; i += VOTE_WIDTH {
		out = append(out, parseVote(data[i:i+VOTE_WIDTH]))
	}
	return out
}

type Fuzz struct {
	cFuzz *C.Fuzz
}

func MkFuzz(verbose bool) Fuzz {
	return Fuzz{
		cFuzz: C.electorium_fuzz_new(C.bool(verbose)),
	}
}

func (f *Fuzz) FuzzCompare(data []byte, verbose bool) {
	if verbose {
		fmt.Println("Generating votes")
	}
	votes := mkVotes(data)
	if verbose {
		fmt.Println("Votes")
		for _, v := range votes {
			fmt.Printf("  - %s %d %s\n", v.VoterId, v.NumberOfVotes, v.VoteFor)
		}
	}
	if verbose {
		fmt.Println("Golang building table")
	}
	vc := electorium.MkVoteCounter(votes, verbose)
	if verbose {
		fmt.Println("Golang finding winner")
	}
	goWin := vc.FindWinner()
	if verbose {
		winner := "<nil>"
		if goWin != nil {
			winner = goWin.VoterId
		}
		fmt.Printf("Golang winner is %s\n", winner)
		fmt.Println("Computing Rust winner")
	}
	goWinNum := idToNum(goWin)
	ptr := unsafe.Pointer(nil)
	if len(data) > 0 {
		ptr = unsafe.Pointer(&data[0])
	}
	rsWin := C.electorium_fuzz_run(f.cFuzz, (*C.uchar)(ptr), C.ulong(len(data)))
	if goWinNum != int16(rsWin) {
		panic("Go found a different winner than rust")
	}
}
