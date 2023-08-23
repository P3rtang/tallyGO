package infobox

import (
	"fmt"
	"log"
	"time"

	. "tallyGo/countable"
	EventBus "tallyGo/eventBus"
	"tallyGo/treeview"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"golang.org/x/exp/slices"
)

type widgetType string

const (
	None              widgetType = ""
	WidgetRevealer               = "WidgetRevealer"
	MainCount                    = "MainCount"
	MainTime                     = "MainTime"
	TimeCountCombined            = "TimeCountCombined"
	ProgressBar                  = "ProgressBar"
	StepTime                     = "StepTime"
	LastStepTime                 = "LastStepTime"
	OverallLuck                  = "OverallLuck"
)

type infoBoxWidget interface {
	setCounter(countable Countable)
	setBorder(isShown bool)
	setTitle(set bool)
	setExpand(set bool)
	connectRevealer(revealer *widgetRevealer)
	addCSSClass(name string)
	removeCSSClass(name string)
}

type InfoBox struct {
	*gtk.Box

	counterList *CounterList

	widgets        map[widgetType]infoBoxWidget
	widgetRevealer *widgetRevealer
	isExpanded     bool
}

func NewInfoBox(counterList *CounterList) (self *InfoBox) {
	self = &InfoBox{
		gtk.NewBox(gtk.OrientationVertical, 0),
		counterList,
		map[widgetType]infoBoxWidget{},
		nil,
		true,
	}

	self.widgetRevealer = newRevealWidget(self.getWidgetSlice())
	self.AddCSSClass("infoBox")
	self.SetVExpand(true)

	EventBus.GetGlobalBus().Subscribe("LayoutChanged", func(...interface{}) {
		self.handleResize()
	})

	self.isExpanded = true
	self.Append(self.widgetRevealer)
	self.SetWidgets([]widgetType{
		MainCount + MainTime,
		ProgressBar,
		StepTime + LastStepTime,
		OverallLuck,
	}, None, true, true)

	EventBus.GetGlobalBus().Subscribe(ListActiveChanged, func(args ...interface{}) {
		if len(args) == 0 {
			self.SetCounter(nil)
		} else {
			self.SetCounter(args[0].(Countable))
		}
	})

	return
}

func (self *InfoBox) GetWidget() *gtk.Widget {
	return &self.Widget
}

func (self *InfoBox) HeaderBar() *gtk.HeaderBar {
	headerBar := gtk.NewHeaderBar()
	backButton := gtk.NewButtonFromIconName("go-previous-symbolic")
	backButton.ConnectClicked(func() {
		EventBus.GetGlobalBus().SendSignal(treeview.DeselectAll)
	})
	headerBar.PackStart(backButton)
	return headerBar
}

func (self *InfoBox) getWidgetSlice() (list []infoBoxWidget) {
	for _, widget := range self.widgets {
		list = append(list, widget)
	}
	return
}

func (self *InfoBox) handleResize() {
	switch {
	case self.Parent().(*gtk.Viewport).Parent().(*gtk.ScrolledWindow).Width() < 560 && self.isExpanded:
		self.isExpanded = false
		self.SetWidgets([]widgetType{
			MainCount,
			MainTime,
			ProgressBar,
			StepTime,
			LastStepTime,
			OverallLuck,
		}, None, true, false)
	case self.Parent().(*gtk.Viewport).Parent().(*gtk.ScrolledWindow).Width() > 560 && !self.isExpanded:
		self.isExpanded = true
		self.SetWidgets([]widgetType{
			MainCount + MainTime,
			ProgressBar,
			StepTime + LastStepTime,
			OverallLuck,
		}, None, true, true)
	}
}

