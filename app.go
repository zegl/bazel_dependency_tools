package main

import (
	"context"
	"errors"
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

type lineReplacement struct {
	filename           string
	line               int32
	find, substitution string
}

func parseWorkspace(path string) []lineReplacement {
	file, _, err := starlark.SourceProgram(path, nil, func(name string) bool {
		log.Printf("isPredeclared: %s", name)
		return false
	})
	if err != nil {
		panic(err)
	}

	var replacements []lineReplacement

	for _, stmt := range file.Stmts {
		switch s := stmt.(type) {
		case *syntax.ExprStmt:
			switch e := s.X.(type) {
			case *syntax.CallExpr:
				if ident, ok := e.Fn.(*syntax.Ident); ok && ident.Name == "http_archive" {
					if archiveReplacements, err := checkHttpArchive(e); err == nil {
						replacements = append(replacements, archiveReplacements...)
					}
				}
			}
		}
	}

	return replacements
}

func checkHttpArchive(e *syntax.CallExpr) ([]lineReplacement, error) {
	for _, arg := range e.Args {
		if binExp, ok := arg.(*syntax.BinaryExpr); ok {
			if binExp.Op == syntax.EQ {
				if xIdent, ok := binExp.X.(*syntax.Ident); ok && xIdent.Name == "urls" {
					if urlsListExpr, ok := binExp.Y.(*syntax.ListExpr); ok {
						for _, urlSingleListExpr := range urlsListExpr.List {
							if urlString, ok := urlSingleListExpr.(*syntax.Literal); ok {
								if gitHubReleaseRegex.MatchString(urlString.Raw) {
									existingVersion, newerVersion, err := findNewerGitHubRelease(urlString.Raw)
									if err != nil {
										// This URL did not match. Keep going with the next url in the list
										continue
									}

									// Create line replacements for all literals in this "urls"
									var replacements []lineReplacement
									for _, u := range urlsListExpr.List {
										if s, ok := u.(*syntax.Literal); ok {
											replacements = append(replacements, lineReplacement{
												filename:     s.TokenPos.Filename(),
												line:         s.TokenPos.Line,
												find:         existingVersion,
												substitution: newerVersion,
											})
										}
									}
									return replacements, nil
								}
							}
						}
					}
				}
			}
		}
	}

	return nil, errors.New("no match")
}

func findNewerGitHubRelease(url string) (oldVersion, newVersion string, err error) {
	submatches := gitHubReleaseRegex.FindStringSubmatch(url)
	owner := submatches[1]
	repo := submatches[2]
	tag := submatches[3]

	oldVersion = tag

	client := github.NewClient(nil)

	releases, _, err := client.Repositories.ListReleases(context.Background(), owner, repo, nil)
	if err != nil {
		panic(err)
	}

	highestVersion, err := semver.New(tag)
	if err != nil {
		log.Println("Could not find version:", err)
		return oldVersion, "", err
	}

	for _, release := range releases {
		if ver, err := semver.New(*release.TagName); err == nil {
			if ver.GT(*highestVersion) {
				highestVersion = ver
			}
		}
	}

	return oldVersion, highestVersion.String(), nil
}
