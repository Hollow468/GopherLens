package types

// DependencyKind classifies external dependencies that need mocking.
type DependencyKind string

const (
	DepSQL       DependencyKind = "sql"
	DepRedis     DependencyKind = "redis"
	DepHTTP      DependencyKind = "http"
	DepGRPC      DependencyKind = "grpc"
	DepFileIO    DependencyKind = "fileio"
	DepGeneric   DependencyKind = "generic"
)

// Dependency describes an external dependency found by the Architect.
type Dependency struct {
	Name     string         `json:"name"`
	Kind     DependencyKind `json:"kind"`
	PkgPath  string         `json:"pkg_path"`
	TypeName string         `json:"type_name"`
}

// CallNode represents one node in the call topology graph.
type CallNode struct {
	FuncName     string       `json:"func_name"`
	File         string       `json:"file"`
	Line         int          `json:"line"`
	Dependencies []Dependency `json:"dependencies"`
	Callees      []string     `json:"callees"`
}

// ArchitectureResult is the output of the Architect Agent.
type ArchitectureResult struct {
	Module       string     `json:"module"`
	TargetFile   string     `json:"target_file"`
	Packages     []string   `json:"packages"`
	CallGraph    []CallNode `json:"call_graph"`
	Dependencies []Dependency `json:"dependencies"`
}

// LogicPath describes a single execution path through a function.
type LogicPath struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Condition   string `json:"condition"`
	HTTPStatus  int    `json:"http_status"`
	Triggers    string `json:"triggers"`
}

// LogicResult is the output of the Logic Miner Agent.
type LogicResult struct {
	Function    string      `json:"function"`
	File        string      `json:"file"`
	Paths       []LogicPath `json:"paths"`
	BranchCount int         `json:"branch_count"`
}

// TestCase specifies a single table-driven test entry.
type TestCase struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	PathID      string            `json:"path_id"`
	Category    string            `json:"category"` // happy, error, boundary
	Input       map[string]any    `json:"input"`
	Mocks       []MockSpec        `json:"mocks"`
	ExpectError bool              `json:"expect_error"`
	ExpectStatus int              `json:"expect_status"`
}

// MockSpec describes a mock that must be set up for a test case.
type MockSpec struct {
	Dependency string `json:"dependency"`
	Method     string `json:"method"`
	Args       []any  `json:"args"`
	Return     []any  `json:"return"`
	Error      string `json:"error,omitempty"`
}

// TestMatrix is the output of the Test Architect.
type TestMatrix struct {
	TargetFunction string     `json:"target_function"`
	Paths          []LogicPath `json:"paths"`
	Cases          []TestCase `json:"cases"`
	CoverageEst    float64    `json:"coverage_estimate"`
}

// GeneratedTest holds a generated _test.go file.
type GeneratedTest struct {
	FilePath    string `json:"file_path"`
	Content     string `json:"content"`
	PackageName string `json:"package_name"`
}

// ValidationResult is the output of the Validator Agent.
type ValidationResult struct {
	Passed      bool     `json:"passed"`
	Coverage    float64  `json:"coverage"`
	Target      float64  `json:"target"`
	FailingTests []string `json:"failing_tests"`
	UncoveredLines []string `json:"uncovered_lines"`
	Stderr      string   `json:"stderr"`
	Iteration   int      `json:"iteration"`
}

// PipelineState tracks the entire pipeline's progress.
type PipelineState struct {
	Phase        string           `json:"phase"`
	Architecture *ArchitectureResult `json:"architecture,omitempty"`
	Logic        *LogicResult     `json:"logic,omitempty"`
	Matrix       *TestMatrix      `json:"matrix,omitempty"`
	Test         *GeneratedTest   `json:"test,omitempty"`
	Validation   *ValidationResult `json:"validation,omitempty"`
	Iterations   int              `json:"iterations"`
	MaxIterations int             `json:"max_iterations"`
}
