package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"os"
	. "tallyGo/countable"
	EventBus "tallyGo/eventBus"
	"tallyGo/input"
	"tallyGo/resizebar"
	"tallyGo/settings"
	"tallyGo/treeview"
	"time"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

const FRAME_TIME = time.Millisecond * 33
const SAVE_STRATEGY = JSON

const INIT_WIDTH = 960
const INIT_HEIGHT = 680

//go:embed styleSheets/style.css
var CSS_FILE string

//go:embed styleSheets/style-light.css
var CSS_LIGHT string

//go:embed styleSheets/style-dark.css
var CSS_DARK string

var APP *adw.Application
var HOME *HomeApplicationWindow

// TODO: instead of just storing the date counter should store diffs with a time
// this will improve the info window
func main() {
	APP = adw.NewApplication("com.github.p3rtang.counter", gio.ApplicationFlagsNone)
	APP.ConnectActivate(func() { activate(APP) })

	EventBus.InitBus()

	if code := APP.Run(os.Args); code > 0 {
		os.Exit(code)
	}
}

func activate(app *adw.Application) (err error) {
	window := newHomeApplicationWindow(app)
	window.Show()
	return
}

type AppLayout string

const (
	LayoutPanes          AppLayout = "ShowPanes"
	LayoutSingleTreeView           = "SingleTreeView"
	LayoutSingleInfoBox            = "SingleInfoBox"
)

const (
	// callback arguments (*gtk.Window)
	LayoutChanged EventBus.Signal = "LayoutChanged"
)

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

