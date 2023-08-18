package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"reflect"
	"strconv"
	. "tallyGo/countable"
	EventBus "tallyGo/eventBus"
	"tallyGo/input"
	"tallyGo/settings"
	"time"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

const FRAME_TIME = time.Millisecond * 33
const SAVE_STRATEGY = JSON

//go:embed style.css
var CSS_FILE string

//go:embed style-dark.css
var CSS_DARK string

var APP *gtk.Application
var HOME *HomeApplicationWindow

// TODO: instead of just storing the date counter should store diffs with a time
// this will improve the info window
func main() {
	APP = gtk.NewApplication("com.github.p3rtang.counter", gio.ApplicationFlagsNone)
	APP.ConnectActivate(func() { activate(APP) })

	EventBus.InitBus()

	if code := APP.Run(os.Args); code > 0 {
		os.Exit(code)
	}
}

func activate(app *gtk.Application) (err error) {
	window := newHomeApplicationWindow(app)
	window.Show()
	return
}

type HomeApplicationWindow struct {
	*gtk.ApplicationWindow

	overlay      *gtk.Overlay
	homeGrid     *gtk.Grid
	settings     *settings.Settings
	settingsGrid *settings.SettingsMenu
	infoBox      *infoBox

	treeViewRevealer     *gtk.Revealer
	isRevealerAutoHidden bool
	collapseButton       *gtk.Button
	settingsButton       *gtk.ToggleButton
	headerBar            *gtk.HeaderBar
	isTimingActive       bool
}

