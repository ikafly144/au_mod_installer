package progress

import (
	"io"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
)

type Progress interface {
	SetValue(value float64)
	Done()
	Start() bool
	Working() bool
	GetValue() float64
}

func NewPhaseProgress(parent Progress, start, scale float64) *PhaseProgress {
	return &PhaseProgress{
		parent: parent,
		start:  start,
		scale:  scale,
	}
}

type PhaseProgress struct {
	l       sync.Mutex
	working bool
	parent  Progress
	start   float64
	scale   float64
}

func (p *PhaseProgress) SetValue(value float64) {
	if p.parent == nil {
		return
	}
	p.parent.SetValue(p.start + min(max(value, 0.0), 1.0)*p.scale)
}

func (p *PhaseProgress) Done() {
	p.l.Lock()
	defer p.l.Unlock()
	p.SetValue(1.0)
	p.working = false
}

func (p *PhaseProgress) Start() bool {
	p.l.Lock()
	defer p.l.Unlock()
	if p.working {
		return false
	}
	p.working = true
	return true
}

func (p *PhaseProgress) Working() bool {
	p.l.Lock()
	defer p.l.Unlock()
	return p.working
}

func (p *PhaseProgress) GetValue() float64 {
	if p.parent == nil || p.scale <= 0 {
		return 0
	}
	return min(max((p.parent.GetValue()-p.start)/p.scale, 0.0), 1.0)
}

type ProgressWriter struct {
	start    float64
	scale    float64
	goal     int64
	progress Progress
	bytes    int64
	writer   io.Writer
}

func NewProgressWriter(start, scale float64, goal int64, progress Progress, writer io.Writer) *ProgressWriter {
	return &ProgressWriter{
		start:    start,
		scale:    scale,
		goal:     goal,
		progress: progress,
		writer:   writer,
	}
}

func (pw *ProgressWriter) SetWriter(writer io.Writer) {
	pw.writer = writer
}

func (pw *ProgressWriter) Write(data []byte) (n int, err error) {
	if pw.writer != nil {
		n, err = pw.writer.Write(data)
	}
	pw.bytes += int64(n)
	if pw.goal > 0 && pw.progress != nil {
		pw.progress.SetValue(min(float64(pw.bytes)/float64(pw.goal), 1.0)*pw.scale + pw.start)
	}
	return
}

func (pw *ProgressWriter) Complete() {
	if pw.scale > 0 && pw.progress != nil {
		pw.progress.SetValue(pw.start + pw.scale)
	}
}

func NewFyneProgress(progress *widget.ProgressBar) *FyneProgress {
	bind := binding.NewFloat()
	progress.Bind(bind)
	return &FyneProgress{
		progress: progress,
		bind:     bind,
	}
}

type FyneProgress struct {
	l        sync.Mutex
	working  bool
	progress *widget.ProgressBar
	bind     binding.Float
}

func (p *FyneProgress) SetValue(value float64) {
	_ = p.bind.Set(value)
}

func (p *FyneProgress) Done() {
	p.l.Lock()
	defer p.l.Unlock()
	p.SetValue(1.0)
	p.working = false
}

func (p *FyneProgress) Canvas() fyne.CanvasObject {
	return p.progress
}

func (p *FyneProgress) Start() bool {
	p.l.Lock()
	defer p.l.Unlock()
	if p.working {
		return false
	}
	p.working = true
	return true
}

func (p *FyneProgress) Working() bool {
	p.l.Lock()
	defer p.l.Unlock()
	return p.working
}

func (p *FyneProgress) GetValue() float64 {
	val, _ := p.bind.Get()
	return val
}
