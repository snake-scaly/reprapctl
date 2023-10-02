package logview

import "fyne.io/fyne/v2"

var _ fyne.WidgetRenderer = (*StackRenderer)(nil)

type StackRenderer struct {
	OnLayout  func(size fyne.Size)
	OnRefresh func()
	objects   []fyne.CanvasObject
}

func NewStackRenderer(o ...fyne.CanvasObject) *StackRenderer {
	return &StackRenderer{objects: o}
}

func (r *StackRenderer) Destroy() {
}

func (r *StackRenderer) Layout(size fyne.Size) {
	for _, o := range r.objects {
		o.Resize(size)
	}
	if r.OnLayout != nil {
		r.OnLayout(size)
	}
}

func (r *StackRenderer) MinSize() fyne.Size {
	var s fyne.Size
	for _, o := range r.objects {
		s = s.Max(o.MinSize())
	}
	return s
}

func (r *StackRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

func (r *StackRenderer) Refresh() {
	for _, o := range r.objects {
		o.Refresh()
	}
	if r.OnRefresh != nil {
		r.OnRefresh()
	}
}
