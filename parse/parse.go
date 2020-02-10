package parse

import (
	"fmt"
	"log"
	"strings"

	"github.com/Masterminds/semver"
	"go.starlark.net/syntax"
)

type FuncHook func(s *syntax.CallExpr, namePrefixFilter string, workspacePath string) error

type MultiPosLiteral struct {
	syntax.Literal
	Positions []syntax.Position
}

func ToMultiPosLiteral(stmt syntax.Expr) *MultiPosLiteral {
	switch s := stmt.(type) {
	case *syntax.Literal:
		return &MultiPosLiteral{Literal: *s, Positions: []syntax.Position{s.TokenPos}}
	case *MultiPosLiteral:
		return s
	default:
		panic("unknown toMultiPosLiteral")
	}
}

func ParseWorkspace(path, namePrefixFilter string, callFuncs map[string]FuncHook) {
	f, err := syntax.Parse(path, nil, syntax.RetainComments)
	if err != nil {
		panic(err)
	}

	vars := make(map[string]syntax.Expr)
	for _, stmt := range f.Stmts {
		eval(stmt, vars, namePrefixFilter, path, callFuncs)
	}
}

func eval(stmt syntax.Stmt, vars map[string]syntax.Expr, namePrefixFilter, workspacePath string, callFuncs map[string]FuncHook) syntax.Stmt {
	switch s := stmt.(type) {
	case *syntax.AssignStmt:
		if s.Op == syntax.EQ {
			key := evalExpr(s.LHS, vars, namePrefixFilter, workspacePath, callFuncs)
			val := evalExpr(s.RHS, vars, namePrefixFilter, workspacePath, callFuncs)
			ks := key.(*syntax.Literal).Value.(string)
			vars[ks] = val
		}
	case *syntax.ExprStmt:
		evalExpr(s.X, vars, namePrefixFilter, workspacePath, callFuncs)
	}
	return nil
}

func pos(stmt syntax.Expr) syntax.Position {
	switch s := stmt.(type) {
	case *syntax.Literal:
		return s.TokenPos
	case *syntax.Ident:
		return s.NamePos
	default:
		panic("unknown pos")
	}
}

// UpgradeRules returns the allowed major, minor, patch versions
// If a version is -1, it's allowed to be upgraded to any version
func UpgradeRules(comments *syntax.Comments) *semver.Constraints {
	if comments == nil {
		c, _ := semver.NewConstraint(">= 0.0.0")
		return c
	}

	for _, co := range [][]syntax.Comment{comments.Before, comments.Suffix} {
		for _, c := range co {
			if strings.HasPrefix(c.Text, "# bazel_dependency_tools:") {
				semverCompare := strings.TrimSpace(c.Text[len("# bazel_dependency_tools:"):])
				if c, err := semver.NewConstraint(semverCompare); err == nil {
					return c
				}
			}
		}
	}

	c, _ := semver.NewConstraint(">= 0.0.0")
	return c
}

func evalExpr(stmt syntax.Expr, vars map[string]syntax.Expr, namePrefixFilter, workspacePath string, callFuncs map[string]FuncHook) syntax.Expr {
	switch s := stmt.(type) {
	case *syntax.Literal:
		return s
	case *syntax.ListExpr:
		return s
	case *syntax.Ident:
		if v, ok := vars[s.Name]; ok {
			return v
		}
		return &syntax.Literal{
			Value: s.Name,
		}
	case *syntax.BinaryExpr:
		switch s.Op {
		case syntax.PERCENT:
			x := evalExpr(s.X, vars, namePrefixFilter, workspacePath, callFuncs)
			y := evalExpr(s.Y, vars, namePrefixFilter, workspacePath, callFuncs)
			val := fmt.Sprintf(
				x.(*syntax.Literal).Value.(string),
				y.(*syntax.Literal).Value.(string),
			)
			r := &MultiPosLiteral{
				Literal:   syntax.Literal{Value: val},
				Positions: []syntax.Position{pos(x), pos(y)},
			}
			log.Printf("%+v", r)
			return r
		default:
			panic("unknown binary expr op")
		}
	case *syntax.CallExpr:
		if ident, ok := s.Fn.(*syntax.Ident); ok {

			// Evaluate / simplify args
			for argI, arg := range s.Args {
				if binExp, ok := arg.(*syntax.BinaryExpr); ok && binExp.Op == syntax.EQ {
					binExp.Y = evalExpr(binExp.Y, vars, namePrefixFilter, workspacePath, callFuncs)
					s.Args[argI] = binExp
				}
			}

			if fn, ok := callFuncs[ident.Name]; ok {
				fn(s, namePrefixFilter, workspacePath)
			}
		}
	default:
		log.Fatalf("unknown expr: %T %+v", s, s)
	}
	return nil
}
