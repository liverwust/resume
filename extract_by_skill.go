package main

import (
	"cmp"
	"fmt"
	"io"
	"log"
	"os"
	"slices"
	"strings"

	mapset "github.com/deckarep/golang-set/v2"
	"gopkg.in/yaml.v3"
)

type Jobs struct {
	Jobs []Job `yaml:"jobs"`
}

type Job struct {
	Company  string `yaml:"company"`
	Location string `yaml:"location"`
	Title    string `yaml:"title"`
	Dates    string `yaml:"dates"`
	Lines    []Line `yaml:"lines"`
}

type Line struct {
	Line         string        `yaml:"line"`
	Skills       []string      `yaml:"skills"`
	Alternatives []Alternative `yaml:"alternatives"`
}

type Alternative struct {
	Line   string   `yaml:"line"`
	Skills []string `yaml:"skills"`
}

// Remove a trailing newline if present.
func trimNewline(s string) string {
	return strings.TrimRight(s, "\n")
}

// Print the "headers" associated with a particular Job.
func (job Job) writeOut() {
	fmt.Println(trimNewline(job.Company))
	fmt.Println(trimNewline(job.Title))
	fmt.Println(trimNewline(job.Location))
	fmt.Println(trimNewline(job.Dates))
	fmt.Println()
}

// Return a set of skills which exist in either the line set (i.e., the one
// shared between all alternative wordings) or the alternative set (i.e., the
// one specific to a particular wording).
func findCombinedSkills(lineLevel []string, alternativeLevel []string) []string {
	lineLevelSet := mapset.NewSet[string]()
	alternativeLevelSet := mapset.NewSet[string]()
	for _, item := range lineLevel {
		lineLevelSet.Add(item)
	}
	for _, item := range alternativeLevel {
		alternativeLevelSet.Add(item)
	}
	intersection := lineLevelSet.Union(alternativeLevelSet)
	return intersection.ToSlice()
}

// Return a set of skills which exist in both the specified set (i.e., the one
// given by the user as args) and the actual set (i.e., the ones listed in the
// job).
func findOverlappingSkills(specified []string, actual []string) []string {
	specifiedSet := mapset.NewSet[string]()
	actualSet := mapset.NewSet[string]()
	for _, item := range specified {
		specifiedSet.Add(item)
	}
	for _, item := range actual {
		actualSet.Add(item)
	}
	intersection := specifiedSet.Intersect(actualSet)
	return intersection.ToSlice()
}

// Return a closure which will sort a slice of Alternatives based on how much
// they (and their line-level skills) overlap with a specified set of skills.
type overlapSortFunc func(Alternative, Alternative) int

func sortUsingOverlaps(specified []string, lineLevel []string) overlapSortFunc {
	return func(a1 Alternative, a2 Alternative) int {
		totalSkills1 := findCombinedSkills(lineLevel, a1.Skills)
		totalSkills2 := findCombinedSkills(lineLevel, a2.Skills)
		overlapSkills1 := findOverlappingSkills(specified, totalSkills1)
		overlapSkills2 := findOverlappingSkills(specified, totalSkills2)
		return cmp.Compare(len(overlapSkills1), len(overlapSkills2))
	}
}

func main() {
	specifiedSkills := make([]string, 0, len(os.Args)-1)
	for _, arg := range os.Args {
		specifiedSkills = append(specifiedSkills, arg)
	}

	f, err := os.Open("jobs.yml")
	if err != nil {
		log.Fatalln(err)
	}

	content, err := io.ReadAll(f)
	if err != nil {
		log.Fatalln(err)
	}

	var jobs Jobs
	err = yaml.Unmarshal(content, &jobs)
	if err != nil {
		log.Fatalln(err)
	}

	for jobIdx, job := range jobs.Jobs {
		didPrintJob := false

		for lineIdx, line := range job.Lines {
			if line.Line != "" && len(line.Alternatives) > 0 {
				log.Fatalf(
					"Cannot give a line: and alternatives: for job %d line %d\n",
					jobIdx,
					lineIdx,
				)
			} else if line.Line != "" {
				overlapSkills := findOverlappingSkills(
					specifiedSkills,
					line.Skills,
				)
				if len(overlapSkills) > 0 {
					if !didPrintJob {
						job.writeOut()
						didPrintJob = true
					}
					fmt.Println(trimNewline(line.Line))
				}
			} else if len(line.Alternatives) > 0 {
				slices.SortStableFunc(
					line.Alternatives,
					sortUsingOverlaps(specifiedSkills, line.Skills),
				)
				bestAlternative := &line.Alternatives[len(line.Alternatives) - 1]
				combinedSkills := findCombinedSkills(
					line.Skills,
					bestAlternative.Skills,
				)
				overlapSkills := findOverlappingSkills(
					specifiedSkills,
					combinedSkills,
				)
				if len(overlapSkills) > 0 {
					if !didPrintJob {
						job.writeOut()
						didPrintJob = true
					}
					fmt.Println(trimNewline(bestAlternative.Line))
				}
			}
		}

		if didPrintJob {
			fmt.Println()
		}
	}
}
