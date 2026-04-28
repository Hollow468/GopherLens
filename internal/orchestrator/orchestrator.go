package orchestrator

import (
	"context"
	"fmt"

	"github.com/hollow/gopherlens/internal/agent"
	"github.com/hollow/gopherlens/pkg/types"
)

// Orchestrator sequences the agent pipeline and manages the
// closed-loop feedback from Validator back to Logic Miner / Coder.
type Orchestrator struct {
	Architect     agent.Architect
	LogicMiner    agent.LogicMiner
	TestArchitect agent.TestArchitect
	Coder         agent.Coder
	Validator     agent.Validator

	MaxIterations int
}

// New creates an Orchestrator with the provided agent implementations.
func New(arch agent.Architect, miner agent.LogicMiner, testArch agent.TestArchitect, coder agent.Coder, validator agent.Validator) *Orchestrator {
	return &Orchestrator{
		Architect:     arch,
		LogicMiner:    miner,
		TestArchitect: testArch,
		Coder:         coder,
		Validator:     validator,
		MaxIterations: 5,
	}
}

// Run executes the full GopherLens pipeline on a target Go file.
//
// Pipeline phases:
//  1. Architect: static analysis → call topology + dependencies
//  2. Logic Miner: extract business logic paths
//  3. Test Architect: design test matrix
//  4. Coder: generate _test.go file
//  5. Validator: run tests, loop if coverage < 80%
func (o *Orchestrator) Run(ctx context.Context, targetFile string) (*types.PipelineState, error) {
	state := &types.PipelineState{
		Phase:         "init",
		MaxIterations: o.MaxIterations,
	}

	// Phase 1: Architecture Analysis
	state.Phase = "architecture"
	arch, err := o.Architect.Analyze(ctx, targetFile)
	if err != nil {
		return state, fmt.Errorf("architect phase: %w", err)
	}
	state.Architecture = arch

	// Phase 2: Logic Mining
	state.Phase = "logic_mining"
	logic, err := o.LogicMiner.Mine(ctx, arch)
	if err != nil {
		return state, fmt.Errorf("logic miner phase: %w", err)
	}
	state.Logic = logic

	// Phase 3: Test Design
	state.Phase = "test_design"
	matrix, err := o.TestArchitect.Design(ctx, logic)
	if err != nil {
		return state, fmt.Errorf("test architect phase: %w", err)
	}
	state.Matrix = matrix

	// Phase 4: Code Generation
	state.Phase = "code_generation"
	test, err := o.Coder.Generate(ctx, matrix, arch)
	if err != nil {
		return state, fmt.Errorf("coder phase: %w", err)
	}
	state.Test = test

	// Phase 5: Validation Loop (closed loop)
	state.Phase = "validation"
	for state.Iterations < o.MaxIterations {
		state.Iterations++

		result, err := o.Validator.Validate(ctx, test)
		if err != nil {
			return state, fmt.Errorf("validator phase: %w", err)
		}
		state.Validation = result

		if result.Passed && result.Coverage >= 80.0 {
			break
		}

		// Feedback loop: feed uncovered lines back to Logic Miner.
		if len(result.UncoveredLines) > 0 {
			newPaths, err := o.LogicMiner.MineUncovered(ctx, state.Architecture, result.UncoveredLines)
			if err != nil {
				continue
			}
			// Append new paths and re-design tests.
			state.Logic.Paths = append(state.Logic.Paths, newPaths...)
			state.Logic.BranchCount = len(state.Logic.Paths)

			matrix, err = o.TestArchitect.Design(ctx, state.Logic)
			if err != nil {
				continue
			}
			state.Matrix = matrix

			test, err = o.Coder.Generate(ctx, matrix, arch)
			if err != nil {
				continue
			}
			state.Test = test
		} else {
			// No uncovered lines to act on; exit loop.
			break
		}
	}

	state.Phase = "complete"
	return state, nil
}
