package logview_test

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"reprapctl/internal/pkg/logview"
	"testing"
)

func TestWrapText(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()

	lines := []string{
		"Lorem ipsum",
		"многа букф",
		"ライスヌードル",
		"",
	}

	measure := func(s string) float32 {
		return fyne.MeasureText(s, 10, fyne.TextStyle{}).Width
	}

	tests := []struct {
		name  string
		width float32
		wrap  fyne.TextWrap
		want  []logview.DisplayLine
	}{
		{
			name:  "NoWrap",
			width: 40,
			wrap:  fyne.TextWrapOff,
			want: []logview.DisplayLine{
				{Text: "Lorem ipsum", SourceIndex: 0},
				{Text: "многа букф", SourceIndex: 1},
				{Text: "ライスヌードル", SourceIndex: 2},
				{Text: "", SourceIndex: 3},
			},
		},
		{
			name:  "BreakAfter40Points",
			width: 40,
			wrap:  fyne.TextWrapBreak,
			want: []logview.DisplayLine{
				{Text: "Lorem i", SourceIndex: 0},
				{Text: "psum", SourceIndex: 0, SourceOffset: 7},
				{Text: "многа б", SourceIndex: 1},
				{Text: "укф", SourceIndex: 1, SourceOffset: 13},
				{Text: "ライスヌー", SourceIndex: 2},
				{Text: "ドル", SourceIndex: 2, SourceOffset: 15},
				{Text: "", SourceIndex: 3},
			},
		},
		{
			name:  "WordWrapAfter40Points",
			width: 40,
			wrap:  fyne.TextWrapWord,
			want: []logview.DisplayLine{
				{Text: "Lorem", SourceIndex: 0},
				{Text: "ipsum", SourceIndex: 0, SourceOffset: 6},
				{Text: "многа", SourceIndex: 1},
				{Text: "букф", SourceIndex: 1, SourceOffset: 11},
				{Text: "ライスヌー", SourceIndex: 2},
				{Text: "ドル", SourceIndex: 2, SourceOffset: 15},
				{Text: "", SourceIndex: 3},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := logview.WrapText(lines, tt.width, tt.wrap, measure)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestWrapText_WordWrapCornerCases(t *testing.T) {
	measure := func(s string) float32 {
		return float32(len(s))
	}

	tests := []struct {
		name  string
		entry string
		want  []logview.DisplayLine
	}{
		{
			name:  "WordEndsOnBoundary",
			entry: "01234 678",
			want: []logview.DisplayLine{
				{Text: "01234"},
				{Text: "678", SourceOffset: 6},
			},
		},
		{
			name:  "WhiteSpaceCrossesBoundary",
			entry: "0123  678",
			want: []logview.DisplayLine{
				{Text: "0123"},
				{Text: "678", SourceOffset: 6},
			},
		},
		{
			name:  "BreakAfterPunctuation",
			entry: "012.45678",
			want: []logview.DisplayLine{
				{Text: "012."},
				{Text: "45678", SourceOffset: 4},
			},
		},
		{
			name:  "LineEndsOnBoundary",
			entry: "01 34",
			want: []logview.DisplayLine{
				{Text: "01 34"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := logview.WrapText([]string{tt.entry}, 5, fyne.TextWrapWord, measure)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestWrapText_ForceWrapNewLine(t *testing.T) {
	measure := func(s string) float32 {
		return float32(len(s))
	}

	tests := []struct {
		name string
		wrap fyne.TextWrap
	}{
		{
			name: "TextWrapOff",
			wrap: fyne.TextWrapOff,
		},
		{
			name: "TextWrapBreak",
			wrap: fyne.TextWrapBreak,
		},
		{
			name: "TextWrapWord",
			wrap: fyne.TextWrapWord,
		},
	}

	want := []logview.DisplayLine{
		{Text: "ab"},
		{Text: "cd", SourceOffset: 3},
		{Text: "ef", SourceOffset: 6},
		{Text: "gh", SourceOffset: 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := logview.WrapText([]string{"ab\ncd\ref\r\ngh"}, 100, tt.wrap, measure)
			assert.Equal(t, want, result)
		})
	}
}
