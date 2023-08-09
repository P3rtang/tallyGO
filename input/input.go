package input

import (
	"container/list"
	"log"
	"os"
	"strings"
	"time"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
	evdev "github.com/gvalkov/golang-evdev"
)

type InputHandler interface {
	Init() error
	NextEvent() Event
	HasEvent() bool
	SimulateKey()
}

type DevInput struct {
	eventStream  *list.List
	callbackList map[callBackKeyType]func()

	isRunning bool
	file      string
}

func NewDevInput() DevInput {
	return DevInput{
		list.New(),
		make(map[callBackKeyType]func()),
		true,
		"",
	}
}

func (self *DevInput) NextEvent() *Event {
	front := self.eventStream.Front()
	if front == nil {
		return nil
	}

	self.eventStream.Remove(front)
	if event, isEvent := front.Value.(Event); isEvent {
		return &event
	}
	return nil
}

func (self *DevInput) HasEvent() bool {
	return self.eventStream.Len() > 0
}

func (self *DevInput) ChangeFile(file string) {
	self.file = "/dev/input/by-id/" + file
}

func (self *DevInput) fileInit() {
	go func() {
		for {
			hasEvent := make(chan bool)
			timedOut := make(chan bool)

			go self.readEvent(hasEvent)
			go func() {
				time.Sleep(time.Millisecond * 200)
			}()

			select {
			case <-timedOut:
			case <-hasEvent:
			}
		}
	}()
}

func (self *DevInput) readEvent(hasEvent chan bool) {
	if device, err := evdev.Open(self.file); err == nil {
		events, _ := device.Read()
		if len(events) == 0 {
			hasEvent <- false
		}
		for _, event := range events {
			if event.Type != 1 {
				continue
			}
			ev := fromEvdev(event)
			ev.Level = DevInputEvent
			self.eventStream.PushBack(ev)
		}
		hasEvent <- true
	} else {
		log.Println("[WARN] Could not open device file, Got Error: ", err)
		hasEvent <- false
	}
}

func (self *DevInput) Init(file string) (err error) {
	self.ChangeFile(file)
	go func() {
		for self.isRunning {
			event := self.NextEvent()
			if event == nil {
				time.Sleep(time.Millisecond * 5)
			} else if callback := self.callbackList[callBackKeyType{event.Code, event.Value, event.Level}]; callback != nil {
				glib.IdleAdd(func() {
					callback()
				})
			}
		}
	}()

	self.fileInit()

	return
}

func (self *DevInput) ConnectKey(key KeyType, evType EventValue, level CallbackLevel, callback func()) {
	self.callbackList[callBackKeyType{key, evType, level}] = callback
}

func (self *DevInput) SimulateKey(key KeyType, evType EventValue) {
	self.eventStream.PushBack(
		Event{
			time.Now().Unix(),
			1,
			key,
			evType,
			WindowEvent,
		},
	)
}

func GetKbdList() []string {
	folder, err := os.ReadDir("/dev/input/by-id/")
	if err != nil {
		return []string{}
	}

	files := []string{}
	for _, file := range folder {
		if !strings.Contains(file.Name(), "kbd") {
			continue
		}
		files = append(files, file.Name())
	}

	return files
}

type callBackKeyType struct {
	Code  CodeType
	Value EventValue
	Level CallbackLevel
}

type CallbackLevel int

const (
	WindowEvent CallbackLevel = iota
	DevInputEvent
)

type Event struct {
	Time  int64
	Type  EventType
	Code  CodeType
	Value EventValue
	Level CallbackLevel
}

func fromEvdev(ev evdev.InputEvent) (event Event) {
	return Event{
		ev.Time.Sec,
		EventType(ev.Type),
		KeyType(ev.Code),
		EventValue(ev.Value),
		DevInputEvent,
	}
}

type EventType uint16

const (
	EventSync EventType = iota
	EventKey
	EventRelative
	EventAbsolute
	EventMisc
	EventSwitch
	EventLED
	EventSound
	EventRepeat
	EventEffect
	EventPower
	EventEffectStatus
)

type CodeType interface {
	GetCode() uint16
}

type KeyType uint16

func (self KeyType) GetCode() uint16 {
	return uint16(self)
}

