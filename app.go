package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	realGithub "github.com/google/go-github/v28/github"
	"go.starlark.net/syntax"
	"golang.org/x/oauth2"

	"github.com/zegl/bazel_dependency_tools/http_archive"
	"github.com/zegl/bazel_dependency_tools/internal"
	"github.com/zegl/bazel_dependency_tools/internal/github"
	"github.com/zegl/bazel_dependency_tools/maven_jar"
	"github.com/zegl/bazel_dependency_tools/parse"
)

func main() {
	flagPrefixFilter := flag.String("prefix", "", "Only attempt to upgrade dependencies with this prefix, if prefix is empty (default) all dependencies will be upgraded")
	flagWorkspace := flag.String("workspace", "WORKSPACE", "Path to the WORKSPACE file")
	flagFindLicenses := flag.Bool("find-licenses", false, "Runin find licenses mode")
	flag.Parse()

	if *flagFindLicenses {
		findLicenses(*flagWorkspace, *flagPrefixFilter)
		return
	}

	versionUpgrades(*flagWorkspace, *flagPrefixFilter)
}

func versionUpgrades(workspace, prefixFilter string) {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GITHUB_TOKEN")},
	)
	tc := oauth2.NewClient(ctx, ts)
	gitHubClient := github.NewGithubClient(realGithub.NewClient(tc))

	var lineReplacements []internal.LineReplacement

	callFuncs := map[string]parse.FuncHook{
		"maven_jar": func(s *syntax.CallExpr, namePrefixFilter string, workspacePath string) error {
			if r, err := maven_jar.Check(s, namePrefixFilter, maven_jar.NewestAvailable); err == nil {
				lineReplacements = append(lineReplacements, r...)
			}
			return nil
		},
		"http_archive": func(s *syntax.CallExpr, namePrefixFilter string, workspacePath string) error {
			if archiveReplacements, err := http_archive.Check(s, namePrefixFilter, gitHubClient); err == nil {
				lineReplacements = append(lineReplacements, archiveReplacements...)
			}
			return nil
		},
	}

	parse.ParseWorkspace(workspace, prefixFilter, callFuncs)

	rawContent, err := ioutil.ReadFile(workspace)
	if err != nil {
		panic(err)
	}

	rows := strings.Split(string(rawContent), "\n")

	// Perform all replacements
	for _, r := range lineReplacements {
		rows[r.Line-1] = strings.Replace(rows[r.Line-1], r.Find, r.Substitution, -1)
	}

	// Write the new file
	err = ioutil.WriteFile(workspace, []byte(strings.Join(rows, "\n")), 0777)
	if err != nil {
		panic(err)
	}
}

func findLicenses(workspace, prefixFilter string) {
	callFuncs := map[string]parse.FuncHook{
		"maven_jar": func(s *syntax.CallExpr, namePrefixFilter string, workspacePath string) error {
			name, license, err := maven_jar.License(s, namePrefixFilter)
			if err == maven_jar.ErrSkipped {
				return nil
			}
			if err != nil {
				log.Println(name, err)
				return nil
			}
			fmt.Printf("%s,%s\n", name, license)
			return nil
		},
		"maven_install": func(s *syntax.CallExpr, namePrefixFilter string, workspacePath string) error {
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
			return nil
		},
	}
	parse.ParseWorkspace(workspace, prefixFilter, callFuncs)
}
