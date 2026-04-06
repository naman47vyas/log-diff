package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/k0kubun/pp"
	"github.com/naman47vyas/log-diff/internal/normalizer"
	"github.com/naman47vyas/log-diff/internal/parser"
)

func main() {
	preFile := flag.String("pre", "", "path to the pre-release file")
	postFile := flag.String("post", "", "path to the post-release file")

	format := flag.String("format", "bracket", "log format: bracket")
	// simTh := flag.Float64("sim-threshold", 0.4, "Drain similarity threshold (0.0-1.0)")

	// freqTh := flag.Float64("freq-threshold", 2.0, "Frequency change ratio to flag as changed")

	if *preFile == "" || *postFile == "" {
		fmt.Fprintln(os.Stderr, "usage: logdiff --pre <file> --post <file>")
		flag.PrintDefaults()
		os.Exit(1)
	}

	p, err := newParser(*format)

	pp.Println(p)

	if err != nil {
		log.Fatalf("invalid format: %v", err)
	}

	norm := normalizer.New()
	pp.Println(norm)

}

func newParser(format string) (parser.Parser, error) {
	switch format {
	case "bracket":
		return parser.NewBracketParser(), nil
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}
