package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"tallyGo/input"
	"time"

	_ "embed"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

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
}

func newHomeApplicationWindow(app *gtk.Application) (window HomeApplicationWindow) {
	builder := gtk.NewBuilderFromFile("counter.ui")
	window = HomeApplicationWindow{gtk.NewWindow()}
	window.SetTitle("Counter")

	mainGrid := gtk.NewGrid()

	counterLabel := newCounterLabel(nil, nil)
	button := builder.GetObject("buttonAddCounter").Cast().(*gtk.Button)
	addInput := newNumericEntry(builder.GetObject("entryNumericAddCounter").Cast().(*gtk.Entry))

	saveDataHandler := NewSaveFileHandler("save.sav")
	saveDataHandler.Restore()
	counters := saveDataHandler.CounterData

	counterTV := newCounterTreeView(counters, counterLabel)
	revealer := gtk.NewRevealer()
	revealer.SetTransitionType(gtk.RevealerTransitionTypeSlideRight)
	revealer.SetChild(counterTV)
	revealer.SetRevealChild(true)
	mainGrid.Attach(revealer, 0, 1, 1, 2)

	header := gtk.NewActionBar()
	collapseButton := gtk.NewButtonFromIconName("open-menu-symbolic")
	collapseButton.ConnectClicked(func() {
		if revealer.RevealChild() {
			revealer.SetRevealChild(false)
		} else {
			revealer.SetRevealChild(true)
		}
	})
	header.PackStart(collapseButton)
	mainGrid.Attach(header, 0, 0, 2, 1)

	button.ConnectClicked(func() {
		if num, err := addInput.Int(); err == nil {
			counterLabel.IncreaseBy(num)
		} else {
			counterLabel.IncreaseBy(1)
		}
	})

	window.SetChild(mainGrid)
	window.SetDefaultSize(960, 600)

	css := gtk.NewCSSProvider()
	css.LoadFromPath("counter.css")
	gtk.StyleContextAddProviderForDisplay(gdk.DisplayGetDefault(), css, gtk.STYLE_PROVIDER_PRIORITY_SETTINGS)

	mainGrid.Attach(counterLabel.labelCount, 1, 1, 1, 1)
	mainGrid.Attach(counterLabel.labelTime, 1, 2, 1, 1)

	app.AddWindow(window.Window)

	inputHandler := input.NewDevInput()
	inputHandler.Init()

	inputHandler.ConnectKey(
		input.KeyEqual,
		input.KeyReleased,
		func() {
			if num, err := addInput.Int(); err == nil {
				counterLabel.IncreaseBy(num)
			} else {
				counterLabel.IncreaseBy(1)
			}
		},
	)
	return
}

type Countable interface {
	IncreaseBy(add int)
	AddTime(time time.Duration)
}

type Counter struct {
	Name   string
	Phases []Phase
}

func newCounter(name string, initValue int) *Counter {
	return &Counter{
		name,
		[]Phase{{"Phase_1", initValue, time.Duration(0)}},
	}
}

func (self *Counter) NewPhase() *Phase {
	phaseName := fmt.Sprintf("Phase_%d", len(self.Phases)+1)
	newPhase := Phase{
		phaseName,
		0,
		time.Duration(0),
	}
	self.Phases = append(self.Phases, newPhase)

	return &newPhase
}

func (self *Counter) IncreaseBy(add int) {
	self.Phases[len(self.Phases)-1].Count += add
}

func (self *Counter) AddTime(time time.Duration) {
	self.Phases[len(self.Phases)-1].Time += time
}

func (self *Counter) Time() (time time.Duration) {
	for _, phase := range self.Phases {
		time += phase.Time
	}
	return
}

type Phase struct {
	Name  string
	Count int
	Time  time.Duration
}

func (self *Phase) IncreaseBy(add int) {
	self.Count += add
}

func (self *Phase) AddTime(time time.Duration) {
	self.Time += time
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
	store     *gio.ListStore
	selection *gtk.SingleSelection
	counters  []*CounterExpander
	mainLabel *LabelMainShowCount
}

func newCounterTreeView(cList []*Counter, mainLabel *LabelMainShowCount) (this CounterTreeView) {
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

	for _, counter := range cList {
		expander := newCounterExpander(counter)
		this.counters = append(this.counters, expander)
		store.Append(expander.Object)
		sep := gtk.NewSeparator(gtk.OrientationHorizontal)
		store.Append(sep.Object)
	}

	return
}