func (self *InfoBox) SetWidgets(widgets []widgetType, setExpand widgetType, showBorder bool, showTitle bool) {
	child := self.LastChild()
	for self.ObserveChildren().NItems() > 0 {
		self.Box.Remove(child)
		child = self.LastChild()
	}

	self.widgets = map[widgetType]infoBoxWidget{}
	self.Append(self.widgetRevealer)

	for _, widget := range widgets {
		self.AddWidget(widget, showBorder)
	}

	if self.widgets[setExpand] != nil {
		self.widgets[setExpand].setExpand(true)
	}

	if self.counterList.HasActive() {
		self.SetCounter(self.counterList.GetActive()[0])
	}

	self.setBorder(showBorder)
	self.setTitle(showTitle)
	self.setRevealer()
}

func (self *InfoBox) setRevealer() {
	for _, widget := range self.widgets {
		widget.connectRevealer(self.widgetRevealer)
	}
}

func (self *InfoBox) AddWidget(type_ widgetType, showBorder bool) {
	countable := Countable(nil)
	if self.counterList.HasActive() {
		countable = self.counterList.GetActive()[0]
	}
	switch type_ {
	case MainCount + MainTime:
		box := newWidgetBox(false)
		box.Append(MainCount, countable)
		box.Append(MainTime, countable)
		self.Box.Append(box)
		self.widgets[MainCount+MainTime] = box
	case MainCount:
		mainCountLabel := newMainCountLabel()
		self.Box.Append(mainCountLabel)
		self.widgets[MainCount] = mainCountLabel
	case MainTime:
		mainTimeLabel := newMainTimeLabel()
		self.Box.Append(mainTimeLabel)
		self.widgets[MainTime] = mainTimeLabel
	case ProgressBar:
		mainProgressBar := newMainProgressBar()
		self.Box.Append(mainProgressBar)
		self.widgets[ProgressBar] = mainProgressBar
	case StepTime + LastStepTime:
		box := newWidgetBox(false)
		box.Append(StepTime, countable)
		box.Append(LastStepTime, countable)
		self.Box.Append(box)
		self.widgets[StepTime+LastStepTime] = box
	case StepTime:
		stepTimeLabel := newStepTime()
		self.Box.Append(stepTimeLabel)
		self.widgets[StepTime] = stepTimeLabel
	case LastStepTime:
		lastStepTime := newLastStepTime()
		self.Box.Append(lastStepTime)
		self.widgets[LastStepTime] = lastStepTime
	case OverallLuck:
		overallLuck := newOverallLuck(self.counterList)
		self.Box.Append(overallLuck)
		self.widgets[OverallLuck] = overallLuck

	default:
		log.Fatal("Unrecognized widget combination")
	}

	self.setBorder(showBorder)
}

func (self *InfoBox) SetCounter(countable Countable) {
	for _, widget := range self.widgets {
		widget.setCounter(countable)
	}

	self.widgetRevealer.setCounter(countable)
}

func (self *InfoBox) setBorder(isShown bool) {
	for _, widget := range self.widgets {
		widget.setBorder(isShown)
	}
}

func (self *InfoBox) setTitle(showTitle bool) {
	for _, widget := range self.widgets {
		widget.setTitle(showTitle)
	}
}

func (self *InfoBox) toggleCSSClass(name string) {
	if slices.Contains(self.CSSClasses(), name) {
		self.RemoveCSSClass(name)
	} else {
		self.AddCSSClass(name)
	}
}

type widgetRevealer struct {
	*gtk.Revealer

	widgetList     []infoBoxWidget
	selectedWidget infoBoxWidget
	widgetType     widgetType
	countable      Countable

	callbacks map[string][]func()
}

func newRevealWidget(widgetList []infoBoxWidget) (self *widgetRevealer) {
	self = &widgetRevealer{
		gtk.NewRevealer(),
		widgetList,
		infoBoxWidget(nil),
		None,
		nil,
		map[string][]func(){},
	}
	self.SetRevealChild(false)
	self.SetTransitionType(gtk.RevealerTransitionTypeSlideUp)
	self.AddCSSClass("WidgetRevealer")
	self.SetHExpand(true)
	return
}

