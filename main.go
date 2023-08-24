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
	"tallyGo/settings"
	"tallyGo/windows"
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
	LayoutChanged EventBus.Signal = "LayoutChanged"
)

type HomeApplicationWindow struct {
	*adw.ApplicationWindow

	overlay      *gtk.Overlay
	settings     *settings.Settings
	settingsGrid *settings.SettingsMenu

	isTimingActive bool
}

func newHomeApplicationWindow(app *adw.Application) (self *HomeApplicationWindow) {
	self = &HomeApplicationWindow{
		adw.NewApplicationWindow(&app.Application),
		gtk.NewOverlay(),
		nil,
		nil,
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

	homeLeaflet := windows.NewHomeLeaflet(counters, self.settings)
	self.overlay.SetChild(homeLeaflet)
	self.SetContent(self.overlay)

	self.SetDefaultSize(INIT_WIDTH, INIT_HEIGHT)
	self.NotifyProperty("default-width", func() { EventBus.GetGlobalBus().SendSignal(LayoutChanged) })
	self.NotifyProperty("default-height", func() { EventBus.GetGlobalBus().SendSignal(LayoutChanged) })

	css := gtk.NewCSSProvider()
	gtk.StyleContextAddProviderForDisplay(gdk.DisplayGetDefault(), css, gtk.STYLE_PROVIDER_PRIORITY_SETTINGS)
	css.LoadFromData(CSS_FILE)

	themeCSS := gtk.NewCSSProvider()
	gtk.StyleContextAddProviderForDisplay(gdk.DisplayGetDefault(), themeCSS, gtk.STYLE_PROVIDER_PRIORITY_SETTINGS+1)

	self.setCSSTheme(themeCSS)
	APP.StyleManager().NotifyProperty("dark", func() { self.setCSSTheme(themeCSS) })

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
			saveDataHandler.Save(counters)

		case key == input.KeyMinus || key == input.KeyKeypadMinus:
			if !self.isTimingActive {
				return
			}
			glib.IdleAdd(func() {
				for _, countable := range counters.GetActive() {
					countable.IncreaseBy(-1)
				}
			})
			saveDataHandler.Save(counters)

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

	APP.ConnectShutdown(func() {
		saveDataHandler.Save(counters)
	})

	return
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

func (self *HomeApplicationWindow) NewHeaderPopoverMenu() (popover *gtk.PopoverMenu) {
	popover = gtk.NewPopoverMenuFromModel(nil)

	menuModel := gio.NewMenu()

	action := gio.NewSimpleAction("preferences.open", nil)
	action.ConnectActivate(func(*glib.Variant) {
		self.overlay.AddOverlay(self.settingsGrid)
	})

	APP.AddAction(action)

	menuModel.Append("Preferences", "app.preferences.open")
	menuModel.Append("About", "")

	popover.SetMenuModel(menuModel)

	return
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

func (self *SaveFileHandler) Save(counters *CounterList) (err error) {
	self.CounterData = counters.List
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
		self.SettingsData.SetupEvents()
	} else {
		log.Printf("[INFO]\tFound Settings data in savefile, loading Settings")
		self.SettingsData.SetupEvents()
	}

	log.Printf("[INFO]\tLoaded %d Counters\n", len(self.CounterData))

	return
}
