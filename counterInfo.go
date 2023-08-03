package main

import (
	"fmt"
	"log"
	"reflect"
	"time"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

type widgetType string

const (
	MainCount         widgetType = "MainCount"
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
}

type infoBox struct {
	*gtk.Box

	countable   Countable
	counterList *CounterList
	widgets     map[widgetType]infoBoxWidget
	isExpanded  bool
}

func NewInfoBox(counter *Counter, counterList *CounterList) (self *infoBox) {
	self = &infoBox{
		gtk.NewBox(gtk.OrientationVertical, 0),
		counter,
		counterList,
		map[widgetType]infoBoxWidget{},
		false,
	}

	self.AddCSSClass("infoBox")
	self.SetVExpand(true)

	infoButton := gtk.NewButton()

	infoButton.SetIconName("info-circle-svgrepo-com")
	infoButton.Child().(*gtk.Image).SetPixelSize(24)
	infoButton.SetHAlign(gtk.AlignEnd)
	infoButton.SetVAlign(gtk.AlignStart)
	infoButton.SetName("infoToggleButton")

	infoButton.ConnectClicked(func() {
		self.isExpanded = !self.isExpanded
		if self.isExpanded {
			self.SetWidgets([]widgetType{
				MainCount + MainTime,
				ProgressBar,
				StepTime + LastStepTime,
				OverallLuck,
			}, "", true, true)
		} else {
			self.SetWidgets([]widgetType{
				MainCount,
				MainTime,
				ProgressBar,
			}, MainCount, false, false)
		}
	})

	HOME.overlay.AddOverlay(infoButton)

	self.SetWidgets([]widgetType{
		MainCount,
		MainTime,
		ProgressBar,
	}, MainCount, false, false)

	return
}

func (self *infoBox) SetWidgets(widgets []widgetType, setExpand widgetType, showBorder bool, showTitle bool) {
	child := self.FirstChild()
	for self.ObserveChildren().NItems() > 0 {
		self.Box.Remove(child)
		child = self.FirstChild()
	}
	self.widgets = map[widgetType]infoBoxWidget{}

	for _, widget := range widgets {
		self.AddWidget(widget, showBorder)
	}

	if self.widgets[setExpand] != nil {
		self.widgets[setExpand].setExpand(true)
	}

	self.SetCounter(self.countable)
	self.setBorder(showBorder)
	self.setTitle(showTitle)
}

func (self *infoBox) AddWidget(type_ widgetType, showBorder bool) {
	switch type_ {
	case MainCount + MainTime:
		box := newWidgetBox(false)
		box.Append(MainCount, self.countable)
		box.Append(MainTime, self.countable)
		self.Box.Append(box)
		self.widgets[MainCount+MainTime] = box
	case MainCount:
		mainCountLabel := newMainCountLabel(self.countable)
		self.Box.Append(mainCountLabel)
		self.widgets[MainCount] = mainCountLabel
		break
	case MainTime:
		mainTimeLabel := newMainTimeLabel(self.countable)
		self.Box.Append(mainTimeLabel)
		self.widgets[MainTime] = mainTimeLabel
		break
	case ProgressBar:
		mainProgressBar := newMainProgressBar(self.countable)
		self.Box.Append(mainProgressBar)
		self.widgets[ProgressBar] = mainProgressBar
		break
	case StepTime + LastStepTime:
		box := newWidgetBox(false)
		box.Append(StepTime, self.countable)
		box.Append(LastStepTime, self.countable)
		self.Box.Append(box)
		self.widgets[StepTime+LastStepTime] = box
		break
	case StepTime:
		stepTimeLabel := newStepTime(self.countable)
		self.Box.Append(stepTimeLabel)
		self.widgets[StepTime] = stepTimeLabel
		break
	case LastStepTime:
		lastStepTime := newLastStepTime(self.countable)
		self.Box.Append(lastStepTime)
		self.widgets[LastStepTime] = lastStepTime
		break
	case OverallLuck:
		overallLuck := newOverallLuck(self.counterList)
		self.Box.Append(overallLuck)
		self.widgets[OverallLuck] = overallLuck
		break

	default:
		log.Fatal("Unrecognized widget combination")
	}

	self.setBorder(showBorder)
}

func (self *infoBox) SetCounter(countable Countable) {
	self.countable = countable
	if !self.countable.IsNil() {
		println(countable.GetName())
	}
	for _, widget := range self.widgets {
		widget.setCounter(countable)
	}
}

func (self *infoBox) setBorder(isShown bool) {
	for _, widget := range self.widgets {
		widget.setBorder(isShown)
	}
}

func (self *infoBox) setTitle(showTitle bool) {
	for _, widget := range self.widgets {
		widget.setTitle(showTitle)
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
		mainCountLabel := newMainCountLabel(countable)
		mainCountLabel.SetHExpand(true)
		self.Box.Append(mainCountLabel)
		self.widgets = append(self.widgets, mainCountLabel)
		break
	case MainTime:
		mainTimeLabel := newMainTimeLabel(countable)
		mainTimeLabel.SetHExpand(true)
		self.Box.Append(mainTimeLabel)
		self.widgets = append(self.widgets, mainTimeLabel)
		break
	case ProgressBar:
		mainProgressBar := newMainProgressBar(countable)
		mainProgressBar.SetHExpand(true)
		self.Box.Append(mainProgressBar)
		self.widgets = append(self.widgets, mainProgressBar)
		break
	case StepTime:
		stepTimeLabel := newStepTime(countable)
		stepTimeLabel.SetHExpand(true)
		self.Box.Append(stepTimeLabel)
		self.widgets = append(self.widgets, stepTimeLabel)
		break
	case LastStepTime:
		lastStep := newLastStepTime(countable)
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

type MainCountLabel struct {
	*gtk.Box
	countable  Countable
	labelTitle *gtk.Label
	labelCount *gtk.Label
}

func newMainCountLabel(countable Countable) (self *MainCountLabel) {
	self = &MainCountLabel{
		gtk.NewBox(gtk.OrientationHorizontal, 0),
		countable,
		gtk.NewLabel("Count"),
		gtk.NewLabel("---"),
	}

	self.Box.AddCSSClass("infoBoxRow")
	self.Box.Append(self.labelTitle)
	self.Box.Append(self.labelCount)

	self.labelTitle.SetVisible(false)
	self.labelTitle.SetName("title")
	self.labelCount.SetHExpand(true)

	return
}

func (self *MainCountLabel) IncreaseBy(add int) {
	if !self.countable.IsNil() {
		self.countable.IncreaseBy(add)
	}
}

func (self *MainCountLabel) setCounter(countable Countable) {
	if countable.IsNil() {
		return
	}
	self.countable = countable
	self.countable.ConnectChanged("Count", reflect.TypeOf(self).String(), self.UpdateCount)
	self.UpdateCount()
}

func (self *MainCountLabel) UpdateCount() {
	if self.countable == nil {
		return
	}
	self.labelCount.SetText(self.String())
}

func (self *MainCountLabel) String() string {
	if self.countable != nil {
		return fmt.Sprintf("%d", self.countable.GetCount())
	}
	return "---"
}

func (self *MainCountLabel) setBorder(setShown bool) {
	if setShown {
		self.AddCSSClass("infoBoxShowBorder")
	} else {
		self.RemoveCSSClass("infoBoxShowBorder")
	}
}

func (self *MainCountLabel) setTitle(set bool) {
	self.labelTitle.SetVisible(set)
}

func (self *MainCountLabel) setExpand(set bool) {
	self.labelCount.SetVExpand(set)
	if set {
		self.labelCount.AddCSSClass("expandWidget")
	} else {
		self.labelCount.RemoveCSSClass("expandWidget")
	}
}

type mainTimeLabel struct {
	*gtk.Box

	labelTitle *gtk.Label
	labelTime  *gtk.Label

	countable Countable
	isPaused  bool
}

func newMainTimeLabel(countable Countable) (self *mainTimeLabel) {
	self = &mainTimeLabel{
		gtk.NewBox(gtk.OrientationHorizontal, 0),
		gtk.NewLabel("Time"),
		gtk.NewLabel("---"),
		countable,
		true,
	}

	self.Box.AddCSSClass("infoBoxRow")
	self.Box.Append(self.labelTitle)
	self.Box.Append(self.labelTime)

	self.labelTitle.SetVisible(false)
	self.labelTitle.SetName("title")
	self.labelTime.SetHExpand(true)
	self.labelTime.AddCSSClass("longTimeLabel")

	return
}

func (self *mainTimeLabel) IncreaseBy(add int) {
	if self.countable != nil {
		self.countable.IncreaseBy(add)
		self.isPaused = false
	}
}

func (self *mainTimeLabel) setCounter(countable Countable) {
	if countable.IsNil() {
		return
	}
	self.countable = countable
	self.countable.ConnectChanged("Time", reflect.TypeOf(self).Name(), self.UpdateTime)
	self.UpdateTime()
}

func (self *mainTimeLabel) UpdateTime() {
	if self.countable == nil {
		return
	}
	self.labelTime.SetText(self.Time())
}

func (self *mainTimeLabel) Time() string {
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

func (self *mainTimeLabel) AddTime(time time.Duration) {
	if self.countable != nil {
		self.countable.AddTime(time)
		self.UpdateTime()
	}
}

func (self *mainTimeLabel) String() string {
	if self.countable != nil {
		return fmt.Sprintf("%d", self.countable.GetCount())
	}
	return "---"
}

func (self *mainTimeLabel) setBorder(setShown bool) {
	if setShown {
		self.AddCSSClass("infoBoxShowBorder")
	} else {
		self.RemoveCSSClass("infoBoxShowBorder")
	}
}

func (self *mainTimeLabel) setTitle(set bool) {
	self.labelTitle.SetVisible(set)
}

func (self *mainTimeLabel) setExpand(set bool) {
	self.labelTime.SetVExpand(set)
	if set {
		self.labelTime.AddCSSClass("expandWidget")
	} else {
		self.labelTime.RemoveCSSClass("expandWidget")
	}
}

type mainProgressBar struct {
	*gtk.Box
	countable   Countable
	title       *gtk.Label
	progressBar *gtk.ProgressBar
}

func newMainProgressBar(countable Countable) (this *mainProgressBar) {
	this = &mainProgressBar{gtk.NewBox(gtk.OrientationHorizontal, 0), countable, nil, nil}
	title := gtk.NewLabel("Progress")
	title.SetVisible(false)
	progressBar := gtk.NewProgressBar()
	progressBar.SetShowText(true)
	progressBar.SetVAlign(gtk.AlignCenter)
	this.Box.AddCSSClass("infoBoxRow")
	progressBar.AddCSSClass("infoBoxRowProgressBar")

	this.title = title
	this.title.SetName("title")
	this.progressBar = progressBar
	this.progressBar.SetHExpand(true)

	this.Box.Append(this.title)
	this.Box.Append(this.progressBar)

	return this
}

func (self *mainProgressBar) setCounter(countable Countable) {
	if countable.IsNil() {
		return
	}
	self.countable = countable
	self.countable.ConnectChanged("Count", reflect.TypeOf(self).String(), self.UpdateCount)
	self.UpdateCount()
}

func (self *mainProgressBar) UpdateCount() {
	if self.countable.IsNil() {
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

func (self *mainProgressBar) setBorder(setShown bool) {
	if setShown {
		self.AddCSSClass("infoBoxShowBorder")
	} else {
		self.RemoveCSSClass("infoBoxShowBorder")
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

type labelStepTime struct {
	*gtk.Box
	countable Countable
	title     *gtk.Label
	label     *gtk.Label
}

func newStepTime(countable Countable) (this *labelStepTime) {
	this = &labelStepTime{nil, countable, nil, nil}
	this.Box = gtk.NewBox(gtk.OrientationHorizontal, 0)
	this.title = gtk.NewLabel("Time per Step")
	this.title.SetName("title")
	this.title.SetVisible(false)
	this.label = gtk.NewLabel("---")
	this.label.AddCSSClass("shortTimeLabel")
	this.Box.Append(this.title)
	this.Box.Append(this.label)
	this.Box.AddCSSClass("infoBoxRow")
	if countable.IsNil() {
		return
	}

	var stepTime time.Duration
	if countable.GetCount() != 0 {
		stepTime = time.Duration(int(countable.GetTime())/countable.GetCount() + 1)
	} else {
		stepTime = 0
	}
	this.label.SetText(shortFormatTime(stepTime))
	this.countable = countable
	return
}

func (self *labelStepTime) setCounter(countable Countable) {
	if countable.IsNil() {
		return
	}
	self.countable = countable

	var stepTime time.Duration
	if countable.GetCount() != 0 {
		stepTime = time.Duration(int(countable.GetTime())/countable.GetCount() + 1)
	} else {
		stepTime = 0
	}
	self.label.SetText(shortFormatTime(stepTime))

	countable.ConnectChanged("Count", reflect.TypeOf(self).String(), func() {
		var stepTime time.Duration
		if countable.GetCount() != 0 {
			stepTime = time.Duration(int(countable.GetTime())/countable.GetCount() + 1)
		} else {
			stepTime = 0
		}
		self.label.SetText(shortFormatTime(stepTime))
	})
}

func (self *labelStepTime) setBorder(setShown bool) {
	if setShown {
		self.AddCSSClass("infoBoxShowBorder")
	} else {
		self.RemoveCSSClass("infoBoxShowBorder")
	}
}

func (self *labelStepTime) setTitle(set bool) {
	if set {
		self.title.SetVisible(true)
	} else {
		self.title.SetVisible(false)
	}
}

func (self *labelStepTime) setExpand(set bool) {
	self.label.SetVExpand(set)
	if set {
		self.label.AddCSSClass("expandWidget")
	} else {
		self.label.RemoveCSSClass("expandWidget")
	}
}

type lastStepTime struct {
	*gtk.Box

	countable Countable
	lastTime  time.Duration

	title         *gtk.Label
	labelLastStep *gtk.Label
}

func newLastStepTime(countable Countable) (self *lastStepTime) {
	self = &lastStepTime{
		gtk.NewBox(gtk.OrientationHorizontal, 0),
		countable,
		-1,
		gtk.NewLabel("Last Step"),
		gtk.NewLabel("---"),
	}

	self.Box.AddCSSClass("infoBoxRow")
	self.Box.Append(self.title)
	self.Box.Append(self.labelLastStep)
	self.title.SetName("title")
	self.title.SetVisible(false)
	self.labelLastStep.AddCSSClass("shortTimeLabel")

	return
}

func (self *lastStepTime) setCounter(countable Countable) {
	self.countable = countable
	if self.countable.IsNil() {
		self.lastTime = 0
		return
	}

	self.countable.ConnectChanged("Count", reflect.TypeOf(self).String(), func() {
		if self.lastTime == -1 {
			self.lastTime = self.countable.GetTime()
			return
		}
		self.labelLastStep.SetText(shortFormatTime(self.countable.GetTime() - self.lastTime))
		self.lastTime = self.countable.GetTime()
	})
}

func (self *lastStepTime) setBorder(setShown bool) {
	if setShown {
		self.Box.AddCSSClass("infoBoxShowBorder")
	} else {
		self.Box.RemoveCSSClass("infoBoxShowBorder")
	}
}

func (self *lastStepTime) setTitle(set bool) {
	self.title.SetVisible(set)
}

func (self *lastStepTime) setExpand(set bool) {
	self.labelLastStep.SetVExpand(set)
	if set {
		self.labelLastStep.AddCSSClass("expandWidget")
	} else {
		self.labelLastStep.RemoveCSSClass("expandWidget")
	}
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
	self.setProgress()

	return
}

func (self *overallLuck) setProgress() {
	luck := self.list.Luck()

	self.luck.SetFraction(luck)
	self.luck.SetText(fmt.Sprintf("%.01f%%", luck*100))
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
	if countable.IsNil() {
		return
	}
	countable.ConnectChanged("Count", reflect.TypeOf(self).String(), func() {
		self.setProgress()
	})
}

func (self *overallLuck) setBorder(setShown bool) {
	if setShown {
		self.Box.AddCSSClass("infoBoxShowBorder")
	} else {
		self.Box.RemoveCSSClass("infoBoxShowBorder")
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
