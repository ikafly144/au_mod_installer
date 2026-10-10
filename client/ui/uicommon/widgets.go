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

// EventCatcherContainer consumes tap events to prevent them from passing to underlying layers
type EventCatcherContainer struct {
	widget.BaseWidget
	Content fyne.CanvasObject
}

func NewEventCatcherContainer(content fyne.CanvasObject) *EventCatcherContainer {
	c := &EventCatcherContainer{Content: content}
	c.ExtendBaseWidget(c)
	return c
}

func (c *EventCatcherContainer) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(c.Content)
}

func (c *EventCatcherContainer) Tapped(_ *fyne.PointEvent) {
	// Consume tap event
}

func (c *EventCatcherContainer) TappedSecondary(_ *fyne.PointEvent) {
	// Consume secondary tap event
}

func (c *EventCatcherContainer) DoubleTapped(_ *fyne.PointEvent) {
	// Consume double tap event
}

func (c *EventCatcherContainer) MouseIn(*desktop.MouseEvent) {
	// Consume mouse in to prevent hover passing through
}

func (c *EventCatcherContainer) MouseOut() {
	// Consume mouse out
}

func (c *EventCatcherContainer) MouseMoved(*desktop.MouseEvent) {
	// Consume mouse move
}

func (c *EventCatcherContainer) Cursor() desktop.Cursor {
	return desktop.DefaultCursor
}

func (c *EventCatcherContainer) Scrolled(_ *fyne.ScrollEvent) {
	// Consume scroll event on empty container areas
}

var _ fyne.Widget = (*EventCatcherContainer)(nil)
var _ fyne.Tappable = (*EventCatcherContainer)(nil)
var _ fyne.SecondaryTappable = (*EventCatcherContainer)(nil)
var _ fyne.DoubleTappable = (*EventCatcherContainer)(nil)
var _ desktop.Hoverable = (*EventCatcherContainer)(nil)
var _ desktop.Cursorable = (*EventCatcherContainer)(nil)
var _ fyne.Scrollable = (*EventCatcherContainer)(nil)
