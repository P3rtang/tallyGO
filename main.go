package main

import (
	"example/hello/input"
	"fmt"
	"os"
	"strconv"
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
	counterLabel := newCounterLabel(getMainLabel(), nil, nil)
	button := builder.GetObject("buttonAddCounter").Cast().(*gtk.Button)
	addInput := newNumericEntry(builder.GetObject("entryNumericAddCounter").Cast().(*gtk.Entry))
	counters := []*Counter{
		newCounter("test1", 0),
		newCounter("test2", 0),
	}
	counterTV := newCounterTreeView(counters, counterLabel)
	mainGrid.Attach(counterTV, 0, 0, 1, 2)

	button.ConnectClicked(func() {
		if num, err := addInput.Int(); err == nil {
			counterLabel.IncreaseBy(num)
		} else {
			counterLabel.IncreaseBy(1)
		}
	})

	window.SetChild(mainGrid)
	window.SetDefaultSize(400, 300)

	css := gtk.NewCSSProvider()
	css.LoadFromPath("counter.css")
	gtk.StyleContextAddProviderForDisplay(gdk.DisplayGetDefault(), css, gtk.STYLE_PROVIDER_PRIORITY_SETTINGS)

	mainGrid.Attach(counterLabel.Label, 1, 0, 2, 1)
	mainGrid.Attach(button, 2, 1, 1, 1)
	mainGrid.Attach(addInput, 1, 1, 1, 1)

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

type Counter struct {
	name      string
	phaseList []Phase
}

func newCounter(name string, initValue int) *Counter {
	return &Counter{
		name,
		[]Phase{Phase{"Phase_1", initValue, time.Duration(0)}},
	}
}

func (self *Counter) NewPhase() *Phase {
	phaseName := fmt.Sprintf("Phase_%d", len(self.phaseList)+1)
	newPhase := Phase{
		phaseName,
		0,
		time.Duration(0),
	}
	self.phaseList = append(self.phaseList, newPhase)

	return &newPhase
}

type Phase struct {
	name  string
	count int
	time  time.Duration
}

func (self *Phase) IncreaseBy(add int) {
	self.count += 1
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
	mainLabel *CounterLabel
}

func newCounterTreeView(cList []*Counter, mainLabel *CounterLabel) (this CounterTreeView) {
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

	tv.SetSizeRequest(360, 0)

	for _, counter := range cList {
		expander := newCounterExpander(counter, &this)
		this.counters = append(this.counters, expander)
		store.Append(expander.Object)
		sep := gtk.NewSeparator(gtk.OrientationHorizontal)
		store.Append(sep.Object)
	}

	return
}

func (self *CounterTreeView) newSelection(position uint, nItems uint) {
	fmt.Printf("selected %d\n", self.selection.Selected())
	row := self.selection.Item(self.selection.Selected()).Cast().(*gtk.TreeListRow)
	var phaseNum uint
	var counter *Counter
	switch row.Depth() {
	case 0:
		exp := row.Item().Cast().(*gtk.TreeExpander)
		counter = getCounterFromExpander(exp, self.counters).counter
		phaseNum = uint(len(counter.phaseList))
		self.mainLabel.SetCounter(counter)
	case 1:
		parentRow := row.Parent()
		exp := parentRow.Item().Cast().(*gtk.TreeExpander)
		counter = getCounterFromExpander(exp, self.counters).counter
		phaseNum = row.Position() - parentRow.Position()
		self.mainLabel.SetPhase(&counter.phaseList[phaseNum-1])
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
		if c.TreeExpander == expander {
			continue
		}
		counter = c
	}
	return
}

type CounterExpander struct {
	*gtk.TreeExpander
	counter *Counter
	store   *gio.ListStore
}

func newCounterExpander(counter *Counter, cTreeView *CounterTreeView) *CounterExpander {
	if counter == nil {
		return nil
	}
	expander := gtk.NewTreeExpander()
	store := gio.NewListStore(glib.TypeObject)

	box := gtk.NewBox(gtk.OrientationHorizontal, 0)
	box.AddCSSClass("counterBoxRow")

	label := gtk.NewLabel(counter.name)
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

	for _, phase := range counter.phaseList {
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
	label := gtk.NewLabel(phase.name)
	label.AddCSSClass("phaseLabel")
	label.SetHAlign(gtk.AlignStart)
	phaseLabel := PhaseLabel{
		label,
		nil,
	}
	return &phaseLabel
}

type CounterLabel struct {
	*gtk.Label
	counter *Counter
	phase   *Phase
}

func newCounterLabel(label *gtk.Label, counter *Counter, phase *Phase) *CounterLabel {
	cLabel := CounterLabel{
		label,
		counter,
		phase,
	}
	return &cLabel
}

func (self *CounterLabel) IncreaseBy(add int) {
	if self.phase != nil {
		self.phase.IncreaseBy(add)
	}
	if self.counter != nil {
		self.counter.phaseList[len(self.counter.phaseList)-1].IncreaseBy(1)
	}
	self.Label.SetText(self.String())
}

func (self *CounterLabel) SetCounter(counter *Counter) {
	self.phase = nil
	self.counter = counter
	self.Label.SetLabel(self.String())
}

func (self *CounterLabel) SetPhase(phase *Phase) {
	self.counter = nil
	self.phase = phase
	self.Label.SetLabel(self.String())
}

func (self *CounterLabel) String() string {
	if self.phase != nil {
		return fmt.Sprintf("%d", self.phase.count)
	}
	if self.counter != nil {
		var sum int
		for _, phase := range self.counter.phaseList {
			sum += phase.count
		}
		return fmt.Sprintf("%d", sum)
	}
	return "None Selected"
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
