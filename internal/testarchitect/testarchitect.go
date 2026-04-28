package testarchitect

import (
	"context"
	"fmt"

	"github.com/hollow/gopherlens/pkg/types"
)

// Agent designs test matrices from mined business logic paths,
// covering happy path, error paths, and boundary values.
type Agent struct{}

func New() *Agent { return &Agent{} }

// Design produces a comprehensive test matrix from the Logic Miner's output.
func (a *Agent) Design(ctx context.Context, logic *types.LogicResult) (*types.TestMatrix, error) {
	if logic == nil || len(logic.Paths) == 0 {
		return nil, fmt.Errorf("no logic paths to design tests for")
	}

	matrix := &types.TestMatrix{
		TargetFunction: logic.Function,
		Paths:          logic.Paths,
	}

	for _, path := range logic.Paths {
		category := classifyPath(path)
		tc := types.TestCase{
			Name:        fmt.Sprintf("Test_%s_%s", logic.Function, sanitizeName(path.ID)),
			Description: path.Description,
			PathID:      path.ID,
			Category:    category,
			Input:       buildDefaultInput(path),
			Mocks:       buildMockSpecs(path),
			ExpectError: category == "error",
			ExpectStatus: path.HTTPStatus,
		}
		matrix.Cases = append(matrix.Cases, tc)
	}

	// Add boundary cases
	matrix.Cases = append(matrix.Cases, boundaryCases(logic.Function)...)

	matrix.CoverageEst = estimateCoverage(matrix.Cases, logic.BranchCount)

	return matrix, nil
}

func classifyPath(path types.LogicPath) string {
	switch path.HTTPStatus {
	case 200, 201, 204:
		return "happy"
	case 400, 404, 422:
		return "boundary"
	case 500, 502, 503:
		return "error"
	default:
		return "error"
	}
}

func buildDefaultInput(path types.LogicPath) map[string]any {
	// Produce sensible zero-value inputs for table-driven tests.
	return map[string]any{
		"description": path.Description,
	}
}

func buildMockSpecs(path types.LogicPath) []types.MockSpec {
	if path.HTTPStatus >= 400 {
		return []types.MockSpec{
			{
				Dependency: "database",
				Method:     "Query",
				Error:      "connection timeout",
			},
		}
	}
	return []types.MockSpec{
		{
			Dependency: "database",
			Method:     "Query",
			Return:     []any{[]string{"row1", "row2"}, nil},
		},
	}
}

func boundaryCases(funcName string) []types.TestCase {
	return []types.TestCase{
		{
			Name:        fmt.Sprintf("Test_%s_EmptyInput", funcName),
			Description: "Boundary: empty or nil input",
			Category:    "boundary",
			Input:       map[string]any{"input": nil},
			ExpectError: true,
			ExpectStatus: 400,
		},
		{
			Name:        fmt.Sprintf("Test_%s_MaxValues", funcName),
			Description: "Boundary: maximum allowed values",
			Category:    "boundary",
			Input:       map[string]any{"input": "maximum_value"},
			ExpectError: false,
			ExpectStatus: 200,
		},
	}
}

func estimateCoverage(cases []types.TestCase, branchCount int) float64 {
	if branchCount == 0 {
		branchCount = 1
	}
	covered := float64(len(cases)) / float64(branchCount+2) // +2 for boundary cases
	if covered > 1.0 {
		covered = 1.0
	}
	return covered * 100
}

func sanitizeName(s string) string {
	result := make([]byte, 0, len(s))
	for _, c := range []byte(s) {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' {
			result = append(result, c)
		}
	}
	return string(result)
}
