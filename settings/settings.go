package settings

import (
	"tallyGo/input"

	"github.com/diamondburned/gotk4/pkg/core/glib"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

type Settings struct {
	Items     map[SettingsKey]interface{}
	callbacks map[SettingsKey][]func(value interface{})
}

func NewSettings() *Settings {
	items := map[SettingsKey]interface{}{}
	items[ActiveKeyboard] = input.GetKbdList()[0]

	callbacks := map[SettingsKey][]func(value interface{}){}

	this := &Settings{items, callbacks}
	return this
}

func (self *Settings) ConnectChanged(key SettingsKey, f func(value interface{})) {
	if self.callbacks == nil {
		self.callbacks = map[SettingsKey][]func(value interface{}){}
	}
	self.callbacks[key] = append(self.callbacks[key], f)
}

func (self *Settings) GetValue(key SettingsKey) interface{} {
	return self.Items[key]
}

func (self *Settings) SetValue(key SettingsKey, value interface{}) {
	self.Items[key] = value
	println(value)
	for _, f := range self.callbacks[key] {
		f(value)
	}
}

type SettingsKey uint

const (
	ActiveKeyboard SettingsKey = 1
)

type SettingsMenu struct {
	*gtk.Grid

	listView *SettingsItems
	settings *Settings
}

func NewSettingsMenu(settings *Settings) *SettingsMenu {
	this := SettingsMenu{nil, nil, settings}

	grid := gtk.NewGrid()

	listView := NewSettingsItems()
	listView.SetSizeRequest(300, -1)
	listView.selectionModel.ConnectSelectionChanged(func(uint, uint) {
		grid.Attach(listView.selection().menuGrid(settings).grid(), 1, 0, 1, 1)
	})

	grid.Attach(listView, 0, 0, 1, 1)
	grid.Attach(Keyboard.menuGrid(settings).grid(), 1, 0, 1, 1)

	this.Grid = grid
	this.listView = listView
	return &this
}

func (self *SettingsMenu) AddItem(key SettingsItemKey) {
	label := gtk.NewLabel(key.String())
	self.listView.store.Append(label.Object)
	self.listView.items[key] = key.menuGrid(self.settings)
}

type SettingsItems struct {
	*gtk.ListView

	store          *gio.ListStore
	selectionModel *gtk.SingleSelection
	items          map[SettingsItemKey]SettingsItemGrid
}

func NewSettingsItems() *SettingsItems {
	store := gio.NewListStore(glib.TypeObject)
	selectionModel := gtk.NewSingleSelection(store)
	selectionModel.UnselectAll()

	itemFactory := gtk.NewSignalListItemFactory()
	list := gtk.NewListView(selectionModel, &itemFactory.ListItemFactory)
	list.AddCSSClass("settingsListView")
	list.SetVExpand(true)

	items := map[SettingsItemKey]SettingsItemGrid{}

	this := SettingsItems{list, store, selectionModel, items}
	itemFactory.ConnectBind(this.bindRow)

	return &this
}

func (self *SettingsItems) createRow(listItem *gtk.ListItem) {
	listItem.SetChild(gtk.NewLabel(""))
}

func (self *SettingsItems) bindRow(listItem *gtk.ListItem) {
	row := listItem.Item()
	listItem.SetChild(row.Cast().(*gtk.Label))
}

func (self *SettingsItems) selection() SettingsItemKey {
	return SettingsItemKey(self.selectionModel.Selected())
}

type SettingsItemKey int

func (self SettingsItemKey) String() string {
	switch self {
	case 0:
		return "Keyboard"
	}
	return "Item Not Found"
}

func (self SettingsItemKey) menuGrid(settings *Settings) SettingsItemGrid {
	switch self {
	case Keyboard:
		return NewKeyboardSettingsGrid(settings)
	}

	return nil
}

const (
	Keyboard SettingsItemKey = iota
)

type SettingsItemGrid interface {
	grid() *gtk.Grid
}

type KeyboardSettingsGrid struct {
	*gtk.Grid
}

func NewKeyboardSettingsGrid(settings *Settings) *KeyboardSettingsGrid {
	box := gtk.NewBox(gtk.OrientationVertical, 0)
	chooser := NewKeyboardChooser(settings)
	box.Append(chooser)

	grid := gtk.NewGrid()
	grid.Attach(box, 0, 0, 1, 1)

	this := KeyboardSettingsGrid{grid}

	chooser.NotifyProperty("selected", func() {
		settings.SetValue(ActiveKeyboard, chooser.ActiveKeyboard())
	})

	return &this
}

func (self *KeyboardSettingsGrid) grid() *gtk.Grid {
	return self.Grid
}

type KeyboardChooser struct {
	*gtk.DropDown

	keybds map[uint]string
}

func NewKeyboardChooser(settings *Settings) *KeyboardChooser {
	value := settings.GetValue(ActiveKeyboard).(string)

	store := gio.NewListStore(glib.TypeObject)
	factory := gtk.NewSignalListItemFactory()
	factory.ConnectSetup(setupKbd)
	factory.ConnectBind(bindKbd)
	dropdown := gtk.NewDropDown(store, nil)
	dropdown.SetFactory(&factory.ListItemFactory)

	keyboards := map[uint]string{}
	selectedIndex := -1
	for i, kbd := range input.GetKbdList() {
		keyboards[uint(i)] = kbd
		if kbd == value {
			selectedIndex = i
		}
		store.Append(gtk.NewLabel(kbd).Object)
	}

	if selectedIndex == -1 {
		store.Insert(0, gtk.NewLabel(value).Object)
	}
	dropdown.SetSelected(uint(selectedIndex))

	this := KeyboardChooser{dropdown, keyboards}
	return &this
}

func setupKbd(listItem *gtk.ListItem) {
	listItem.SetChild(gtk.NewLabel(""))
}

func bindKbd(listItem *gtk.ListItem) {
	label := listItem.Item().Cast().(*gtk.Label)
	listItem.SetChild(gtk.NewLabel(label.Label()))
}

func (self *KeyboardChooser) ActiveKeyboard() string {
	return self.keybds[self.Selected()]
}
