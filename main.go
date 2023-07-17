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

type HomeApplicationWindow struct {
	*gtk.Window
}

func newHomeApplicationWindow(app *gtk.Application) (window HomeApplicationWindow) {
	builder := gtk.NewBuilderFromFile("counter.ui")
	window = HomeApplicationWindow{gtk.NewWindow()}
	window.SetTitle("Counter")

	mainGrid := gtk.NewGrid()
	counterLabel := newCounterLabel(builder.GetObject("labelCounter").Cast().(*gtk.Label))
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
	gtk.StyleContextAddProviderForDisplay(gdk.DisplayGetDefault(), css, 0)

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
	selection *gtk.SingleSelection
	counters  []*Counter
	mainLabel *CounterLabel
}

func newCounterTreeView(cList []*Counter, mainLabel *CounterLabel) (this CounterTreeView) {
	store := gio.NewListStore(glib.TypeObject)
	this = CounterTreeView{nil, nil, cList, mainLabel}

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

	ssel.ConnectSelectionChanged(this.newSelection)

	tv.SetSizeRequest(360, 0)

	for _, counter := range cList {
		expander := newCounterExpander(counter)
		store.Append(expander.Object)
	}

	return
}

func (self CounterTreeView) newSelection(position uint, nItems uint) {
	fmt.Printf("selected %d\n", self.selection.Selected())
	row := self.selection.Item(self.selection.Selected()).Cast().(*gtk.TreeListRow)
	switch row.Depth() {
	case 0:
		exp := row.Item().Cast().(*gtk.TreeExpander)
		lbl := exp.Child().(*gtk.Label)
		for _, counter := range self.counters {
			if counter.name != lbl.Text() {
				continue
			}
			self.mainLabel.ChangePhase(&counter.phaseList[len(counter.phaseList)-1])
		}
	case 1:
		parentRow := row.Parent()
		exp := parentRow.Item().Cast().(*gtk.TreeExpander)
		lbl := exp.Child().(*gtk.Label)
		for _, counter := range self.counters {
			if counter.name != lbl.Text() {
				continue
			}
			phaseNum := row.Position() - parentRow.Position()
			self.mainLabel.ChangePhase(&counter.phaseList[phaseNum-1])
		}
	}

}

func (self CounterTreeView) createTreeModel(gObj *glib.Object) *gio.ListModel {
	if gObj.Type().Name() != "GtkTreeExpander" {
		return nil
	}
	expander, _ := gObj.Cast().(*gtk.TreeExpander)

	store := gio.NewListStore(glib.TypeObject)
	for _, counter := range self.counters {
		if !(counter.name == expander.Child().(*gtk.Label).Text()) {
			continue
		}
		for _, phase := range counter.phaseList {
			label := newPhaseLabel(&phase)
			store.Append(label.Object)
		}
	}

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
	}
}

type CounterExpander struct {
	*gtk.TreeExpander
	counter *Counter
}

func newCounterExpander(counter *Counter) *CounterExpander {
	if counter == nil {
		return nil
	}
	expander := gtk.NewTreeExpander()
	label := gtk.NewLabel(counter.name)
	expander.SetChild(label)
	counterExpander := CounterExpander{
		expander,
		counter,
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
	phase *Phase
}

func newCounterLabel(label *gtk.Label) *CounterLabel {
	cLabel := CounterLabel{
		label,
		nil,
	}
	return &cLabel
}

func (self *CounterLabel) IncreaseBy(add int) {
	if self.phase == nil {
		return
	}
	self.phase.IncreaseBy(add)
	self.Label.SetText(self.String())
}

func (self *CounterLabel) ChangePhase(phase *Phase) {
	self.phase = phase
	self.SetText(self.String())
}

func (self *CounterLabel) String() string {
	if self.phase == nil {
		return "None Selected"
	}
	return fmt.Sprintf("%d", self.phase.count)
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
