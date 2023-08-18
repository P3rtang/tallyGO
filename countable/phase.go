package countable

import (
	"encoding/json"
	"fmt"
	"log"
	EventBus "tallyGo/eventBus"
	"time"
)

type Phase struct {
	Name     string
	Count    int
	Time     time.Duration
	Progress Progress

	IsCompleted bool
}

func (self *Phase) GetName() (name string) {
	return self.Name
}

func (self *Phase) SetName(name string) {
	self.Name = name
	EventBus.GetGlobalBus().SendSignal(NameChanged, self.Name)
}

func (self *Phase) GetRolls() int {
	return self.Progress.GetRolls()
}

func (self *Phase) GetCount() int {
	return self.Count
}

func (self *Phase) SetCount(num int) {
	self.Count = num
	self.UpdateProgress()
	EventBus.GetGlobalBus().SendSignal(CountChanged, self.Count)
}

func (self *Phase) IncreaseBy(add int) {
	if self.IsCompleted {
		return
	}
	self.Count += add
	self.UpdateProgress()
	EventBus.GetGlobalBus().SendSignal(CountChanged, self.Count)
}

func (self *Phase) GetTime() time.Duration {
	return self.Time
}

func (self *Phase) SetTime(time time.Duration) {
	self.Time = time
	EventBus.GetGlobalBus().SendSignal(TimeChanged, self.Time)
}

func (self *Phase) AddTime(time time.Duration) {
	if self.IsCompleted {
		return
	}
	self.Time += time
	EventBus.GetGlobalBus().SendSignal(TimeChanged, self.Time)
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
	self.UpdateProgress()
}

func (self *Phase) GetProgress() float64 {
	pr := self.Progress.GetProgress()
	return pr
}

func (self *Phase) GetProgressType() ProgressType {
	return self.Progress.GetType()
}

func (self *Phase) UpdateProgress() {
	self.Progress.SetRollsFromCount(self.Count)
}

func (self *Phase) HasCharm() bool {
	return self.Progress.Charm()
}

func (self *Phase) SetCharm(hasCharm bool) {
	self.Progress.SetCharm(hasCharm)
	self.UpdateProgress()
}

func (self *Phase) SetCompleted(isCompleted bool) {
	self.IsCompleted = isCompleted
	EventBus.GetGlobalBus().SendSignal(CompletedStatus, self)
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
