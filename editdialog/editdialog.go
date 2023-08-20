package editdialog

import (
	"fmt"
	"log"
	"reflect"
	"strconv"
	. "tallyGo/countable"
	"time"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

type EditDialog struct {
	*gtk.Dialog
	list      *gtk.Box
	buttonRow *gtk.Box

	rows map[string]interface{}

	countable Countable
}

func NewEditDialog(countable Countable) *EditDialog {
	var mainWindow *gtk.ApplicationWindow
	var ok bool
	if mainWindow, ok = gtk.WindowListToplevels()[0].(*gtk.ApplicationWindow); !ok {
		log.Println("[WARN]\tCould not find the root ApplicationWindow,\nis there more than one window opened")
	}

	window := gtk.NewDialog()
	window.SetResizable(false)
	window.SetTransientFor(&mainWindow.Window)
	window.SetModal(true)

	listBox := gtk.NewBox(gtk.OrientationVertical, 0)
	listBox.AddCSSClass("editDialogBox")
	window.SetChild(listBox)

	rows := make(map[string]interface{})
	this := EditDialog{window, listBox, gtk.NewBox(gtk.OrientationHorizontal, 0), rows, countable}

	switch countable.(type) {
	case *Phase:
		phase := countable.(*Phase)
		this.NewRow("Name", phase.Name)
		this.NewRow("Count", phase.Count)
		this.NewRow("Time", phase.Time)
		this.NewRow("HuntType", fmt.Sprint(phase.Progress.GetType()))
		this.AddButton("cancel", func() {
			this.Close()
		})
		this.AddButton("confirm", func() {
			if name, ok := this.rows["Name"].(string); ok {
				phase.SetName(name)
			}
			if count, ok := this.rows["Count"].(int); ok {
				println(count)
				phase.SetCount(count)
			}
			if duration, ok := this.rows["Time"].(time.Duration); ok {
				phase.SetTime(duration)
			}
			if type_str, ok := this.rows["HuntType"].(string); ok {
				type_, _ := strconv.Atoi(type_str)
				phase.SetProgressType(ProgressType(type_))
			}
			this.Close()
		})
		break
	case *Counter:
		counter := countable.(*Counter)
		this.NewRow("Name", counter.Name)
		this.NewRow("Count", counter.GetCount())
		this.NewRow("HuntType", fmt.Sprint(counter.ProgressType))
		this.NewRow("Shiny Charm", counter.HasCharm())
		this.AddButton("cancel", func() {
			this.Close()
		})
		this.AddButton("confirm", func() {
			if name, ok := this.rows["Name"].(string); ok {
				counter.SetName(name)
			}
			if count, ok := this.rows["Count"].(int); ok {
				counter.SetCount(count)
			}
			if type_str, ok := this.rows["HuntType"].(string); ok {
				type_, _ := strconv.Atoi(type_str)
				counter.SetProgressType(ProgressType(type_))
			}
			if hasCharm, ok := this.rows["Shiny Charm"].(bool); ok {
				counter.SetCharm(hasCharm)
			}
			this.Close()
		})
		break
	}

	this.list.Append(this.buttonRow)
	this.buttonRow.AddCSSClass("editDialogButtonRow")
	this.buttonRow.SetHAlign(gtk.AlignEnd)

	return &this
}

func (self *EditDialog) NewRow(title string, value interface{}) {
	self.rows[title] = value

	switch value.(type) {
	case int:
		row := NewDialogRow(title, value.(int))
		self.list.Append(row)
		row.entry.ConnectChanged(func() { self.rows[title] = int(row.entry.Int()) })
		break
	case string:
		row := NewDialogRow(title, value.(string))
		self.list.Append(row)
		row.entry.ConnectChanged(func() { self.rows[title] = row.entry.Text() })
		break
	case time.Duration:
		row := NewDialogTimeRow(title, value.(time.Duration))
		self.list.Append(row)

		row.ConnectChanged(func() {
			dur := time.Hour.Nanoseconds()*row.hours.Int() + time.Minute.Nanoseconds()*row.mins.Int()
			self.rows[title] = time.Duration(dur)
		})
	case bool:
		row := NewDialogBoolRow(title, value.(bool))
		self.list.Append(row)

		row.ConnectChanged(func() {
			self.rows[title] = row.state.Active()
		})
	}
}

func (self *EditDialog) AddButton(name string, clickCallback func()) {
	button := gtk.NewButtonWithLabel(name)
	button.ConnectClicked(clickCallback)
	self.buttonRow.Append(button)
}

type DialogBoolRow struct {
	*gtk.Box
	state *gtk.CheckButton
}

func NewDialogBoolRow(title string, value bool) (self *DialogBoolRow) {
	self = &DialogBoolRow{
		Box:   gtk.NewBox(gtk.OrientationHorizontal, 0),
		state: nil,
	}
	self.Box.AddCSSClass("editDialogRow")

	titleLabel := gtk.NewLabel(title)
	titleLabel.SetHExpand(true)
	titleLabel.SetHAlign(gtk.AlignStart)
	self.Append(titleLabel)

	switchButton := gtk.NewCheckButton()
	switchButton.SetActive(value)
	self.Append(switchButton)
	self.state = switchButton

	return
}

func (self *DialogBoolRow) ConnectChanged(callback func()) {
	self.state.ConnectToggled(callback)
}

type DialogTimeRow struct {
	*gtk.Box
	hours *TypedEntry[int]
	mins  *TypedEntry[int]
}

func NewDialogTimeRow(title string, value time.Duration) *DialogTimeRow {
	row := gtk.NewBox(gtk.OrientationHorizontal, 0)
	row.AddCSSClass("editDialogRow")
	entryHour := NewTypedEntry(int(value.Hours()))
	entryMins := NewTypedEntry(int(value.Minutes()) % 60)

	entryHour.AddCSSClass("entryEditDialog")
	entryMins.AddCSSClass("entryEditDialog")
	entryHour.SetMaxWidthChars(4)
	entryMins.SetMaxWidthChars(2)
	entryHour.SetAlignment(1)
	entryMins.SetAlignment(1)
	entryMins.SetMaxLength(2)

	titleLabel := gtk.NewLabel(title)
	titleLabel.SetHExpand(true)
	titleLabel.SetHAlign(gtk.AlignStart)
	row.Append(titleLabel)
	row.Append(entryHour)
	row.Append(gtk.NewLabel(" h :   "))
	row.Append(entryMins)
	row.Append(gtk.NewLabel(" m "))

	return &DialogTimeRow{row, entryHour, entryMins}
}

func (self *DialogTimeRow) ConnectChanged(f func()) {
	self.hours.ConnectChanged(f)
	self.mins.ConnectChanged(f)
}

type DialogRow[T EntryType] struct {
	*gtk.Box
	entry *TypedEntry[T]
}

func NewDialogRow[T EntryType](title string, value T) *DialogRow[T] {
	row := gtk.NewBox(gtk.OrientationHorizontal, 0)
	row.AddCSSClass("editDialogRow")
	entry := NewTypedEntry(value)
	entry.SetAlignment(1)
	entry.AddCSSClass("entryEditDialog")
	entry.SetMaxWidthChars(14)

	titleLabel := gtk.NewLabel(title)
	titleLabel.SetHExpand(true)
	titleLabel.SetHAlign(gtk.AlignStart)
	row.Append(titleLabel)
	row.Append(entry)

	return &DialogRow[T]{row, entry}
}

type EntryType interface {
	int | ~string
}

type TypedEntry[T EntryType] struct {
	*gtk.Entry
}

func NewTypedEntry[T EntryType](value T) *TypedEntry[T] {
	entry := gtk.NewEntry()

	switch reflect.TypeOf(value).Kind() {
	case reflect.Int:
		entry.ConnectChanged(func() {
			text := entry.Text()
			if _, err := strconv.Atoi(text); err != nil && text != "" {
				glib.IdleAdd(func() {
					entry.DeleteText(len(text)-1, len(text))
				})
			}
		})

	}

	this := &TypedEntry[T]{entry}
	this.SetValue(value)

	return &TypedEntry[T]{entry}
}

func (self *TypedEntry[T]) SetValue(value T) {
	self.SetText(fmt.Sprint(value))
}

func (self *TypedEntry[int]) Int() int64 {
	input := self.Entry.Text()
	value, _ := strconv.Atoi(input)
	return int64(value)
}
