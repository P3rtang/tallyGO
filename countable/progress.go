package countable

import (
	"encoding/json"
	"log"
	"math"
)

type Progress interface {
	SetRollsFromCount(count int)
	SetCharm(hasCharm bool)
	GetProgress() float64
	GetRolls() int
	GetType() ProgressType
	MarshalJSON() (bytes []byte, err error)
}

type DefaultOdds struct {
	Odds  float64
	Rolls int

	HasCharm bool
	Progress float64
}

func NewDefaultOdds(odds float64, count int, hasCharm bool) (self *DefaultOdds) {
	self = &DefaultOdds{odds, 0, hasCharm, 0.0}
	self.SetRollsFromCount(count)
	return
}

func (self *DefaultOdds) SetRollsFromCount(count int) {
	if self.HasCharm {
		count *= 3
	}
	self.Rolls = count
	self.GetProgress()
}

func (self *DefaultOdds) SetCharm(hasCharm bool) {
	self.HasCharm = hasCharm
}

func (self *DefaultOdds) GetProgress() float64 {
	self.Progress = math.Pow(1-1/self.Odds, float64(self.Rolls))
	return self.Progress
}

func (self *DefaultOdds) GetRolls() int {
	return self.Rolls
}

func (self *DefaultOdds) GetType() ProgressType {
	if self.Odds == 8192 {
		return OldOdds
	} else {
		return NewOdds
	}
}

func (self *DefaultOdds) MarshalJSON() (bytes []byte, err error) {
	m := map[string]interface{}{}
	if err != nil {
		log.Println("Could not marshal DefaultOdds ProgressType, Got Error: ", err)
		return
	}

	m["type"] = "DefaultOdds"
	m["Odds"] = self.Odds
	m["Rolls"] = self.Rolls
	m["HasCharm"] = self.HasCharm
	m["Progress"] = self.Progress
	return json.Marshal(m)
}

type SOSBattle struct {
	Rolls int

	HasCharm bool
	Progress float64
}

func NewSOSBattle(count int, hasCharm bool) (self *SOSBattle) {
	self = &SOSBattle{0, hasCharm, 0.0}
	self.SetRollsFromCount(count)
	return
}

func (self *SOSBattle) SetRollsFromCount(count int) {
	self.Rolls = 0
	for i := 0; i <= count; i++ {
		switch {
		case i > 30:
			self.Rolls += 13
		case i > 20:
			self.Rolls += 9
		case i > 10:
			self.Rolls += 5
		default:
			self.Rolls += 1
		}
	}

	if self.HasCharm {
		self.Rolls += count * 2
	}

	self.GetProgress()
}

func (self *SOSBattle) SetCharm(hasCharm bool) {
	self.HasCharm = hasCharm
}

func (self *SOSBattle) GetProgress() float64 {
	self.Progress = math.Pow(4095.0/4096.0, float64(self.Rolls))
	return self.Progress
}

func (self *SOSBattle) GetRolls() int {
	return self.Rolls
}

func (self *SOSBattle) GetType() ProgressType {
	return SOS
}

func (self *SOSBattle) MarshalJSON() (bytes []byte, err error) {
	m := map[string]interface{}{}

	m["type"] = "SOSBattle"
	m["Rolls"] = self.Rolls
	m["HasCharm"] = self.HasCharm
	m["Progress"] = self.Progress
	return json.Marshal(m)
}
