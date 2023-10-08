package doc

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

func (a Anchor) Between(b, c Anchor) bool {
	if b.Compare(c) > 0 {
		b, c = c, b
	}
	return a.Compare(b) >= 0 && a.Compare(c) < 0
}
