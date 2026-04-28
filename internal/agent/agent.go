package agent

import (
	"context"

	"github.com/hollow/gopherlens/pkg/types"
)

// Architect analyzes Go source files and identifies dependencies needing mocks.
type Architect interface {
	Analyze(ctx context.Context, targetFile string) (*types.ArchitectureResult, error)
}

// LogicMiner extracts business logic paths from Go functions.
type LogicMiner interface {
	Mine(ctx context.Context, arch *types.ArchitectureResult) (*types.LogicResult, error)
	MineUncovered(ctx context.Context, arch *types.ArchitectureResult, uncovered []string) ([]types.LogicPath, error)
}

// TestArchitect designs test matrices from mined logic paths.
type TestArchitect interface {
	Design(ctx context.Context, logic *types.LogicResult) (*types.TestMatrix, error)
}

// Coder generates Go test files from test specifications.
type Coder interface {
	Generate(ctx context.Context, matrix *types.TestMatrix, arch *types.ArchitectureResult) (*types.GeneratedTest, error)
	Fix(ctx context.Context, test *types.GeneratedTest, failures *types.ValidationResult) (*types.GeneratedTest, error)
}

// Validator runs tests and drives the coverage feedback loop.
type Validator interface {
	Validate(ctx context.Context, test *types.GeneratedTest) (*types.ValidationResult, error)
}