const (
	KeyReserved          KeyType = 0
	KeyEscape            KeyType = 1
	Key1                 KeyType = 2
	Key2                 KeyType = 3
	Key3                 KeyType = 4
	Key4                 KeyType = 5
	Key5                 KeyType = 6
	Key6                 KeyType = 7
	Key7                 KeyType = 8
	Key8                 KeyType = 9
	Key9                 KeyType = 10
	Key0                 KeyType = 11
	KeyMinus             KeyType = 12
	KeyEqual             KeyType = 13
	KeyBackSpace         KeyType = 14
	KeyTab               KeyType = 15
	KeyQ                 KeyType = 16
	KeyW                 KeyType = 17
	KeyE                 KeyType = 18
	KeyR                 KeyType = 19
	KeyT                 KeyType = 20
	KeyY                 KeyType = 21
	KeyU                 KeyType = 22
	KeyI                 KeyType = 23
	KeyO                 KeyType = 24
	KeyP                 KeyType = 25
	KeyLeftBrace         KeyType = 26
	KeyRightBrace        KeyType = 27
	KeyEnter             KeyType = 28
	KeyLeftCtrl          KeyType = 29
	KeyA                 KeyType = 30
	KeyS                 KeyType = 31
	KeyD                 KeyType = 32
	KeyF                 KeyType = 33
	KeyG                 KeyType = 34
	KeyH                 KeyType = 35
	KeyJ                 KeyType = 36
	KeyK                 KeyType = 37
	KeyL                 KeyType = 38
	KeySemiColon         KeyType = 39
	KeyApostrophe        KeyType = 40
	KeyGrave             KeyType = 41
	KeyLeftShift         KeyType = 42
	KeyBackSlash         KeyType = 43
	KeyZ                 KeyType = 44
	KeyX                 KeyType = 45
	KeyC                 KeyType = 46
	KeyV                 KeyType = 47
	KeyB                 KeyType = 48
	KeyN                 KeyType = 49
	KeyM                 KeyType = 50
	KeyComma             KeyType = 51
	KeyDot               KeyType = 52
	KeySlash             KeyType = 53
	KeyRightShift        KeyType = 54
	KeyKeypadAsterisk    KeyType = 55
	KeyLeftAlt           KeyType = 56
	KeySpace             KeyType = 57
	KeyCapsLock          KeyType = 58
	KeyF1                KeyType = 59
	KeyF2                KeyType = 60
	KeyF3                KeyType = 61
	KeyF4                KeyType = 62
	KeyF5                KeyType = 63
	KeyF6                KeyType = 64
	KeyF7                KeyType = 65
	KeyF8                KeyType = 66
	KeyF9                KeyType = 67
	KeyF10               KeyType = 68
	KeyNumLock           KeyType = 69
	KeyScrollLock        KeyType = 70
	KeyKeypad7           KeyType = 71
	KeyKeypad8           KeyType = 72
	KeyKeypad9           KeyType = 73
	KeyKeypadMinus       KeyType = 74
	KeyKeypad4           KeyType = 75
	KeyKeypad5           KeyType = 76
	KeyKeypad6           KeyType = 77
	KeyKeypadPlus        KeyType = 78
	KeyKeypad1           KeyType = 79
	KeyKeypad2           KeyType = 80
	KeyKeypad3           KeyType = 81
	KeyKeypad0           KeyType = 82
	KeyKeypadDot         KeyType = 83
	KeyZenkakuHankaku    KeyType = 85
	Key102ND             KeyType = 86
	KeyF11               KeyType = 87
	KeyF12               KeyType = 88
	KeyRO                KeyType = 89
	KeyKatakana          KeyType = 90
	KeyHiragana          KeyType = 91
	KeyHenkan            KeyType = 92
	KeyKatakanaHiragana  KeyType = 93
	KeyMuhenkan          KeyType = 94
	KeyKeypadJPComma     KeyType = 95
	KeyKeypadEnter       KeyType = 96
	KeyRightCtrl         KeyType = 97
	KeyKeypadSlash       KeyType = 98
	KeySysRQ             KeyType = 99
	KeyRightAlt          KeyType = 100
	KeyLineFeed          KeyType = 101
	KeyHome              KeyType = 102
	KeyUp                KeyType = 103
	KeyPageUp            KeyType = 104
	KeyLeft              KeyType = 105
	KeyRight             KeyType = 106
	KeyEnd               KeyType = 107
	KeyDown              KeyType = 108
	KeyPageDown          KeyType = 109
	KeyInsert            KeyType = 110
	KeyDelete            KeyType = 111
	KeyMacro             KeyType = 112
	KeyMute              KeyType = 113
	KeyVolumeDown        KeyType = 114
	KeyVolumeUp          KeyType = 115
	KeyPower             KeyType = 116 // SC System Power Down
	KeyKeypadEqual       KeyType = 117
	KeyKeypadPlusMinus   KeyType = 118
	KeyPause             KeyType = 119
	KeyScale             KeyType = 120 // AL Compiz Scale (Expose)
	KeyKeypadComma       KeyType = 121
	KeyHangul            KeyType = 122
	KeyHanja             KeyType = 123
	KeyYen               KeyType = 124
	KeyLeftMeta          KeyType = 125
	KeyRightMeta         KeyType = 126
	KeyCompose           KeyType = 127
	KeyStop              KeyType = 128 // AC Stop
	KeyAgain             KeyType = 129
	KeyProps             KeyType = 130 // AC Properties
	KeyUndo              KeyType = 131 // AC Undo
	KeyFront             KeyType = 132
	KeyCopy              KeyType = 133 // AC Copy
	KeyOpen              KeyType = 134 // AC Open
	KeyPaste             KeyType = 135 // AC Paste
	KeyFind              KeyType = 136 // AC Search
	KeyCut               KeyType = 137 // AC Cut
	KeyHelp              KeyType = 138 // AL Integrated Help Center
	KeyMenu              KeyType = 139 // Menu (show menu)
	KeyCalc              KeyType = 140 // AL Calculator
	KeySetup             KeyType = 141
	KeySleep             KeyType = 142 // SC System Sleep
	KeyWakeup            KeyType = 143 // System Wake Up
	KeyFile              KeyType = 144 // AL Local Machine Browser
	KeySendFile          KeyType = 145
	KeyDeleteFile        KeyType = 146
	KeyXfer              KeyType = 147
	KeyProg1             KeyType = 148
	KeyProg2             KeyType = 149
	KeyWWW               KeyType = 150 // AL Internet Browser
	KeyMSDOS             KeyType = 151
	KeyScreenlock        KeyType = 152
	KeyDirection         KeyType = 153
	KeyCycleWindows      KeyType = 154
	KeyMail              KeyType = 155
	KeyBookmarks         KeyType = 156 // AC Bookmarks
	KeyComputer          KeyType = 157
	KeyBack              KeyType = 158 // AC Back
	KeyForward           KeyType = 159 // AC Forward
	KeyCloseCD           KeyType = 160
	KeyEjectCD           KeyType = 161
	KeyEjectCloseCD      KeyType = 162
	KeyNextSong          KeyType = 163
	KeyPlayPause         KeyType = 164
	KeyPreviousSong      KeyType = 165
	KeyStopCD            KeyType = 166
	KeyRecord            KeyType = 167
	KeyRewind            KeyType = 168
	KeyPhone             KeyType = 169 // Media Select Telephone
	KeyISO               KeyType = 170
	KeyConfig            KeyType = 171 // AL Consumer Control Configuration
	KeyHomepage          KeyType = 172 // AC Home
	KeyRefresh           KeyType = 173 // AC Refresh
	KeyExit              KeyType = 174 // AC Exit
	KeyMove              KeyType = 175
	KeyEdit              KeyType = 176
	KeyScrollUp          KeyType = 177
	KeyScrollDown        KeyType = 178
	KeyKeypadLeftParen   KeyType = 179
	KeyKeypadRightParen  KeyType = 180
	KeyNew               KeyType = 181 // AC New
	KeyRedo              KeyType = 182 // AC Redo/Repeat
	KeyF13               KeyType = 183
	KeyF14               KeyType = 184
	KeyF15               KeyType = 185
	KeyF16               KeyType = 186
	KeyF17               KeyType = 187
	KeyF18               KeyType = 188
	KeyF19               KeyType = 189
	KeyF20               KeyType = 190
	KeyF21               KeyType = 191
	KeyF22               KeyType = 192
	KeyF23               KeyType = 193
	KeyF24               KeyType = 194
	KeyPlayCD            KeyType = 200
	KeyPauseCD           KeyType = 201
	KeyProg3             KeyType = 202
	KeyProg4             KeyType = 203
	KeyDashboard         KeyType = 204 // AL Dashboard
	KeySuspend           KeyType = 205
	KeyClose             KeyType = 206 // AC Close
	KeyPlay              KeyType = 207
	KeyFastForward       KeyType = 208
	KeyBassBoost         KeyType = 209
	KeyPrint             KeyType = 210 // AC Print
	KeyHP                KeyType = 211
	KeyCamera            KeyType = 212
	KeySound             KeyType = 213
	KeyQuestion          KeyType = 214
	KeyEmail             KeyType = 215
	KeyChat              KeyType = 216
	KeySearch            KeyType = 217
	KeyConnect           KeyType = 218
	KeyFinance           KeyType = 219 // AL Checkbook/Finance
	KeySport             KeyType = 220
	KeyShop              KeyType = 221
	KeyAltErase          KeyType = 222
	KeyCancel            KeyType = 223 // AC Cancel
	KeyBrightnessDown    KeyType = 224
	KeyBrightnessUp      KeyType = 225
	KeyMedia             KeyType = 226
	KeySwitchVideoMode   KeyType = 227 // Cycle between available video  outputs (Monitor/LCD/TV-out/etc)
	KeyKbdIllumToggle    KeyType = 228
	KeyKbdIllumDown      KeyType = 229
	KeyKbdIllumUp        KeyType = 230
	KeySend              KeyType = 231 // AC Send
	KeyReply             KeyType = 232 // AC Reply
	KeyForwardMail       KeyType = 233 // AC Forward Msg
	KeySave              KeyType = 234 // AC Save
	KeyDocuments         KeyType = 235
	KeyBattery           KeyType = 236
	KeyBluetooth         KeyType = 237
	KeyWLAN              KeyType = 238
	KeyUWB               KeyType = 239
	KeyUnknown           KeyType = 240
	KeyVideoNext         KeyType = 241 // drive next video source
	KeyVideoPrevious     KeyType = 242 // drive previous video source
	KeyBrightnessCycle   KeyType = 243 // brightness up, after max is min
	KeyBrightnessZero    KeyType = 244 // brightness off, use ambient
	KeyDisplayOff        KeyType = 245 // display device to off state
	KeyWiMax             KeyType = 246
	KeyRFKill            KeyType = 247 // Key that controls all radios
	KeyMicMute           KeyType = 248 // Mute / unmute the microphone
	KeyOk                KeyType = 0x160
	KeySelect            KeyType = 0x161
	KeyGoto              KeyType = 0x162
	KeyClear             KeyType = 0x163
	KeyPower2            KeyType = 0x164
	KeyOption            KeyType = 0x165
	KeyInfo              KeyType = 0x166 // AL OEM Features/Tips/Tutorial
	KeyTime              KeyType = 0x167
	KeyVendor            KeyType = 0x168
	KeyArchive           KeyType = 0x169
	KeyProgram           KeyType = 0x16a // Media Select Program Guide
	KeyChannel           KeyType = 0x16b
	KeyFavorites         KeyType = 0x16c
	KeyEPG               KeyType = 0x16d
	KeyPVR               KeyType = 0x16e // Media Select Home
	KeyMHP               KeyType = 0x16f
	KeyLanguage          KeyType = 0x170
	KeyTitle             KeyType = 0x171
	KeySubtitle          KeyType = 0x172
	KeyAngle             KeyType = 0x173
	KeyZoom              KeyType = 0x174
	KeyMode              KeyType = 0x175
	KeyKeyboard          KeyType = 0x176
	KeyScreen            KeyType = 0x177
	KeyPC                KeyType = 0x178 // Media Select Computer
	KeyTV                KeyType = 0x179 // Media Select TV
	KeyTV2               KeyType = 0x17a // Media Select Cable
	KeyVCR               KeyType = 0x17b // Media Select VCR
	KeyVCR2              KeyType = 0x17c // VCR Plus
	KeySAT               KeyType = 0x17d // Media Select Satellite
	KeySAT2              KeyType = 0x17e
	KeyCD                KeyType = 0x17f // Media Select CD
	KeyTape              KeyType = 0x180 // Media Select Tape
	KeyRadio             KeyType = 0x181
	KeyTuner             KeyType = 0x182 // Media Select Tuner
	KeyPlayer            KeyType = 0x183
	KeyText              KeyType = 0x184
	KeyDVD               KeyType = 0x185 // Media Select DVD
	KeyAUX               KeyType = 0x186
	KeyMP3               KeyType = 0x187
	KeyAudio             KeyType = 0x188 // AL Audio Browser
	KeyVideo             KeyType = 0x189 // AL Movie Browser
	KeyDirectory         KeyType = 0x18a
	KeyList              KeyType = 0x18b
	KeyMemo              KeyType = 0x18c // Media Select Messages
	KeyCalender          KeyType = 0x18d
	KeyRed               KeyType = 0x18e
	KeyGreen             KeyType = 0x18f
	KeyYellow            KeyType = 0x190
	KeyBlue              KeyType = 0x191
	KeyChannelUp         KeyType = 0x192 // Channel Increment
	KeyChannelDown       KeyType = 0x193 // Channel Decrement
	KeyFirst             KeyType = 0x194
	KeyLast              KeyType = 0x195 // Recall Last
	KeyAB                KeyType = 0x196
	KeyNext              KeyType = 0x197
	KeyRestart           KeyType = 0x198
	KeySlow              KeyType = 0x199
	KeyShuffle           KeyType = 0x19a
	KeyBreak             KeyType = 0x19b
	KeyPrevious          KeyType = 0x19c
	KeyDigits            KeyType = 0x19d
	KeyTeen              KeyType = 0x19e
	KeyTwen              KeyType = 0x19f
	KeyVideoPhone        KeyType = 0x1a0 // Media Select Video Phone
	KeyGames             KeyType = 0x1a1 // Media Select Games
	KeyZoomIn            KeyType = 0x1a2 // AC Zoom In
	KeyZoomOut           KeyType = 0x1a3 // AC Zoom Out
	KeyZoomReset         KeyType = 0x1a4 // AC Zoom
	KeyWordProcessor     KeyType = 0x1a5 // AL Word Processor
	KeyEditor            KeyType = 0x1a6 // AL Text Editor
	KeySpreadsheet       KeyType = 0x1a7 // AL Spreadsheet
	KeyGraphicsEditor    KeyType = 0x1a8 // AL Graphics Editor
	KeyPresentation      KeyType = 0x1a9 // AL Presentation App
	KeyDatabase          KeyType = 0x1aa // AL Database App
	KeyNews              KeyType = 0x1ab // AL Newsreader
	KeyVoiceMail         KeyType = 0x1ac // AL Voicemail
	KeyAddressBook       KeyType = 0x1ad // AL Contacts/Address Book
	KeyMessenger         KeyType = 0x1ae // AL Instant Messaging
	KeyDisplayToggle     KeyType = 0x1af // Turn display (LCD) on and off
	KeySpellCheck        KeyType = 0x1b0 // AL Spell Check
	KeyLogoff            KeyType = 0x1b1 // AL Logoff
	KeyDollar            KeyType = 0x1b2
	KeyEuro              KeyType = 0x1b3
	KeyFrameBack         KeyType = 0x1b4 // Consumer - transport controls
	KeyframeForward      KeyType = 0x1b5
	KeyContextMenu       KeyType = 0x1b6 // GenDesc - system context menu
	KeyMediaRepeat       KeyType = 0x1b7 // Consumer - transport control
	Key10ChannelsUp      KeyType = 0x1b8 // 10 channels up (10+)
	Key10ChannelsDown    KeyType = 0x1b9 // 10 channels down (10-)
	KeyImages            KeyType = 0x1ba // AL Image Browser
	KeyDelEOL            KeyType = 0x1c0
	KeyDelEOS            KeyType = 0x1c1
	KeyInsLine           KeyType = 0x1c2
	KeyDelLine           KeyType = 0x1c3
	KeyFunc              KeyType = 0x1d0
	KeyFuncEsc           KeyType = 0x1d1
	KeyFuncF1            KeyType = 0x1d2
	KeyFuncF2            KeyType = 0x1d3
	KeyFuncF3            KeyType = 0x1d4
	KeyFuncF4            KeyType = 0x1d5
	KeyFuncF5            KeyType = 0x1d6
	KeyFuncF6            KeyType = 0x1d7
	KeyFuncF7            KeyType = 0x1d8
	KeyFuncF8            KeyType = 0x1d9
	KeyFuncF9            KeyType = 0x1da
	KeyFuncF10           KeyType = 0x1db
	KeyFuncF11           KeyType = 0x1dc
	KeyFuncF12           KeyType = 0x1dd
	KeyFunc1             KeyType = 0x1de
	KeyFunc2             KeyType = 0x1df
	KeyFuncD             KeyType = 0x1e0
	KeyFuncE             KeyType = 0x1e1
	KeyFuncF             KeyType = 0x1e2
	KeyFuncS             KeyType = 0x1e3
	KeyFuncB             KeyType = 0x1e4
	KeyBrailleDot1       KeyType = 0x1f1
	KeyBrailleDot2       KeyType = 0x1f2
	KeyBrailleDot3       KeyType = 0x1f3
	KeyBrailleDot4       KeyType = 0x1f4
	KeyBrailleDot5       KeyType = 0x1f5
	KeyBrailleDot6       KeyType = 0x1f6
	KeyBrailleDot7       KeyType = 0x1f7
	KeyBrailleDot8       KeyType = 0x1f8
	KeyBrailleDot9       KeyType = 0x1f9
	KeyBrailleDot10      KeyType = 0x1fa
	KeyNumeric0          KeyType = 0x200 // used by phones, remote controls,
	KeyNumeric1          KeyType = 0x201 // and other keypads
	KeyNumeric2          KeyType = 0x202
	KeyNumeric3          KeyType = 0x203
	KeyNumeric4          KeyType = 0x204
	KeyNumeric5          KeyType = 0x205
	KeyNumeric6          KeyType = 0x206
	KeyNumeric7          KeyType = 0x207
	KeyNumeric8          KeyType = 0x208
	KeyNumeric9          KeyType = 0x209
	KeyNumericStar       KeyType = 0x20a
	KeyNumericPound      KeyType = 0x20b
	KeyNumericA          KeyType = 0x20c // Phone key A - HUT Telephony 0xb9
	KeyNumericB          KeyType = 0x20d
	KeyNumericC          KeyType = 0x20e
	KeyNumericD          KeyType = 0x20f
	KeyCameraFocus       KeyType = 0x210
	KeyWPSButton         KeyType = 0x211 // WiFi Protected Setup key
	KeyTouchpadToggle    KeyType = 0x212 // Request switch touchpad on or off
	KeyTouchpadOn        KeyType = 0x213
	KeyTouchpadOff       KeyType = 0x214
	KeyCameraZoomIn      KeyType = 0x215
	KeyCameraZoomOut     KeyType = 0x216
	KeyCameraUp          KeyType = 0x217
	KeyCameraDown        KeyType = 0x218
	KeyCameraLeft        KeyType = 0x219
	KeyCameraRight       KeyType = 0x21a
	KeyAttendantOn       KeyType = 0x21b
	KeyAttendantOff      KeyType = 0x21c
	KeyAttendantToggle   KeyType = 0x21d // Attendant call on or off
	KeyLightsToggle      KeyType = 0x21e // Reading light on or off
	KeyAlsToggle         KeyType = 0x230 // Ambient light sensor
	KeyButtonConfig      KeyType = 0x240 // AL Button Configuration
	KeyTaskManager       KeyType = 0x241 // AL Task/Project Manager
	KeyJournal           KeyType = 0x242 // AL Log/Journal/Timecard
	KeyControlPanel      KeyType = 0x243 // AL Control Panel
	KeyAppSelect         KeyType = 0x244 // AL Select Task/Application
	KeyScreensaver       KeyType = 0x245 // AL Screen Saver
	KeyVoiceCommand      KeyType = 0x246 // Listening Voice Command
	KeyAssistant         KeyType = 0x247 // AL Context-aware desktop assistant
	KeyBrightnessMin     KeyType = 0x250 // Set Brightness to Minimum
	KeyBrightnessMax     KeyType = 0x251 // Set Brightness to Maximum
	KeyKbdInputPrev      KeyType = 0x260
	KeyKbdInputNext      KeyType = 0x261
	KeyKbdInputPrevGroup KeyType = 0x262
	KeyKbdInputNextGroup KeyType = 0x263
	KeyKbdInputAccept    KeyType = 0x264
	KeyKbdInputCancel    KeyType = 0x265
	KeyRightUp           KeyType = 0x266
	KeyRightDown         KeyType = 0x267
	KeyLeftUp            KeyType = 0x268
	KeyLeftDown          KeyType = 0x269
	KeyRootMenu          KeyType = 0x26a // Show Device's Root Menu
	KeyMediaTopMenu      KeyType = 0x26b
	KeyNumeric11         KeyType = 0x26c
	KeyNumeric12         KeyType = 0x26d
	KeyAudioDesc         KeyType = 0x26e
	Key3dMode            KeyType = 0x26f
	KeyNextFavorite      KeyType = 0x270
	KeyStopRecord        KeyType = 0x271
	KeyPauseRecord       KeyType = 0x272
)

type EventValue int32

const (
	KeyPressed EventValue = iota
	KeyReleased
)
