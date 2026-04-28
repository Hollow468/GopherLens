package architect

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/hollow/gopherlens/pkg/types"
)

type Agent struct{}

func New() *Agent { return &Agent{} }

func (a *Agent) Analyze(ctx context.Context, targetFile string) (*types.ArchitectureResult, error) {
	absPath, err := filepath.Abs(targetFile)
	if err != nil {
		return nil, fmt.Errorf("resolve path: %w", err)
	}

	src, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, absPath, src, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse file: %w", err)
	}

	impMap := buildImportMap(file)

	result := &types.ArchitectureResult{
		TargetFile: absPath,
		Module:     extractModule(absPath),
		Packages:   collectImports(file),
	}

	deps := analyzeDependencies(file, impMap)
	result.Dependencies = deduplicateDeps(deps)
	result.CallGraph = buildCallGraph(file, deps)

	return result, nil
}

func extractModule(filePath string) string {
	dir := filepath.Dir(filePath)
	for {
		modPath := filepath.Join(dir, "go.mod")
		if data, err := os.ReadFile(modPath); err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				if strings.HasPrefix(line, "module ") {
					return strings.TrimSpace(strings.TrimPrefix(line, "module "))
				}
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "unknown"
}

func collectImports(file *ast.File) []string {
	var pkgs []string
	for _, imp := range file.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		pkgs = append(pkgs, path)
	}
	return pkgs
}

// buildImportMap maps package identifier names to their full import paths.
func buildImportMap(file *ast.File) map[string]string {
	m := make(map[string]string)
	for _, imp := range file.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		name := pkgNameFromImport(path)
		if imp.Name != nil {
			name = imp.Name.Name
		}
		m[name] = path
	}
	return m
}

func pkgNameFromImport(impPath string) string {
	idx := strings.LastIndex(impPath, "/")
	if idx >= 0 {
		return impPath[idx+1:]
	}
	return impPath
}

var mockablePathKinds = map[string]types.DependencyKind{
	"database/sql":              types.DepSQL,
	"gorm.io/gorm":              types.DepSQL,
	"github.com/go-redis/redis": types.DepRedis,
	"github.com/redis/go-redis": types.DepRedis,
	"net/http":                  types.DepHTTP,
	"google.golang.org/grpc":    types.DepGRPC,
	"github.com/aws/aws-sdk-go": types.DepHTTP,
	"cloud.google.com/go":       types.DepHTTP,
}

func classifyImport(impPath string) types.DependencyKind {
	for prefix, kind := range mockablePathKinds {
		if strings.HasPrefix(impPath, prefix) {
			return kind
		}
	}
	return types.DepGeneric
}

func analyzeDependencies(file *ast.File, impMap map[string]string) []types.Dependency {
	var deps []types.Dependency
	seen := make(map[string]bool)

	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.SelectorExpr:
			if ident, ok := node.X.(*ast.Ident); ok {
				key := ident.Name + "." + node.Sel.Name
				if seen[key] {
					return true
				}
				seen[key] = true

				impPath, known := impMap[ident.Name]
				kind := types.DepGeneric
				if known {
					kind = classifyImport(impPath)
				}

				deps = append(deps, types.Dependency{
					Name:     key,
					Kind:     kind,
					PkgPath:  impPath,
					TypeName: ident.Name,
				})
			}
		case *ast.Field:
			if star, ok := node.Type.(*ast.StarExpr); ok {
				if sel, ok := star.X.(*ast.SelectorExpr); ok {
					if ident, ok := sel.X.(*ast.Ident); ok {
						key := ident.Name + "." + sel.Sel.Name
						if seen[key] {
							return true
						}
						seen[key] = true

						impPath, known := impMap[ident.Name]
						kind := types.DepGeneric
						if known {
							kind = classifyImport(impPath)
						}

						deps = append(deps, types.Dependency{
							Name:     key,
							Kind:     kind,
							PkgPath:  impPath,
							TypeName: ident.Name,
						})
					}
				}
			}
		}
		return true
	})

	return deps
}

func buildCallGraph(file *ast.File, deps []types.Dependency) []types.CallNode {
	var nodes []types.CallNode

	depSet := make(map[string]types.Dependency, len(deps))
	for _, d := range deps {
		depSet[d.Name] = d
	}

	ast.Inspect(file, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			return true
		}

		node := types.CallNode{
			FuncName: fn.Name.Name,
			File:     fn.Name.Name,
		}

		ast.Inspect(fn.Body, func(inner ast.Node) bool {
			call, ok := inner.(*ast.CallExpr)
			if !ok {
				return true
			}
			switch fun := call.Fun.(type) {
			case *ast.SelectorExpr:
				if ident, ok := fun.X.(*ast.Ident); ok {
					key := ident.Name + "." + fun.Sel.Name
					if d, found := depSet[key]; found {
						node.Dependencies = append(node.Dependencies, d)
					}
					node.Callees = append(node.Callees, fun.Sel.Name)
				}
			case *ast.Ident:
				node.Callees = append(node.Callees, fun.Name)
			}
			return true
		})

		nodes = append(nodes, node)
		return true
	})

	return nodes
}

func deduplicateDeps(deps []types.Dependency) []types.Dependency {
	seen := make(map[string]bool)
	var result []types.Dependency
	for _, d := range deps {
		if !seen[d.Name] {
			seen[d.Name] = true
			result = append(result, d)
		}
	}
	return result
}
