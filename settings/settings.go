package settings

import (
	"tallyGo/input"

	"github.com/diamondburned/gotk4/pkg/core/glib"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

type Settings struct {
	Items     map[SettingsKey]any
	callbacks map[SettingsKey][]func(value any)
}

func NewSettings() *Settings {
	items := map[SettingsKey]any{}
	items[ActiveKeyboard] = input.GetKbdList()[0]

	callbacks := map[SettingsKey][]func(value any){}

	this := &Settings{items, callbacks}
	return this
}

func (self *Settings) ConnectChanged(key SettingsKey, f func(value any)) {
	if self.callbacks == nil {
		self.callbacks = map[SettingsKey][]func(value any){}
	}
	self.callbacks[key] = append(self.callbacks[key], f)
}

func (self *Settings) GetValue(key SettingsKey) any {
	return self.Items[key]
}

func (self *Settings) SetValue(key SettingsKey, value any) {
	self.Items[key] = value
	for _, f := range self.callbacks[key] {
		f(value)
	}
}

type SettingsKey uint

const (
	None SettingsKey = iota
	ActiveKeyboard
	DarkMode
)

type SettingsMenu struct {
	*gtk.Box

	listView *SettingsItems
	settings *Settings
}

func NewSettingsMenu(settings *Settings) (self *SettingsMenu) {
	self = &SettingsMenu{gtk.NewBox(gtk.OrientationHorizontal, 0), nil, settings}

	listView := NewSettingsItems()
	listView.SetSizeRequest(300, -1)
	listView.selectionModel.ConnectSelectionChanged(func(uint, uint) {
		self.Box.Remove(self.Box.LastChild())
		self.Box.Append(listView.selection().menuGrid(settings).grid())
	})

	self.Box.Append(listView)
	self.Box.Append(Keyboard.menuGrid(settings).grid())

	self.listView = listView
	return
}

func (self *SettingsMenu) AddItem(key SettingsItemKey) {
	label := gtk.NewLabel(string(key))
	self.listView.store.Append(label.Object)
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

func (self *SettingsItems) selection() (key SettingsItemKey) {
	switch self.selectionModel.Selected() {
	case 0:
		key = Keyboard
	case 1:
		key = Theme
	}
	return
}

type SettingsItemKey string

func (self SettingsItemKey) menuGrid(settings *Settings) SettingsItemGrid {
	switch self {
	case Keyboard:
		return NewKeyboardSettingsGrid(settings)
	case Theme:
		return NewThemeSettingsGrid(settings)
	}

	return nil
}

const (
	Keyboard SettingsItemKey = "Keyboard"
	Theme                    = "Theme"
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
	var value string
	var ok bool
	if value, ok = settings.GetValue(ActiveKeyboard).(string); !ok {
		value = input.GetKbdList()[0]
	}

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

type ThemeSettingsGrid struct {
	*gtk.Grid

	darkModeToggle *gtk.ToggleButton
	settings       *Settings
}

func NewThemeSettingsGrid(settings *Settings) (self *ThemeSettingsGrid) {
	self = &ThemeSettingsGrid{
		gtk.NewGrid(),
		gtk.NewToggleButton(),
		settings,
	}

	self.Grid.Attach(self.darkModeToggle, 0, 0, 1, 1)

	if self.settings.GetValue(DarkMode) == nil {
		self.settings.SetValue(DarkMode, false)
	}

	self.darkModeToggle.ConnectClicked(func() {
		self.settings.SetValue(DarkMode, !self.settings.GetValue(DarkMode).(bool))
		self.setIcon()
	})

	self.setIcon()

	return
}

func (self *ThemeSettingsGrid) grid() *gtk.Grid {
	return self.Grid
}

func (self *ThemeSettingsGrid) setIcon() {
	if self.settings.GetValue(DarkMode) == true {
		self.darkModeToggle.SetIconName("dark-mode-night-moon-svgrepo-com")
	} else {
		self.darkModeToggle.SetIconName("sun-svgrepo-com")
	}

}