func newHomeApplicationWindow(app *adw.Application) (self *HomeApplicationWindow) {
	self = &HomeApplicationWindow{
		gtk.NewApplicationWindow(&app.Application),
		gtk.NewOverlay(),
		gtk.NewGrid(),
		nil,
		nil,
		nil,
		gtk.NewRevealer(),
		false,
		gtk.NewButtonFromIconName("sidebar-show-symbolic"),
		gtk.NewToggleButton(),
		gtk.NewHeaderBar(),
		false,
	}

	fmt.Println(APP.ActiveWindow().Settings().ObjectProperty("gtk-theme-name"))

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

	counterTV := treeview.NewCounterTreeView(counters)

	scrollView := gtk.NewScrolledWindow()
	scrollView.SetPropagateNaturalWidth(true)
	scrollView.SetChild(counterTV)
	scrollView.SetName("treeViewScrollWindow")

	self.treeViewRevealer.SetTransitionType(gtk.RevealerTransitionTypeSlideRight)
	self.treeViewRevealer.SetChild(scrollView)
	self.treeViewRevealer.SetRevealChild(true)

	self.Window.ConnectShow(func() {
		if self.settings.HasValue(settings.SideBarSize) {
			if self.settings.GetValue(settings.SideBarSize).(float64) > INIT_WIDTH {
				counterTV.SetSizeRequest(240, -1)
			} else {
				counterTV.SetSizeRequest(
					int(self.settings.GetValue(settings.SideBarSize).(float64)), -1)
			}
		}
	})

	self.settingsButton.SetIconName("open-menu-symbolic")
	self.settingsButton.SetName("settingsButton")
	image := self.settingsButton.Child().(*gtk.Image)
	image.SetPixelSize(18)
	self.settingsButton.ConnectToggled(func() {
		if self.settingsButton.Active() {
			self.overlay.SetChild(self.settingsGrid)
		} else {
			self.overlay.SetChild(self.homeGrid)
		}
	})

	self.headerBar.PackStart(self.collapseButton)
	self.headerBar.PackEnd(self.settingsButton)
	self.SetTitlebar(self.headerBar)

	self.SetChild(self.overlay)
	self.overlay.SetChild(self.homeGrid)
	self.SetDefaultSize(INIT_WIDTH, INIT_HEIGHT)
	self.NotifyProperty("default-width", func() { EventBus.GetGlobalBus().SendSignal(LayoutChanged, &self.Window) })
	self.NotifyProperty("default-height", func() { EventBus.GetGlobalBus().SendSignal(LayoutChanged, &self.Window) })

	css := gtk.NewCSSProvider()
	gtk.StyleContextAddProviderForDisplay(gdk.DisplayGetDefault(), css, gtk.STYLE_PROVIDER_PRIORITY_SETTINGS)
	css.LoadFromData(CSS_FILE)

	themeCSS := gtk.NewCSSProvider()
	gtk.StyleContextAddProviderForDisplay(gdk.DisplayGetDefault(), themeCSS, gtk.STYLE_PROVIDER_PRIORITY_SETTINGS+1)

	self.setCSSTheme(themeCSS)
	APP.StyleManager().NotifyProperty("dark", func() { self.setCSSTheme(themeCSS) })

	resizeBar := resizebar.NewResizeBar(gtk.OrientationVertical)
	resizeBar.Attach(&counterTV.Widget)
	resizeBar.SetMaxWidth(0.4, &self.Window)

	resizeGesture := gtk.NewGestureDrag()
	resizeBar.AddController(resizeGesture)
	resizeBar.Gesture().ConnectDragUpdate(func(offsetX float64, _ float64) {
		self.settings.SetValue(
			settings.SideBarSize,
			float64(counterTV.Width())+offsetX,
		)
		eventBus.SendSignal(LayoutChanged, &self.Window)
	})

	self.infoBox = NewInfoBox(counters)
	infoScrollView := gtk.NewScrolledWindow()
	infoScrollView.SetChild(self.infoBox)

	self.homeGrid.Attach(self.treeViewRevealer, 0, 0, 1, 1)
	self.homeGrid.Attach(resizeBar, 1, 0, 1, 1)
	self.homeGrid.Attach(infoScrollView, 2, 0, 1, 1)

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

	self.collapseButton.ConnectClicked(func() {
		if self.treeViewRevealer.RevealChild() {
			counterTV.SetSizeRequest(-1, -1)
			resizeBar.Hide()
			self.treeViewRevealer.SetRevealChild(false)
		} else {
			counterTV.SetSizeRequest(
				int(self.settings.GetValue(settings.SideBarSize).(float64)), -1)
			resizeBar.Show()
			self.treeViewRevealer.SetRevealChild(true)
		}
		eventBus.SendSignal(LayoutChanged, &self.Window)
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

	EventBus.GetGlobalBus().Subscribe(LayoutChanged, func(...interface{}) {
		var sidebarWidth int = 0
		if self.settings.HasValue(settings.SideBarSize) {
			sidebarWidth = int(self.settings.GetValue(settings.SideBarSize).(float64))
		}

		switch {
		case self.infoBox.Width() < 400:
			if self.treeViewRevealer.RevealChild() {
				resizeBar.Hide()
				self.treeViewRevealer.SetRevealChild(false)
				self.isRevealerAutoHidden = true
			}
			self.collapseButton.SetSensitive(false)
		case self.Width() > 420+sidebarWidth:
			if self.isRevealerAutoHidden {
				resizeBar.Show()
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
	})

	return
}

func (self *HomeApplicationWindow) ChangeLayout(layout AppLayout) {
	switch layout {
	case LayoutPanes:
		self.overlay.SetChild(self.homeGrid)
	case LayoutSingleTreeView:
		self.overlay.SetChild(self.treeViewRevealer)
	case LayoutSingleInfoBox:
		self.overlay.SetChild(self.infoBox)
	}
}

func (self *HomeApplicationWindow) setCSSTheme(css *gtk.CSSProvider) {
	if APP.StyleManager().Dark() {
		gtk.SettingsGetDefault().SetObjectProperty("gtk-theme-name", "Adwaita-dark")
		css.LoadFromData(CSS_DARK)
	} else {
		gtk.SettingsGetDefault().SetObjectProperty("gtk-theme-name", "Adwaita")
		css.LoadFromData(CSS_LIGHT)
	}
}

func (self *HomeApplicationWindow) HandleNotify() {
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
		log.Fatal("[FATAL]\tCould not Read save file, Got Error: ", err)
		return
	}

	err = json.Unmarshal(saveData, self)
	if err != nil {
		self.SettingsData = settings.NewSettings()
		log.Println("[WARN] Could not Unmarshal SaveData, Got Error: ", err)
		return
	}

	if self.SettingsData == nil {
		log.Printf("[INFO]\tFound no Settings data in savefile, generating default Settings")
		self.SettingsData = settings.NewSettings()
	} else {
		log.Printf("[INFO]\tFound Settings data in savefile, loading Settings")
	}

	log.Printf("[INFO]\tLoaded %d Counters\n", len(self.CounterData))

	return
}
