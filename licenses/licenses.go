package licenses

import (
	"fmt"
	"log"

	"github.com/zegl/bazel_dependency_tools/maven_jar"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

func ParseWorkspace(path, namePrefixFilter string) {
	file, _, err := starlark.SourceProgram(path, nil, func(name string) bool {
		log.Printf("isPredeclared: %s", name)
		return true
	})
	if err != nil {
		// panic(err)
	}

	vars := make(map[string]syntax.Expr)
	for _, stmt := range file.Stmts {
		eval(stmt, vars, namePrefixFilter, path)
	}
}

func eval(stmt syntax.Stmt, vars map[string]syntax.Expr, namePrefixFilter, workspacePath string) syntax.Stmt {
	switch s := stmt.(type) {
	case *syntax.AssignStmt:
		if s.Op == syntax.EQ {
			key := evalExpr(s.LHS, vars, namePrefixFilter, workspacePath)
			val := evalExpr(s.RHS, vars, namePrefixFilter, workspacePath)
			ks := key.(*syntax.Literal).Value.(string)
			vars[ks] = val
		}
	case *syntax.ExprStmt:
		evalExpr(s.X, vars, namePrefixFilter, workspacePath)
	}
	return nil
}

func evalExpr(stmt syntax.Expr, vars map[string]syntax.Expr, namePrefixFilter, workspacePath string) syntax.Expr {
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
			val := fmt.Sprintf(
				evalExpr(s.X, vars, namePrefixFilter, workspacePath).(*syntax.Literal).Value.(string),
				evalExpr(s.Y, vars, namePrefixFilter, workspacePath).(*syntax.Literal).Value.(string),
			)
			return &syntax.Literal{
				Value: val,
			}
		default:
			panic("unknown binary expr op")
		}
	case *syntax.CallExpr:
		if ident, ok := s.Fn.(*syntax.Ident); ok {

			// Evaluate / simplify args
			for argI, arg := range s.Args {
				if binExp, ok := arg.(*syntax.BinaryExpr); ok && binExp.Op == syntax.EQ {
					binExp.Y = evalExpr(binExp.Y, vars, namePrefixFilter, workspacePath)
					s.Args[argI] = binExp
				}
			}

			switch ident.Name {
			// case "http_archive":
			// 	if archiveReplacements, err := http_archive.Check(e, namePrefixFilter, gitHubClient); err == nil {
			// 		replacements = append(replacements, archiveReplacements...)
			// 	}
			case "maven_jar":
				name, license, err := maven_jar.License(s, namePrefixFilter)
				if err == maven_jar.ErrSkipped {
					return nil
				}
				if err != nil {
					log.Println(name, err)
					return nil
				}
				fmt.Printf("%s,%s\n", name, license)
			case "maven_install":
				licenses, err := maven_jar.LicenseMavenInstall(s, namePrefixFilter, workspacePath)
				if err == maven_jar.ErrSkipped {
					return nil
				}
				if err != nil {
					log.Println(err)
					return nil
				}
				for _, l := range licenses {
					fmt.Printf("%s,%s\n", l.Art, l.License)
				}
			}
		}
	default:
		log.Fatalf("unknown expr: %T %+v", s, s)
	}
	return nil
}
