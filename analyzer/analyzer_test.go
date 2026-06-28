package analyzer

import (
	"path/filepath"
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
		want    string
	}

	testCases := []testCase{
		{
			name:    "1",
			docLine: "// use #constructor[ConstructB].",
			want:    "ConstructB",
		},
		{
			name:    "2",
			docLine: "// #constructor[BazInit]",
			want:    "BazInit",
		},
		{
			name:    "3",
			docLine: "// #constructor[]",
			want:    "",
		},
		{
			name:    "4",
			docLine: "//    #constructor[12] ",
			want:    "12",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := constructorNameFromDocLine(tc.docLine)

			if got != tc.want {
				t.Fatalf("got: %q, want: %q", got, tc.want)
			}
		})
	}
}
