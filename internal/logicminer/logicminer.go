package logicminer

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strconv"

	"github.com/hollow/gopherlens/pkg/types"
)

type Agent struct{}

func New() *Agent { return &Agent{} }

func (a *Agent) Mine(ctx context.Context, arch *types.ArchitectureResult) (*types.LogicResult, error) {
	src, err := os.ReadFile(arch.TargetFile)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, arch.TargetFile, src, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}

	var result *types.LogicResult
	ast.Inspect(file, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			return true
		}
		paths := extractPaths(fn, fset)
		result = &types.LogicResult{
			Function:    fn.Name.Name,
			File:        arch.TargetFile,
			Paths:       paths,
			BranchCount: len(paths),
		}
		return false
	})

	if result == nil {
		return nil, fmt.Errorf("no function found in %s", arch.TargetFile)
	}

	return result, nil
}

func (a *Agent) MineUncovered(ctx context.Context, arch *types.ArchitectureResult, uncovered []string) ([]types.LogicPath, error) {
	return nil, fmt.Errorf("not yet implemented")
}

func extractPaths(fn *ast.FuncDecl, fset *token.FileSet) []types.LogicPath {
	p := &pathExtractor{
		funcName: fn.Name.Name,
		fset:     fset,
	}

	// Happy path always exists.
	p.addPath(
		fmt.Sprintf("Happy path: %s succeeds, returns 200", fn.Name.Name),
		200,
		"valid inputs, all dependencies succeed",
	)

	ast.Walk(p, fn.Body)
	return p.paths
}

type pathExtractor struct {
	funcName string
	fset     *token.FileSet
	paths    []types.LogicPath
	nextID   int
	visited  map[string]bool
}

func (p *pathExtractor) addPath(desc string, status int, trigger string) {
	key := fmt.Sprintf("%d-%s", status, desc)
	if p.visited == nil {
		p.visited = make(map[string]bool)
	}
	if p.visited[key] {
		return
	}
	p.visited[key] = true

	p.paths = append(p.paths, types.LogicPath{
		ID:          fmt.Sprintf("path_%d", p.nextID),
		Description: desc,
		HTTPStatus:  status,
		Triggers:    trigger,
	})
	p.nextID++
}

func (p *pathExtractor) Visit(n ast.Node) ast.Visitor {
	if n == nil {
		return nil
	}

	switch node := n.(type) {
	case *ast.IfStmt:
		p.visitIf(node)
	case *ast.SwitchStmt:
		for _, clause := range node.Body.List {
			if cc, ok := clause.(*ast.CaseClause); ok {
				label := exprToString(cc.List)
				p.addPath(
					fmt.Sprintf("Switch case: %s", label),
					200,
					fmt.Sprintf("switch matches: %s", label),
				)
			}
		}
	}

	return p
}

func (p *pathExtractor) visitIf(ifStmt *ast.IfStmt) {
	cond := exprToString(ifStmt.Cond)
	status, msg := extractErrorResponse(ifStmt.Body)

	if status == 0 {
		status = 500
	}

	p.addPath(msg, status, cond)
}

func extractErrorResponse(body *ast.BlockStmt) (int, string) {
	if body == nil {
		return 0, ""
	}

	for _, stmt := range body.List {
		switch s := stmt.(type) {
		case *ast.ExprStmt:
			if call, ok := s.X.(*ast.CallExpr); ok {
				if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
					if sel.Sel.Name == "Error" && len(call.Args) >= 3 {
						// http.Error(w, message, statusCode) — status is args[2]
						sc := extractStatusCode(call.Args[2])
						return sc, fmt.Sprintf("%s returns %d", pkgFuncName(sel), sc)
					}
				}
			}
		case *ast.ReturnStmt:
			for _, expr := range s.Results {
				if lit, ok := expr.(*ast.BasicLit); ok && lit.Kind == token.INT {
					if v, err := strconv.Atoi(lit.Value); err == nil {
						return v, fmt.Sprintf("returns %d", v)
					}
				}
			}
		}
	}

	return 0, ""
}

func extractStatusCode(expr ast.Expr) int {
	switch e := expr.(type) {
	case *ast.SelectorExpr:
		name := fmt.Sprintf("%s.%s", identName(e.X), e.Sel.Name)
		switch name {
		case "http.StatusBadRequest":
			return 400
		case "http.StatusNotFound":
			return 404
		case "http.StatusInternalServerError":
			return 500
		case "http.StatusOK":
			return 200
		case "http.StatusUnauthorized":
			return 401
		case "http.StatusForbidden":
			return 403
		case "http.StatusConflict":
			return 409
		case "http.StatusUnprocessableEntity":
			return 422
		default:
			return 500
		}
	case *ast.BasicLit:
		if v, err := strconv.Atoi(e.Value); err == nil {
			return v
		}
	}
	return 0
}

func pkgFuncName(sel *ast.SelectorExpr) string {
	return identName(sel.X) + "." + sel.Sel.Name
}

func identName(expr ast.Expr) string {
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name
	}
	return "?"
}

func exprToString(expr any) string {
	switch e := expr.(type) {
	case ast.Expr:
		return formatNode(e)
	case []ast.Expr:
		if len(e) > 0 {
			return formatNode(e[0])
		}
		return "default"
	default:
		return fmt.Sprintf("%v", expr)
	}
}

func formatNode(n ast.Expr) string {
	switch x := n.(type) {
	case *ast.BinaryExpr:
		return fmt.Sprintf("%s %s %s", formatNode(x.X), x.Op, formatNode(x.Y))
	case *ast.Ident:
		return x.Name
	case *ast.BasicLit:
		return x.Value
	case *ast.CallExpr:
		return exprToString(x.Fun)
	case *ast.SelectorExpr:
		return fmt.Sprintf("%s.%s", formatNode(x.X), x.Sel.Name)
	default:
		return fmt.Sprintf("%T", n)
	}
}
