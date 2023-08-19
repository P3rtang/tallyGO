package treeview

import (
	. "tallyGo/countable"
	EventBus "tallyGo/eventBus"

	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

type CounterRow struct {
	*gtk.TreeExpander
	store *gio.ListStore

	counter     *Counter
	contextMenu *TreeRowContextMenu
}

func NewCounterRow(counter *Counter, objMap map[*glib.Object]TreeRowObject) (self *CounterRow) {
	self = &CounterRow{
		TreeExpander: gtk.NewTreeExpander(),
		store:        gio.NewListStore(glib.TypeObject),
		counter:      counter,
		contextMenu:  newTreeRowContextMenu(),
	}

	// remove the default shortcuts as it will mess with increasing and decreasing counters
	if shortcutCtrl, ok := self.TreeExpander.ObserveControllers().Item(1).Cast().(*gtk.ShortcutController); ok {
		self.TreeExpander.RemoveController(shortcutCtrl)
	}

	box := NewCounterRowBox(self.counter)
	self.TreeExpander.SetChild(box)

	self.SetupContextMenu()

	for _, p := range counter.Phases {
		self.NewPhase(p, objMap)
	}

	EventBus.GetGlobalBus().Subscribe(PhaseAdded, func(args ...interface{}) {
		counter := args[0].(*Counter)
		newPhase := args[1].(*Phase)

		if self.counter == counter {
			self.NewPhase(newPhase, objMap)
		}
	})

	EventBus.GetGlobalBus().Subscribe(PhaseRemoved, func(args ...interface{}) {
		counter := args[0].(*Counter)
		phase := args[1].(*Phase)

		if self.counter == counter {
			for i := uint(0); i < self.store.NItems(); i++ {
				if phase == findRowObj(objMap, self.store.Item(i)).Countable().(*Phase) {
					self.store.Remove(i)
				}
			}
		}
	})

	return
}

func (self *CounterRow) Store() *gio.ListStore {
	return self.store
}

func (self *CounterRow) Model() *gio.ListModel {
	return &self.store.ListModel
}

func (self *CounterRow) GetWidget() *gtk.Widget {
	return &self.Widget
}

func (self *CounterRow) Expander() *gtk.TreeExpander {
	return self.TreeExpander
}

func (self *CounterRow) Countable() Countable {
	return self.counter
}

func (self *CounterRow) SetupContextMenu() {
	self.contextMenu.SetParent(&self.TreeExpander.Widget)
	self.contextMenu.NewRow("mark complete", func() {
		if self.counter.IsCompleted() {
			self.contextMenu.rows["mark complete"].SetText("mark complete")
			self.counter.SetCompleted(false)
		} else {
			self.contextMenu.rows["mark complete"].SetText("mark incomplete")
			self.counter.SetCompleted(true)
		}
	})
	if self.counter.IsCompleted() {
		self.contextMenu.rows["mark complete"].SetText("mark incomplete")
	}
	self.contextMenu.NewRow("edit", func() {
	})
	self.contextMenu.NewRow("delete", func() {
		EventBus.GetGlobalBus().SendSignal(RemoveCounter, self.counter)
	})
}

func (self *CounterRow) NewPhase(phase *Phase, objMap map[*glib.Object]TreeRowObject) {
	row := NewPhaseRow(phase)
	objMap[row.Object] = row
	self.store.Append(row.Object)
}

type CounterRowBox struct {
	*gtk.Box
	label  *gtk.Label
	button *gtk.Button
}

func NewCounterRowBox(counter *Counter) (self *CounterRowBox) {
	self = &CounterRowBox{
		Box:    gtk.NewBox(gtk.OrientationHorizontal, 0),
		label:  gtk.NewLabel(counter.Name),
		button: gtk.NewButtonWithLabel("+"),
	}

	self.Box.Append(self.label)
	self.Box.Append(self.button)
	self.Box.AddCSSClass("counterBoxRow")

	self.label.SetHExpand(true)
	self.label.SetXAlign(0)

	self.button.SetHAlign(gtk.AlignEnd)
	self.button.ConnectClicked(func() {
		counter.NewPhase()
	})

	return
}
