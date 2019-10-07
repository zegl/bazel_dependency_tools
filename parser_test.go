package main

import (
	"testing"

	"github.com/blang/semver"
	"github.com/stretchr/testify/assert"

	"github.com/zegl/bazel_dependency_tools/http_archive"
	"github.com/zegl/bazel_dependency_tools/internal"
	"github.com/zegl/bazel_dependency_tools/parse"
)

func TestParseWorkspace(t *testing.T) {
	replacements := parse.ParseWorkspace("testdata/rules_go_0_19_3_WORKSPACE", nil, nil)
	assert.Equal(t, []internal.LineReplacement{
		{Filename: "testdata/rules_go_0_19_3_WORKSPACE", Line: 6, Find: "0.19.3", Substitution: "0.19.4"},
		{Filename: "testdata/rules_go_0_19_3_WORKSPACE", Line: 7, Find: "0.19.3", Substitution: "0.19.4"},
		{Filename: "testdata/rules_go_0_19_3_WORKSPACE", Line: 20, Find: "0.18.3", Substitution: "0.19.4"},
	}, replacements)
}

func TestFindNewerVersion(t *testing.T) {
	existing, newest, shasum, err := http_archive.FindNewerGitHubRelease(nil, "https://github.com/bazelbuild/rules_go/releases/download/0.19.3/rules_go-0.19.3.tar.gz")
	assert.Nil(t, err)
	assert.True(t, semver.MustParse(newest).GT(semver.MustParse("0.19.3")))
	assert.Equal(t, "0.19.3", existing)
	assert.Equal(t, "aaa", shasum)
}

func TestReplace(t *testing.T) {
	replacements := parse.ParseWorkspace("testdata/maven_jar_WORKSPACE", nil, func(c string) (string, error) {
		return "11.22.33", nil
	})

	assert.Equal(t, []internal.LineReplacement{
		{Filename: "testdata/maven_jar_WORKSPACE", Line: 3, Find: "com.google.zxing:core:3.3.3", Substitution: "com.google.zxing:core:11.22.33"},
	}, replacements)
}
