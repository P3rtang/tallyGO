package main

import (
	_ "embed"
	"encoding/json"
	"log"
	"os"
	. "tallyGo/countable"
	EventBus "tallyGo/eventBus"
	"tallyGo/input"
	"tallyGo/settings"
	"tallyGo/treeview"
	"time"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

const FRAME_TIME = time.Millisecond * 33
const SAVE_STRATEGY = JSON

//go:embed styleSheets/style.css
var CSS_FILE string

//go:embed styleSheets/style-light.css
var CSS_LIGHT string

//go:embed styleSheets/style-dark.css
var CSS_DARK string

var APP *gtk.Application
var HOME *HomeApplicationWindow

// TODO: instead of just storing the date counter should store diffs with a time
// this will improve the info window
func main() {
	APP = gtk.NewApplication("com.github.p3rtang.counter", gio.ApplicationFlagsNone)
	APP.ConnectActivate(func() { activate(APP) })

	EventBus.InitBus()

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

func newHomeApplicationWindow(app *gtk.Application) (self *HomeApplicationWindow) {
	self = &HomeApplicationWindow{
		gtk.NewApplicationWindow(app),
		gtk.NewOverlay(),
		gtk.NewGrid(),
		nil,
		nil,
		nil,
		gtk.NewRevealer(),
		false,
		gtk.NewButtonFromIconName("open-menu-symbolic"),
		gtk.NewToggleButton(),
		gtk.NewHeaderBar(),
		false,
	}

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
	scrollView.SetChild(counterTV)
	scrollView.SetName("treeViewScrollWindow")

	self.treeViewRevealer.SetTransitionType(gtk.RevealerTransitionTypeSlideRight)
	self.treeViewRevealer.SetChild(scrollView)
	self.treeViewRevealer.SetRevealChild(true)

	self.collapseButton.ConnectClicked(func() {
		if self.treeViewRevealer.RevealChild() {
			self.treeViewRevealer.SetRevealChild(false)
		} else {
			self.treeViewRevealer.SetRevealChild(true)
		}
	})

	self.settingsButton.SetIconName("applications-system-symbolic")
	self.settingsButton.SetName("settingsButton")
	image := self.settingsButton.Child().(*gtk.Image)
	image.SetPixelSize(20)
	self.settingsButton.ConnectToggled(func() {
		if self.settingsButton.Active() {
			self.overlay.SetChild(self.settingsGrid)
		} else {
			self.overlay.SetChild(self.homeGrid)
		}
	})

	self.headerBar.PackStart(self.collapseButton)
	self.headerBar.PackEnd(self.settingsButton)
	self.Window.SetTitlebar(self.headerBar)

	self.SetChild(self.overlay)
	self.overlay.SetChild(self.homeGrid)
	self.SetDefaultSize(900, 660)
	self.NotifyProperty("default-width", self.HandleNotify)

	css := gtk.NewCSSProvider()
	gtk.StyleContextAddProviderForDisplay(gdk.DisplayGetDefault(), css, gtk.STYLE_PROVIDER_PRIORITY_SETTINGS)
	css.LoadFromData(CSS_FILE)

	themeCSS := gtk.NewCSSProvider()
	gtk.StyleContextAddProviderForDisplay(gdk.DisplayGetDefault(), themeCSS, gtk.STYLE_PROVIDER_PRIORITY_SETTINGS+1)
	self.setCSSTheme(themeCSS)

	saveDataHandler.SettingsData.ConnectChanged(settings.DarkMode, func(_ any) {
		self.setCSSTheme(themeCSS)
	})

	self.infoBox = NewInfoBox(counters)
	infoBoxScroll := gtk.NewScrolledWindow()
	infoBoxScroll.SetChild(self.infoBox)

	self.homeGrid.Attach(self.treeViewRevealer, 0, 1, 1, 9)
	self.homeGrid.Attach(infoBoxScroll, 1, 2, 1, 1)

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

	return
}

func (self *HomeApplicationWindow) setCSSTheme(css *gtk.CSSProvider) {
	if self.settings.GetValue(settings.DarkMode) == nil {
		self.settings.SetValue(settings.DarkMode, false)
	}
	if self.settings.GetValue(settings.DarkMode).(bool) {
		gtk.SettingsGetDefault().SetObjectProperty("gtk-theme-name", "Adwaita-dark")
		css.LoadFromData(CSS_DARK)
	} else {
		gtk.SettingsGetDefault().SetObjectProperty("gtk-theme-name", "Adwaita")
		css.LoadFromData(CSS_LIGHT)
	}
}

func (self *HomeApplicationWindow) HandleNotify() {
	switch {
	case self.Width() < 500:
		if self.treeViewRevealer.RevealChild() {
			self.treeViewRevealer.SetRevealChild(false)
			self.isRevealerAutoHidden = true
		}
		self.collapseButton.SetSensitive(false)
	case self.Width() > 500:
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
