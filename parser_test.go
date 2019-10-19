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
	client.AddRelease("bazelbuild", "rules_go", "0.19.4", "https://github.com/bazelbuild/rules_go/releases/download/0.19.4/rules_go-0.19.4.tar.gz") // https://github.com/bazelbuild/rules_go/releases/download/0.19.3/rules_go-0.19.3.tar.gz
	// client.AddRelease("bazelbuild", "rules_go", "0.19.4", "xxx") // https://github.com/bazelbuild/rules_go/releases/download/0.19.3/rules_go-0.19.3.tar.gz

	replacements := parse.ParseWorkspace("testdata/rules_go_0_19_3_WORKSPACE", "", client, nil)
	assert.Equal(t, []internal.LineReplacement{
		{Filename: "testdata/rules_go_0_19_3_WORKSPACE", Line: 6, Find: "0.19.3", Substitution: "0.19.4"},
		{Filename: "testdata/rules_go_0_19_3_WORKSPACE", Line: 7, Find: "0.19.3", Substitution: "0.19.4"},
		{Filename: "testdata/rules_go_0_19_3_WORKSPACE", Line: 9, Find: "313f2c7a23fecc33023563f082f381a32b9b7254f727a7dd2d6380ccc6dfe09b", Substitution: "ae8c36ff6e565f674c7a3692d6a9ea1096e4c1ade497272c2108a810fb39acd2"},
		{Filename: "testdata/rules_go_0_19_3_WORKSPACE", Line: 20, Find: "0.18.3", Substitution: "0.19.4"},
		{Filename: "testdata/rules_go_0_19_3_WORKSPACE", Line: 21, Find: "86ae934bd4c43b99893fc64be9d9fc684b81461581df7ea8fc291c816f5ee8c5", Substitution: "ae8c36ff6e565f674c7a3692d6a9ea1096e4c1ade497272c2108a810fb39acd2"},
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
