// SPDX-License-Identifier: MIT OR ISC
package electorium

import (
	"encoding/binary"
	"fmt"
	"sort"

	"golang.org/x/crypto/blake2b"
	"golang.org/x/exp/slices"
)

type Vote struct {
	/// The unique ID of the voter/candidate
	VoterId string

	/// The unique ID of the candidate who they are voting for
	VoteFor string

	/// How many votes they have - in a typical national election this would be 1
	/// In the case of stock companies, for instance, this would be number of shares.
	NumberOfVotes uint64

	/// If this voter willing to also be a candidate for election?
	WillingCandidate bool
}

type candidate struct {
	/// A reference to the Vote object which corrisponds to this candidate
	vote *Vote
	/// The index of the Candidate who they voted for, if any
	voteFor *candidate
	/// The number of indirect votes which would be received if every candidate
	/// delegated their votes.
	totalIndirectVotes uint64
	/// The first candidate who voted for voted for this candidate.
	/// This and voting_for_same are used to create a linked list.
	votedForMe []*candidate
	/// True if this is someone who is willing to potentially win the election.
	willingCandidate bool
	/// Forms a linked list of candidates ordered by total indirect votes, descending
	/// Non-willing candidates are not included.
	nextByTotalIndirectVotes *candidate
}

func mkCandidates(votes []Vote, verbose bool) ([]candidate, int) {
	candidateIdxByName := make(map[string]*candidate)
	cands := make([]candidate, 0, len(votes))
	totalWilling := 0
	for _, willing := range []bool{true, false} {
		for i, v := range votes {
			if v.WillingCandidate != willing {
				continue
			}
			if _, ok := candidateIdxByName[v.VoterId]; ok {
				if verbose {
					fmt.Printf("Duplicate voter %s, invalid\n", v.VoterId)
				}
				continue
			}
			if willing {
				totalWilling += 1
			}
			cands = append(cands, candidate{
				vote:                     &votes[i],
				voteFor:                  nil,
				totalIndirectVotes:       v.NumberOfVotes,
				votedForMe:               make([]*candidate, 0),
				willingCandidate:         willing,
				nextByTotalIndirectVotes: nil,
			})
			candidateIdxByName[v.VoterId] = &cands[len(cands)-1]
		}
	}
	for i := 0; i < len(cands); i++ {
		if cands[i].vote.VoteFor == cands[i].vote.VoterId {
			if verbose {
				fmt.Printf("Candidate %s voted for themselves\n", cands[i].vote.VoteFor)
			}
			continue
		}
		ok := false
		cands[i].voteFor, ok = candidateIdxByName[cands[i].vote.VoteFor]
		if verbose {
			if ok {
				fmt.Printf("Candidate %s voted for %s\n", cands[i].vote.VoterId, cands[i].voteFor.vote.VoterId)
			} else {
				fmt.Printf("Candidate %s voted for nonexistent %s\n", cands[i].vote.VoterId, cands[i].vote.VoteFor)
			}
		}
	}
	return cands, totalWilling
}

func computeDelegatedVotes(cand []candidate, verbose bool) {
	delegationPath := make([]*candidate, 0)
	for i := 0; i < len(cand); i++ {
		c := cand[i]
		voteFor := c.voteFor
		origVote := c.vote
		votes := origVote.NumberOfVotes
		if verbose {
			fmt.Printf("Delegating [%d] votes from %s\n", votes, c.vote.VoterId)
		}
		if voteFor != nil {
			voteFor.votedForMe = append(voteFor.votedForMe, &cand[i])
		} else if verbose {
			fmt.Printf("  - %s did not vote for anyone\n", c.vote.VoterId)
		}
		delegationPath = delegationPath[:0]
		delegationPath = append(delegationPath, &cand[i])
		for voteFor != nil {
			if slices.Contains(delegationPath, voteFor) {
				if verbose {
					fmt.Printf("  - Encountered a ring at %s\n", voteFor.vote.VoterId)
				}
				// It's a ring, we already delegated to them, abort.
				break
			}
			delegationPath = append(delegationPath, voteFor)
			voteFor.totalIndirectVotes += votes
			if verbose {

				fmt.Printf("  - Delegating to %s (new score: %d)\n",
					voteFor.vote.VoterId, voteFor.totalIndirectVotes)
			}
			voteFor = voteFor.voteFor
		}
	}
}

