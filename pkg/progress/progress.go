package progress

import (
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
