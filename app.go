package main

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/blang/semver"
	"github.com/google/go-github/v28/github"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
	"golang.org/x/oauth2"
)

var gitHubReleaseRegex = regexp.MustCompile(`https://github\.com/([a-zA-Z0-9_-]+)/([a-zA-Z0-9_-]+)/releases/download/v?([a-z0-9\.]+)/(.*)\.tar\.gz`)

func main() {
	ctx := context.Background()

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GITHUB_TOKEN")},
	)
	tc := oauth2.NewClient(ctx, ts)
	gitHubClient := github.NewClient(tc)

	workspaceFile := os.Args[1]

	lineReplacements := parseWorkspace(workspaceFile, gitHubClient)

	rawContent, err := ioutil.ReadFile(workspaceFile)
	if err != nil {
		panic(err)
	}

	rows := strings.Split(string(rawContent), "\n")

	// Perform all replacements
	for _, r := range lineReplacements {
		// fmt.Printf("%s -> %s\n", r.find, r.substitution)
		rows[r.line-1] = strings.Replace(rows[r.line-1], r.find, r.substitution, -1)
	}

	// Write the new file
	err = ioutil.WriteFile(workspaceFile, []byte(strings.Join(rows, "\n")), 0777)
	if err != nil {
		panic(err)
	}
}

type lineReplacement struct {
	filename           string
	line               int32
	find, substitution string
}

func parseWorkspace(path string, gitHubClient *github.Client) []lineReplacement {
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
					if archiveReplacements, err := checkHttpArchive(e, gitHubClient); err == nil {
						replacements = append(replacements, archiveReplacements...)
					}
				}
			}
		}
	}

	return replacements
}

func checkHttpArchive(e *syntax.CallExpr, gitHubClient *github.Client) ([]lineReplacement, error) {
	var replacements []lineReplacement
	var newSha256sum string

	// Find name of this dependency, used in logging
	for _, arg := range e.Args {
		if binExp, ok := arg.(*syntax.BinaryExpr); ok && binExp.Op == syntax.EQ {
			if xIdent, ok := binExp.X.(*syntax.Ident); ok && xIdent.Name == "name" {
				if rhs, ok := binExp.Y.(*syntax.Literal); ok {
					log.Printf("Checking %s", rhs.Value.(string))
					break
				}
			}
		}
	}

argsLoop:
	for _, arg := range e.Args {
		if binExp, ok := arg.(*syntax.BinaryExpr); ok {
			if binExp.Op == syntax.EQ {

				// Single URL
				if xIdent, ok := binExp.X.(*syntax.Ident); ok && xIdent.Name == "url" {
					if urlString, ok := binExp.Y.(*syntax.Literal); ok {
						if gitHubReleaseRegex.MatchString(urlString.Raw) {
							existingVersion, newerVersion, sha256sum, err := findNewerGitHubRelease(gitHubClient, urlString.Raw)
							if err != nil {
								break argsLoop
							}

							replacements = append(replacements, lineReplacement{
								filename:     urlString.TokenPos.Filename(),
								line:         urlString.TokenPos.Line,
								find:         existingVersion,
								substitution: newerVersion,
							})

							newSha256sum = sha256sum
							break argsLoop
						}
					}
				}

				// Multiple URLs
				if xIdent, ok := binExp.X.(*syntax.Ident); ok && xIdent.Name == "urls" {
					if urlsListExpr, ok := binExp.Y.(*syntax.ListExpr); ok {
						for _, urlSingleListExpr := range urlsListExpr.List {
							if urlString, ok := urlSingleListExpr.(*syntax.Literal); ok {

								if gitHubReleaseRegex.MatchString(urlString.Raw) {
									existingVersion, newerVersion, sha256sum, err := findNewerGitHubRelease(gitHubClient, urlString.Raw)
									if err != nil {
										// This URL did not match. Keep going with the next url in the list
										continue
									}

									// Create line replacements for all literals in this "urls"
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

									newSha256sum = sha256sum
									break argsLoop
								}
							}
						}
					}
				}
			}
		}
	}

	if newSha256sum != "" {
		// Find line num of where the sha265sum is defined
		var foundOldSha256Row bool
		for _, arg := range e.Args {
			if binExp, ok := arg.(*syntax.BinaryExpr); ok && binExp.Op == syntax.EQ {
				if xIdent, ok := binExp.X.(*syntax.Ident); ok && xIdent.Name == "sha256" {
					if rhs, ok := binExp.Y.(*syntax.Literal); ok {
						replacements = append(replacements, lineReplacement{
							filename:     rhs.TokenPos.Filename(),
							line:         rhs.TokenPos.Line,
							find:         rhs.Value.(string),
							substitution: newSha256sum,
						})
						foundOldSha256Row = true
						break
					}
				}
			}
		}

		if !foundOldSha256Row {
			return nil, errors.New("row of existing sha256 not found")
		}
	}

	// TODO: Check if there is a strip_prefix configuration that needs to be updated

	if len(replacements) != 0 {
		return replacements, nil
	}

	return nil, errors.New("no match")
}

func findNewerGitHubRelease(githubClient *github.Client, url string) (oldVersion, newVersion, sha256sum string, err error) {
	submatches := gitHubReleaseRegex.FindStringSubmatch(url)
	owner := submatches[1]
	repo := submatches[2]
	tag := submatches[3]

	oldVersion = tag

	releases, _, err := githubClient.Repositories.ListReleases(context.Background(), owner, repo, nil)
	if err != nil {
		panic(err)
	}

	highestVersion, err := semver.New(strings.TrimLeft(tag, "v"))
	if err != nil {
		return "", "", "", err
	}

	var highestRelease *github.RepositoryRelease

	for _, release := range releases {
		if ver, err := semver.New(strings.TrimLeft(*release.TagName, "v")); err == nil {
			if ver.GT(*highestVersion) {
				highestVersion = ver
				highestRelease = release
			}
		}
	}

	// var sha256sum string
	if highestRelease != nil {
		resp, err := http.Get(*highestRelease.TarballURL)
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()

		allData, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}

		sha256sum = fmt.Sprintf("%x", sha256.Sum256(allData))
	}

	if oldVersion != highestVersion.String() {
		log.Printf("Found: version=%s sha256=%s", highestVersion.String(), sha256sum)
	}

	return oldVersion, highestVersion.String(), sha256sum, nil
}
