package main

import (
	"context"
	"log"
	"regexp"

	"github.com/blang/semver"
	"github.com/google/go-github/v28/github"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

var gitHubReleaseRegex = regexp.MustCompile(`https://github\.com/([a-zA-Z0-9_-]+)/([a-zA-Z0-9_-]+)/releases/download/([a-z0-9\.]+)/(.*)\.tar\.gz`)

func main() {
	parseWorkspace("testdata/rules_go_0_19_3_WORKSPACE")
}

func parseWorkspace(path string) {
	file, _, err := starlark.SourceProgram(path, nil, func(name string) bool {
		log.Printf("isPredeclared: %s", name)
		return false
	})
	if err != nil {
		panic(err)
	}

	for _, stmt := range file.Stmts {
		switch s := stmt.(type) {
		case *syntax.ExprStmt:
			switch e := s.X.(type) {
			case *syntax.CallExpr:
				if ident, ok := e.Fn.(*syntax.Ident); ok && ident.Name == "http_archive" {
					log.Println("http_archive!")

					for _, arg := range e.Args {
						//log.Printf("arg: %T %+v", arg, arg)

						if binExp, ok := arg.(*syntax.BinaryExpr); ok {
							//log.Printf("binExp: %+v", binExp, binExp)

							if binExp.Op == syntax.EQ {
								//log.Printf("x: %T %+v", binExp.X, binExp.X)

								if xIdent, ok := binExp.X.(*syntax.Ident); ok && xIdent.Name == "urls" {
									// log.Printf("y: %T %+v", binExp.Y, binExp.Y)

									if urlsListExpr, ok := binExp.Y.(*syntax.ListExpr); ok {
										for _, urlSingleListExpr := range urlsListExpr.List {

											// log.Printf("url: %T %+v", urlSingleListExpr, urlSingleListExpr)

											if urlString, ok := urlSingleListExpr.(*syntax.Literal); ok {

												if gitHubReleaseRegex.MatchString(urlString.Raw) {
													findNewerGitHubRelease(urlString.Raw)
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
}

func findNewerGitHubRelease(url string) (string, error) {
	log.Printf("findNewerGitHubRelease: %s", url)
	submatches := gitHubReleaseRegex.FindStringSubmatch(url)
	owner := submatches[1]
	repo := submatches[2]
	tag := submatches[3]
	log.Printf("%+v", tag)

	client := github.NewClient(nil)

	releases, _, err := client.Repositories.ListReleases(context.Background(), owner, repo, nil)
	if err != nil {
		panic(err)
	}

	highestVersion, err := semver.New(tag)
	if err != nil {
		log.Println("Could not find version:", err)
		return "", err
	}

	for _, release := range releases {
		if ver, err := semver.New(*release.TagName); err == nil {
			if ver.GT(*highestVersion) {
				highestVersion = ver
			}
		}
	}

	return highestVersion.String(), nil
}