func (self *widgetRevealer) setCounter(countable Countable) {
	self.countable = countable
	if self.selectedWidget != infoBoxWidget(nil) {
		self.selectedWidget.setCounter(self.countable)
	}
}

func (self *widgetRevealer) setWidget(widget widgetType) {
	for _, f := range self.callbacks["ChangeWidget"] {
		f()
	}

	self.widgetType = widget

	box := gtk.NewBox(gtk.OrientationVertical, 0)
	self.SetChild(box)
	label := gtk.NewLabel("")
	label.SetName("RevealerTitle")
	box.Append(label)

	switch widget {
	case None:
		self.SetRevealChild(false)
		self.SetVExpand(false)
		self.selectedWidget = infoBoxWidget(nil)
		return
	case MainCount:
		mainCountLabel := newMainCountLabel()
		mainCountLabel.setCounter(self.countable)
		self.selectedWidget = mainCountLabel
		label.SetText("Count")
		box.Append(mainCountLabel)
	case MainTime:
		mainTimeLabel := newMainTimeLabel()
		mainTimeLabel.setCounter(self.countable)
		self.selectedWidget = mainTimeLabel
		label.SetText("Time")
		box.Append(mainTimeLabel)
	case ProgressBar:
		mainProgressBar := newMainProgressBar()
		mainProgressBar.setCounter(self.countable)
		self.selectedWidget = mainProgressBar
		label.SetText("Progress")
		box.Append(mainProgressBar)
	case StepTime:
		stepTimeLabel := newStepTime()
		stepTimeLabel.setCounter(self.countable)
		self.selectedWidget = stepTimeLabel
		label.SetText("Time per Step")
		box.Append(stepTimeLabel)
	case LastStepTime:
		lastStep := newLastStepTime()
		lastStep.setCounter(self.countable)
		self.selectedWidget = lastStep
		label.SetText("Last Step")
		box.Append(lastStep)
	}
	self.SetRevealChild(true)
}

func (self *widgetRevealer) ConnectChanged(name string, f func()) {
	callbackNames := []string{"ChangeWidget"}
	if !slices.Contains(callbackNames, name) {
		log.Printf("Unrecognized callback Name %s\n", name)
		log.Printf("Name should be one of the following %v", callbackNames)
		return
	}
	self.callbacks[name] = append(self.callbacks[name], f)
}

func (self *widgetRevealer) callback(name string) {
	for _, f := range self.callbacks[name] {
		f()
	}
}

type WidgetBox struct {
	*gtk.Box
	widgets []infoBoxWidget

	showDivider bool
}

func newWidgetBox(showDivider bool) (self *WidgetBox) {
	self = &WidgetBox{gtk.NewBox(gtk.OrientationHorizontal, 0), []infoBoxWidget{}, showDivider}
	return
}

func (self *WidgetBox) Append(widget widgetType, countable Countable) {
	if self.Box.ObserveChildren().NItems() > 0 && self.showDivider {
		self.Box.Append(gtk.NewSeparator(gtk.OrientationVertical))
	}
	switch widget {
	case MainCount:
		mainCountLabel := newMainCountLabel()
		mainCountLabel.SetHExpand(true)
		self.Box.Append(mainCountLabel)
		self.widgets = append(self.widgets, mainCountLabel)
		break
	case MainTime:
		mainTimeLabel := newMainTimeLabel()
		mainTimeLabel.SetHExpand(true)
		self.Box.Append(mainTimeLabel)
		self.widgets = append(self.widgets, mainTimeLabel)
		break
	case ProgressBar:
		mainProgressBar := newMainProgressBar()
		mainProgressBar.SetHExpand(true)
		self.Box.Append(mainProgressBar)
		self.widgets = append(self.widgets, mainProgressBar)
		break
	case StepTime:
		stepTimeLabel := newStepTime()
		stepTimeLabel.SetHExpand(true)
		self.Box.Append(stepTimeLabel)
		self.widgets = append(self.widgets, stepTimeLabel)
		break
	case LastStepTime:
		lastStep := newLastStepTime()
		lastStep.SetHExpand(true)
		self.Box.Append(lastStep)
		self.widgets = append(self.widgets, lastStep)
		break
	}
	self.FirstChild().(*gtk.Box).SetHExpand(true)
}

