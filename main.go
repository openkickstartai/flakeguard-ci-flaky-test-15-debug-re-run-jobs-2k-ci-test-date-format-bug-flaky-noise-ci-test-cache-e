package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

func main() {
	pattern := flag.String("results", "*.xml", "Glob pattern for JUnit XML result files")
	costPerMin := flag.Float64("cost", 0.008, "CI cost per minute in USD")
	threshold := flag.Float64("threshold", 0.5, "Minimum flaky probability to report")
	jsonOut := flag.Bool("json", false, "Output as JSON")
	flag.Parse()

	files, _ := filepath.Glob(*pattern)
	if len(files) == 0 {
		fmt.Fprintf(os.Stderr, "error: no files matching %q\n", *pattern)
		os.Exit(1)
	}
	var runs [][]TestCase
	for _, f := range files {
		cases, err := ParseJUnitXML(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warn: skip %s: %v\n", f, err)
			continue
		}
		runs = append(runs, cases)
	}
	if len(runs) < 2 {
		fmt.Fprintln(os.Stderr, "error: need >= 2 result files for statistical detection")
		os.Exit(1)
	}

	records := Aggregate(runs)
	flaky := Detect(records, *costPerMin, *threshold)
	sort.Slice(flaky, func(i, j int) bool { return flaky[i].FlakyProb > flaky[j].FlakyProb })

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(map[string]interface{}{
			"flaky_tests": flaky, "total_runs": len(runs),
			"total_tests": len(records), "flaky_count": len(flaky),
		})
		return
	}

	fmt.Printf("FlakeGuard — %d runs, %d tests, %d flaky\n\n", len(runs), len(records), len(flaky))
	if len(flaky) == 0 {
		fmt.Println("  No flaky tests detected.")
		return
	}
	totalCost := 0.0
	for _, f := range flaky {
		totalCost += f.CICostUSD
		fmt.Printf("  %s::%s\n", f.Class, f.Name)
		fmt.Printf("   pass=%.0f%% flaky=%.0f%% runs=%d cause=%s cost=$%.2f\n",
			f.PassRate*100, f.FlakyProb*100, f.Runs, f.RootCause, f.CICostUSD)
		fmt.Printf("   -> %s\n\n", f.Suggestion)
	}
	fmt.Printf("  Total CI waste: $%.2f\n", totalCost)
	os.Exit(2)
}
