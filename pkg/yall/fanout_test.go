package yall_test

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"log/slog"
	"reprapctl/pkg/yall"
	"testing"
)

func TestFanOutSink_Handle(t *testing.T) {
	tests := []struct {
		name  string
		sinks []*testSink
	}{
		{
			name: "Empty",
		},
		{
			name:  "SingleDisabled",
			sinks: []*testSink{{enabled: false}},
		},
		{
			name:  "FirstDisabled",
			sinks: []*testSink{{enabled: false}, {enabled: true}},
		},
		{
			name:  "SecondDisabled",
			sinks: []*testSink{{enabled: true}, {enabled: false}},
		},
		{
			name:  "MiddleDisabled",
			sinks: []*testSink{{enabled: true}, {enabled: false}, {enabled: true}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := newFanOutSinkWithTestSinks(tt.sinks)
			r := rec()

			e := d.Handle(someCtx, r)

			assert.Nil(t, e)

			for _, s := range tt.sinks {
				if s.enabled {
					assert.Equal(t, 1, len(s.calls))
					assert.Equal(t, someCtx, s.calls[0].ctx)
					assert.Equal(t, r, s.calls[0].record)
				} else {
					assert.Zero(t, len(s.calls))
				}
			}
		})
	}
}

func TestFanOutSink_Handle_Errors(t *testing.T) {
	e1 := errors.New("e1")
	e2 := errors.New("e2")
	e3 := errors.New("e3")

	tests := []struct {
		name  string
		sinks []*testSink
		is    []error
		isNot []error
	}{
		{
			name:  "SingleError",
			sinks: []*testSink{{enabled: true, err: e1}},
			is:    []error{e1},
		},
		{
			name:  "TwoErrors",
			sinks: []*testSink{{enabled: true, err: e1}, {enabled: true, err: e2}},
			is:    []error{e1, e2},
		},
		{
			name:  "FirstThirdError",
			sinks: []*testSink{{enabled: true, err: e1}, {enabled: true}, {enabled: true, err: e3}},
			is:    []error{e1, e3},
		},
		{
			name:  "SecondDisabled",
			sinks: []*testSink{{enabled: true, err: e1}, {enabled: false, err: e2}, {enabled: true, err: e3}},
			is:    []error{e1, e3},
			isNot: []error{e2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := newFanOutSinkWithTestSinks(tt.sinks)
			r := rec()

			e := d.Handle(someCtx, r)

			for _, is := range tt.is {
				assert.ErrorIs(t, e, is)
			}
			for _, isNot := range tt.isNot {
				assert.NotErrorIs(t, e, isNot)
			}
		})
	}
}

func TestFanOutSink_Enabled(t *testing.T) {
	tests := []struct {
		name  string
		sinks []*testSink
		want  bool
	}{
		{
			name: "EmptyDisabled",
			want: false,
		},
		{
			name:  "SingleEnabled",
			sinks: []*testSink{{enabled: true}},
			want:  true,
		},
		{
			name:  "SingleDisabled",
			sinks: []*testSink{{enabled: false}},
			want:  false,
		},
		{
			name:  "FirstEnabled",
			sinks: []*testSink{{enabled: true}, {enabled: false}},
			want:  true,
		},
		{
			name:  "SecondEnabled",
			sinks: []*testSink{{enabled: false}, {enabled: true}},
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := newFanOutSinkWithTestSinks(tt.sinks)
			e := d.Enabled(someCtx, slog.LevelInfo)
			assert.Equal(t, tt.want, e)
		})
	}
}

func TestFanOutSink_AddSink(t *testing.T) {
	s1 := &testSink{enabled: true}
	s2 := &testSink{enabled: true}
	d := yall.NewFanOutSink()
	d.AddSink(s1)
	d.AddSink(s2)

	e := d.Handle(someCtx, rec())

	assert.Nil(t, e)
	assert.Equal(t, 1, len(s1.calls))
	assert.Equal(t, 1, len(s2.calls))
}

func TestFanOutSink_RemoveSink(t *testing.T) {
	s1 := &testSink{enabled: true}
	s2 := &testSink{enabled: true}
	d := yall.NewFanOutSink()
	d.AddSink(s1)
	d.AddSink(s2)

	removed := d.RemoveSink(s1)
	err := d.Handle(someCtx, rec())

	assert.True(t, removed)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(s1.calls))
	assert.Equal(t, 1, len(s2.calls))
}

func TestFanOutSink_RemoveAllSinks(t *testing.T) {
	s1 := &testSink{enabled: true}
	s2 := &testSink{enabled: true}
	d := yall.NewFanOutSink()
	d.AddSink(s1)
	d.AddSink(s2)

	removed1 := d.RemoveSink(s1)
	removed2 := d.RemoveSink(s2)
	err := d.Handle(someCtx, rec())

	assert.True(t, removed1)
	assert.True(t, removed2)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(s1.calls))
	assert.Equal(t, 0, len(s2.calls))
}

func TestFanOutSink_RemoveInitialSink(t *testing.T) {
	s1 := &testSink{enabled: true}
	s2 := &testSink{enabled: true}
	d := yall.NewFanOutSink(s1, s2)

	removed := d.RemoveSink(s1)
	err := d.Handle(someCtx, rec())

	assert.True(t, removed)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(s1.calls))
	assert.Equal(t, 1, len(s2.calls))
}

func TestFanOutSink_RemoveUnknownSink(t *testing.T) {
	d := yall.NewFanOutSink(&testSink{enabled: true})
	removed := d.RemoveSink(&testSink{enabled: false})
	assert.False(t, removed)
}

func newFanOutSinkWithTestSinks(sinks []*testSink) *yall.FanOutSink {
	ss := make([]yall.Sink, len(sinks))
	for i, s := range sinks {
		ss[i] = s
	}
	return yall.NewFanOutSink(ss...)
}
