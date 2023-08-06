package countable

import (
	"fmt"
	"math"
	"math/big"
	"time"
)

type Counter struct {
	Name         string
	Phases       []*Phase
	ProgressType ProgressType

	callbackChange map[string][]func()
}

func NewCounter(name string, _ int, progressType ProgressType) (counter *Counter) {
	counter = &Counter{name, []*Phase{}, progressType, nil}
	counter.NewPhase(progressType)
	return
}

func (self *Counter) ConnectChanged(field string, f func()) {
	if self.callbackChange == nil {
		self.callbackChange = map[string][]func(){}
	}
	if field == "Name" {
		self.callbackChange[field] = append(self.callbackChange[field], f)
	} else {
		for _, phase := range self.Phases {
			phase.ConnectChanged(field, f)
		}
	}
}

func (self *Counter) GetName() (name string) {
	return self.Name
}

func (self *Counter) SetName(name string) {
	self.Name = name
	self.callback("Name")
}

func (self *Counter) NewPhase(progressType ProgressType) *Phase {
	phaseName := fmt.Sprintf("Phase_%d", len(self.Phases)+1)
	newPhase := &Phase{
		phaseName,
		0,
		time.Duration(0),
		nil,
		false,
		map[string][]func(){},
	}

	newPhase.SetProgressType(progressType)
	self.Phases = append(self.Phases, newPhase)

	return newPhase
}

func (self *Counter) GetChance() (chance float64) {
	chance = math.Pow(1-1/float64(self.GetOdds()), float64(self.GetCount()))
	return
}

func (self *Counter) GetOdds() (odds float64) {
	switch self.ProgressType {
	case OldOdds:
		odds = 8192
	case NewOdds:
		odds = 4096
	case SOS:
		var totalRolls int
		for _, chain := range self.Phases {
			totalRolls += chain.Progress.GetRolls()
		}
		odds = 1 / (1 - math.Pow((1-1/float64(4096)), float64(totalRolls)/float64(self.GetCount())))
	}

	return
}

func (self *Counter) GetCount() (count int) {
	for _, phase := range self.Phases {
		count += phase.Count
	}
	return
}

func (self *Counter) SetCount(num int) {
	diff := num - self.GetCount()
	self.Phases[len(self.Phases)-1].IncreaseBy(diff)
}

func (self *Counter) IncreaseBy(add int) {
	self.Phases[len(self.Phases)-1].IncreaseBy(add)
	self.UpdateProgress()
}

func (self *Counter) GetTime() (time time.Duration) {
	for _, phase := range self.Phases {
		time += phase.Time
	}
	return
}

func (self *Counter) SetTime(time time.Duration) {
	diff := self.GetTime() - time
	self.Phases[len(self.Phases)-1].AddTime(diff)
}

func (self *Counter) AddTime(time time.Duration) {
	self.Phases[len(self.Phases)-1].AddTime(time)
}

func (self *Counter) GetProgress() (progress float64) {
	averageOdds := self.GetOdds()
	count := self.GetCount()

	var deviation float64

	for i := 0; i < len(self.Phases); i++ {
		var nChooseK big.Int
		nChooseK.Binomial(int64(count), int64(i))

		deviation += float64(nChooseK.Int64()) *
			math.Pow((1/averageOdds), float64(i)) *
			math.Pow(1-1/averageOdds, float64(count))
	}

	return deviation
}

func (self *Counter) GetProgressType() ProgressType {
	return self.ProgressType
}

func (self *Counter) SetProgressType(type_ ProgressType) {
	self.ProgressType = type_
}

func (self *Counter) UpdateProgress() {
	for _, phase := range self.Phases {
		phase.UpdateProgress()
	}
}

func (self *Counter) IsCompleted() (isCompleted bool) {
	for _, p := range self.Phases {
		if !p.IsCompleted {
			return false
		}
	}
	return true
}

func (self *Counter) SetCompleted(isCompleted bool) {
	for _, p := range self.Phases {
		p.SetCompleted(true)
	}
	self.callback("IsCompleted")
}

func (self *Counter) Deviation() (deviation float64) {
	for _, p := range self.Phases {
		deviation += (float64(p.GetCount()) / self.GetOdds())
	}
	return
}

func (self *Counter) callback(field string) {
	if field == "Name" {
		for _, f := range self.callbackChange[field] {
			f()
		}
	} else {
		for _, p := range self.Phases {
			p.callback(field)
		}
	}
}

func (self *Counter) IsNil() bool {
	return self == nil
}
