package doc

import (
	"fyne.io/fyne/v2"
	"reprapctl/pkg/alg"
	"unicode"
	"unicode/utf8"
)

// WrapString breaks a string into lines that fit the specified width according
// to the measure function and wrapping style. Three wrapping styles are supported:
//
//   - [fyne.TextWrapOff] ignores the width parameter and produces unbroken lines.
//
//   - [fyne.TextWrapBreak] breaks a line at the first rune that exceeds width
//     without any consideration for context.
//
//   - [fyne.TextWrapWord] breaks a line at the first word that exceeds width.
//     Words are separated by white space and punctuation. If a break happens at
//     white space, the run of white space that separates the lines is dropped
//     both from the current and the next line. At punctuation, the break is made
//     after the punctuation. Initial white space in text is always preserved.
//     If the first word on the line, including the initial white space, does not
//     fit width, the algorithm falls back to [fyne.TextWrapBreak].
//
// Regardless of the wrapping style, explicit EOLs are always honored: every
// NL ('\n'), CR ('\r'), or a sequence of CR followed by NL ("\r\n"), ends a line.
// EOL is considered to be logically part of a line. This means that if text ends
// in EOL, no additional empty line at the end is produced. The EOL itself is not
// included in the resulting line.
//
// Measure function must calculate rendered width of a given string in the same
// units as the width parameter. Implementations will usually call [fyne.MeasureText]
// with the appropriate text style.
//
// LineHandler is called for each line found in text, in order of their appearance.
// Handler receives a slice of text representing the line and a byte offset in text
// where the line starts.
func WrapString(
	text string,
	width float32,
	wrap fyne.TextWrap,
	measure func(string) float32,
	lineHandler func(string, int)) {

	for _, runeOffsets := range buildRuneOffsets(text) {
		if wrap == fyne.TextWrapOff || len(runeOffsets) == 1 {
			lineHandler(text[runeOffsets[0]:runeOffsets[len(runeOffsets)-1]], runeOffsets[0])
			continue
		}

		metric := func(i int) float32 {
			return measure(text[runeOffsets[0]:runeOffsets[i]])
		}

		for len(runeOffsets) > 1 {
			end, _ := alg.BinarySearch(len(runeOffsets)-1, width, metric)
			next := end
			if wrap == fyne.TextWrapWord {
				end, next = trimPartialWord(end, len(runeOffsets)-1, func(i int) rune {
					r, _ := utf8.DecodeRune([]byte(text)[runeOffsets[i]:])
					return r
				})
			}
			lineHandler(text[runeOffsets[0]:runeOffsets[end]], runeOffsets[0])
			runeOffsets = runeOffsets[next:]
		}
	}
}

func WrapDocument(
	lines []string,
	width float32,
	wrap fyne.TextWrap,
	measure func(string) float32) []DocumentFragment {

	display := make([]DocumentFragment, 0, len(lines))

	for i, e := range lines {
		WrapString(e, width, wrap, measure, func(line string, offset int) {
			display = append(display, DocumentFragment{
				Text:   line,
				Anchor: Anchor{LineIndex: i, LineOffset: offset},
			})
		})
	}

	return display
}

// buildRuneOffsets finds offsets of each rune in a string, and returns an array of offsets
// for each segment separated by EOL (cr/lf/crlf).
// The last element in each array is an offset beyond the last rune of the segment.
//
// An EOL is considered part of a line. This means that a string ending in EOL does not
// produce an additional empty segment at the end.
func buildRuneOffsets(s string) [][]int {
	if len(s) == 0 {
		return [][]int{{0}}
	}

	offsets := make([]int, 0, len(s)+1)
	lines := make([][]int, 0)
	lineStart := 0
	cr := false

	for i, r := range s {
		offsets = append(offsets, i)

		if r == '\r' {
			lines = append(lines, offsets[lineStart:])
			lineStart = len(offsets)
			cr = true
		} else {
			if r == '\n' {
				if !cr {
					lines = append(lines, offsets[lineStart:])
				}
				lineStart = len(offsets)
			}
			cr = false
		}
	}

	if lineStart != len(offsets) {
		offsets = append(offsets, len(s))
		lines = append(lines, offsets[lineStart:])
	}

	return lines
}

// trimPartialWord finds the latest word boundary in a rune sequence.
// Fit points past the last rune that fits. Size is the total number of runes in the line.
// It returns two rune indices: where the word ends, and where the next word begins.
func trimPartialWord(fit, size int, getRune func(int) rune) (trim, next int) {
	trim, next = fit, fit
	isPartialWord := true

	if next == size || unicode.IsSpace(getRune(next)) {
		isPartialWord = false
		if next < size {
			next++
		}
		for next < size && unicode.IsSpace(getRune(next)) {
			next++
		}
	}

	for i := fit - 1; i >= 0; i-- {
		r := getRune(i)
		if unicode.IsSpace(r) {
			isPartialWord = false
		} else if !isPartialWord || unicode.IsPunct(r) {
			return
		} else {
			next = i
		}
		trim = i
	}

	// couldn't find a word boundary
	return fit, fit
}