func newHomeApplicationWindow(app *gtk.Application) (self *HomeApplicationWindow) {
	self = &HomeApplicationWindow{
		gtk.NewApplicationWindow(app),
		gtk.NewOverlay(),
		gtk.NewGrid(),
		nil,
		nil,
		nil,
		gtk.NewRevealer(),
		false,
		gtk.NewButtonFromIconName("open-menu-symbolic"),
		gtk.NewToggleButton(),
		gtk.NewHeaderBar(),
		false,
	}

	HOME = self
	eventBus := EventBus.GetGlobalBus()

	self.SetTitle("tallyGo")

	savePath, _ := os.UserHomeDir()
	savePath += "/.local/share/tallyGo/ProgramData.json"
	saveDataHandler := NewSaveFileHandler(savePath, SAVE_STRATEGY)
	saveDataHandler.Restore()

	self.settings = saveDataHandler.SettingsData
	self.settingsGrid = settings.NewSettingsMenu(self.settings)
	self.settingsGrid.AddItem(settings.Keyboard)
	self.settingsGrid.AddItem(settings.Theme)

	counters := NewCounterList(saveDataHandler.CounterData)
	app.ConnectShutdown(func() {
		saveDataHandler.CounterData = counters.List
		saveDataHandler.Save()
	})

	counterTV := newCounterTreeView(counters, self)

	scrollView := gtk.NewScrolledWindow()
	scrollView.SetChild(counterTV)
	scrollView.SetName("treeViewScrollWindow")

	self.treeViewRevealer.SetTransitionType(gtk.RevealerTransitionTypeSlideRight)
	self.treeViewRevealer.SetChild(scrollView)
	self.treeViewRevealer.SetRevealChild(true)

	self.collapseButton.ConnectClicked(func() {
		if self.treeViewRevealer.RevealChild() {
			self.treeViewRevealer.SetRevealChild(false)
		} else {
			self.treeViewRevealer.SetRevealChild(true)
		}
	})

	self.settingsButton.SetIconName("applications-system-symbolic")
	self.settingsButton.SetName("settingsButton")
	image := self.settingsButton.Child().(*gtk.Image)
	image.SetPixelSize(20)
	self.settingsButton.ConnectToggled(func() {
		if self.settingsButton.Active() {
			self.overlay.SetChild(self.settingsGrid)
		} else {
			self.overlay.SetChild(self.homeGrid)
		}
	})

	self.headerBar.PackStart(self.collapseButton)
	self.headerBar.PackEnd(self.settingsButton)
	self.Window.SetTitlebar(self.headerBar)

	self.SetChild(self.overlay)
	self.overlay.SetChild(self.homeGrid)
	self.SetDefaultSize(900, 660)
	self.NotifyProperty("default-width", self.HandleNotify)

	css := gtk.NewCSSProvider()
	gtk.StyleContextAddProviderForDisplay(gdk.DisplayGetDefault(), css, gtk.STYLE_PROVIDER_PRIORITY_SETTINGS)
	css.LoadFromData(CSS_FILE)
	self.setCSSTheme(css)

	saveDataHandler.SettingsData.ConnectChanged(settings.DarkMode, func(_ any) {
		self.setCSSTheme(css)
	})

	self.infoBox = NewInfoBox(counters)
	infoBoxScroll := gtk.NewScrolledWindow()
	infoBoxScroll.SetChild(self.infoBox)

	self.homeGrid.Attach(self.treeViewRevealer, 0, 1, 1, 9)
	self.homeGrid.Attach(infoBoxScroll, 1, 2, 1, 1)

	inputHandler := input.NewDevInput()
	err := inputHandler.Init(self.settings.GetValue(settings.ActiveKeyboard).(string))
	if err != nil {
		log.Println("[WARN] Could not initialize keyboard. Got Error: ", err)
	}
	self.settings.ConnectChanged(settings.ActiveKeyboard, func(value interface{}) {
		inputHandler.ChangeFile(value.(string))
		if err != nil {
			log.Println("[WARN] Could not initialize keyboard. Got Error: ", err)
		}
	})

	eventController := gtk.NewEventControllerKey()
	self.Window.AddController(eventController)
	eventController.ConnectKeyReleased(func(keyval uint, _ uint, _ gdk.ModifierType) {
		var key input.KeyType
		switch keyval {
		case 112:
			key = input.KeyP
		}
		inputHandler.SimulateKey(key, input.SimKeyReleased)
	})

	go func() {
		for {
			startInstant := time.Now()
			time.Sleep(FRAME_TIME)
			if self.isTimingActive && counters.HasActive() {
				glib.IdleAdd(func() {
					for _, countable := range counters.GetActive() {
						countable.AddTime(time.Now().Sub(startInstant))
					}
				})
			}
		}
	}()

	eventBus.Subscribe(input.DevKeyReleased, func(args ...interface{}) {
		key := args[0].(input.KeyType)

		switch {
		case key == input.KeyEqual || key == input.KeyKeypadPlus:
			if !self.isTimingActive {
				return
			}
			glib.IdleAdd(func() {
				for _, countable := range counters.GetActive() {
					countable.IncreaseBy(1)
				}
			})
			saveDataHandler.Save()

		case key == input.KeyMinus || key == input.KeyKeypadMinus:
			if !self.isTimingActive {
				return
			}
			glib.IdleAdd(func() {
				for _, countable := range counters.GetActive() {
					countable.IncreaseBy(-1)
				}
			})
			saveDataHandler.Save()

		case key == input.KeyQ:
			self.isTimingActive = false

		}
	})

	eventBus.Subscribe(input.SimKeyReleased, func(args ...interface{}) {
		key := args[0].(input.KeyType)

		switch {
		case key == input.KeyP:
			self.isTimingActive = !self.isTimingActive
		}
	})
	counterTV.CollapseAll()

	return
}

func (self *HomeApplicationWindow) setCSSTheme(css *gtk.CSSProvider) {
	if self.settings.GetValue(settings.DarkMode) == nil {
		self.settings.SetValue(settings.DarkMode, false)
	}
	if self.settings.GetValue(settings.DarkMode).(bool) {
		gtk.SettingsGetDefault().SetObjectProperty("gtk-theme-name", "Adwaita-dark")
		css.LoadFromData(CSS_DARK)
	} else {
		gtk.SettingsGetDefault().SetObjectProperty("gtk-theme-name", "Adwaita")
		css.LoadFromData(CSS_FILE)
	}
}

func (self *HomeApplicationWindow) HandleNotify() {
	switch {
	case self.Width() < 500:
		if self.treeViewRevealer.RevealChild() {
			self.treeViewRevealer.SetRevealChild(false)
			self.isRevealerAutoHidden = true
		}
		self.collapseButton.SetSensitive(false)
	case self.Width() > 500:
		if self.isRevealerAutoHidden {
			self.treeViewRevealer.SetRevealChild(true)
			self.isRevealerAutoHidden = false
		}
		self.collapseButton.SetSensitive(true)
	}

	if self.Height() < 360 {
		self.headerBar.SetVisible(false)
	} else {
		self.headerBar.SetVisible(true)
	}
}

