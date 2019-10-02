package main

import (
	"testing"

	"github.com/blang/semver"
	"github.com/stretchr/testify/assert"
)

func TestParseWorkspace(t *testing.T) {
	parseWorkspace("testdata/rules_go_0_19_3_WORKSPACE")
}

func TestFindNewerVersion(t *testing.T) {
	newest, err := findNewerGitHubRelease("https://github.com/bazelbuild/rules_go/releases/download/0.19.3/rules_go-0.19.3.tar.gz")
	assert.Nil(t, err)
	assert.True(t, semver.MustParse(newest).GT(semver.MustParse("0.19.3")))
}
