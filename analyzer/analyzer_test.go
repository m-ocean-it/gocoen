package analyzer

import (
	"path/filepath"
	"slices"
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	t.Parallel()

	analyzer := NewAnalyzer()

	analysistest.Run(
		t,
		filepath.Join(analysistest.TestData(), "base"),
		analyzer,
	)
}

func Test_constructorNameFromDocLine(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name    string
		docLine string
		want    []string
	}

	testCases := []testCase{
		{
			name:    "1",
			docLine: "// use #constructor[ConstructB].",
			want:    []string{"ConstructB"},
		},
		{
			name:    "2",
			docLine: "// #constructor[BazInit]",
			want:    []string{"BazInit"},
		},
		{
			name:    "3",
			docLine: "// #constructor[]",
			want:    nil,
		},
		{
			name:    "4",
			docLine: "//    #constructor[12] ",
			want:    []string{"12"},
		},
		{
			name:    "5",
			docLine: "//    #constructor[ 12  , abc] ",
			want:    []string{"12", "abc"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := constructorNamesFromDocLine(tc.docLine)

			if !slices.Equal(got, tc.want) {
				t.Fatalf("got: %v, want: %v", got, tc.want)
			}
		})
	}
}
