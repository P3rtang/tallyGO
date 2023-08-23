package resizebar

import (
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

type ResizeBar struct {
	*gtk.Box

	separator    *gtk.Separator
	gesture      *gtk.GestureDrag
	resizeWidget *gtk.Widget

	maxWidthPercentage float64
	window             *gtk.Window
}

func NewResizeBar(orientation gtk.Orientation) (self *ResizeBar) {
	self = &ResizeBar{
		Box:                gtk.NewBox(orientation, 0),
		separator:          gtk.NewSeparator(orientation),
		gesture:            gtk.NewGestureDrag(),
		maxWidthPercentage: -1,
	}

	self.SetCursorFromName("ew-resize")
	if orientation == gtk.OrientationHorizontal {
		self.separator.SetHExpand(true)
		self.separator.SetVAlign(gtk.AlignCenter)
	} else {
		self.separator.SetVExpand(true)
		self.separator.SetHAlign(gtk.AlignCenter)
	}

	self.Box.Append(self.separator)

	self.AddController(self.gesture)
	self.gesture.ConnectDragUpdate(self.gestureUpdate)

	return
}

func (self *ResizeBar) Attach(widget *gtk.Widget) {
	self.resizeWidget = widget
}

func (self *ResizeBar) SetMaxWidth(percentage float64, window *gtk.Window) {
	self.maxWidthPercentage = percentage
	self.window = window
}

func (self *ResizeBar) Gesture() *gtk.GestureDrag {
	return self.gesture
}

func (self *ResizeBar) gestureUpdate(offsetX float64, offsetY float64) {
	if self.resizeWidget == nil {
		return
	}
	switch self.Orientation() {
	case gtk.OrientationHorizontal:
	case gtk.OrientationVertical:
		if self.maxWidthPercentage == -1 {
			self.resizeWidget.SetSizeRequest(self.resizeWidget.Width()+int(offsetX), -1)
		} else if self.resizeWidget.Width() < int(float64(self.window.Width())/2)-1 || offsetX < 0 {
			self.resizeWidget.SetSizeRequest(self.resizeWidget.Width()+int(offsetX), -1)
		}
	}
}
