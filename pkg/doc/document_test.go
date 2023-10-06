package doc_test

import (
	"github.com/stretchr/testify/assert"
	"reprapctl/pkg/doc"
	"testing"
)

func TestDocument_Empty(t *testing.T) {
	d := doc.New()

	d.Read(func(lines []string) {
		assert.Empty(t, lines)
	})

	assert.Zero(t, d.Capacity())
	assert.Zero(t, d.Version())

	var ok bool
	var b doc.Anchor
	var s string

	b, ok = d.GetBookmark(doc.BookmarkStart)
	assert.True(t, ok)
	assert.Equal(t, doc.Anchor{}, b)

	b, ok = d.GetBookmark(doc.BookmarkEnd)
	assert.True(t, ok)
	assert.Equal(t, doc.Anchor{}, b)

	s, ok = d.String(doc.BookmarkStart, doc.BookmarkEnd, "separator")
	assert.True(t, ok)
	assert.Empty(t, s)
}

func TestDocument_Add(t *testing.T) {
	tests := []struct {
		name       string
		cap        int
		add        [][]string
		wantLines  []string
		wantString string
	}{
		{
			name:      "1",
			cap:       0,
			add:       [][]string{{"line1"}},
			wantLines: []string{"line1"},
		},
		{
			name:      "2",
			cap:       0,
			add:       [][]string{{"line1", "line2"}},
			wantLines: []string{"line1", "line2"},
		},
		{
			name:      "2Plus3",
			cap:       0,
			add:       [][]string{{"line1", "line2"}, {"line3", "line4", "line5"}},
			wantLines: []string{"line1", "line2", "line3", "line4", "line5"},
		},
		{
			name:      "2Plus3Cap4",
			cap:       4,
			add:       [][]string{{"line1", "line2"}, {"line3", "line4", "line5"}},
			wantLines: []string{"line2", "line3", "line4", "line5"},
		},
		{
			name:      "2Plus3Cap2",
			cap:       2,
			add:       [][]string{{"line1", "line2"}, {"line3", "line4", "line5"}},
			wantLines: []string{"line4", "line5"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := doc.New()
			d.SetCapacity(tt.cap)
			for _, args := range tt.add {
				d.Add(args...)
			}
			d.Read(func(lines []string) {
				assert.Equal(t, tt.wantLines, lines)
			})
		})
	}
}

func TestDocument_SetSmallerCapacity(t *testing.T) {
	d := doc.New()
	d.Add("line1", "line2", "line3", "line4", "line5")
	d.SetCapacity(4)
	d.Read(func(lines []string) {
		assert.Equal(t, []string{"line2", "line3", "line4", "line5"}, lines)
	})
}

func TestDocument_Version(t *testing.T) {
	var versions []uint64
	d := doc.New()

	checkVersion := func() {
		v := d.Version()
		assert.Equal(t, v, d.Version())
		assert.NotContains(t, versions, v)
		versions = append(versions, v)
	}

	checkVersion()
	d.Add("line")
	checkVersion()
	d.Add("line", "line")
	checkVersion()
	d.Add("line1", "line2", "line3")
	checkVersion()
}

func TestDocument_Bookmarks_GetSet(t *testing.T) {
	tests := []struct {
		name string
		mark doc.Anchor
		want doc.Anchor
	}{
		{
			name: "Inside",
			mark: doc.Anchor{LineIndex: 1, LineOffset: 2},
			want: doc.Anchor{LineIndex: 1, LineOffset: 2},
		},
		{
			name: "Above",
			mark: doc.Anchor{LineIndex: -1, LineOffset: 5},
			want: doc.Anchor{},
		},
		{
			name: "BeforeLineStart",
			mark: doc.Anchor{LineIndex: 1, LineOffset: -5},
			want: doc.Anchor{LineIndex: 1, LineOffset: 0},
		},
		{
			name: "AfterLineEnd",
			mark: doc.Anchor{LineIndex: 1, LineOffset: 8},
			want: doc.Anchor{LineIndex: 1, LineOffset: 5},
		},
		{
			name: "AfterEnd",
			mark: doc.Anchor{LineIndex: 3, LineOffset: 1},
			want: doc.Anchor{LineIndex: 2, LineOffset: 5},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := doc.New()
			d.Add("line1", "line2", "line3")
			d.SetBookmark("bookmark", tt.mark)
			a, ok := d.GetBookmark("bookmark")
			assert.True(t, ok)
			assert.Equal(t, tt.want, a)
		})
	}
}

func TestDocument_Bookmarks_AdjustOnMutation(t *testing.T) {
	d := doc.New()
	d.SetCapacity(4)
	d.Add("line1", "line2", "line3", "line4")
	d.SetBookmark("b1", doc.Anchor{LineIndex: 2, LineOffset: 4})
	d.SetBookmark("b2", doc.Anchor{LineIndex: 3, LineOffset: 2})

	var a doc.Anchor
	var ok bool

	d.Add("line5")
	a, ok = d.GetBookmark("b1")
	assert.True(t, ok)
	assert.Equal(t, doc.Anchor{LineIndex: 1, LineOffset: 4}, a)
	a, ok = d.GetBookmark("b2")
	assert.True(t, ok)
	assert.Equal(t, doc.Anchor{LineIndex: 2, LineOffset: 2}, a)

	d.SetCapacity(3)
	a, ok = d.GetBookmark("b1")
	assert.True(t, ok)
	assert.Equal(t, doc.Anchor{LineIndex: 0, LineOffset: 4}, a)
	a, ok = d.GetBookmark("b2")
	assert.True(t, ok)
	assert.Equal(t, doc.Anchor{LineIndex: 1, LineOffset: 2}, a)

	d.Add("line6")
	a, ok = d.GetBookmark("b1")
	assert.True(t, ok)
	assert.Equal(t, doc.Anchor{}, a)
	a, ok = d.GetBookmark("b2")
	assert.True(t, ok)
	assert.Equal(t, doc.Anchor{LineIndex: 0, LineOffset: 2}, a)
}

