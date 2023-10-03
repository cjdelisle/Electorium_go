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

### Debugging a fuzz test failure

If you see error output like the following:

```
            reflect.Value.Call({0x105276380?, 0x14000132090?, 0x140000d85b0?}, {0x140002f4f00, 0x2, 0x2})                                                                                                   
                /opt/homebrew/Cellar/go/1.18.1/libexec/src/reflect/value.go:339 +0x98
            testing.(*F).Fuzz.func1.1(0x1400020e180?)                     
                /opt/homebrew/Cellar/go/1.18.1/libexec/src/testing/fuzz.go:337 +0x1e0
            testing.tRunner(0x140000aeb60, 0x14000078f30)                     
                /opt/homebrew/Cellar/go/1.18.1/libexec/src/testing/testing.go:1439 +0x110                                                                                                                   
            created by testing.(*F).Fuzz.func1                                                        
                /opt/homebrew/Cellar/go/1.18.1/libexec/src/testing/fuzz.go:324 +0x4cc
                                                                                                      
                                                                                                      
    Failing input written to testdata/fuzz/FuzzVsRust/f4f77909101233b0d1bfff0599004e3d9286ec23bf91f6d657acb310e600b295                                                                                      
    To re-run:             
    go test -run=FuzzVsRust/f4f77909101233b0d1bfff0599004e3d9286ec23bf91f6d657acb310e600b295
FAIL                               
exit status 1                             
FAIL    github.com/cjdelisle/Electorium_go/test 0.496s 
```

You can re-run that test in "manual mode" where it will print verbose output of it's entire decision
making process. Simply re-run the test with `-- --manual` at the end.

```bash
go test -run=FuzzVsRust/f4f77909101233b0d1bfff0599004e3d9286ec23bf91f6d657acb310e600b295 -- --manual  
```

## License
MIT OR ISC at your preference