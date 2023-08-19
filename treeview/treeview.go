package treeview

import (
	"log"
	. "tallyGo/countable"
	EventBus "tallyGo/eventBus"

	"github.com/diamondburned/gotk4/pkg/core/glib"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

type TreeRowObject interface {
	Store() *gio.ListStore
	Model() *gio.ListModel
	GetWidget() *gtk.Widget
	Expander() *gtk.TreeExpander
	Countable() Countable
}

type CounterTreeView struct {
	*gtk.ListView

	store   *gio.ListStore
	objects map[*glib.Object]TreeRowObject
}

func NewCounterTreeView(counters *CounterList) (self *CounterTreeView) {
	self = &CounterTreeView{
		ListView: nil,
		store:    nil,
		objects:  map[*glib.Object]TreeRowObject{},
	}

	self.ListView = gtk.NewListView(nil, nil)
	self.AddCSSClass("counterTreeView")

	self.store = gio.NewListStore(glib.TypeObject)
	treeListModel := gtk.NewTreeListModel(self.store, false, false, self.createTreeModel)
	selectionModel := gtk.NewMultiSelection(treeListModel)
	selectionModel.ConnectSelectionChanged(func(uint, uint) {
		var selection []Countable
		for i := uint(0); i < selectionModel.NItems(); i++ {
			if selectionModel.IsSelected(i) {
				rowObj := findRowObj(self.objects, selectionModel.Item(i).Cast().(*gtk.TreeListRow).Item())
				if rowObj == nil {
					log.Fatal("[FATAL]\tCould not get Underlying Counter from Selection")
				}
				selection = append(selection, rowObj.Countable())
			}
		}
		counters.SetActive(selection...)
	})
	self.SetModel(selectionModel)

	factory := gtk.NewSignalListItemFactory()
	factory.ConnectBind(self.bindRow)
	self.SetFactory(&factory.ListItemFactory)

	for _, c := range counters.List {
		row := NewCounterRow(c, self.objects)
		self.objects[row.Object] = row
		self.store.Append(row.Object)
		sep := gtk.NewSeparator(gtk.OrientationHorizontal)
		self.store.Append(sep.Object)
	}

	EventBus.GetGlobalBus().Subscribe(CounterRemoved, func(args ...interface{}) {
		counter := args[0].(*Counter)
		if idx, ok := self.GetIdxFromCounter(counter); ok {
			self.store.Remove(idx)
		}
	})

	return
}

func (self *CounterTreeView) createTreeModel(gObj *glib.Object) *gio.ListModel {
	return findRowObj(self.objects, gObj).Model()
}

func (self *CounterTreeView) bindRow(listItem *gtk.ListItem) {
	row := listItem.Item().Cast().(*gtk.TreeListRow)
	gObj := row.Item()

	switch row.Item().Type().Name() {
	case "GtkTreeExpander":
		if rowObj := findRowObj(self.objects, gObj); rowObj != nil {
			rowObj.Expander().SetListRow(row)
			listItem.SetChild(rowObj.GetWidget())
		}
	case "GtkBox":
		if rowObj := findRowObj(self.objects, gObj); rowObj != nil {
			listItem.SetChild(rowObj.GetWidget())
		}
	case "GtkSeparator":
		listItem.SetSelectable(false)
		listItem.SetActivatable(false)
		sep := row.Item().Cast().(*gtk.Separator)
		listItem.SetChild(sep)
	case "GtkButton":
		listItem.SetSelectable(false)
		listItem.SetActivatable(false)
		button := row.Item().Cast().(*gtk.Button)
		listItem.SetChild(button)
	}
}

func (self *CounterTreeView) GetIdxFromCounter(counter *Counter) (uint, bool) {
	for i := uint(0); i < self.store.NItems(); i++ {
		rowObj := findRowObj(self.objects, self.store.Item(i))
		if c, ok := rowObj.Countable().(*Counter); ok {
			if c == counter {
				return i, true
			}
		}
	}
	return 0, false
}

func findRowObj(objects map[*glib.Object]TreeRowObject, obj *glib.Object) TreeRowObject {
	for key, value := range objects {
		if key.Eq(obj) {
			return value
		}
	}
	return nil
}