package main

import (
	"testing"

	"github.com/blang/semver"
	"github.com/stretchr/testify/assert"
)

func TestParseWorkspace(t *testing.T) {
	replacements := parseWorkspace("testdata/rules_go_0_19_3_WORKSPACE")
	assert.Equal(t, []lineReplacement{
		{filename: "testdata/rules_go_0_19_3_WORKSPACE", line: 6, find: "0.19.3", substitution: "0.19.4"},
		{filename: "testdata/rules_go_0_19_3_WORKSPACE", line: 7, find: "0.19.3", substitution: "0.19.4"},
		{filename: "testdata/rules_go_0_19_3_WORKSPACE", line: 20, find: "0.18.3", substitution: "0.19.4"},
	}, replacements)
}

func TestFindNewerVersion(t *testing.T) {
	existing, newest, err := findNewerGitHubRelease("https://github.com/bazelbuild/rules_go/releases/download/0.19.3/rules_go-0.19.3.tar.gz")
	assert.Nil(t, err)
	assert.True(t, semver.MustParse(newest).GT(semver.MustParse("0.19.3")))
	assert.Equal(t, "0.19.3", existing)
}