func (self *CounterTreeView) newSelection(position uint, nItems uint) {
	row := self.selection.Item(self.selection.Selected()).Cast().(*gtk.TreeListRow)
	var phaseNum uint
	var counter *Counter
	switch row.Depth() {
	case 0:
		exp := row.Item().Cast().(*gtk.TreeExpander)
		counter = getCounterFromExpander(exp, self.counters).counter
		phaseNum = uint(len(counter.Phases))
		self.mainLabel.SetCounter(counter)
	case 1:
		parentRow := row.Parent()
		exp := parentRow.Item().Cast().(*gtk.TreeExpander)
		counter = getCounterFromExpander(exp, self.counters).counter
		phaseNum = row.Position() - parentRow.Position()
		phase := &counter.Phases[phaseNum-1]
		self.mainLabel.SetPhase(phase)
	}
}

func (self *CounterTreeView) createTreeModel(gObj *glib.Object) *gio.ListModel {
	if gObj.Type().Name() != "GtkTreeExpander" {
		return nil
	}

	expander, _ := gObj.Cast().(*gtk.TreeExpander)
	store := getCounterFromExpander(expander, self.counters).store

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
	}
}

func getCounterFromExpander(expander *gtk.TreeExpander, counters []*CounterExpander) (counter *CounterExpander) {
	for _, c := range counters {
		if !(c.counter.Name == getStringFromExpander(expander)) {
			continue
		}
		counter = c
	}
	return
}

func getStringFromExpander(expander *gtk.TreeExpander) string {
	return expander.Child().(*gtk.Box).FirstChild().(*gtk.Label).Text()
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
		label := newPhaseLabel(&phase)
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
	*gtk.Box
	labelCount *gtk.Label
	labelTime  *gtk.Label
	//TODO: add countable interface here
	counter *Counter
	phase   *Phase
}

func newCounterLabel(counter *Counter, phase *Phase) *LabelMainShowCount {
	labelCount := gtk.NewLabel("---")
	labelCount.SetHExpand(true)
	labelCount.SetVExpand(true)
	labelCount.AddCSSClass("labelMainCount")
	labelTime := gtk.NewLabel("--:--:--,---")
	labelTime.AddCSSClass("labelMainTime")
	cLabel := LabelMainShowCount{
		nil,
		labelCount,
		labelTime,
		counter,
		phase,
	}
	go func() {
		for {
			frameTime := time.Millisecond * 33
			time.Sleep(frameTime)
			cLabel.labelCount.SetLabel(cLabel.String())
			cLabel.labelTime.SetLabel(cLabel.Time())
			cLabel.UpdateTime(frameTime)
		}
	}()
	return &cLabel
}

func (self *LabelMainShowCount) IncreaseBy(add int) {
	if self.phase != nil {
		self.phase.IncreaseBy(add)
	}
	if self.counter != nil {
		self.counter.Phases[len(self.counter.Phases)-1].IncreaseBy(add)
	}
	self.labelCount.SetText(self.String())
}

func (self *LabelMainShowCount) SetCounter(counter *Counter) {
	self.phase = nil
	self.counter = counter
}

func (self *LabelMainShowCount) SetPhase(phase *Phase) {
	self.counter = nil
	self.phase = phase
}

func (self *LabelMainShowCount) UpdateTime(time time.Duration) {
	self.labelTime.SetText(self.Time())
	if self.phase != nil {
		self.phase.AddTime(time)
	}
	if self.counter != nil {
		self.counter.AddTime(time)
	}
}

func (self *LabelMainShowCount) Time() string {
	var time time.Duration
	if self.phase != nil {
		time = self.phase.Time
	}
	if self.counter != nil {
		time = self.counter.Time()
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
	if self.phase != nil {
		self.phase.AddTime(time)
	}
	if self.counter != nil {
		self.counter.AddTime(time)
	}
}

func (self *LabelMainShowCount) String() string {
	if self.phase != nil {
		return fmt.Sprintf("%d", self.phase.Count)
	}
	if self.counter != nil {
		var sum int
		for _, phase := range self.counter.Phases {
			sum += phase.Count
		}
		return fmt.Sprintf("%d", sum)
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
