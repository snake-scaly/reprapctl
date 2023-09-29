package logview

import (
	"fyne.io/fyne/v2"
	"unicode"
	"unicode/utf8"
)

type DisplayLine struct {
	Text         string
	SourceIndex  int
	SourceOffset int
}

func WrapText(entries []string, width float32, wrap fyne.TextWrap, measure func(string) float32) []DisplayLine {
	display := make([]DisplayLine, 0, len(entries))

	for i, e := range entries {
		for _, runeOffsets := range buildRuneOffsets(e) {
			if wrap == fyne.TextWrapOff || len(runeOffsets) == 1 {
				display = append(display, DisplayLine{
					Text:         e[runeOffsets[0]:runeOffsets[len(runeOffsets)-1]],
					SourceIndex:  i,
					SourceOffset: runeOffsets[0],
				})
				continue
			}

			metric := func(i int) float32 {
				return measure(e[runeOffsets[0]:runeOffsets[i]])
			}

			for len(runeOffsets) > 1 {
				end, _ := BinarySearch(len(runeOffsets)-1, width, metric)
				next := end

				if wrap == fyne.TextWrapWord {
					end, next = trimPartialWord(end, len(runeOffsets)-1, func(i int) rune {
						r, _ := utf8.DecodeRune([]byte(e)[runeOffsets[i]:])
						return r
					})
				}

				display = append(display, DisplayLine{
					Text:         e[runeOffsets[0]:runeOffsets[end]],
					SourceIndex:  i,
					SourceOffset: runeOffsets[0],
				})

				runeOffsets = runeOffsets[next:]
			}
		}
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