func (self *WidgetBox) setCounter(countable Countable) {
	for _, widget := range self.widgets {
		widget.setCounter(countable)
	}
}

func (self *WidgetBox) setBorder(set bool) {
	for _, widget := range self.widgets {
		widget.setBorder(set)
	}
}

func (self *WidgetBox) setTitle(set bool) {
	for _, widget := range self.widgets {
		widget.setTitle(set)
	}
}

func (self *WidgetBox) setExpand(set bool) {
	for _, widget := range self.widgets {
		widget.setExpand(set)
	}
}

func (self *WidgetBox) connectRevealer(revealer *widgetRevealer) {
	for _, widget := range self.widgets {
		widget.connectRevealer(revealer)
	}
}

func (self *WidgetBox) addCSSClass(name string) {
	self.AddCSSClass(name)
}

func (self *WidgetBox) removeCSSClass(name string) {
	self.RemoveCSSClass(name)
}

type countLabel struct {
	*gtk.Box
	countable  Countable
	labelTitle *gtk.Label
	labelCount *gtk.Label
}

func newMainCountLabel() (self *countLabel) {
	self = &countLabel{
		gtk.NewBox(gtk.OrientationHorizontal, 0),
		Countable(nil),
		gtk.NewLabel("Count"),
		gtk.NewLabel("---"),
	}

	self.Box.AddCSSClass("infoBoxRow")
	self.Box.Append(self.labelTitle)
	self.Box.Append(self.labelCount)

	self.labelTitle.SetVisible(false)
	self.labelTitle.SetName("title")
	self.labelCount.SetHExpand(true)

	EventBus.GetGlobalBus().Subscribe(CountChanged, self.UpdateCount)

	return
}

func (self *countLabel) IncreaseBy(add int) {
	if self.countable != Countable(nil) {
		self.countable.IncreaseBy(add)
	}
}

func (self *countLabel) setCounter(countable Countable) {
	self.countable = countable
	self.UpdateCount()
}

func (self *countLabel) UpdateCount(...interface{}) {
	self.labelCount.SetText(self.String())
}

func (self *countLabel) String() string {
	if self.countable != nil {
		return fmt.Sprintf("%d", self.countable.GetCount())
	}
	return "---"
}

func (self *countLabel) setBorder(setShown bool) {
	if setShown {
		self.AddCSSClass("infoBoxShowBackground")
	} else {
		self.RemoveCSSClass("infoBoxShowBackground")
	}
}

func (self *countLabel) setTitle(set bool) {
	self.labelTitle.SetVisible(set)
}

func (self *countLabel) setExpand(set bool) {
	self.labelCount.SetVExpand(set)
	if set {
		self.labelCount.AddCSSClass("expandWidget")
	} else {
		self.labelCount.RemoveCSSClass("expandWidget")
	}
}

func (self *countLabel) connectRevealer(revealer *widgetRevealer) {
	clickController := gtk.NewGestureClick()
	clickController.ConnectPressed(func(_ int, _ float64, _ float64) {
		if revealer.widgetType == MainCount {
			revealer.setWidget(None)
		} else {
			self.AddCSSClass("selected")
			revealer.setWidget(MainCount)
			revealer.ConnectChanged("ChangeWidget", func() {
				if revealer.widgetType == MainCount {
					self.removeCSSClass("selected")
				}
			})
		}
	})
	self.AddController(clickController)
}

func (self *countLabel) addCSSClass(name string) {
	self.AddCSSClass(name)
}