func createColumn(title string, id int) *gtk.TreeViewColumn {
	cellRenderer := gtk.NewCellRendererText()
	column := gtk.NewTreeViewColumn()
	column.SetTitle(title)

	column.PackEnd(cellRenderer, false)
	column.AddAttribute(cellRenderer, "text", int(id))
	column.SetResizable(true)

	return column
}

type CounterTreeView struct {
	*gtk.ListView
	store      *gio.ListStore
	selection  *gtk.SingleSelection
	expanders  []*CounterExpander
	homeWindow *HomeApplicationWindow
}

func newCounterTreeView(cList *CounterList, homeWindow *HomeApplicationWindow) (self *CounterTreeView) {
	store := gio.NewListStore(glib.TypeObject)
	var expanderList []*CounterExpander

	self = &CounterTreeView{nil, store, nil, expanderList, homeWindow}

	treeStore := gtk.NewTreeListModel(store, false, true, self.createTreeModel)
	ssel := gtk.NewSingleSelection(treeStore)
	tv := gtk.NewListView(ssel, nil)
	tv.AddCSSClass("counterTreeView")
	tv.SetVExpand(true)

	self.ListView = tv
	self.selection = ssel

	factory := gtk.NewSignalListItemFactory()
	factory.ConnectBind(self.bindRow)
	tv.SetFactory(&factory.ListItemFactory)

	ssel.SetAutoselect(false)
	ssel.ConnectSelectionChanged(self.newSelection)

	newCounterButton := gtk.NewButtonWithLabel("New Counter")
	newCounterButton.ConnectClicked(func() {
		counter := NewCounter("Test", 0, OldOdds)
		self.addCounter(counter, cList)

		cList.List = append(cList.List, counter)
	})

	store.Append(newCounterButton.Object)

	for _, counter := range cList.List {
		self.addCounter(counter, cList)
	}

	return
}

func (self *CounterTreeView) CollapseAll() {
	for _, exp := range self.expanders {
		if row := exp.ListRow(); row != nil {
			row.SetExpanded(false)
		}
	}
}

func (self *CounterTreeView) addCounter(counter *Counter, cList *CounterList) {
	expander := newCounterExpander(counter)
	expander.contextMenu.ConnectRowClick("delete", func() {
		if idx, ok := self.store.Find(expander.Object); ok {
			self.store.Remove(idx)
			// remove the separator
			self.store.Remove(idx)
			cList.Remove(counter)
		} else {
			fmt.Printf("[4040] [WARN] Could not find Counter to delete")
		}
	})
	expander.contextMenu.ConnectRowClick("edit", func() {
		dialog := NewEditDialog(counter)
		dialog.Show()
	})
	expander.contextMenu.ConnectRowClick("mark complete", func() {
		if counter.IsCompleted() {
			counter.SetCompleted(false)
			expander.contextMenu.rows["mark complete"].SetText("mark complete")
		} else {
			counter.SetCompleted(true)
			expander.contextMenu.rows["mark complete"].SetText("mark in progress")
		}
	})

	self.expanders = append(self.expanders, expander)

	items := self.store.NItems()
	self.store.Insert(items-1, expander.Object)
	sep := gtk.NewSeparator(gtk.OrientationHorizontal)
	self.store.Insert(items, sep.Object)
}

func (self *CounterTreeView) newSelection(position uint, nItems uint) {
	row := self.selection.Item(self.selection.Selected()).Cast().(*gtk.TreeListRow)
	self.homeWindow.isTimingActive = false

	var phaseNum uint
	var counter *Counter
	switch row.Depth() {
	case 0:
		// exp := row.Item().Cast().(*gtk.TreeExpander)
		counter = getCounterExpander(row.Item(), self.expanders).counter
		phaseNum = uint(len(counter.Phases))
		self.homeWindow.infoBox.counterList.SetActive(counter)
	case 1:
		parentRow := row.Parent()
		// exp := parentRow.Item().Cast().(*gtk.TreeExpander)
		counter = getCounterExpander(parentRow.Item(), self.expanders).counter
		phaseNum = row.Position() - parentRow.Position()
		phase := counter.Phases[phaseNum-1]
		self.homeWindow.infoBox.counterList.SetActive(phase)
	}
}