func orderByTotalIndirect(cand []candidate, totalWillingCandidates int) *candidate {
	sortable := make([]*candidate, 0, totalWillingCandidates)
	for i := 0; i < totalWillingCandidates; i++ {
		c := &cand[i]
		if !c.willingCandidate {
			panic("Non-willing candidate in beginning of list")
		}
		sortable = append(sortable, c)
	}
	sort.Slice(sortable, func(a, b int) bool {
		return sortable[a].totalIndirectVotes < sortable[b].totalIndirectVotes
	})
	var last *candidate
	for _, c := range sortable {
		//fmt.Printf("c = %s\n", c.vote.VoterId)
		c.nextByTotalIndirectVotes = last
		last = c
	}
	return last
}

type ringComputer struct {
	rings       [][]*candidate
	unorganized []*candidate
}

func (rc *ringComputer) removeFromUnorganized(cand *candidate) bool {
	for i, c := range rc.unorganized {
		if c == cand {
			rc.unorganized[i] = rc.unorganized[len(rc.unorganized)-1]
			rc.unorganized = rc.unorganized[0 : len(rc.unorganized)-1]
			return true
		}
	}
	return false
}
func (rc *ringComputer) addToRingForward(ringN int, cand *candidate) {
	if cand == nil {
		return
	}
	if !rc.removeFromUnorganized(cand) {
		return
	}
	rc.addToRingReverse(ringN, cand)
	rc.rings[ringN] = append(rc.rings[ringN], cand)
	rc.addToRingForward(ringN, cand.voteFor)
}
func (rc *ringComputer) addToRingReverse(ringN int, cand *candidate) {
	for _, c := range cand.votedForMe {
		if rc.removeFromUnorganized(c) {
			rc.rings[ringN] = append(rc.rings[ringN], c)
			rc.addToRingReverse(ringN, c)
		}
	}
}
func computeRingMembers(ring []*candidate) [][]*candidate {
	rc := ringComputer{}
	rc.unorganized = append(rc.unorganized, ring...)
	for len(rc.unorganized) > 0 {
		n := len(rc.rings)
		rc.rings = append(rc.rings, make([]*candidate, 0))
		c := rc.unorganized[0]
		rc.addToRingForward(n, c)
	}
	return rc.rings
}

func getBestCandidates(
	cand []candidate,
	best *candidate,
	verbose bool,
) ([]*candidate, int) {
	var bestRing []*candidate
	nextC := best
	score := best.totalIndirectVotes
	for nextC != nil && nextC.totalIndirectVotes == score {
		bestRing = append(bestRing, nextC)
		nextC = nextC.nextByTotalIndirectVotes
	}
	if verbose {
		fmt.Println("Best ring:")
		for _, c := range bestRing {
			fmt.Printf("  - %s with %d possible votes\n", c.vote.VoterId, c.totalIndirectVotes)
		}
	}
	members := computeRingMembers(bestRing)
	if verbose {
		fmt.Printf("Best ring made up of %d distinct rings:\n", len(members))
		for i, m := range members {
			fmt.Printf("  - %d:\n", i)
			for _, mm := range m {
				fmt.Printf("    - %s\n", mm.vote.VoterId)
			}
		}
	}
	return bestRing, len(members)
}

