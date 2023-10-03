// SPDX-License-Identifier: MIT OR ISC
package electorium_testable_test

import (
	"testing"

	electorium "github.com/cjdelisle/Electorium_go"
)

func TestSimple(t *testing.T) {
	var votes []electorium.Vote
	votes = append(votes, electorium.Vote{
		/// The unique ID of the voter/candidate
		VoterId: "Alice",

		/// The unique ID of the candidate who they are voting for
		VoteFor: "Bob",

		/// How many votes they have - in a typical national election this would be 1
		/// In the case of stock companies, for instance, this would be number of shares.
		NumberOfVotes: 1,

		/// If this voter willing to also be a candidate for election?
		WillingCandidate: false,
	})
	votes = append(votes, electorium.Vote{
		VoterId:          "Bob",
		WillingCandidate: true,
	})
	vc := electorium.MkVoteCounter(votes, true)
	win := vc.FindWinner()
	if win == nil || win.VoterId != "Bob" {
		t.Fail()
	}
}
