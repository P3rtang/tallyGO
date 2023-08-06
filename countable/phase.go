package countable

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
)

type Phase struct {
	Name     string
	Count    int
	Time     time.Duration
	Progress Progress

	IsCompleted bool

	callbackChange map[string][]func()
}

func (self *Phase) ConnectChanged(field string, f func()) {
	if self.callbackChange == nil {
		self.callbackChange = map[string][]func(){}
	}
	self.callbackChange[field] = append(self.callbackChange[field], f)
}

func (self *Phase) GetName() (name string) {
	return self.Name
}

func (self *Phase) SetName(name string) {
	self.Name = name
	self.callback("Name")
}

func (self *Phase) GetCount() int {
	return self.Count
}

func (self *Phase) SetCount(num int) {
	self.Count = num
	self.UpdateProgress()
	self.callback("Count")
}

func (self *Phase) IncreaseBy(add int) {
	if self.IsCompleted {
		return
	}
	self.Count += add
	self.UpdateProgress()
	self.callback("Count")
}

func (self *Phase) GetTime() time.Duration {
	return self.Time
}

func (self *Phase) SetTime(time time.Duration) {
	self.Time = time
	self.callback("Time")
}

func (self *Phase) AddTime(time time.Duration) {
	if self.IsCompleted {
		return
	}
	self.Time += time
	self.callback("Time")
}

func (self *Phase) SetProgressType(type_ ProgressType) {
	switch type_ {
	case NewOdds:
		self.Progress = NewDefaultOdds(4096, self.Count, false)
	case SOS:
		self.Progress = NewSOSBattle(self.Count, false)
	default:
		self.Progress = NewDefaultOdds(8192, self.Count, false)
	}
}

func (self *Phase) GetProgress() float64 {
	return self.Progress.GetProgress()
}

func (self *Phase) GetProgressType() ProgressType {
	return self.Progress.GetType()
}

func (self *Phase) UpdateProgress() {
	self.Progress.SetRollsFromCount(self.Count)
}

func (self *Phase) SetCompleted(isCompleted bool) {
	self.IsCompleted = isCompleted
	self.callback("IsCompleted")
}

func (self *Phase) callback(field string) {
	for _, f := range self.callbackChange[field] {
		f()
	}
}

func (self *Phase) IsNil() bool {
	return self == nil
}

func (self *Phase) UnmarshalJSON(bytes []byte) (err error) {
	var objMap map[string]*json.RawMessage
	err = json.Unmarshal(bytes, &objMap)

	var phaseObjMap map[string]interface{}
	err = json.Unmarshal(bytes, &phaseObjMap)

	self.Name = phaseObjMap["Name"].(string)
	self.Count = int(phaseObjMap["Count"].(float64))
	self.Time = time.Duration(phaseObjMap["Time"].(float64))
	self.IsCompleted = phaseObjMap["IsCompleted"].(bool)

	var progressObjMap map[string]interface{}
	err = json.Unmarshal(*objMap["Progress"], &progressObjMap)

	fmt.Println("ProgressMap: ", progressObjMap)

	switch progressObjMap["type"] {
	case "DefaultOdds":
		var progress Progress = &DefaultOdds{
			Odds:     progressObjMap["Odds"].(float64),
			Rolls:    int(progressObjMap["Rolls"].(float64)),
			HasCharm: progressObjMap["HasCharm"].(bool),
			Progress: progressObjMap["Progress"].(float64),
		}
		self.Progress = progress
	case "SOSBattle":
		var progress Progress = &SOSBattle{
			Rolls:    int(progressObjMap["Rolls"].(float64)),
			HasCharm: progressObjMap["HasCharm"].(bool),
			Progress: progressObjMap["Progress"].(float64),
		}
		self.Progress = progress
	default:
		log.SetFlags(log.Llongfile)
		log.Fatal("Unhandled ProgressType")
	}

	return
}
