package main

import (
	_ "embed"
	"encoding/gob"
	"fmt"
	"math"
	"os"
	"reflect"
	"strconv"
	"tallyGo/input"
	"tallyGo/settings"
	"time"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

const FRAME_TIME = time.Millisecond * 33

//go:embed counter.css
var CSS_FILE string
var APP *gtk.Application

func main() {
	APP = gtk.NewApplication("com.github.p3rtang.counter", gio.ApplicationFlagsNone)
	APP.ConnectActivate(func() { activate(APP) })

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

	homeGrid     *gtk.Grid
	settingsGrid *settings.SettingsMenu

	treeViewRevealer     *gtk.Revealer
	isRevealerAutoHidden bool
	collapseButton       *gtk.Button
	headerBar            *gtk.HeaderBar
}

func newHomeApplicationWindow(app *gtk.Application) (this HomeApplicationWindow) {
	this.ApplicationWindow = gtk.NewApplicationWindow(app)
	this.SetTitle("Counter")

	savePath, _ := os.UserHomeDir()
	savePath += "/.local/share/tallyGo/ProgramData"
	saveDataHandler := NewSaveFileHandler(savePath)
	saveDataHandler.Restore()

	appSettings := saveDataHandler.SettingsData

	this.homeGrid = gtk.NewGrid()
	this.settingsGrid = settings.NewSettingsMenu(appSettings)
	this.settingsGrid.AddItem(settings.Keyboard)

	counterLabel := newCounterLabel()

	counters := NewCounterList(saveDataHandler.CounterData)
	app.ConnectShutdown(func() {
		saveDataHandler.CounterData = counters.list
		saveDataHandler.Save()
	})

	counterTV := newCounterTreeView(counters, counterLabel)
	scrollView := gtk.NewScrolledWindow()
	scrollView.SetChild(counterTV)
	scrollView.SetName("treeViewScrollWindow")
	revealer := gtk.NewRevealer()
	revealer.SetTransitionType(gtk.RevealerTransitionTypeSlideRight)
	revealer.SetChild(scrollView)
	revealer.SetRevealChild(true)
	this.homeGrid.Attach(revealer, 0, 1, 1, 3)
	this.treeViewRevealer = revealer

	header := gtk.NewHeaderBar()
	collapseButton := gtk.NewButtonFromIconName("open-menu-symbolic")
	collapseButton.ConnectClicked(func() {
		if revealer.RevealChild() {
			revealer.SetRevealChild(false)
		} else {
			revealer.SetRevealChild(true)
		}
	})

	settingsButton := gtk.NewToggleButton()
	settingsButton.SetIconName("settings-svgrepo-com")
	settingsButton.SetName("settingsButton")
	image := settingsButton.Child().(*gtk.Image)
	image.SetPixelSize(24)
	settingsButton.ConnectToggled(func() {
		if settingsButton.Active() {
			this.Window.SetChild(this.settingsGrid)
		} else {
			this.Window.SetChild(this.homeGrid)
		}
	})

	header.PackStart(collapseButton)
	header.PackEnd(settingsButton)

	this.Window.SetTitlebar(header)
	this.collapseButton = collapseButton
	this.headerBar = header

	this.SetChild(this.homeGrid)
	this.SetDefaultSize(900, 600)
	this.Object.NotifyProperty("default-width", this.HandleNotify)

	css := gtk.NewCSSProvider()
	css.LoadFromData(CSS_FILE)
	gtk.StyleContextAddProviderForDisplay(gdk.DisplayGetDefault(), css, gtk.STYLE_PROVIDER_PRIORITY_SETTINGS)

	this.homeGrid.Attach(counterLabel.labelCount, 1, 1, 1, 1)
	this.homeGrid.Attach(counterLabel.labelTime, 1, 2, 1, 1)
	this.homeGrid.Attach(counterLabel.progressBar, 1, 3, 1, 1)

	inputHandler := input.NewDevInput()
	inputHandler.Init(appSettings.GetValue(settings.ActiveKeyboard).(string))
	appSettings.ConnectChanged(settings.ActiveKeyboard, func(value interface{}) {
		inputHandler.ChangeFile(value.(string))
	})

	eventController := gtk.NewEventControllerKey()
	this.Window.AddController(eventController)
	eventController.ConnectKeyReleased(func(keyval uint, _ uint, _ gdk.ModifierType) {
		key := input.KeyType(uint16(keyval))
		inputHandler.SimulateKey(key, 1)
	})

	inputHandler.ConnectKey(input.KeyEqual, input.KeyReleased, input.DevInputEvent, func() {
		if counterLabel.isPaused {
			return
		}
		counterLabel.IncreaseBy(1)
		saveDataHandler.Save()
	})
	inputHandler.ConnectKey(input.KeyKeypadPlus, input.KeyReleased, input.DevInputEvent, func() {
		if counterLabel.isPaused {
			return
		}
		counterLabel.IncreaseBy(1)
		saveDataHandler.Save()
	})
	inputHandler.ConnectKey(input.KeyMinus, input.KeyReleased, input.DevInputEvent, func() {
		if counterLabel.isPaused {
			return
		}
		counterLabel.IncreaseBy(-1)
		saveDataHandler.Save()
	})
	inputHandler.ConnectKey(input.KeyKeypadMinus, input.KeyReleased, input.DevInputEvent, func() {
		if counterLabel.isPaused {
			return
		}
		counterLabel.IncreaseBy(-1)
		saveDataHandler.Save()
	})
	inputHandler.ConnectKey(112, input.KeyReleased, input.WindowEvent, func() {
		counterLabel.isPaused = !counterLabel.isPaused
	})
	inputHandler.ConnectKey(input.KeyQ, input.KeyReleased, input.DevInputEvent, func() {
		counterLabel.isPaused = true
	})
	return
}

func (self *HomeApplicationWindow) HandleNotify() {
	if self.Width() < 600 {
		if self.treeViewRevealer.RevealChild() {
			self.treeViewRevealer.SetRevealChild(false)
			self.isRevealerAutoHidden = true
		}
		self.collapseButton.SetSensitive(false)
	} else if self.Width() > 600 {
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

type CounterList struct {
	list []*Counter
}

func NewCounterList(list []*Counter) *CounterList {
	return &CounterList{list}
}

func (self *CounterList) Remove(counter *Counter) {
	if idx, ok := self.GetIdx(counter); ok {
		if idx < len(self.list)-1 {
			self.list = append(self.list[:idx], self.list[idx+1:len(self.list)]...)

		} else {
			self.list = self.list[:idx]
		}
	}
}

func (self *CounterList) GetIdx(counter *Counter) (int, bool) {
	for idx, c := range self.list {
		if c == counter {
			return idx, true
		}
	}
	return 0, false
}

type Countable interface {
	SetName(name string)

	GetCount() int
	SetCount(num int)
	IncreaseBy(add int)

	AddTime(time time.Duration)
	SetTime(time time.Duration)
	GetTime() time.Duration

	GetProgress() float64
	GetProgressType() ProgressType

	ConnectChanged(field string, f func())
	callback(field string)
}

type Counter struct {
	Name         string
	Phases       []*Phase
	ProgressType ProgressType

	callbackChange map[string][]func()
}

func newCounter(name string, _ int, progressType ProgressType) (counter *Counter) {
	counter = &Counter{name, []*Phase{}, progressType, nil}
	counter.NewPhase(progressType)
	return
}

func (self *Counter) ConnectChanged(field string, f func()) {
	if self.callbackChange == nil {
		self.callbackChange = make(map[string][]func())
	}
	if field == "Name" {
		self.callbackChange[field] = append(self.callbackChange[field], f)
	} else {
		for _, phase := range self.Phases {
			phase.ConnectChanged(field, f)
		}
	}
}

func (self *Counter) SetName(name string) {
	self.Name = name
	self.callback("Name")
}

func (self *Counter) NewPhase(progressType ProgressType) *Phase {
	phaseName := fmt.Sprintf("Phase_%d", len(self.Phases)+1)
	newPhase := &Phase{
		phaseName,
		0,
		time.Duration(0),
		newProgress(progressType),
		false,
		make(map[string][]func()),
	}
	self.Phases = append(self.Phases, newPhase)

	return newPhase
}

func (self *Counter) GetCount() (count int) {
	for _, phase := range self.Phases {
		count += phase.Count
	}
	return
}

func (self *Counter) SetCount(num int) {
	diff := num - self.GetCount()
	self.Phases[len(self.Phases)-1].IncreaseBy(diff)
}

func (self *Counter) IncreaseBy(add int) {
	self.Phases[len(self.Phases)-1].IncreaseBy(add)
	self.UpdateProgress()
}

func (self *Counter) GetTime() (time time.Duration) {
	for _, phase := range self.Phases {
		time += phase.Time
	}
	return
}

func (self *Counter) SetTime(time time.Duration) {
	diff := self.GetTime() - time
	self.Phases[len(self.Phases)-1].AddTime(diff)
}

func (self *Counter) AddTime(time time.Duration) {
	self.Phases[len(self.Phases)-1].AddTime(time)
}

func (self *Counter) GetProgress() (progress float64) {
	for _, phase := range self.Phases {
		progress += phase.Progress.Progress
	}
	return
}

func (self *Counter) GetProgressType() ProgressType {
	return self.Phases[0].Progress.Type
}

func (self *Counter) SetProgressType(type_ ProgressType) {
	self.ProgressType = type_
}

func (self *Counter) UpdateProgress() {
	for _, phase := range self.Phases {
		phase.UpdateProgress()
	}
}

func (self *Counter) callback(field string) {
	if field == "Name" {
		for _, f := range self.callbackChange[field] {
			f()
		}
	} else {
		for _, p := range self.Phases {
			p.callback(field)
		}
	}
}

type Phase struct {
	Name     string
	Count    int
	Time     time.Duration
	Progress *Progress
	IsLocked bool

	callbackChange map[string][]func()
}

func (self *Phase) ConnectChanged(field string, f func()) {
	if self.callbackChange == nil {
		self.callbackChange = make(map[string][]func())
	}
	self.callbackChange[field] = append(self.callbackChange[field], f)
}

func (self *Phase) SetName(name string) {
	self.Name = name
	self.callback("Name")
}

func (self *Phase) GetCount() int {
	return self.Count
}

func (self *Phase) SetCount(num int) {
	self.Count = num
	self.UpdateProgress()
	self.callback("Count")
}

func (self *Phase) IncreaseBy(add int) {
	if self.IsLocked {
		return
	}
	self.Count += add
	self.UpdateProgress()
	self.callback("Count")
}

func (self *Phase) GetTime() time.Duration {
	return self.Time
}

func (self *Phase) SetTime(time time.Duration) {
	self.Time = time
	self.callback("Time")
}

func (self *Phase) AddTime(time time.Duration) {
	if self.IsLocked {
		return
	}
	self.Time += time
	self.callback("Time")
}

func (self *Phase) GetProgress() float64 {
	return self.Progress.Progress
}

func (self *Phase) GetProgressType() ProgressType {
	return self.Progress.Type
}

func (self *Phase) SetProgressType(type_ ProgressType) {
	self.Progress.Type = type_
}

func (self *Phase) UpdateProgress() {
	switch self.Progress.Type {
	case OldOdds:
		self.Progress.Progress = math.Pow(float64(1.0-1.0/8192.0), float64(self.Count))
	case NewOdds:
		self.Progress.Progress = math.Pow(float64(1.0-1.0/4096.0), float64(self.Count))
	}
}

func (self *Phase) callback(field string) {
	for _, f := range self.callbackChange[field] {
		f()
	}
}

type callBackType int

const (
	Count callBackType = iota
	Time
)

type Progress struct {
	Type     ProgressType
	Progress float64
}

func newProgress(type_ ProgressType) *Progress {
	return &Progress{
		type_,
		0.0,
	}
}

type ProgressType int

const (
	OldOdds ProgressType = iota
	NewOdds
	SOS
	DexNav
)

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
	store     *gio.ListStore
	selection *gtk.SingleSelection
	expanders []*CounterExpander
	mainLabel *LabelMainShowCount
}

func newCounterTreeView(cList *CounterList, mainLabel *LabelMainShowCount) (this CounterTreeView) {
	store := gio.NewListStore(glib.TypeObject)
	var expanderList []*CounterExpander

	this = CounterTreeView{nil, store, nil, expanderList, mainLabel}

	treeStore := gtk.NewTreeListModel(store, false, true, this.createTreeModel)
	ssel := gtk.NewSingleSelection(treeStore)
	tv := gtk.NewListView(ssel, nil)
	tv.AddCSSClass("counterTreeView")

	this.ListView = tv
	this.selection = ssel

	factory := gtk.NewSignalListItemFactory()
	factory.ConnectSetup(createRow)
	factory.ConnectBind(bindRow)
	tv.SetFactory(&factory.ListItemFactory)

	ssel.SetAutoselect(false)
	ssel.ConnectSelectionChanged(this.newSelection)

	newCounterButton := gtk.NewButtonWithLabel("New Counter")
	newCounterButton.ConnectClicked(func() {
		counter := newCounter("Test", 0, OldOdds)
		this.addCounter(counter, cList)

		cList.list = append(cList.list, counter)
	})

	store.Append(newCounterButton.Object)

	for _, counter := range cList.list {
		this.addCounter(counter, cList)
	}

	return
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
	self.expanders = append(self.expanders, expander)

	items := self.store.NItems()
	self.store.Insert(items-1, expander.Object)
	sep := gtk.NewSeparator(gtk.OrientationHorizontal)
	self.store.Insert(items, sep.Object)
}

func (self *CounterTreeView) newSelection(position uint, nItems uint) {
	row := self.selection.Item(self.selection.Selected()).Cast().(*gtk.TreeListRow)
	var phaseNum uint
	var counter *Counter
	switch row.Depth() {
	case 0:
		// exp := row.Item().Cast().(*gtk.TreeExpander)
		counter = getCounterExpander(row.Item(), self.expanders).counter
		phaseNum = uint(len(counter.Phases))
		self.mainLabel.SetCounter(counter)
	case 1:
		parentRow := row.Parent()
		// exp := parentRow.Item().Cast().(*gtk.TreeExpander)
		counter = getCounterExpander(parentRow.Item(), self.expanders).counter
		phaseNum = row.Position() - parentRow.Position()
		phase := counter.Phases[phaseNum-1]
		self.mainLabel.SetCounter(phase)
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

func createRow(listItem *gtk.ListItem) {
	label := gtk.NewLabel("")
	listItem.SetChild(label)
}

func bindRow(listItem *gtk.ListItem) {
	row := listItem.Item().Cast().(*gtk.TreeListRow)
	switch row.Item().Type().Name() {
	case "GtkTreeExpander":

		expander := row.Item().Cast().(*gtk.TreeExpander)

		expander.SetListRow(row)
		listItem.SetChild(expander)
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

func newCounterExpander(counter *Counter) *CounterExpander {
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
		store.Append(newPhaseRow(phase).Object)
	})

	box.Append(label)
	box.Append(button)
	expander.SetChild(box)

	contextMenu := newTreeRowContextMenu()
	contextMenu.NewRow("edit")
	contextMenu.NewRow("delete")
	contextMenu.SetParent(&box.Widget)

	for _, phase := range counter.Phases {
		label := newPhaseRow(phase)
		store.Append(label.Object)
	}

	counterExpander := CounterExpander{expander, counter, store, contextMenu}
	counter.ConnectChanged("Name", func() {
		label.SetText(counter.Name)
	})

	return &counterExpander
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
		f()
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

func newPhaseRow(phase *Phase) *PhaseRow {
	if phase == nil {
		return nil
	}

	box := gtk.NewBox(gtk.OrientationHorizontal, 0)
	box.AddCSSClass("phaseRow")

	label := gtk.NewLabel(phase.Name)
	label.SetHAlign(gtk.AlignStart)

	phase.ConnectChanged("Name", func() {
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

	phaseLabel := PhaseRow{
		box,
		label,
		lock,
		contextMenu,
		phase,
	}
	phaseLabel.UpdateLock()

	contextMenu.ConnectRowClick("edit", func() {
		dialog := NewEditDialog(phase)
		dialog.Present()
	})

	contextMenu.ConnectRowClick("lock", func() {
		phase.IsLocked = !phase.IsLocked
		phaseLabel.UpdateLock()
	})

	return &phaseLabel
}

func (self *PhaseRow) UpdateLock() {
	label := self.contextMenu.rows["lock"]

	if !self.phase.IsLocked {
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
		this.NewRow("HuntType", fmt.Sprint(phase.Progress.Type))
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
	}
}

func (self *EditDialog) AddButton(name string, clickCallback func()) {
	button := gtk.NewButtonWithLabel(name)
	button.ConnectClicked(clickCallback)
	self.buttonRow.Append(button)
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

type LabelMainShowCount struct {
	labelCount  *gtk.Label
	labelTime   *gtk.Label
	progressBar *gtk.ProgressBar
	countable   Countable
	isPaused    bool
}

func newCounterLabel() *LabelMainShowCount {
	labelCount := gtk.NewLabel("---")
	labelCount.SetHExpand(true)
	labelCount.SetVExpand(true)
	labelCount.AddCSSClass("labelMainCount")

	labelTime := gtk.NewLabel("--:--:--,---")
	labelTime.AddCSSClass("labelMainTime")

	progressBar := gtk.NewProgressBar()
	progressBar.SetShowText(true)

	this := LabelMainShowCount{
		labelCount,
		labelTime,
		progressBar,
		nil,
		true,
	}

	go func() {
		for {
			time.Sleep(FRAME_TIME)
			if !this.isPaused {
				glib.IdleAdd(func() {
					this.countable.AddTime(FRAME_TIME)
				})
			}
		}
	}()
	return &this
}

func (self *LabelMainShowCount) IncreaseBy(add int) {
	if self.countable != nil && self.isPaused != true {
		self.countable.IncreaseBy(add)
	}
}

func (self *LabelMainShowCount) SetCounter(countable Countable) {
	self.countable = countable
	self.countable.ConnectChanged("Count", self.UpdateCount)
	self.countable.ConnectChanged("Time", self.UpdateTime)
	self.UpdateCount()
	self.UpdateTime()
}

func (self *LabelMainShowCount) UpdateCount() {
	if self.countable == nil {
		return
	}
	self.labelCount.SetText(self.String())
	fraction := 1.0 - self.countable.GetProgress()

	self.progressBar.SetFraction(fraction)
	self.progressBar.SetText(fmt.Sprintf("%.03f%%", fraction*100))

	self.progressBar.RemoveCSSClass("progressGreen")
	self.progressBar.RemoveCSSClass("progressYellow")
	self.progressBar.RemoveCSSClass("progressOrange")
	self.progressBar.RemoveCSSClass("progressRed")

	var odds int
	switch self.countable.GetProgressType() {
	case OldOdds:
		odds = 8192
	case NewOdds:
		odds = 4096
	}

	switch {
	case fraction < .4:
		self.progressBar.AddCSSClass("progressGreen")
		break
	case self.countable.GetCount() < odds && odds != 0:
		self.progressBar.AddCSSClass("progressYellow")
		break
	case fraction < .75:
		self.progressBar.AddCSSClass("progressOrange")
		break
	case fraction < 1.0:
		self.progressBar.AddCSSClass("progressRed")
		break
	}
}

func (self *LabelMainShowCount) UpdateTime() {
	if self.countable == nil {
		return
	}
	self.labelTime.SetText(self.Time())
}

func (self *LabelMainShowCount) Time() string {
	var time time.Duration
	if self.countable != nil {
		time = self.countable.GetTime()
	}
	return fmt.Sprintf(
		"%d:%02d:%02d,%03d",
		int(time.Hours()),
		int(time.Minutes())%60,
		int(time.Seconds())%60,
		time.Milliseconds()%1000,
	)
}

func (self *LabelMainShowCount) AddTime(time time.Duration) {
	if self.countable != nil {
		self.countable.AddTime(time)
		self.UpdateTime()
	}
}

func (self *LabelMainShowCount) String() string {
	if self.countable != nil {
		return fmt.Sprintf("%d", self.countable.GetCount())
	}
	return "---"
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

type SaveFileHandler struct {
	filePath     string
	CounterData  []*Counter
	SettingsData *settings.Settings
}

func NewSaveFileHandler(path string) *SaveFileHandler {
	return &SaveFileHandler{
		path,
		nil,
		nil,
	}
}

func (self *SaveFileHandler) Save() (err error) {
	file, err := os.OpenFile(self.filePath, os.O_RDWR, 0666)
	encoder := gob.NewEncoder(file)
	if err = encoder.Encode(self); err != nil {
		fmt.Printf("Could not save program got error: %v\n", err)
	}
	return
}

func (self *SaveFileHandler) Restore() (err error) {
	file, err := os.Open(self.filePath)
	decoder := gob.NewDecoder(file)
	decoder.Decode(self)
	if self.SettingsData == nil {
		self.SettingsData = settings.NewSettings()
	}
	return
}
