package main

import (
	"testing"

	"github.com/blang/semver"
	"github.com/stretchr/testify/assert"

	"github.com/zegl/bazel_dependency_tools/http_archive"
	"github.com/zegl/bazel_dependency_tools/internal"
	"github.com/zegl/bazel_dependency_tools/internal/github"
	"github.com/zegl/bazel_dependency_tools/parse"
)

func TestParseWorkspace(t *testing.T) {

	client := github.NewFakeClient()
	client.AddRelease("bazelbuild", "rules_go", "0.19.4", "https://github.com/bazelbuild/rules_go/releases/download/0.19.4/rules_go-0.19.4.tar.gz")
	client.AddRelease("bazelbuild", "rules_sass", "1.23.1", "https://github.com/bazelbuild/rules_sass/archive/1.23.1.zip")

	replacements := parse.ParseWorkspace("testdata/rules_go_0_19_3_WORKSPACE", "", client, nil)
	assert.Equal(t, []internal.LineReplacement{
		// rules_go multiple urls (tar.gz from release artifacts)
		{Filename: "testdata/rules_go_0_19_3_WORKSPACE", Line: 6, Find: "0.19.3", Substitution: "0.19.4"},
		{Filename: "testdata/rules_go_0_19_3_WORKSPACE", Line: 7, Find: "0.19.3", Substitution: "0.19.4"},
		{Filename: "testdata/rules_go_0_19_3_WORKSPACE", Line: 9, Find: "313f2c7a23fecc33023563f082f381a32b9b7254f727a7dd2d6380ccc6dfe09b", Substitution: "ae8c36ff6e565f674c7a3692d6a9ea1096e4c1ade497272c2108a810fb39acd2"},

		// rules_go single URL
		{Filename: "testdata/rules_go_0_19_3_WORKSPACE", Line: 20, Find: "0.18.3", Substitution: "0.19.4"},
		{Filename: "testdata/rules_go_0_19_3_WORKSPACE", Line: 21, Find: "86ae934bd4c43b99893fc64be9d9fc684b81461581df7ea8fc291c816f5ee8c5", Substitution: "ae8c36ff6e565f674c7a3692d6a9ea1096e4c1ade497272c2108a810fb39acd2"},

		// rules_sass (zip archive)
		{Filename: "testdata/rules_go_0_19_3_WORKSPACE", Line: 29, Find: "1.15.2", Substitution: "1.23.1"},
		{Filename: "testdata/rules_go_0_19_3_WORKSPACE", Line: 27, Find: "96cedd370d8b87759c8b4a94e6e1c3bef7c17762770215c65864d9fba40f07cf", Substitution: "2ad5580e1ab6dabc6bea40699a7d78f8cae3f98b48d112812f43a0e2beec3eef"},
		{Filename: "testdata/rules_go_0_19_3_WORKSPACE", Line: 28, Find: "1.15.2", Substitution: "1.23.1"},
	}, replacements)
}

func TestFindNewerVersion(t *testing.T) {
	client := github.NewFakeClient()
	client.AddRelease("bazelbuild", "rules_go", "0.19.4", "https://github.com/bazelbuild/rules_go/releases/download/0.19.4/rules_go-0.19.4.tar.gz") // https://github.com/bazelbuild/rules_go/releases/download/0.19.3/rules_go-0.19.3.tar.gz

	existing, newest, shasum, err := http_archive.FindNewerGitHubRelease(client, "https://github.com/bazelbuild/rules_go/releases/download/0.19.3/rules_go-0.19.3.tar.gz")
	assert.Nil(t, err)
	assert.True(t, semver.MustParse(newest).GT(semver.MustParse("0.19.3")))
	assert.Equal(t, "0.19.3", existing)
	assert.Equal(t, "ae8c36ff6e565f674c7a3692d6a9ea1096e4c1ade497272c2108a810fb39acd2", shasum)
}

func TestReplace(t *testing.T) {
	replacements := parse.ParseWorkspace("testdata/maven_jar_WORKSPACE", "", nil, func(c string) (string, string, error) {
		return "11.22.33", "deadbeef", nil
	})

	assert.Equal(t, []internal.LineReplacement{
		{Filename: "testdata/maven_jar_WORKSPACE", Line: 3, Find: "com.google.zxing:core:3.3.3", Substitution: "com.google.zxing:core:11.22.33"},
		{Filename: "testdata/maven_jar_WORKSPACE", Line: 4, Find: "b640badcc97f18867c4dfd249ef8d20ec0204c07", Substitution: "deadbeef"},
	}, replacements)
}
