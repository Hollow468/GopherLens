package validator

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hollow/gopherlens/pkg/types"
)

const defaultCoverageTarget = 80.0
const maxIterations = 5

// Agent runs generated Go tests, measures coverage, and drives
// the feedback loop to improve coverage above the target threshold.
type Agent struct {
	TargetCoverage float64
	MaxIterations  int
}

func New() *Agent {
	return &Agent{
		TargetCoverage: defaultCoverageTarget,
		MaxIterations:  maxIterations,
	}
}

// Validate runs go test with coverage and returns the result.
// If coverage is below target, it identifies uncovered lines for
// the Logic Miner to re-analyze.
func (a *Agent) Validate(ctx context.Context, test *types.GeneratedTest) (*types.ValidationResult, error) {
	if test == nil || test.Content == "" {
		return nil, fmt.Errorf("no test content to validate")
	}

	// Write the generated test file.
	testDir := filepath.Dir(test.FilePath)
	if err := os.MkdirAll(testDir, 0755); err != nil {
		return nil, fmt.Errorf("create test dir: %w", err)
	}
	if err := os.WriteFile(test.FilePath, []byte(test.Content), 0644); err != nil {
		return nil, fmt.Errorf("write test file: %w", err)
	}

	result := &types.ValidationResult{
		Target: a.TargetCoverage,
	}

	for iteration := 1; iteration <= a.MaxIterations; iteration++ {
		result.Iteration = iteration

		coverage, stderr, err := runTest(testDir)
		result.Stderr = stderr
		result.Coverage = coverage

		if err != nil {
			result.Passed = false
			result.FailingTests = extractFailingTests(stderr)
		} else {
			result.Passed = true
		}

		if coverage >= a.TargetCoverage {
			break
		}

		result.UncoveredLines = extractUncoveredLines(stderr)
	}

	return result, nil
}

// runTest executes go test -v -cover in the target directory.
func runTest(dir string) (float64, string, error) {
	var stdout, stderr bytes.Buffer

	cmd := exec.Command("go", "test", "-v", "-cover", "./...")
	cmd.Dir = dir
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := stdout.String() + "\n" + stderr.String()

	coverage := parseCoverage(output)
	return coverage, output, err
}

// parseCoverage extracts the coverage percentage from go test output.
// Looks for "coverage: X.X% of statements" in the output.
func parseCoverage(output string) float64 {
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "coverage:") && strings.Contains(line, "%") {
			// Extract the percentage value.
			start := strings.Index(line, "coverage:") + len("coverage:")
			end := strings.Index(line[start:], "%")
			if end >= 0 {
				val := strings.TrimSpace(line[start : start+end])
				if f, err := strconv.ParseFloat(val, 64); err == nil {
					return f
				}
			}
		}
	}
	return 0
}

func extractFailingTests(stderr string) []string {
	var failures []string
	for _, line := range strings.Split(stderr, "\n") {
		if strings.HasPrefix(line, "--- FAIL:") {
			failures = append(failures, strings.TrimSpace(line))
		}
	}
	return failures
}

func extractUncoveredLines(stderr string) []string {
	var lines []string
	// Parse go test -coverprofile output for lines with 0 coverage.
	for _, line := range strings.Split(stderr, "\n") {
		if strings.Contains(line, ".go:") && strings.Contains(line, "0") {
			lines = append(lines, strings.TrimSpace(line))
		}
	}
	return lines
}
