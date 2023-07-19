package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strconv"
	"tallyGo/input"
	"time"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

const FRAME_TIME = time.Millisecond * 33

func main() {
	app := gtk.NewApplication("com.github.p3rtang.counter", gio.ApplicationFlagsNone)
	app.ConnectActivate(func() { activate(app) })

	if code := app.Run(os.Args); code > 0 {
		os.Exit(code)
	}
}

func activate(app *gtk.Application) (err error) {
	window := newHomeApplicationWindow(app)
	window.Show()
	return
}

func getMainLabel() *gtk.Label {
	builder := gtk.NewBuilderFromFile("counter.ui")
	return builder.GetObject("labelCounter").Cast().(*gtk.Label)
}

type HomeApplicationWindow struct {
	*gtk.Window
	treeViewRevealer     *gtk.Revealer
	isRevealerAutoHidden bool
	collapseButton       *gtk.Button
	headerBar            *gtk.ActionBar
}

func newHomeApplicationWindow(app *gtk.Application) (this HomeApplicationWindow) {
	this = HomeApplicationWindow{gtk.NewWindow(), nil, false, nil, nil}
	this.SetTitle("Counter")

	mainGrid := gtk.NewGrid()

	counterLabel := newCounterLabel()

	saveDataHandler := NewSaveFileHandler("save.sav")
	saveDataHandler.Restore()
	counters := NewCounterList(saveDataHandler.CounterData)
	app.ConnectShutdown(func() {
		saveDataHandler.Save()
	})

	counterTV := newCounterTreeView(counters, counterLabel)
	revealer := gtk.NewRevealer()
	revealer.SetTransitionType(gtk.RevealerTransitionTypeSlideRight)
	revealer.SetChild(counterTV)
	revealer.SetRevealChild(true)
	mainGrid.Attach(revealer, 0, 1, 1, 3)
	this.treeViewRevealer = revealer

	header := gtk.NewActionBar()
	collapseButton := gtk.NewButtonFromIconName("open-menu-symbolic")
	collapseButton.ConnectClicked(func() {
		if revealer.RevealChild() {
			revealer.SetRevealChild(false)
		} else {
			revealer.SetRevealChild(true)
		}
	})
	this.collapseButton = collapseButton
	header.PackStart(collapseButton)
	mainGrid.Attach(header, 0, 0, 2, 1)
	this.headerBar = header

	this.SetChild(mainGrid)
	this.SetDefaultSize(900, 600)
	this.Object.NotifyProperty("default-width", this.HandleNotify)
	css := gtk.NewCSSProvider()
	css.LoadFromPath("counter.css")
	gtk.StyleContextAddProviderForDisplay(gdk.DisplayGetDefault(), css, gtk.STYLE_PROVIDER_PRIORITY_SETTINGS)

	mainGrid.Attach(counterLabel.labelCount, 1, 1, 1, 1)
	mainGrid.Attach(counterLabel.labelTime, 1, 2, 1, 1)

	progressBar := gtk.NewProgressBar()
	progressBar.SetFraction(0.4)
	progressBar.SetText("0.4")
	progressBar.SetShowText(true)
	mainGrid.Attach(progressBar, 1, 3, 1, 1)

	app.AddWindow(this.Window)

	inputHandler := input.NewDevInput()
	inputHandler.Init()

	inputHandler.ConnectKey(input.KeyEqual, input.KeyReleased, func() { counterLabel.IncreaseBy(1) })
	inputHandler.ConnectKey(input.KeyKeypadPlus, input.KeyReleased, func() { counterLabel.IncreaseBy(1) })
	inputHandler.ConnectKey(input.KeyMinus, input.KeyReleased, func() { counterLabel.IncreaseBy(-1) })
	inputHandler.ConnectKey(input.KeyKeypadMinus, input.KeyReleased, func() { counterLabel.IncreaseBy(-1) })
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

func (self *CounterList) ConnectChange(type_ callBackType, f func()) {
	for _, counter := range self.list {
		counter.ConnectChange(type_, f)
	}
}

type Countable interface {
	IncreaseBy(add int)
	AddTime(time time.Duration)
	GetTime() time.Duration
	GetCount() int

	ConnectChange(type_ callBackType, f func())
}

type Counter struct {
	Name     string
	Phases   []*Phase
	progress *Progress
}

func newCounter(name string, initValue int, progressType ProgressType) (counter *Counter) {
	counter = &Counter{name, []*Phase{}, newProgress(progressType)}
	counter.NewPhase()
	return
}

func (self *Counter) NewPhase() *Phase {
	phaseName := fmt.Sprintf("Phase_%d", len(self.Phases)+1)
	newPhase := &Phase{
		phaseName,
		0,
		time.Duration(0),
		newProgress(self.progress.Type),
		make(map[callBackType][]func()),
	}
	self.Phases = append(self.Phases, newPhase)

	return newPhase
}

func (self *Counter) IncreaseBy(add int) {
	self.Phases[len(self.Phases)-1].IncreaseBy(add)
}

func (self *Counter) AddTime(time time.Duration) {
	self.Phases[len(self.Phases)-1].AddTime(time)
}

func (self *Counter) GetTime() (time time.Duration) {
	for _, phase := range self.Phases {
		time += phase.Time
	}
	return
}

func (self *Counter) GetCount() (count int) {
	for _, phase := range self.Phases {
		count += phase.Count
	}
	return
}

func (self *Counter) ConnectChange(type_ callBackType, f func()) {
	for _, phase := range self.Phases {
		phase.ConnectChange(type_, f)
	}
}

func (self *Counter) UpdateProgress() {
	for _, phase := range self.Phases {
		phase.UpdateProgress()
	}
}

type Phase struct {
	Name      string
	Count     int
	Time      time.Duration
	Progress  *Progress
	callbacks map[callBackType][]func()
}

func (self *Phase) IncreaseBy(add int) {
	self.Count += add
	self.callback(Count)
}

func (self *Phase) AddTime(time time.Duration) {
	self.Time += time
	self.callback(Time)
}

func (self *Phase) GetTime() time.Duration {
	return self.Time
}

func (self *Phase) GetCount() int {
	return self.Count
}

func (self *Phase) ConnectChange(type_ callBackType, f func()) {
	if self.callbacks == nil {
		self.callbacks = make(map[callBackType][]func())
	}
	self.callbacks[type_] = append(self.callbacks[type_], f)
}

func (self *Phase) callback(type_ callBackType) {
	for _, f := range self.callbacks[type_] {
		f()
	}
}

func (self *Phase) UpdateProgress() {
	switch self.Progress.Type {
	case OldOdds:
		self.Progress.Progress = math.Pow(float64(1.0-1.0/8192.0), float64(self.Count))
	case NewOdds:
		self.Progress.Progress = math.Pow(float64(1.0-1.0/4096.0), float64(self.Count))
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
	counters  []*CounterExpander
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

	tv.SetSizeRequest(300, 0)

	newCounterButton := gtk.NewButtonWithLabel("New Counter")
	newCounterButton.ConnectClicked(func() {
		counter := newCounter("Test", 0, OldOdds)
		this.addCounter(counter)
	})
	store.Append(newCounterButton.Object)

	for _, counter := range cList.list {
		this.addCounter(counter)
	}

	return
}

func (self *CounterTreeView) addCounter(counter *Counter) {
	counter.ConnectChange(Time, self.mainLabel.Update)
	counter.ConnectChange(Count, self.mainLabel.Update)
	expander := newCounterExpander(counter)
	self.counters = append(self.counters, expander)
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
		counter = getCounterExpander(row.Item(), self.counters).counter
		phaseNum = uint(len(counter.Phases))
		self.mainLabel.SetCounter(counter)
	case 1:
		parentRow := row.Parent()
		// exp := parentRow.Item().Cast().(*gtk.TreeExpander)
		counter = getCounterExpander(parentRow.Item(), self.counters).counter
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
	store := getCounterExpander(gObj, self.counters).store

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
	store   *gio.ListStore
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
		phase := counter.NewPhase()
		store.Append(newPhaseLabel(phase).Object)
	})

	box.Append(label)
	box.Append(button)

	expander.SetChild(box)

	for _, phase := range counter.Phases {
		label := newPhaseLabel(phase)
		store.Append(label.Object)
	}

	counterExpander := CounterExpander{
		expander,
		counter,
		store,
	}

	return &counterExpander
}