func (self *countLabel) removeCSSClass(name string) {
	self.RemoveCSSClass(name)
}

type timeLabel struct {
	*gtk.Box

	labelTitle *gtk.Label
	labelTime  *gtk.Label

	countable Countable
	isPaused  bool
}

func newMainTimeLabel() (self *timeLabel) {
	self = &timeLabel{
		gtk.NewBox(gtk.OrientationHorizontal, 0),
		gtk.NewLabel("Time"),
		gtk.NewLabel("---"),
		Countable(nil),
		true,
	}

	self.Box.AddCSSClass("infoBoxRow")
	self.Box.Append(self.labelTitle)
	self.Box.Append(self.labelTime)

	self.labelTitle.SetVisible(false)
	self.labelTitle.SetName("title")
	self.labelTime.SetHExpand(true)
	self.labelTime.AddCSSClass("longTimeLabel")

	EventBus.GetGlobalBus().Subscribe(TimeChanged, self.UpdateTime)

	return
}

func (self *timeLabel) IncreaseBy(add int) {
	if self.countable != nil {
		self.countable.IncreaseBy(add)
		self.isPaused = false
	}
}

func (self *timeLabel) setCounter(countable Countable) {
	self.countable = countable
	self.UpdateTime()
}

func (self *timeLabel) UpdateTime(...interface{}) {
	self.labelTime.SetText(self.Time())
}

func (self *timeLabel) Time() string {
	if self.countable != nil {
		time := self.countable.GetTime()
		return fmt.Sprintf(
			"%d:%02d:%02d,%03d",
			int(time.Hours()),
			int(time.Minutes())%60,
			int(time.Seconds())%60,
			time.Milliseconds()%1000,
		)
	} else {
		return "---"
	}
}

func (self *timeLabel) AddTime(time time.Duration) {
	if self.countable != nil {
		self.countable.AddTime(time)
		self.UpdateTime()
	}
}

func (self *timeLabel) String() string {
	if self.countable != nil {
		return fmt.Sprintf("%d", self.countable.GetCount())
	}
	return "---"
}

func (self *timeLabel) setBorder(setShown bool) {
	if setShown {
		self.AddCSSClass("infoBoxShowBackground")
	} else {
		self.RemoveCSSClass("infoBoxShowBackground")
	}
}

func (self *timeLabel) setTitle(set bool) {
	self.labelTitle.SetVisible(set)
}

func (self *timeLabel) setExpand(set bool) {
	self.labelTime.SetVExpand(set)
	if set {
		self.labelTime.AddCSSClass("expandWidget")
	} else {
		self.labelTime.RemoveCSSClass("expandWidget")
	}
}

func (self *timeLabel) connectRevealer(revealer *widgetRevealer) {
	clickController := gtk.NewGestureClick()
	clickController.ConnectPressed(func(_ int, _ float64, _ float64) {
		if revealer.widgetType == MainTime {
			revealer.setWidget(None)
		} else {
			self.AddCSSClass("selected")
			revealer.setWidget(MainTime)
			revealer.ConnectChanged("ChangeWidget", func() {
				if revealer.widgetType == MainTime {
					self.removeCSSClass("selected")
				}
			})
		}
	})
	self.AddController(clickController)
}

func (self *timeLabel) addCSSClass(name string) {
	self.AddCSSClass(name)
}

func (self *timeLabel) removeCSSClass(name string) {
	self.RemoveCSSClass(name)
}

type mainProgressBar struct {
	*gtk.Box
	countable   Countable
	title       *gtk.Label
	progressBar *gtk.ProgressBar
}

