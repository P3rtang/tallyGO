package countable

import (
	"fmt"
	"math"
	EventBus "tallyGo/eventBus"
	"time"

	"gonum.org/v1/gonum/stat/distuv"
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

func (self *Counter) GetName() (name string) {
	return self.Name
}

func (self *Counter) SetName(name string) {
	self.Name = name
	EventBus.GetGlobalBus().SendSignal(NameChanged, self.Name)
}

func (self *Counter) NewPhase(progressType ProgressType) *Phase {
	phaseName := fmt.Sprintf("Phase_%d", len(self.Phases)+1)
	newPhase := &Phase{
		phaseName,
		0,
		time.Duration(0),
		nil,
		false,
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
		odds = 4096
	}

	return
}

func (self *Counter) GetRolls() (rolls int) {
	for _, phase := range self.Phases {
		rolls += phase.GetRolls()
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
	self.GetProgress()
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
	rolls := self.GetRolls()
	var completed int

	switch self.ProgressType {
	case SOS:
		completed = 1
	default:
		completed = len(self.Phases)
	}

	for _, phase := range self.Phases {
		phase.GetProgress()
	}

	binomial := distuv.Binomial{
		N:   float64(rolls),
		P:   1 / averageOdds,
		Src: nil,
	}

	return binomial.CDF(float64(completed - 1))
}

func (self *Counter) GetProgressType() ProgressType {
	return self.ProgressType
}

func (self *Counter) SetProgressType(type_ ProgressType) {
	self.ProgressType = type_
	for _, p := range self.Phases {
		p.SetProgressType(type_)
	}
	self.GetProgress()
}

func (self *Counter) HasCharm() bool {
	for _, p := range self.Phases {
		if !p.HasCharm() {
			return false
		}
	}
	return true
}

func (self *Counter) SetCharm(hasCharm bool) {
	for _, p := range self.Phases {
		p.SetCharm(hasCharm)
	}
	self.GetProgress()
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
	// when settings a counter to incomplete only the last phase should be unlocked
	if !isCompleted {
		self.Phases[len(self.Phases)-1].SetCompleted(false)
		return
	}

	// otherwise lock every phase when settings completed
	for _, p := range self.Phases {
		p.SetCompleted(isCompleted)
	}
}

func (self *Counter) Deviation() (deviation float64) {
	for _, p := range self.Phases {
		deviation += (float64(p.GetCount()) / self.GetOdds())
	}
	return
}