type PhaseLabel struct {
	*gtk.Label
	phase *Phase
}

func newPhaseLabel(phase *Phase) *PhaseLabel {
	if phase == nil {
		return nil
	}
	label := gtk.NewLabel(phase.Name)
	label.AddCSSClass("phaseLabel")
	label.SetHAlign(gtk.AlignStart)
	phaseLabel := PhaseLabel{
		label,
		nil,
	}
	return &phaseLabel
}

type LabelMainShowCount struct {
	labelCount *gtk.Label
	labelTime  *gtk.Label
	countable  Countable
}

func newCounterLabel() *LabelMainShowCount {
	labelCount := gtk.NewLabel("---")
	labelCount.SetHExpand(true)
	labelCount.SetVExpand(true)
	labelCount.AddCSSClass("labelMainCount")
	labelTime := gtk.NewLabel("--:--:--,---")
	labelTime.AddCSSClass("labelMainTime")
	cLabel := LabelMainShowCount{
		labelCount,
		labelTime,
		nil,
	}
	go func() {
		for {
			time.Sleep(FRAME_TIME)
			cLabel.AddTime(FRAME_TIME)
		}
	}()
	return &cLabel
}

func (self *LabelMainShowCount) IncreaseBy(add int) {
	if self.countable != nil {
		self.countable.IncreaseBy(add)
	}
}

