# Electorium Go
This is an implementation of the Electorium delegated vote algorithm in Golang.

A detailed explanation of the algorithm can be found at https://github.com/cjdelisle/Electorium

## Usage

```go
import electorium "github.com/cjdelisle/Electorium_go"

func main() {
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
	if win == nil {
        fmt.Printf("No winner\n")
    } else {
        fmt.Printf("Winner is %s\n", win.VoterId)
	}
}
```

## Fuzz testing
You can fuzz test this algorithm *against* the Rust implementation. To do this, you must first
get this code and the Rust implementation in folders next to eachother.

```bash
mkdir electorium_fuzz
cd electorium_fuzz
git clone https://github.com/cjdelisle/Electorium
git clone https://github.com/cjdelisle/Electorium_go
```

Then you need to compile `Electorium/fuzzable`

```bash
cd Electorium/fuzzable
cargo build
cd ../../
```

Then you can fuzz test from the `Electorium_go/test` folder.

```bash
cd Electorium_go/test
go test -fuzz FuzzVsRust
```

## License
MIT OR ISC at your preference