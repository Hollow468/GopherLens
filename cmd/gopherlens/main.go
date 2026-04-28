package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/hollow/gopherlens/internal/architect"
	"github.com/hollow/gopherlens/internal/coder"
	"github.com/hollow/gopherlens/internal/logicminer"
	"github.com/hollow/gopherlens/internal/orchestrator"
	"github.com/hollow/gopherlens/internal/testarchitect"
	"github.com/hollow/gopherlens/internal/validator"
)

func main() {
	targetFile := flag.String("file", "", "Path to the Go source file to analyze")
	coverageTarget := flag.Float64("coverage", 80.0, "Target coverage percentage (default: 80.0)")
	flag.Parse()

	if *targetFile == "" {
		fmt.Fprintf(os.Stderr, "Usage: gopherlens -file <path/to/file.go> [-coverage 80.0]\n")
		os.Exit(1)
	}

	ctx := context.Background()

	// Wire the agent pipeline.
	arch := architect.New()
	miner := logicminer.New()
	testArch := testarchitect.New()
	codeGen := coder.New()
	val := validator.New()
	val.TargetCoverage = *coverageTarget

	orch := orchestrator.New(arch, miner, testArch, codeGen, val)

	fmt.Printf("GopherLens: analyzing %s (target coverage: %.1f%%)\n", *targetFile, *coverageTarget)

	state, err := orch.Run(ctx, *targetFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Pipeline error: %v\n", err)
		os.Exit(1)
	}

	// Output results.
	fmt.Printf("\n=== GopherLens Report ===\n")
	fmt.Printf("File: %s\n", *targetFile)
	fmt.Printf("Module: %s\n", state.Architecture.Module)
	fmt.Printf("Dependencies found: %d\n", len(state.Architecture.Dependencies))
	for _, dep := range state.Architecture.Dependencies {
		fmt.Printf("  - %s (%s)\n", dep.Name, dep.Kind)
	}

	fmt.Printf("\nLogic paths discovered: %d\n", state.Logic.BranchCount)
	for _, path := range state.Logic.Paths {
		fmt.Printf("  [%s] %s\n", path.ID, path.Description)
	}

	fmt.Printf("\nTest cases designed: %d\n", len(state.Matrix.Cases))
	fmt.Printf("Estimated coverage: %.1f%%\n", state.Matrix.CoverageEst)

	if state.Test != nil {
		fmt.Printf("\nTest file: %s\n", state.Test.FilePath)
		fmt.Printf("Generated %d bytes of test code\n", len(state.Test.Content))
	}

	if state.Validation != nil {
		fmt.Printf("\nValidation (iterations=%d):\n", state.Validation.Iteration)
		fmt.Printf("  Passed: %v\n", state.Validation.Passed)
		fmt.Printf("  Coverage: %.1f%%\n", state.Validation.Coverage)
		if len(state.Validation.FailingTests) > 0 {
			fmt.Printf("  Failing tests:\n")
			for _, ft := range state.Validation.FailingTests {
				fmt.Printf("    - %s\n", ft)
			}
		}
	}

	fmt.Printf("\nPhase: %s\n", state.Phase)
}