func bestOfRing(ring []*candidate, verbose bool) []*candidate {
	type candScore struct {
		c     *candidate
		score uint64
	}
	var scores []candScore
	for _, c := range ring {
		score := c.vote.NumberOfVotes
		for _, vfm := range c.votedForMe {
			if !slices.Contains(ring, vfm) {
				score += vfm.totalIndirectVotes
			}
		}
		scores = append(scores, candScore{c, score})
	}
	winningCount := uint64(0)
	var out []*candidate
	for _, cs := range scores {
		if verbose {
			fmt.Printf("  - %s has score %d (without ring)\n", cs.c.vote.VoterId, cs.score)
		}
		if cs.score >= winningCount {
			if cs.score > winningCount {
				out = out[:0]
				winningCount = cs.score
			}
			out = append(out, cs.c)
		}
	}
	return out
}

func getRunnerUp(
	tenativeWinner *candidate,
	excludeRing []*candidate,
) *candidate { // may return nil
	ru := tenativeWinner.nextByTotalIndirectVotes
	for ru != nil {
		if !slices.Contains(excludeRing, ru) {
			return ru
		}
		ru = ru.nextByTotalIndirectVotes
	}
	return nil
}

func getPotentialPatron(
	current *candidate,
	excludeRing []*candidate,
	verbose bool,
) *candidate { // maybe nil
	bestScore := uint64(0)
	var bestCand *candidate
	for _, vfm := range current.votedForMe {
		if verbose {
			fmt.Printf("getPotentialPatron %s\n", vfm.vote.VoterId)
		}
		if slices.Contains(excludeRing, vfm) {
			if verbose {
				fmt.Printf("  - Excluded because part of ring\n")
			}
		} else if vfm.totalIndirectVotes > bestScore {
			if verbose {
				fmt.Printf("  - Candidate with score %d\n", vfm.totalIndirectVotes)
			}
			bestScore = vfm.totalIndirectVotes
			bestCand = vfm
		}
	}
	return bestCand
}

func isValidPatron(
	tenativeWinner *candidate,
	patron *candidate,
	runnerUp *candidate, // maybe nil
	verbose bool,
) bool {
	markToBeat := tenativeWinner.totalIndirectVotes / 2
	if verbose {
		fmt.Printf("Trying potential patron %s with score %d, mark to beat is %d\n",
			patron.vote.VoterId, patron.totalIndirectVotes, markToBeat)
	}
	if !patron.willingCandidate {
		if verbose {
			fmt.Printf("  - Potential patron is not willing candidate, discard\n")
		}
		return false
	} else if patron.totalIndirectVotes <= markToBeat {
		if verbose {
			fmt.Printf("  - Potential patron only has score %d, mark to beat is %d\n",
				patron.totalIndirectVotes, markToBeat)
		}
		return false
	} else {
		if runnerUp != nil {
			if patron.totalIndirectVotes <= runnerUp.totalIndirectVotes {
				if verbose {
					fmt.Printf("  - Potential patron can't beat runner-up %s with %d\n",
						runnerUp.vote.VoterId, runnerUp.totalIndirectVotes)
				}
				return false
			} else {
				if verbose {
					fmt.Printf("  - Potential patron BEATS runner-up %s with %d\n",
						runnerUp.vote.VoterId, runnerUp.totalIndirectVotes)
				}
				return true
			}
		} else {
			if verbose {
				fmt.Printf("  - No runner-up found, potential patron beats tenative winner\n")
			}
			return true
		}
	}
}

func getPatron(
	tenativeWinner *candidate,
	excludeRing []*candidate,
	verbose bool,
) *candidate { // maybe nil
	runnerUp := getRunnerUp(tenativeWinner, excludeRing)

	if verbose {
		fmt.Printf("Tenative winner is %s with %d potential votes\n",
			tenativeWinner.vote.VoterId, tenativeWinner.totalIndirectVotes)
	}

	potentialPatron := getPotentialPatron(tenativeWinner, excludeRing, verbose)
	if potentialPatron == nil {
		return nil
	}

	var patron *candidate
	for {
		if runnerUp == potentialPatron {
			runnerUp = runnerUp.nextByTotalIndirectVotes
		}
		if !isValidPatron(tenativeWinner, potentialPatron, runnerUp, verbose) {
			break
		}
		patron = potentialPatron
		potentialPatron = getPotentialPatron(potentialPatron, excludeRing, verbose)
		if potentialPatron == nil {
			break
		}
	}

	return patron
}

