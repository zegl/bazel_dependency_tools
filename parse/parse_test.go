package parse

import "testing"

import "go.starlark.net/syntax"
import "github.com/stretchr/testify/assert"

func TestUpgradeRulesCommentBefore(t *testing.T) {
	for _, tc := range []struct {
		Text          string
		ExpectedMajor int
		ExpectedMinor int
		ExpectedPatch int
	}{
		{"# bazel_dependency_tools: major=8", 8, -1, -1},
		{"# bazel_dependency_tools: major=8 minor=3", 8, 3, -1},
		{"# bazel_dependency_tools: major=8 minor=3 patch=155", 8, 3, 155},
	} {
		ma, mi, pa := UpgradeRules(&syntax.Comments{
			Before: []syntax.Comment{
				{Text: tc.Text},
			},
		})
		assert.Equal(t, tc.ExpectedMajor, ma)
		assert.Equal(t, tc.ExpectedMinor, mi)
		assert.Equal(t, tc.ExpectedPatch, pa)
	}
}
func TestUpgradeRulesCommentSuffix(t *testing.T) {
	for _, tc := range []struct {
		Text          string
		ExpectedMajor int
		ExpectedMinor int
		ExpectedPatch int
	}{
		{"# bazel_dependency_tools: major=8", 8, -1, -1},
		{"# bazel_dependency_tools: major=8 minor=3", 8, 3, -1},
		{"# bazel_dependency_tools: major=8 minor=3 patch=155", 8, 3, 155},
	} {
		ma, mi, pa := UpgradeRules(&syntax.Comments{
			Suffix: []syntax.Comment{
				{Text: tc.Text},
			},
		})
		assert.Equal(t, tc.ExpectedMajor, ma)
		assert.Equal(t, tc.ExpectedMinor, mi)
		assert.Equal(t, tc.ExpectedPatch, pa)
	}
}