func (self *CounterTreeView) createTreeModel(gObj *glib.Object) *gio.ListModel {
	if gObj.Type().Name() != "GtkTreeExpander" {
		return nil
	}

	// expander, _ := gObj.Cast().(*gtk.TreeExpander)
	store := getCounterExpander(gObj, self.expanders).store

	return &store.ListModel
}

func (self *CounterTreeView) bindRow(listItem *gtk.ListItem) {
	row := listItem.Item().Cast().(*gtk.TreeListRow)
	switch row.Item().Type().Name() {
	case "GtkTreeExpander":
		for _, exp := range self.expanders {
			if exp.Object.Eq(row.Item()) {
				exp.SetListRow(row)
				listItem.SetChild(exp)
			}
		}

		break
	case "GtkLabel":
		label := row.Item().Cast().(*gtk.Label)
		listItem.SetChild(label)
	case "GtkBox":
		box := row.Item().Cast().(*gtk.Box)
		listItem.SetChild(box)
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

func getCounterExpander(object *glib.Object, counters []*CounterExpander) (counter *CounterExpander) {
	for _, c := range counters {
		if c.Object.Eq(object) {
			return c
		}
	}
	return
}

type CounterExpander struct {
	*gtk.TreeExpander
	counter *Counter

	store       *gio.ListStore
	contextMenu *TreeRowContextMenu
}

func newCounterExpander(counter *Counter) (self *CounterExpander) {
	if counter == nil {
		return nil
	}

	expander := gtk.NewTreeExpander()
	shortcutCtrl, ok := expander.ObserveControllers().Item(1).Cast().(*gtk.ShortcutController)
	if ok {
		expander.RemoveController(shortcutCtrl)
	}

	store := gio.NewListStore(glib.TypeObject)

	box := gtk.NewBox(gtk.OrientationHorizontal, 0)
	box.AddCSSClass("counterBoxRow")

	label := gtk.NewLabel(counter.Name)
	label.SetHExpand(true)
	label.SetXAlign(0)

	button := gtk.NewButtonWithLabel("+")
	button.SetName("buttonAddPhase")
	button.ConnectClicked(func() {
		phase := counter.NewPhase(counter.ProgressType)
		row := newPhaseRow(phase)
		row.contextMenu.ConnectRowClick("delete", func() {
			fmt.Println(self.counter.Phases)
			for i, p := range self.counter.Phases {
				if p == phase {
					self.counter.Phases = append(self.counter.Phases[:i], self.counter.Phases[i+1:]...)
					idx, ok := store.Find(row.Object)
					if ok {
						store.Remove(idx)
					}
				}
			}
		})
		store.Append(row.Object)
	})

	box.Append(label)
	box.Append(button)
	expander.SetChild(box)

	contextMenu := newTreeRowContextMenu()
	contextMenu.NewRow("mark complete")
	contextMenu.NewRow("edit")
	contextMenu.NewRow("delete")
	contextMenu.SetParent(&box.Widget)

	if !counter.IsCompleted() {
		contextMenu.rows["mark complete"].SetText("mark complete")
	} else {
		contextMenu.rows["mark complete"].SetText("mark in progress")
	}

	for _, phase := range counter.Phases {
		row := newPhaseRow(phase)
		row.contextMenu.ConnectRowClick("delete", func() {
			for i, p := range self.counter.Phases {
				if p == phase {
					println(i)
					self.counter.Phases = append(self.counter.Phases[:i])
					idx, ok := store.Find(row.Object)
					if ok {
						store.Remove(idx)
					}
				}
			}
		})
		store.Append(row.Object)
	}

	self = &CounterExpander{expander, counter, store, contextMenu}
	EventBus.GetGlobalBus().Subscribe(NameChanged, func(args ...interface{}) {
		label.SetText(counter.Name)
	})

	return
}

type TreeRowContextMenu struct {
	*gtk.Popover
	items *gtk.ListBox
	rows  map[string]*gtk.Label
}

func newTreeRowContextMenu() *TreeRowContextMenu {
	rows := make(map[string]*gtk.Label)

	contextItems := gtk.NewListBox()
	contextItems.UnselectAll()
	contextItems.SetSelectionMode(gtk.SelectionNone)

	contextMenu := gtk.NewPopover()
	contextMenu.SetName("treeViewContext")
	contextMenu.SetChild(contextItems)
	contextMenu.SetHasArrow(false)

	return &TreeRowContextMenu{
		contextMenu,
		contextItems,
		rows,
	}
}

func (self *TreeRowContextMenu) SetParent(parent *gtk.Widget) {
	self.Popover.SetParent(parent)

	gesture := gtk.NewGestureClick()
	gesture.SetButton(3)
	gesture.ConnectPressed(func(int, float64, float64) {
		self.Popover.Show()
	})
	parent.AddController(gesture)
}

func (self *TreeRowContextMenu) NewRow(text string) {
	button := gtk.NewLabel(text)
	button.SetXAlign(0)
	self.rows[text] = button
	self.items.Append(button)
}

func (self *TreeRowContextMenu) ConnectRowClick(rowName string, f func()) {
	gesture := gtk.NewGestureClick()
	gesture.ConnectPressed(func(int, float64, float64) {
		self.Unmap()
		glib.IdleAdd(func() {
			f()
		})
	})
	self.rows[rowName].AddController(gesture)
}

type ContextMenuRow uint

const (
	RowDelete ContextMenuRow = iota
)

type PhaseRow struct {
	*gtk.Box
	label       *gtk.Label
	lock        *gtk.Image
	contextMenu *TreeRowContextMenu

	phase *Phase
}

func newPhaseRow(phase *Phase) (self *PhaseRow) {
	if phase == nil {
		return nil
	}

	box := gtk.NewBox(gtk.OrientationHorizontal, 0)
	box.AddCSSClass("phaseRow")

	label := gtk.NewLabel(phase.Name)
	label.SetHAlign(gtk.AlignStart)

	EventBus.GetGlobalBus().Subscribe(NameChanged, func(args ...interface{}) {
		label.SetText(phase.Name)
	})

	contextMenu := newTreeRowContextMenu()
	contextMenu.NewRow("lock")
	contextMenu.NewRow("edit")
	contextMenu.NewRow("delete")

	lock := gtk.NewImageFromIconName("padlock-unlocked")
	box.Append(lock)
	box.Append(label)

	contextMenu.SetParent(&box.Widget)

	self = &PhaseRow{
		box,
		label,
		lock,
		contextMenu,
		phase,
	}
	self.UpdateLock()

	contextMenu.ConnectRowClick("edit", func() {
		dialog := NewEditDialog(phase)
		dialog.Present()
	})

	contextMenu.ConnectRowClick("lock", func() {
		phase.SetCompleted(!phase.IsCompleted)
	})

	EventBus.GetGlobalBus().Subscribe(CompletedStatus, func(...interface{}) {
		self.UpdateLock()
	})

	return
}

func (self *PhaseRow) UpdateLock() {
	label := self.contextMenu.rows["lock"]

	if !self.phase.IsCompleted {
		self.lock.SetFromIconName("padlock-unlocked")
		label.SetText("lock")
	} else {
		self.lock.SetFromIconName("padlock")
		label.SetText("unlock")
	}
}

type EditDialog struct {
	*gtk.Dialog
	list      *gtk.Box
	buttonRow *gtk.Box

	rows map[string]interface{}

	countable Countable
}

func NewEditDialog(countable Countable) *EditDialog {
	window := gtk.NewDialog()
	window.SetResizable(false)

	mainWindow := APP.ActiveWindow()
	window.SetTransientFor(mainWindow)
	window.SetModal(true)

	listBox := gtk.NewBox(gtk.OrientationVertical, 0)
	listBox.AddCSSClass("editDialogBox")
	window.SetChild(listBox)

	rows := make(map[string]interface{})
	this := EditDialog{window, listBox, gtk.NewBox(gtk.OrientationHorizontal, 0), rows, countable}

	switch countable.(type) {
	case *Phase:
		phase := countable.(*Phase)
		this.NewRow("Name", phase.Name)
		this.NewRow("Count", phase.Count)
		this.NewRow("Time", phase.Time)
		this.NewRow("HuntType", fmt.Sprint(phase.Progress.GetType()))
		this.AddButton("cancel", func() {
			this.Close()
		})
		this.AddButton("confirm", func() {
			if name, ok := this.rows["Name"].(string); ok {
				phase.SetName(name)
			}
			if count, ok := this.rows["Count"].(int); ok {
				println(count)
				phase.SetCount(count)
			}
			if duration, ok := this.rows["Time"].(time.Duration); ok {
				phase.SetTime(duration)
			}
			if type_str, ok := this.rows["HuntType"].(string); ok {
				type_, _ := strconv.Atoi(type_str)
				phase.SetProgressType(ProgressType(type_))
			}
			this.Close()
		})
		break
	case *Counter:
		counter := countable.(*Counter)
		this.NewRow("Name", counter.Name)
		this.NewRow("Count", counter.GetCount())
		this.NewRow("HuntType", fmt.Sprint(counter.ProgressType))
		this.NewRow("Shiny Charm", counter.HasCharm())
		this.AddButton("cancel", func() {
			this.Close()
		})
		this.AddButton("confirm", func() {
			if name, ok := this.rows["Name"].(string); ok {
				counter.SetName(name)
			}
			if count, ok := this.rows["Count"].(int); ok {
				counter.SetCount(count)
			}
			if type_str, ok := this.rows["HuntType"].(string); ok {
				type_, _ := strconv.Atoi(type_str)
				counter.SetProgressType(ProgressType(type_))
			}
			if hasCharm, ok := this.rows["Shiny Charm"].(bool); ok {
				counter.SetCharm(hasCharm)
			}
			this.Close()
		})
		break
	}

	this.list.Append(this.buttonRow)
	this.buttonRow.AddCSSClass("editDialogButtonRow")
	this.buttonRow.SetHAlign(gtk.AlignEnd)

	return &this
}

func (self *EditDialog) NewRow(title string, value interface{}) {
	self.rows[title] = value

	switch value.(type) {
	case int:
		row := NewDialogRow(title, value.(int))
		self.list.Append(row)
		row.entry.ConnectChanged(func() { self.rows[title] = int(row.entry.Int()) })
		break
	case string:
		row := NewDialogRow(title, value.(string))
		self.list.Append(row)
		row.entry.ConnectChanged(func() { self.rows[title] = row.entry.Text() })
		break
	case time.Duration:
		row := NewDialogTimeRow(title, value.(time.Duration))
		self.list.Append(row)

		row.ConnectChanged(func() {
			dur := time.Hour.Nanoseconds()*row.hours.Int() + time.Minute.Nanoseconds()*row.mins.Int()
			self.rows[title] = time.Duration(dur)
		})
	case bool:
		row := NewDialogBoolRow(title, value.(bool))
		self.list.Append(row)

		row.ConnectChanged(func() {
			self.rows[title] = row.state.Active()
		})
	}
}

func (self *EditDialog) AddButton(name string, clickCallback func()) {
	button := gtk.NewButtonWithLabel(name)
	button.ConnectClicked(clickCallback)
	self.buttonRow.Append(button)
}

type DialogBoolRow struct {
	*gtk.Box
	state *gtk.CheckButton
}

func NewDialogBoolRow(title string, value bool) (self *DialogBoolRow) {
	self = &DialogBoolRow{
		Box:   gtk.NewBox(gtk.OrientationHorizontal, 0),
		state: nil,
	}
	self.Box.AddCSSClass("editDialogRow")

	titleLabel := gtk.NewLabel(title)
	titleLabel.SetHExpand(true)
	titleLabel.SetHAlign(gtk.AlignStart)
	self.Append(titleLabel)

	switchButton := gtk.NewCheckButton()
	switchButton.SetActive(value)
	self.Append(switchButton)
	self.state = switchButton

	return
}

func (self *DialogBoolRow) ConnectChanged(callback func()) {
	self.state.ConnectToggled(callback)
}

type DialogTimeRow struct {
	*gtk.Box
	hours *TypedEntry[int]
	mins  *TypedEntry[int]
}

func NewDialogTimeRow(title string, value time.Duration) *DialogTimeRow {
	row := gtk.NewBox(gtk.OrientationHorizontal, 0)
	row.AddCSSClass("editDialogRow")
	entryHour := NewTypedEntry(int(value.Hours()))
	entryMins := NewTypedEntry(int(value.Minutes()) % 60)

	entryHour.AddCSSClass("entryEditDialog")
	entryMins.AddCSSClass("entryEditDialog")
	entryHour.SetMaxWidthChars(4)
	entryMins.SetMaxWidthChars(2)
	entryHour.SetAlignment(1)
	entryMins.SetAlignment(1)
	entryMins.SetMaxLength(2)

	titleLabel := gtk.NewLabel(title)
	titleLabel.SetHExpand(true)
	titleLabel.SetHAlign(gtk.AlignStart)
	row.Append(titleLabel)
	row.Append(entryHour)
	row.Append(gtk.NewLabel(" h :   "))
	row.Append(entryMins)
	row.Append(gtk.NewLabel(" m "))

	return &DialogTimeRow{row, entryHour, entryMins}
}

func (self *DialogTimeRow) ConnectChanged(f func()) {
	self.hours.ConnectChanged(f)
	self.mins.ConnectChanged(f)
}

type DialogRow[T EntryType] struct {
	*gtk.Box
	entry *TypedEntry[T]
}

func NewDialogRow[T EntryType](title string, value T) *DialogRow[T] {
	row := gtk.NewBox(gtk.OrientationHorizontal, 0)
	row.AddCSSClass("editDialogRow")
	entry := NewTypedEntry(value)
	entry.SetAlignment(1)
	entry.AddCSSClass("entryEditDialog")
	entry.SetMaxWidthChars(14)

	titleLabel := gtk.NewLabel(title)
	titleLabel.SetHExpand(true)
	titleLabel.SetHAlign(gtk.AlignStart)
	row.Append(titleLabel)
	row.Append(entry)

	return &DialogRow[T]{row, entry}
}

type EntryType interface {
	int | ~string
}

type TypedEntry[T EntryType] struct {
	*gtk.Entry
}

func NewTypedEntry[T EntryType](value T) *TypedEntry[T] {
	entry := gtk.NewEntry()

	switch reflect.TypeOf(value).Kind() {
	case reflect.Int:
		entry.ConnectChanged(func() {
			text := entry.Text()
			if _, err := strconv.Atoi(text); err != nil && text != "" {
				glib.IdleAdd(func() {
					entry.DeleteText(len(text)-1, len(text))
				})
			}
		})

	}

	this := &TypedEntry[T]{entry}
	this.SetValue(value)

	return &TypedEntry[T]{entry}
}

func (self *TypedEntry[T]) SetValue(value T) {
	self.SetText(fmt.Sprint(value))
}

func (self *TypedEntry[int]) Int() int64 {
	input := self.Entry.Text()
	value, _ := strconv.Atoi(input)
	return int64(value)
}

type SaveStrategy string

const (
	Binary SaveStrategy = "Binary"
	JSON                = "JSON"
)

type SaveFileHandler struct {
	filePath     string
	CounterData  []*Counter
	SettingsData *settings.Settings

	strategy SaveStrategy
}

func NewSaveFileHandler(path string, strategy SaveStrategy) *SaveFileHandler {
	return &SaveFileHandler{
		path,
		nil,
		nil,
		strategy,
	}
}

func (self *SaveFileHandler) Save() (err error) {
	var saveData []byte
	if saveData, err = json.Marshal(self); err != nil {
		return
	}
	os.WriteFile(self.filePath, saveData, 0666)
	return
}

func (self *SaveFileHandler) Restore() (err error) {
	var saveData []byte
	if saveData, err = os.ReadFile(self.filePath); err != nil {
		log.Fatal("[FATAL] Could not Read save file, Got Error: ", err)
		return
	}

	err = json.Unmarshal(saveData, self)
	if err != nil {
		self.SettingsData = settings.NewSettings()
		log.Println("[WARN] Could not Unmarshal SaveData, Got Error: ", err)
		return
	}

	if self.SettingsData == nil {
		self.SettingsData = settings.NewSettings()
	}

	return
}
