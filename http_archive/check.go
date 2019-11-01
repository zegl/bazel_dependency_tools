package http_archive

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/blang/semver"
	"go.starlark.net/syntax"

	"github.com/zegl/bazel_dependency_tools/internal"
	"github.com/zegl/bazel_dependency_tools/internal/github"

	realGithub "github.com/google/go-github/v28/github"
)

var gitHubReleaseRegex = regexp.MustCompile(`https://github\.com/([a-zA-Z0-9_-]+)/([a-zA-Z0-9_-]+)/releases/download/v?([a-z0-9\.]+)/(.*)\.tar\.gz`)
var githubArchiveRegex = regexp.MustCompile(`https://github\.com/([a-zA-Z0-9_-]+)/([a-zA-Z0-9_-]+)/archive/v?([a-z0-9\.]+)\.zip`)

func Check(e *syntax.CallExpr, namePrefixFilter string, gitHubClient github.Client) ([]internal.LineReplacement, error) {
	var replacements []internal.LineReplacement

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

	// Don't attempt to upgrade this dependency
	if !strings.HasPrefix(archiveName, namePrefixFilter) {
		return nil, nil
	}

	log.Printf("Checking %s", archiveName)

	for _, url := range archiveUrls {
		log.Println(url.Raw, gitHubReleaseRegex.MatchString(url.Raw), githubArchiveRegex.MatchString(url.Raw))
		if gitHubReleaseRegex.MatchString(url.Raw) || githubArchiveRegex.MatchString(url.Raw) {
			existingVersion, newerVersion, sha256sum, err := FindNewerGitHubRelease(gitHubClient, url.Raw)
			if err != nil {
				log.Println(err)
				continue
			}

			// Create replacements for all urls
			for _, subUrl := range archiveUrls {
				replacements = append(replacements, internal.LineReplacement{
					Filename:     subUrl.TokenPos.Filename(),
					Line:         subUrl.TokenPos.Line,
					Find:         existingVersion,
					Substitution: newerVersion,
				})
			}

			// Create substitution for sha256
			if archiveSha256 != nil {
				replacements = append(replacements, internal.LineReplacement{
					Filename:     archiveSha256.TokenPos.Filename(),
					Line:         archiveSha256.TokenPos.Line,
					Find:         archiveSha256.Value.(string),
					Substitution: sha256sum,
				})
			}

			// Create substitution for strip_prefix
			if archiveStripPrefix != nil {
				replacements = append(replacements, internal.LineReplacement{
					Filename:     archiveStripPrefix.TokenPos.Filename(),
					Line:         archiveStripPrefix.TokenPos.Line,
					Find:         existingVersion,
					Substitution: newerVersion,
				})
			}
		}
	}

	if len(replacements) != 0 {
		return replacements, nil
	}

	return nil, errors.New("no match")
}

func FindNewerGitHubRelease(githubClient github.Client, url string) (oldVersion, newVersion, sha256sum string, err error) {
	var owner, repo, tag, extension string

	if gitHubReleaseRegex.MatchString(url) {
		submatches := gitHubReleaseRegex.FindStringSubmatch(url)
		owner = submatches[1]
		repo = submatches[2]
		tag = submatches[3]
		extension = "tar.gz"
	} else if githubArchiveRegex.MatchString(url) {
		submatches := githubArchiveRegex.FindStringSubmatch(url)
		owner = submatches[1]
		repo = submatches[2]
		tag = submatches[3]
		extension = "zip"
	} else {
		return "", "", "", errors.New("No pattern matches")
	}

	oldVersion = tag

	releases, err := githubClient.ListReleases(owner, repo)
	if err != nil {
		return "", "", "", err
	}

	highestVersion, err := semver.New(strings.TrimLeft(tag, "v"))
	if err != nil {
		return "", "", "", err
	}

	var highestRelease *realGithub.RepositoryRelease

	for _, release := range releases {
		if ver, err := semver.New(strings.TrimLeft(*release.TagName, "v")); err == nil {
			if ver.GT(*highestVersion) {
				highestVersion = ver
				highestRelease = release
			}
		}
	}

	if highestRelease != nil {
		for _, r := range highestRelease.Assets {
			if strings.HasSuffix(r.GetBrowserDownloadURL(), "."+extension) {
				resp, err := http.Get(r.GetBrowserDownloadURL())
				if err != nil {
					panic(err)
				}
				defer resp.Body.Close()

				allData, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					panic(err)
				}

				sha256sum = fmt.Sprintf("%x", sha256.Sum256(allData))
				break
			}
		}
	}

	if oldVersion == highestVersion.String() {
		return "", "", "", errors.New("no newer version found")
	}

	log.Printf("Found: version=%s sha256=%s", highestVersion.String(), sha256sum)
	return oldVersion, highestVersion.String(), sha256sum, nil
}