func newMainProgressBar() (self *mainProgressBar) {
	self = &mainProgressBar{
		Box:         gtk.NewBox(gtk.OrientationHorizontal, 0),
		countable:   Countable(nil),
		title:       gtk.NewLabel("Progress"),
		progressBar: gtk.NewProgressBar(),
	}

	self.title.SetVisible(false)
	self.title.SetName("title")

	self.progressBar.SetShowText(true)
	self.progressBar.SetHExpand(true)
	self.progressBar.SetVAlign(gtk.AlignCenter)
	self.progressBar.AddCSSClass("infoBoxRowProgressBar")
	self.progressBar.SetFraction(0)
	self.progressBar.SetText("---")

	self.Box.Append(self.title)
	self.Box.Append(self.progressBar)
	self.Box.AddCSSClass("infoBoxRow")

	EventBus.GetGlobalBus().Subscribe(TimeChanged, self.Update)

	return self
}

func (self *mainProgressBar) setCounter(countable Countable) {
	self.countable = countable
	self.Update()
}

func (self *mainProgressBar) Update(...interface{}) {
	if self.countable == Countable(nil) {
		self.progressBar.SetFraction(0)
		self.progressBar.SetText("---")
		return
	}
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
	case fraction < .5:
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

func (self *mainProgressBar) setBorder(setShown bool) {
	if setShown {
		self.AddCSSClass("infoBoxShowBackground")
	} else {
		self.RemoveCSSClass("infoBoxShowBackground")
	}
}

func (self *mainProgressBar) setTitle(set bool) {
	if set {
		self.title.SetVisible(true)
	} else {
		self.title.SetVisible(false)
	}
}

func (self *mainProgressBar) setExpand(set bool) {
	self.progressBar.SetVExpand(set)
	if set {
		self.progressBar.AddCSSClass("expandWidget")
	} else {
		self.progressBar.RemoveCSSClass("expandWidget")
	}
}

func (self *mainProgressBar) connectRevealer(revealer *widgetRevealer) {
	clickController := gtk.NewGestureClick()
	clickController.ConnectPressed(func(_ int, _ float64, _ float64) {
		if revealer.widgetType == ProgressBar {
			revealer.setWidget(None)
		} else {
			self.AddCSSClass("selected")
			revealer.setWidget(ProgressBar)
			revealer.ConnectChanged("ChangeWidget", func() {
				if revealer.widgetType == ProgressBar {
					self.removeCSSClass("selected")
				}
			})
		}
	})
	self.AddController(clickController)
}

func (self *mainProgressBar) addCSSClass(name string) {
	self.AddCSSClass(name)
}

func (self *mainProgressBar) removeCSSClass(name string) {
	self.RemoveCSSClass(name)
}

type stepTime struct {
	*gtk.Box
	countable Countable
	title     *gtk.Label
	label     *gtk.Label
}

func newStepTime() (self *stepTime) {
	self = &stepTime{
		gtk.NewBox(gtk.OrientationHorizontal, 0),
		Countable(nil),
		gtk.NewLabel("Time per Step"),
		gtk.NewLabel("---"),
	}
	self.title.SetName("title")
	self.title.SetVisible(false)
	self.label.AddCSSClass("shortTimeLabel")
	self.label.SetHExpand(true)
	self.Box.Append(self.title)
	self.Box.Append(self.label)
	self.Box.AddCSSClass("infoBoxRow")

	EventBus.GetGlobalBus().Subscribe(CountChanged, self.Update)

	return
}

func (self *stepTime) setCounter(countable Countable) {
	if countable == Countable(nil) {
		return
	}
	self.countable = countable
}

func (self *stepTime) Update(...interface{}) {
	var stepTime time.Duration
	if self.countable.GetCount() != 0 {
		stepTime = time.Duration(int(self.countable.GetTime())/self.countable.GetCount() + 1)
	} else {
		stepTime = 0
	}
	self.label.SetText(shortFormatTime(stepTime))
}

func (self *stepTime) setBorder(setShown bool) {
	if setShown {
		self.AddCSSClass("infoBoxShowBackground")
	} else {
		self.RemoveCSSClass("infoBoxShowBackground")
	}
}

func (self *stepTime) setTitle(set bool) {
	if set {
		self.title.SetVisible(true)
	} else {
		self.title.SetVisible(false)
	}
}

func (self *stepTime) setExpand(set bool) {
	self.label.SetVExpand(set)
	if set {
		self.label.AddCSSClass("expandWidget")
	} else {
		self.label.RemoveCSSClass("expandWidget")
	}
}

func (self *stepTime) connectRevealer(revealer *widgetRevealer) {
	clickController := gtk.NewGestureClick()
	clickController.ConnectPressed(func(_ int, _ float64, _ float64) {
		if revealer.widgetType == StepTime {
			revealer.setWidget(None)
		} else {
			self.AddCSSClass("selected")
			revealer.setWidget(StepTime)
			revealer.ConnectChanged("ChangeWidget", func() {
				if revealer.widgetType == StepTime {
					self.removeCSSClass("selected")
				}
			})
		}
	})
	self.AddController(clickController)
}

func (self *stepTime) addCSSClass(name string) {
	self.AddCSSClass(name)
}

func (self *stepTime) removeCSSClass(name string) {
	self.RemoveCSSClass(name)
}

type lastStepTime struct {
	*gtk.Box

	countable Countable
	lastTime  time.Duration

	title *gtk.Label
	label *gtk.Label
}

func newLastStepTime() (self *lastStepTime) {
	self = &lastStepTime{
		gtk.NewBox(gtk.OrientationHorizontal, 0),
		Countable(nil),
		-1,
		gtk.NewLabel("Last Step"),
		gtk.NewLabel("---"),
	}

	self.Box.AddCSSClass("infoBoxRow")
	self.Box.Append(self.title)
	self.Box.Append(self.label)
	self.title.SetName("title")
	self.title.SetVisible(false)
	self.label.AddCSSClass("shortTimeLabel")
	self.label.SetHExpand(true)

	EventBus.GetGlobalBus().Subscribe(CountChanged, self.Update)

	return
}

func (self *lastStepTime) setCounter(countable Countable) {
	self.countable = countable
	if self.countable == Countable(nil) {
		self.lastTime = 0
		return
	}
}

func (self *lastStepTime) Update(...interface{}) {
	if self.lastTime == -1 {
		self.lastTime = self.countable.GetTime()
		return
	}
	self.label.SetText(shortFormatTime(self.countable.GetTime() - self.lastTime))
	self.lastTime = self.countable.GetTime()
}

func (self *lastStepTime) setBorder(setShown bool) {
	if setShown {
		self.Box.AddCSSClass("infoBoxShowBackground")
	} else {
		self.Box.RemoveCSSClass("infoBoxShowBackground")
	}
}

func (self *lastStepTime) setTitle(set bool) {
	self.title.SetVisible(set)
}

func (self *lastStepTime) setExpand(set bool) {
	self.label.SetVExpand(set)
	if set {
		self.label.AddCSSClass("expandWidget")
	} else {
		self.label.RemoveCSSClass("expandWidget")
	}
}

func (self *lastStepTime) connectRevealer(revealer *widgetRevealer) {
	clickController := gtk.NewGestureClick()
	clickController.ConnectPressed(func(_ int, _ float64, _ float64) {
		if revealer.widgetType == LastStepTime {
			revealer.setWidget(None)
		} else {
			self.AddCSSClass("selected")
			revealer.setWidget(LastStepTime)
			revealer.ConnectChanged("ChangeWidget", func() {
				if revealer.widgetType == LastStepTime {
					self.removeCSSClass("selected")
				}
			})
		}
	})
	self.AddController(clickController)
}

func (self *lastStepTime) addCSSClass(name string) {
	self.AddCSSClass(name)
}

func (self *lastStepTime) removeCSSClass(name string) {
	self.RemoveCSSClass(name)
}

type overallLuck struct {
	*gtk.Box

	list *CounterList

	title *gtk.Label
	luck  *gtk.ProgressBar
}

func newOverallLuck(list *CounterList) (self *overallLuck) {
	self = &overallLuck{
		gtk.NewBox(gtk.OrientationHorizontal, 0),
		list,
		gtk.NewLabel("Overall Luck"),
		gtk.NewProgressBar(),
	}

	self.Box.Append(self.title)
	self.Box.Append(self.luck)
	self.Box.AddCSSClass("infoBoxRow")

	self.title.SetName("title")
	self.luck.SetVAlign(gtk.AlignCenter)
	self.setProgress()

	EventBus.GetGlobalBus().Subscribe(CountChanged, self.setProgress)

	return
}

func (self *overallLuck) setProgress(...interface{}) {
	luck := self.list.Luck()

	self.luck.SetFraction(luck)
	self.luck.SetText(fmt.Sprintf("%.03f%%", luck*100-50))
	self.luck.SetShowText(true)
	self.luck.SetHExpand(true)

	switch {
	case luck < 0.3:
		self.luck.AddCSSClass("progressRed")
		break
	case luck < 0.4:
		self.luck.AddCSSClass("progressOrange")
		break
	case luck < 0.5:
		self.luck.AddCSSClass("progressYellow")
		break
	default:
		self.luck.AddCSSClass("progressGreen")
	}

}

func (self *overallLuck) setCounter(countable Countable) {
	if countable == Countable(nil) {
		return
	}
	self.setProgress()
}

func (self *overallLuck) setBorder(setShown bool) {
	if setShown {
		self.Box.AddCSSClass("infoBoxShowBackground")
	} else {
		self.Box.RemoveCSSClass("infoBoxShowBackground")
	}
}

func (self *overallLuck) setTitle(set bool) {
	self.title.SetVisible(set)
}

func (self *overallLuck) setExpand(set bool) {
	self.luck.SetVExpand(set)
	if set {
		self.luck.AddCSSClass("expandWidget")
	} else {
		self.luck.RemoveCSSClass("expandWidget")
	}
}

func (self *overallLuck) connectRevealer(revealer *widgetRevealer) {
	clickController := gtk.NewGestureClick()
	clickController.ConnectPressed(func(_ int, _ float64, _ float64) {
		if revealer.widgetType == OverallLuck {
			revealer.setWidget(None)
		} else {
			self.AddCSSClass("selected")
			revealer.setWidget(OverallLuck)
			revealer.ConnectChanged("ChangeWidget", func() {
				if revealer.widgetType == OverallLuck {
					self.removeCSSClass("selected")
				}
			})
		}
	})
	self.AddController(clickController)
}

func formatTime(duration time.Duration) (format string) {
	format = fmt.Sprintf(
		"%02dh %02dm %02ds %03d",
		int(duration.Hours()),
		int(duration.Minutes())%60,
		int(duration.Seconds())%60,
		duration.Milliseconds()%1000,
	)

	return
}

func (self *overallLuck) addCSSClass(name string) {
	self.AddCSSClass(name)
}

func (self *overallLuck) removeCSSClass(name string) {
	self.RemoveCSSClass(name)
}

func shortFormatTime(duration time.Duration) (format string) {
	stepMins := duration.Minutes()
	switch {
	case stepMins < 1:
		format = fmt.Sprintf(
			"%02ds %03d",
			int(duration.Seconds()),
			duration.Milliseconds()%1000,
		)
	case stepMins < 60:
		format = fmt.Sprintf(
			"%02dm %02ds %03d",
			int(duration.Minutes()),
			int(duration.Seconds())%60,
			duration.Milliseconds()%1000,
		)
	default:
		format = fmt.Sprintf(
			"%02dh %02dm %02ds %03d",
			int(duration.Hours()),
			int(duration.Minutes())%60,
			int(duration.Seconds())%60,
			duration.Milliseconds()%1000,
		)
	}
	return
}
