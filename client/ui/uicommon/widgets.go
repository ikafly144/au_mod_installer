package uicommon

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

// TappableContainer is a container that handles tap events
type TappableContainer struct {
	widget.BaseWidget
	Content           fyne.CanvasObject
	OnTapped          func()
	OnSecondaryTapped func(*fyne.PointEvent)
}

func NewTappableContainer(content fyne.CanvasObject, tapped func()) *TappableContainer {
	c := &TappableContainer{Content: content, OnTapped: tapped}
	c.ExtendBaseWidget(c)
	return c
}

func NewTappableContainerWithSecondary(content fyne.CanvasObject, tapped func(), secondaryTapped func(*fyne.PointEvent)) *TappableContainer {
	c := &TappableContainer{Content: content, OnTapped: tapped, OnSecondaryTapped: secondaryTapped}
	c.ExtendBaseWidget(c)
	return c
}

func (c *TappableContainer) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(c.Content)
}

func (c *TappableContainer) Tapped(_ *fyne.PointEvent) {
	if c.OnTapped != nil {
		c.OnTapped()
	}
}

func (c *TappableContainer) TappedSecondary(ev *fyne.PointEvent) {
	if c.OnSecondaryTapped != nil {
		c.OnSecondaryTapped(ev)
	}
}

func (c *TappableContainer) Cursor() desktop.Cursor {
	return desktop.PointerCursor
}

func (c *TappableContainer) MouseIn(*desktop.MouseEvent) {
	// Optional: hover effect
}

func (c *TappableContainer) MouseOut() {
	// Optional: hover effect
}

func (c *TappableContainer) MouseMoved(*desktop.MouseEvent) {
}

// Ensure TappableContainer implements necessary interfaces
var _ fyne.Widget = (*TappableContainer)(nil)
var _ fyne.Tappable = (*TappableContainer)(nil)
var _ fyne.SecondaryTappable = (*TappableContainer)(nil)
var _ desktop.Hoverable = (*TappableContainer)(nil)
