package parse

import "testing"

import "go.starlark.net/syntax"
import "github.com/stretchr/testify/assert"

import "github.com/Masterminds/semver"

func TestUpgradeRulesCommentBefore(t *testing.T) {
	for _, tc := range []struct {
		Text               string
		ExpectedOK         string
		ExpectedNotAllowed string
	}{
		{"# bazel_dependency_tools: ~8", "8.2.3", "9.0.1"},
		{"# bazel_dependency_tools: ~8.3", "8.3.314", "9.0.1"},
		{"# bazel_dependency_tools: 8.3.155", "8.3.155", "8.3.4"},
	} {
		pinning := UpgradeRules(&syntax.Comments{
			Before: []syntax.Comment{
				{Text: tc.Text},
			},
		})
		assert.True(t, pinning.Check(semver.MustParse(tc.ExpectedOK)), tc.Text)
		assert.False(t, pinning.Check(semver.MustParse(tc.ExpectedNotAllowed)), tc.Text)
	}
}
func TestUpgradeRulesCommentSuffix(t *testing.T) {
	for _, tc := range []struct {
		Text               string
		ExpectedOK         string
		ExpectedNotAllowed string
	}{
		{"# bazel_dependency_tools: ~8", "8.2.3", "9.0.1"},
		{"# bazel_dependency_tools: ~8.3", "8.3.314", "9.0.1"},
		{"# bazel_dependency_tools: 8.3.155", "8.3.155", "8.3.4"},
	} {
		pinning := UpgradeRules(&syntax.Comments{
			Suffix: []syntax.Comment{
				{Text: tc.Text},
			},
		})
		assert.True(t, pinning.Check(semver.MustParse(tc.ExpectedOK)), tc.Text)
		assert.False(t, pinning.Check(semver.MustParse(tc.ExpectedNotAllowed)), tc.Text)
	}
}