func solveWinner(
	tenativeWinners []*candidate,
	bestRing []*candidate,
	verbose bool,
) []*candidate {
	if len(tenativeWinners) != 1 {
		return tenativeWinners
	}
	patron := getPatron(tenativeWinners[0], bestRing, verbose)
	if patron == nil {
		patron = tenativeWinners[0]
	}
	return []*candidate{patron}
}

func tieBreakerHash(c *candidate, verbose bool) []byte {
	hasher, _ := blake2b.New512(nil)
	if verbose {
		fmt.Printf("Deterministic Tie Breaker Hash for %s w/ %d -> ",
			c.vote.VoterId, c.totalIndirectVotes)
		for _, b := range []byte(c.vote.VoterId) {
			fmt.Printf("%02x", b)
		}
	}
	hasher.Write([]byte(c.vote.VoterId))
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, c.totalIndirectVotes)
	if verbose {
		for _, b := range b {
			fmt.Printf("%02x", b)
		}
		fmt.Println()
	}
	hasher.Write(b)
	return hasher.Sum(nil)
}

func tieBreaker(winners []*candidate, verbose bool) *candidate {
	if len(winners) == 0 {
		return nil
	} else if len(winners) == 1 {
		return winners[0]
	} else {
		type hashCand struct {
			h []byte
			c *candidate
		}
		if verbose {
			fmt.Println("Tie breaker needed:")
		}
		hashes := make([]hashCand, 0, len(winners))
		for _, c := range winners {
			h := tieBreakerHash(c, verbose)
			hashes = append(hashes, hashCand{c: c, h: h})
		}
		sort.Slice(hashes, func(a, b int) bool {
			return slices.Compare(hashes[a].h, hashes[b].h) < 0
		})
		if verbose {
			for _, hc := range hashes {
				h := hc.h
				c := hc.c
				fmt.Printf("  - Hash %02x%02x%02x%02x for %s with %d\n",
					h[0], h[1], h[2], h[3], c.vote.VoterId, c.totalIndirectVotes)
			}
		}
		return hashes[0].c
	}
}

type VoteCounter struct {
	cand                   []candidate
	totalWillingCandidates int
	best                   *candidate
	verbose                bool
}

func MkVoteCounter(votes []Vote, verbose bool) VoteCounter {
	cand, totalWillingCandidates := mkCandidates(votes, verbose)
	out := VoteCounter{
		cand:                   cand,
		totalWillingCandidates: totalWillingCandidates,
		best:                   nil,
		verbose:                verbose,
	}
	out.computeDelegatedVotes()
	return out
}

func (vc *VoteCounter) computeDelegatedVotes() {
	computeDelegatedVotes(vc.cand, vc.verbose)
	vc.best = orderByTotalIndirect(vc.cand, vc.totalWillingCandidates)
}

func (vc *VoteCounter) FindWinner() *Vote {
	if vc.best == nil {
		if vc.verbose {
			fmt.Println("Best is nil so winner is nil")
		}
		return nil
	}
	if vc.verbose {
		fmt.Println("Vote ranking")
		c := vc.best
		for c != nil {
			fmt.Printf("  - %s with %d possible votes\n", c.vote.VoterId, c.totalIndirectVotes)
			c = c.nextByTotalIndirectVotes
		}
	}
	bestRing, ringCount := getBestCandidates(vc.cand, vc.best, vc.verbose)
	tenativeWinners := bestOfRing(bestRing, vc.verbose)
	if ringCount < 2 {
		tenativeWinners = solveWinner(tenativeWinners, bestRing, vc.verbose)
	} else if vc.verbose {
		fmt.Printf("Skipping patron detection because ringCount = %d\n", ringCount)
	}
	winner := tieBreaker(tenativeWinners, vc.verbose)
	if winner == nil {
		return nil
	}
	return winner.vote
}