func TestDocument_BookmarksStartEnd(t *testing.T) {
	tests := []struct {
		name    string
		add     []string
		wantEnd doc.Anchor
	}{
		{
			name:    "Empty",
			wantEnd: doc.Anchor{},
		},
		{
			name:    "SingleLine",
			add:     []string{"line1"},
			wantEnd: doc.Anchor{LineIndex: 0, LineOffset: 5},
		},
		{
			name:    "MultipleLines",
			add:     []string{"line1", "l2", "l3"},
			wantEnd: doc.Anchor{LineIndex: 2, LineOffset: 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := doc.New()
			d.Add(tt.add...)

			var a doc.Anchor
			var ok bool

			a, ok = d.GetBookmark(doc.BookmarkStart)
			assert.True(t, ok)
			assert.Equal(t, doc.Anchor{}, a)

			a, ok = d.GetBookmark(doc.BookmarkEnd)
			assert.True(t, ok)
			assert.Equal(t, tt.wantEnd, a)
		})
	}
}

func TestDocument_UnknownBookmark(t *testing.T) {
	d := doc.New()
	a, ok := d.GetBookmark("foo")
	assert.False(t, ok)
	assert.Equal(t, doc.Anchor{}, a)
}

func TestDocument_SetBookmark_EmptyDocument(t *testing.T) {
	d := doc.New()
	d.SetBookmark("b", doc.Anchor{})
	a, ok := d.GetBookmark("b")
	assert.True(t, ok)
	assert.Equal(t, doc.Anchor{}, a)
}

func TestDocument_RemoveBookmark(t *testing.T) {
	d := doc.New()
	d.SetBookmark("b", doc.Anchor{})
	d.RemoveBookmark("b")
	_, ok := d.GetBookmark("b")
	assert.False(t, ok)
}

func TestDocument_String_EmptyDocument(t *testing.T) {
	d := doc.New()
	s, ok := d.String(doc.BookmarkStart, doc.BookmarkEnd, "EOL")
	assert.True(t, ok)
	assert.Equal(t, "", s)
}

func TestDocument_String_SingleLine(t *testing.T) {
	d := doc.New()
	d.Add("line1")
	s, ok := d.String(doc.BookmarkStart, doc.BookmarkEnd, "EOL")
	assert.True(t, ok)
	assert.Equal(t, "line1", s)
}

func TestDocument_String_Multiline(t *testing.T) {
	d := doc.New()
	d.Add("line1", "line2", "line3")
	s, ok := d.String(doc.BookmarkStart, doc.BookmarkEnd, "EOL")
	assert.True(t, ok)
	assert.Equal(t, "line1EOLline2EOLline3", s)
}

func TestDocument_String_BetweenBookmarks(t *testing.T) {
	tests := []struct {
		name       string
		start, end doc.Anchor
		want       string
	}{
		{
			name:  "WithinOneLine",
			start: doc.Anchor{LineIndex: 1, LineOffset: 1},
			end:   doc.Anchor{LineIndex: 1, LineOffset: 4},
			want:  "ine",
		},
		{
			name:  "WithinOneLineBackward",
			start: doc.Anchor{LineIndex: 1, LineOffset: 4},
			end:   doc.Anchor{LineIndex: 1, LineOffset: 1},
			want:  "ine",
		},
		{
			name:  "MultiLine",
			start: doc.Anchor{LineIndex: 1, LineOffset: 1},
			end:   doc.Anchor{LineIndex: 2, LineOffset: 3},
			want:  "ine2EOLlin",
		},
		{
			name:  "MultiLineBackward",
			start: doc.Anchor{LineIndex: 2, LineOffset: 3},
			end:   doc.Anchor{LineIndex: 1, LineOffset: 1},
			want:  "ine2EOLlin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := doc.New()
			d.Add("line1", "line2", "line3", "line4")
			d.SetBookmark("start", tt.start)
			d.SetBookmark("end", tt.end)
			s, ok := d.String("start", "end", "EOL")
			assert.True(t, ok)
			assert.Equal(t, tt.want, s)
		})
	}
}

func TestDocument_String_NoBookmark(t *testing.T) {
	tests := []struct {
		name       string
		start, end any
	}{
		{
			name:  "UnknownStart",
			start: "foo",
			end:   doc.BookmarkEnd,
		},
		{
			name:  "UnknownEnd",
			start: doc.BookmarkStart,
			end:   "foo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := doc.New()
			_, ok := d.String(tt.start, tt.end, "EOL")
			assert.False(t, ok)
		})
	}
}
