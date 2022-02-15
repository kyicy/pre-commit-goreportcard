package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sort"

	"github.com/gojp/goreportcard/check"
)

var (
	dir     = flag.String("d", ".", "Root directory of your Go application")
	verbose = flag.Bool("v", false, "Verbose output")
	th      = flag.Float64("t", 90, "Threshold of failure command")
)

func run(dir string) (check.ChecksResult, error) {
	filenames, skipped, err := check.GoFiles(dir)
	if err != nil {
		return check.ChecksResult{}, fmt.Errorf("could not get filenames: %v", err)
	}
	if len(filenames) == 0 {
		return check.ChecksResult{}, fmt.Errorf("no .go files found")
	}

	err = check.RenameFiles(skipped)
	if err != nil {
		log.Println("Could not remove files:", err)
	}
	defer check.RevertFiles(skipped)

	checks := []check.Check{
		check.GoFmt{Dir: dir, Filenames: filenames},
		check.GoVet{Dir: dir, Filenames: filenames},
		check.GoLint{Dir: dir, Filenames: filenames},
		check.GoCyclo{Dir: dir, Filenames: filenames},
		// License{Dir: dir, Filenames: []string{}},
		check.Misspell{Dir: dir, Filenames: filenames},
		check.IneffAssign{Dir: dir, Filenames: filenames},
		// ErrCheck{Dir: dir, Filenames: filenames}, // disable errcheck for now, too slow and not finalized
	}

	ch := make(chan check.Score)
	for _, c := range checks {
		go func(c check.Check) {
			p, summaries, err := c.Percentage()
			errMsg := ""
			if err != nil {
				log.Printf("ERROR: (%s) %v", c.Name(), err)
				errMsg = err.Error()
			}
			s := check.Score{
				Name:          c.Name(),
				Description:   c.Description(),
				FileSummaries: summaries,
				Weight:        c.Weight(),
				Percentage:    p,
				Error:         errMsg,
			}
			ch <- s
		}(c)
	}

	resp := check.ChecksResult{
		Files: len(filenames),
	}

	var total, totalWeight float64
	var issues = make(map[string]bool)
	for i := 0; i < len(checks); i++ {
		s := <-ch
		resp.Checks = append(resp.Checks, s)
		total += s.Percentage * s.Weight
		totalWeight += s.Weight
		for _, fs := range s.FileSummaries {
			issues[fs.Filename] = true
		}
		if s.Error != "" {
			resp.DidError = true
		}
	}
	total /= totalWeight

	sort.Sort(check.ByWeight(resp.Checks))
	resp.Average = total
	resp.Issues = len(issues)
	resp.Grade = check.GradeFromPercentage(total * 100)

	return resp, nil
}

func main() {
	flag.Parse()

	result, err := run(*dir)
	if err != nil {
		log.Fatalf("Fatal error checking %s: %s", *dir, err.Error())
	}

	fmt.Printf("Grade: %s (%.1f%%)\n", result.Grade, result.Average*100)
	fmt.Printf("Files: %d\n", result.Files)
	fmt.Printf("Issues: %d\n", result.Issues)

	for _, c := range result.Checks {
		fmt.Printf("%s: %d%%\n", c.Name, int64(c.Percentage*100))
		if *verbose && len(c.FileSummaries) > 0 {
			for _, f := range c.FileSummaries {
				fmt.Printf("\t%s\n", f.Filename)
				for _, e := range f.Errors {
					fmt.Printf("\t\tLine %d: %s\n", e.LineNumber, e.ErrorString)
				}
			}
		}
	}

	if result.Average*100 < *th {
		os.Exit(1)
	}
}
