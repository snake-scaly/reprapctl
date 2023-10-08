package doc_test

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"reprapctl/pkg/doc"
	"testing"
)

func TestWrapString(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()

	measure := func(s string) fyne.Size {
		return fyne.MeasureText(s, 10, fyne.TextStyle{})
	}

	type line struct {
		text   string
		offset int
	}

	tests := []struct {
		name  string
		text  string
		width float32
		wrap  fyne.TextWrap
		want  []line
	}{
		{
			name:  "Latin_WrapOff",
			text:  "Lorem ipsum",
			width: 40,
			wrap:  fyne.TextWrapOff,
			want: []line{
				{text: "Lorem ipsum"},
			},
		},
		{
			name:  "Latin_WrapBreak",
			text:  "Lorem ipsum",
			width: 40,
			wrap:  fyne.TextWrapBreak,
			want: []line{
				{text: "Lorem i"},
				{text: "psum", offset: 7},
			},
		},
		{
			name:  "Latin_WrapWord",
			text:  "Lorem ipsum",
			width: 40,
			wrap:  fyne.TextWrapWord,
			want: []line{
				{text: "Lorem"},
				{text: "ipsum", offset: 6},
			},
		},
		{
			name:  "Russian_WrapOff",
			text:  "многа букф",
			width: 40,
			wrap:  fyne.TextWrapOff,
			want: []line{
				{text: "многа букф"},
			},
		},
		{
			name:  "Russian_WrapBreak",
			text:  "многа букф",
			width: 40,
			wrap:  fyne.TextWrapBreak,
			want: []line{
				{text: "многа б"},
				{text: "укф", offset: 13},
			},
		},
		{
			name:  "Russian_WrapWord",
			text:  "многа букф",
			width: 40,
			wrap:  fyne.TextWrapWord,
			want: []line{
				{text: "многа"},
				{text: "букф", offset: 11},
			},
		},
		{
			name:  "Japanese_WrapOff",
			text:  "ライスヌードル",
			width: 40,
			wrap:  fyne.TextWrapOff,
			want: []line{
				{text: "ライスヌードル"},
			},
		},
		{
			name:  "Japanese_WrapBreak",
			text:  "ライスヌードル",
			width: 40,
			wrap:  fyne.TextWrapBreak,
			want: []line{
				{text: "ライスヌー"},
				{text: "ドル", offset: 15},
			},
		},
		{
			name:  "Japanese_WrapWord",
			text:  "ライスヌードル",
			width: 40,
			wrap:  fyne.TextWrapWord,
			want: []line{
				{text: "ライスヌー"},
				{text: "ドル", offset: 15},
			},
		},
		{
			name:  "Emptpy_WrapOff",
			text:  "",
			width: 40,
			wrap:  fyne.TextWrapOff,
			want:  []line{{text: ""}},
		},
		{
			name:  "Emptpy_WrapBreak",
			text:  "",
			width: 40,
			wrap:  fyne.TextWrapBreak,
			want:  []line{{text: ""}},
		},
		{
			name:  "Emptpy_WrapWord",
			text:  "",
			width: 40,
			wrap:  fyne.TextWrapWord,
			want:  []line{{text: ""}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var lines []line
			doc.WrapString(tt.text, tt.width, tt.wrap, measure, func(s string, i int, z fyne.Size) {
				lines = append(lines, line{text: s, offset: i})
			})
			assert.Equal(t, tt.want, lines)
		})
	}
}

func TestWrapString_WordWrapCornerCases(t *testing.T) {
	measure := func(s string) fyne.Size {
		return fyne.Size{Width: float32(len(s))}
	}

	type line struct {
		text   string
		offset int
		width  float32
	}

	tests := []struct {
		name string
		text string
		want []line
	}{
		{
			name: "WordEndsOnBoundary",
			text: "01234 678",
			want: []line{
				{text: "01234", width: 5},
				{text: "678", offset: 6, width: 3},
			},
		},
		{
			name: "WhiteSpaceCrossesBoundary",
			text: "0123  678",
			want: []line{
				{text: "0123", width: 4},
				{text: "678", offset: 6, width: 3},
			},
		},
		{
			name: "BreakAfterPunctuation",
			text: "012.45678",
			want: []line{
				{text: "012.", width: 4},
				{text: "45678", offset: 4, width: 5},
			},
		},
		{
			name: "LineEndsOnBoundary",
			text: "01 34",
			want: []line{
				{text: "01 34", width: 5},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var lines []line
			doc.WrapString(tt.text, 5, fyne.TextWrapWord, measure, func(s string, i int, z fyne.Size) {
				lines = append(lines, line{text: s, offset: i, width: z.Width})
			})
			assert.Equal(t, tt.want, lines)
		})
	}
}

func TestWrapString_ForceWrapNewLine(t *testing.T) {
	measure := func(s string) fyne.Size {
		return fyne.Size{Width: float32(len(s))}
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

	type line struct {
		text   string
		offset int
		width  float32
	}

	want := []line{
		{text: "ab", width: 2},
		{text: "cd", offset: 3, width: 2},
		{text: "ef", offset: 6, width: 2},
		{text: "gh", offset: 10, width: 2},
		{text: "", offset: 13, width: 0},
		{text: "", offset: 15, width: 0},
		{text: "", offset: 16, width: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var lines []line
			doc.WrapString(
				"ab\ncd\ref\r\ngh\r\r\n\n\r", 5, fyne.TextWrapWord, measure,
				func(s string, i int, z fyne.Size,
				) {
					lines = append(lines, line{text: s, offset: i, width: z.Width})
				})
			assert.Equal(t, want, lines)
		})
	}
}
