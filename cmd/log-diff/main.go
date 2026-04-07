package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/naman47vyas/log-diff/internal/differ"
	"github.com/naman47vyas/log-diff/internal/drain"
	"github.com/naman47vyas/log-diff/internal/normalizer"
	"github.com/naman47vyas/log-diff/internal/parser"
)

func main() {
	preFile := flag.String("pre", "", "path to pre-release log file")
	postFile := flag.String("post", "", "path to post-release log file")
	format := flag.String("format", "bracket", "log format: bracket")
	simTh := flag.Float64("sim-threshold", 0.4, "Drain similarity threshold (0.0-1.0)")
	freqTh := flag.Float64("freq-threshold", 2.0, "frequency change ratio to flag as changed")
	flag.Parse()

	if *preFile == "" || *postFile == "" {
		fmt.Fprintln(os.Stderr, "usage: logdiff --pre <file> --post <file>")
		flag.PrintDefaults()
		os.Exit(1)
	}

	p, err := newParser(*format)
	if err != nil {
		log.Fatalf("invalid format: %v", err)
	}

	norm := normalizer.NewFast()
	drainCfg := drain.DefaultConfig()
	drainCfg.SimThreshold = *simTh

	preDrain, preErrs := processFile(*preFile, p, norm, drainCfg)
	postDrain, postErrs := processFile(*postFile, p, norm, drainCfg)

	if preErrs > 0 {
		fmt.Fprintf(os.Stderr, "warning: %d unparseable lines in pre-release log\n", preErrs)
	}
	if postErrs > 0 {
		fmt.Fprintf(os.Stderr, "warning: %d unparseable lines in post-release log\n", postErrs)
	}

	diffCfg := differ.DefaultConfig()
	diffCfg.FreqRatioThreshold = *freqTh

	report := differ.Diff(preDrain.Clusters(), postDrain.Clusters(), diffCfg)
	printReport(report)
}

func newParser(format string) (parser.Parser, error) {
	switch format {
	case "bracket":
		return parser.NewBracketParser(), nil
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

func processFile(path string, p parser.Parser, norm normalizer.Norm, cfg drain.Config) (*drain.Drain, int) {
	f, err := os.Open(path)
	if err != nil {
		log.Fatalf("cannot open %s: %v", path, err)
	}
	defer f.Close()

	return processReader(f, p, norm, cfg)
}

func processReader(r io.Reader, p parser.Parser, norm normalizer.Norm, cfg drain.Config) (*drain.Drain, int) {
	numWorkers := runtime.NumCPU()

	lines := make(chan string, 4096)
	results := make(chan string, 4096)

	var errCount atomic.Int64

	// Stage 1: scanner goroutine reads lines into channel
	go func() {
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			line := scanner.Text()
			if line != "" {
				lines <- line
			}
		}
		if err := scanner.Err(); err != nil {
			log.Fatalf("error reading input: %v", err)
		}
		close(lines)
	}()

	// Stage 2: N workers parse + normalize in parallel
	var workerWg sync.WaitGroup
	workerWg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go func() {
			defer workerWg.Done()
			for line := range lines {
				content, err := p.Parse(line)
				if err != nil {
					errCount.Add(1)
					continue
				}
				// normalized := norm.Normalize(entry.Severity + " " + entry.Message)
				normalized := norm.Normalize(content)
				results <- normalized
			}
		}()
	}

	// Close results channel when all workers are done
	go func() {
		workerWg.Wait()
		close(results)
	}()

	// Stage 3: single goroutine feeds Drain
	d := drain.New(cfg)
	for normalized := range results {
		d.Train(normalized)
	}

	return d, int(errCount.Load())
}

func printReport(r *differ.Report) {
	fmt.Printf("=== Log Diff Report ===\n")
	fmt.Printf("Pre-release:  %d total lines, %d templates\n", r.PreTotal, r.PreTemplates)
	fmt.Printf("Post-release: %d total lines, %d templates\n\n", r.PostTotal, r.PostTemplates)

	if len(r.New) > 0 {
		fmt.Printf("--- NEW templates (%d) ---\n", len(r.New))
		for _, e := range r.New {
			fmt.Printf("  [count: %d, %.1f%%] %s\n", e.PostCount, e.PostFreqPct, e.Template)
			printSamples(e.Samples)
		}
		fmt.Println()
	}

	if len(r.Gone) > 0 {
		fmt.Printf("--- GONE templates (%d) ---\n", len(r.Gone))
		for _, e := range r.Gone {
			fmt.Printf("  [was: %d, %.1f%%] %s\n", e.PreCount, e.PreFreqPct, e.Template)
			printSamples(e.Samples)
		}
		fmt.Println()
	}

	if len(r.Changed) > 0 {
		fmt.Printf("--- CHANGED templates (%d) ---\n", len(r.Changed))
		for _, e := range r.Changed {
			fmt.Printf("  [pre: %d (%.1f%%) → post: %d (%.1f%%)] %s\n",
				e.PreCount, e.PreFreqPct, e.PostCount, e.PostFreqPct, e.Template)
			printSamples(e.Samples)
		}
		fmt.Println()
	}

	if len(r.New) == 0 && len(r.Gone) == 0 && len(r.Changed) == 0 {
		fmt.Println("No differences found.")
	}
}

func printSamples(samples []string) {
	for _, s := range samples {
		fmt.Printf("    → %s\n", s)
	}
}
