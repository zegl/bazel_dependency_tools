package main

import (
	"context"
	"io/ioutil"
	"os"
	"strings"

	realGithub "github.com/google/go-github/v28/github"
	"golang.org/x/oauth2"

	"github.com/zegl/bazel_dependency_tools/internal/github"
	"github.com/zegl/bazel_dependency_tools/maven_jar"
	"github.com/zegl/bazel_dependency_tools/parse"
)

func main() {
	ctx := context.Background()

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GITHUB_TOKEN")},
	)
	tc := oauth2.NewClient(ctx, ts)
	gitHubClient := github.NewGithubClient(realGithub.NewClient(tc))

	workspaceFile := os.Args[1]

	lineReplacements := parse.ParseWorkspace(workspaceFile, gitHubClient, maven_jar.NewestAvailable)

	rawContent, err := ioutil.ReadFile(workspaceFile)
	if err != nil {
		panic(err)
	}

	rows := strings.Split(string(rawContent), "\n")

	// Perform all replacements
	for _, r := range lineReplacements {
		rows[r.Line-1] = strings.Replace(rows[r.Line-1], r.Find, r.Substitution, -1)
	}

	// Write the new file
	err = ioutil.WriteFile(workspaceFile, []byte(strings.Join(rows, "\n")), 0777)
	if err != nil {
		panic(err)
	}
}
