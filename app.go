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

	var archiveName string
	var archiveUrls []*syntax.Literal
	var archiveSha256 *syntax.Literal
	var archiveStripPrefix *syntax.Literal

	for _, arg := range e.Args {
		if binExp, ok := arg.(*syntax.BinaryExpr); ok && binExp.Op == syntax.EQ {
			if xIdent, ok := binExp.X.(*syntax.Ident); ok {
				switch xIdent.Name {
				case "name":
					if rhs, ok := binExp.Y.(*syntax.Literal); ok {
						archiveName = rhs.Value.(string)
					}
				case "url":
					if urlString, ok := binExp.Y.(*syntax.Literal); ok {
						archiveUrls = append(archiveUrls, urlString)
					}
				case "urls":
					if urlsListExpr, ok := binExp.Y.(*syntax.ListExpr); ok {
						for _, urlSingleListExpr := range urlsListExpr.List {
							if urlString, ok := urlSingleListExpr.(*syntax.Literal); ok {
								archiveUrls = append(archiveUrls, urlString)
							}
						}
					}
				case "sha256":
					if rhs, ok := binExp.Y.(*syntax.Literal); ok {
						archiveSha256 = rhs
					}
				case "strip_prefix":
					if rhs, ok := binExp.Y.(*syntax.Literal); ok {
						archiveStripPrefix = rhs
					}
				}
			}
		}
	}

	log.Printf("Checking %s", archiveName)

	for _, url := range archiveUrls {
		if gitHubReleaseRegex.MatchString(url.Raw) {
			existingVersion, newerVersion, sha256sum, err := findNewerGitHubRelease(gitHubClient, url.Raw)
			if err != nil {
				continue
			}

			// Create replacements for all urls
			for _, subUrl := range archiveUrls {
				replacements = append(replacements, lineReplacement{
					filename:     subUrl.TokenPos.Filename(),
					line:         subUrl.TokenPos.Line,
					find:         existingVersion,
					substitution: newerVersion,
				})

			}

			// Create substitution for sha256
			if archiveSha256 != nil {
				replacements = append(replacements, lineReplacement{
					filename:     archiveSha256.TokenPos.Filename(),
					line:         archiveSha256.TokenPos.Line,
					find:         archiveSha256.Value.(string),
					substitution: sha256sum,
				})
			}

			// Create substitution for strip_prefix
			if archiveStripPrefix != nil {
				replacements = append(replacements, lineReplacement{
					filename:     archiveStripPrefix.TokenPos.Filename(),
					line:         archiveStripPrefix.TokenPos.Line,
					find:         existingVersion,
					substitution: newerVersion,
				})
			}
		}
	}

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

	if oldVersion == highestVersion.String() {
		return "", "", "", errors.New("no newer version found")
	}

	log.Printf("Found: version=%s sha256=%s", highestVersion.String(), sha256sum)
	return oldVersion, highestVersion.String(), sha256sum, nil
}
