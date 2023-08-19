package treeview

import (
	. "tallyGo/countable"
	EventBus "tallyGo/eventBus"

	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

type PhaseRow struct {
	*gtk.Box
	phase       *Phase
	contextMenu *TreeRowContextMenu
}

func NewPhaseRow(phase *Phase) (self *PhaseRow) {
	self = &PhaseRow{
		Box:         gtk.NewBox(gtk.OrientationHorizontal, 0),
		phase:       phase,
		contextMenu: newTreeRowContextMenu(),
	}

	box := NewPhaseRowBox(phase)
	self.Box.Append(box)
	self.Box.SetName("PhaseRow")

	EventBus.GetGlobalBus().Subscribe(NameChanged, func(args ...interface{}) {
		if self.phase == args[0] {
			box.setLabel(phase.Name)
		}
	})

	EventBus.GetGlobalBus().Subscribe(CompletedStatus, func(args ...interface{}) {
		if self.phase == args[0] {
			box.setPadlock(phase)
		}
	})

	self.SetupContextMenu()

	return
}

func (self *PhaseRow) Store() *gio.ListStore {
	return nil
}

func (self *PhaseRow) Model() *gio.ListModel {
	return nil
}

func (self *PhaseRow) GetWidget() *gtk.Widget {
	return &self.Box.Widget
}

func (self *PhaseRow) Expander() *gtk.TreeExpander {
	return nil
}

func (self *PhaseRow) Countable() Countable {
	return self.phase
}

func (self *PhaseRow) SetupContextMenu() {
	self.contextMenu.SetParent(&self.Box.Widget)
	self.contextMenu.NewRow("mark complete", func() {
		if self.phase.IsCompleted {
			self.contextMenu.rows["mark complete"].SetText("mark complete")
			self.phase.SetCompleted(false)
		} else {
			self.contextMenu.rows["mark complete"].SetText("mark incomplete")
			self.phase.SetCompleted(true)
		}
	})
	if self.phase.IsCompleted {
		self.contextMenu.rows["mark complete"].SetText("mark incomplete")
	}
	self.contextMenu.NewRow("edit", func() {
	})
	self.contextMenu.NewRow("delete", func() {
		EventBus.GetGlobalBus().SendSignal(RemovePhase, self.phase)
	})
}

type PhaseRowBox struct {
	*gtk.Box
	label *gtk.Label
	image *gtk.Image
}

func NewPhaseRowBox(phase *Phase) (self *PhaseRowBox) {
	self = &PhaseRowBox{
		Box:   gtk.NewBox(gtk.OrientationHorizontal, 0),
		label: gtk.NewLabel(phase.Name),
		image: gtk.NewImage(),
	}

	self.Box.Append(self.image)
	self.Box.Append(self.label)

	self.setPadlock(phase)

	return
}

func (self *PhaseRowBox) setPadlock(phase *Phase) {
	if phase.IsCompleted {
		self.image.SetFromIconName("padlock")
	} else {
		self.image.SetFromIconName("padlock-unlocked")
	}
}

func (self *PhaseRowBox) setLabel(name string) {
	self.label.SetText(name)
}
