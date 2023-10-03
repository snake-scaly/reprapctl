package doc

type Fragment struct {
	Text   string
	Anchor Anchor
}

type Anchor struct {
	LineIndex  int
	LineOffset int
}

func (a Anchor) Compare(b Anchor) int {
	if a.LineIndex != b.LineIndex {
		return a.LineIndex - b.LineIndex
	}
	return a.LineOffset - b.LineOffset
}
