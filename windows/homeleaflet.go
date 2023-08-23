package windows

import (
	"tallyGo/countable"
	EventBus "tallyGo/eventBus"
	infobox "tallyGo/infoBox"
	"tallyGo/resizebar"
	"tallyGo/settings"
	"tallyGo/treeview"

	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

type HomeLeaflet struct {
	*gtk.Box

	counters *countable.CounterList
	settings *settings.Settings

	sidebar         *LeafletPane
	sidebarRevealer *gtk.Revealer

	infoBox         *LeafletPane
	infoBoxRevealer *gtk.Revealer

	resizeBar       *resizebar.ResizeBar
	isAutoCollapsed bool
}

func NewHomeLeaflet(counters *countable.CounterList, settings *settings.Settings) (self *HomeLeaflet) {
	self = &HomeLeaflet{
		Box:             gtk.NewBox(gtk.OrientationHorizontal, 0),
		counters:        counters,
		settings:        settings,
		sidebarRevealer: gtk.NewRevealer(),
		infoBoxRevealer: gtk.NewRevealer(),
		resizeBar:       resizebar.NewResizeBar(gtk.OrientationVertical),
	}

	hideSidebar := gio.NewSimpleAction("toggleSidebar", nil)
	hideSidebar.ConnectActivate(func(*glib.Variant) {
		if self.IsCollapsed() {
			self.UnCollapse()
		} else {
			self.Collapse()
		}
	})

	aGroup := gio.NewSimpleActionGroup()
	aGroup.AddAction(hideSidebar)
	self.Box.InsertActionGroup("leaflet", aGroup)

	self.sidebar = NewLeafletPane(treeview.NewCounterTreeView(counters), false)
	self.sidebarRevealer.SetChild(self.sidebar)
	self.sidebarRevealer.SetTransitionType(gtk.RevealerTransitionTypeSlideRight)
	self.sidebarRevealer.SetRevealChild(true)
	self.infoBox = NewLeafletPane(infobox.NewInfoBox(counters), true)
	self.infoBoxRevealer.SetChild(self.infoBox)
	self.infoBoxRevealer.SetTransitionType(gtk.RevealerTransitionTypeSlideLeft)
	self.infoBoxRevealer.SetRevealChild(true)

	self.Box.Append(self.sidebarRevealer)
	self.Box.Append(self.resizeBar)
	self.Box.Append(self.infoBoxRevealer)

	self.resizeBar.Attach(&self.sidebar.Widget)

	self.SetupEvents()
	self.UnCollapse()

	return
}

func (self *HomeLeaflet) SetupEvents() {
	evtBus := EventBus.GetGlobalBus()
	evtBus.Subscribe(countable.ListActiveChanged, func(...interface{}) {
		if !self.IsCollapsed() {
			return
		}
		self.UnCollapse()
		self.Collapse()
	})

	evtBus.Subscribe(treeview.LayoutChanged, func(...interface{}) {
		sidebarMinWidth := int(self.settings.GetValue(settings.SideBarSize).(float64))
		infoboxMinWidth := 250
		if (self.sidebar.Width() < sidebarMinWidth || self.infoBox.Width() < infoboxMinWidth) && !self.IsCollapsed() {
			self.ActionSetEnabled("leaflet.toggleSidebar", false)
			self.Collapse()
			self.isAutoCollapsed = true
		} else if self.sidebar.Width() > sidebarMinWidth+infoboxMinWidth+40 || self.infoBox.Width() > sidebarMinWidth+infoboxMinWidth+40 {
			self.ActionSetEnabled("leaflet.toggleSidebar", true)
			if self.isAutoCollapsed {
				self.UnCollapse()
			}
			self.isAutoCollapsed = false
		}
	})
}

func (self *HomeLeaflet) IsCollapsed() bool {
	return !(self.sidebarRevealer.RevealChild() && self.infoBoxRevealer.RevealChild())
}

func (self *HomeLeaflet) Collapse() {
	if self.counters.HasActive() {
		self.sidebarRevealer.SetRevealChild(false)
		self.sidebar.SetHExpand(false)
		self.infoBox.SetHExpand(true)
	} else {
		self.infoBoxRevealer.SetRevealChild(false)
		self.sidebar.SetSizeRequest(-1, -1)
		self.infoBoxRevealer.ConnectStateFlagsChanged(func(flags gtk.StateFlags) {

		})
		self.sidebar.SetHExpand(true)
		self.infoBox.SetHExpand(false)
	}
	self.resizeBar.Hide()
}

func (self *HomeLeaflet) UnCollapse() {
	self.infoBoxRevealer.SetRevealChild(true)
	self.sidebarRevealer.SetRevealChild(true)
	self.sidebar.SetHExpand(false)
	self.infoBox.SetHExpand(true)
	if self.settings.HasValue(settings.SideBarSize) {
		self.sidebar.SetSizeRequest(int(self.settings.GetValue(settings.SideBarSize).(float64)), -1)
	} else {
		self.sidebar.SetSizeRequest(240, -1)
	}
	self.resizeBar.Show()
}

type LeafletBody interface {
	GetWidget() *gtk.Widget
	HeaderBar() *gtk.HeaderBar
}

type LeafletPane struct {
	*gtk.Box

	header *gtk.HeaderBar
	body   LeafletBody
}

func NewLeafletPane(body LeafletBody, hasWindowButtons bool) (self *LeafletPane) {
	self = &LeafletPane{
		Box:    gtk.NewBox(gtk.OrientationVertical, 0),
		header: body.HeaderBar(),
		body:   body,
	}

	self.header.SetShowTitleButtons(hasWindowButtons)
	scrollView := gtk.NewScrolledWindow()
	scrollView.SetChild(body.GetWidget())
	scrollView.SetPropagateNaturalWidth(true)
	self.body.GetWidget().SetVExpand(true)

	self.Box.Append(self.header)
	self.Box.Append(scrollView)

	return
}
