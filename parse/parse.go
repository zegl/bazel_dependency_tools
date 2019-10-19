package parse

import (
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
	"log"

	"github.com/zegl/bazel_dependency_tools/http_archive"
	"github.com/zegl/bazel_dependency_tools/internal"
	"github.com/zegl/bazel_dependency_tools/internal/github"
	"github.com/zegl/bazel_dependency_tools/maven_jar"
)

func ParseWorkspace(path, namePrefixFilter string, gitHubClient github.Client, mavenJarNewestFunc maven_jar.NewestVersionResolver) []internal.LineReplacement {
	file, _, err := starlark.SourceProgram(path, nil, func(name string) bool {
		log.Printf("isPredeclared: %s", name)
		return true
	})
	if err != nil {
		// panic(err)
	}

	var replacements []internal.LineReplacement

	for _, stmt := range file.Stmts {
		switch s := stmt.(type) {
		case *syntax.ExprStmt:
			switch e := s.X.(type) {
			case *syntax.CallExpr:
				if ident, ok := e.Fn.(*syntax.Ident); ok {
					switch ident.Name {
					case "http_archive":
						if archiveReplacements, err := http_archive.Check(e, namePrefixFilter, gitHubClient); err == nil {
							replacements = append(replacements, archiveReplacements...)
						}
					case "maven_jar":
						if r, err := maven_jar.Check(e, namePrefixFilter, mavenJarNewestFunc); err == nil {
							replacements = append(replacements, r...)
						}
					}
				}
			}
		}
	}

	return replacements
}