func (self *LabelMainShowCount) SetCounter(countable Countable) {
	self.countable = countable
	self.Update()
}

func (self *LabelMainShowCount) Update() {
	if self.countable == nil {
		return
	}
	self.UpdateCount()
	self.UpdateTime()
}

func (self *LabelMainShowCount) UpdateCount() {
	if self.countable == nil {
		return
	}
	self.labelCount.SetText(self.String())
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
	}
}

func (self *LabelMainShowCount) String() string {
	if self.countable != nil {
		return fmt.Sprintf("%d", self.countable.GetCount())
	}
	return "---"
}

type NumericEntry struct {
	*gtk.Entry
}

func newNumericEntry(entry *gtk.Entry) NumericEntry {
	entry.ConnectChanged(func() {
		text := entry.Text()
		if _, err := strconv.Atoi(text); err != nil {
			glib.IdleAdd(func() {
				entry.DeleteText(len(text)-1, len(text))
			})
		}
	})

	return NumericEntry{entry}
}

func (self *NumericEntry) Int() (int, error) {
	input := self.Entry.Text()
	return strconv.Atoi(input)
}

type Settings struct{}

type SaveFileHandler struct {
	filePath     string
	CounterData  []*Counter
	SettingsData *Settings
}

func NewSaveFileHandler(path string) *SaveFileHandler {
	return &SaveFileHandler{
		path,
		nil,
		nil,
	}
}

func (self *SaveFileHandler) Save() (err error) {
	var saveData []byte
	if saveData, err = json.Marshal(self.CounterData); err != nil {
		return
	}
	os.WriteFile(self.filePath, saveData, 0666)
	return
}

func (self *SaveFileHandler) Restore() (err error) {
	var saveData []byte
	if saveData, err = os.ReadFile(self.filePath); err != nil {
		return
	}
	err = json.Unmarshal(saveData, &self.CounterData)
	return
}

type SaveData interface {
	SaveAsBytes() string
}
