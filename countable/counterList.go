package countable

import (
	EventBus "tallyGo/eventBus"

	"gonum.org/v1/gonum/stat/distuv"
)

type CounterList struct {
	List   []*Counter
	active []Countable
}

func NewCounterList(list []*Counter) (self *CounterList) {
	self = &CounterList{list, nil}
	self.setupListeners()
	return
}

func (self *CounterList) setupListeners() {
	EventBus.GetGlobalBus().Subscribe(RemoveCounter, func(args ...interface{}) {
		self.RemoveCounter(args[0].(*Counter))
	})

	EventBus.GetGlobalBus().Subscribe(RemovePhase, func(args ...interface{}) {
		self.RemovePhase(args[0].(*Phase))
	})
}

func (self *CounterList) GetActive() []Countable {
	return self.active
}

func (self *CounterList) HasActive() bool {
	return len(self.active) > 0
}

func (self *CounterList) SetActive(countables ...Countable) {
	self.active = countables
	data := []interface{}{}
	for _, c := range self.active {
		data = append(data, c)
	}
	EventBus.GetGlobalBus().SendSignal(ListActiveChanged, data...)
}

func (self *CounterList) RemoveCounter(counter *Counter) {
	if idx, ok := self.GetIdx(counter); ok {
		if idx < len(self.List)-1 {
			self.List = append(self.List[:idx], self.List[idx+1:len(self.List)]...)

		} else {
			self.List = self.List[:idx]
		}
	}
	EventBus.GetGlobalBus().SendSignal(CounterRemoved, counter)
}

func (self *CounterList) RemovePhase(phase *Phase) {
	for _, c := range self.List {
		if c.hasPhase(phase) {
			c.RemovePhase(phase)
		}
	}
}

func (self *CounterList) GetIdx(counter *Counter) (int, bool) {
	for idx, c := range self.List {
		if c == counter {
			return idx, true
		}
	}
	return 0, false
}

func (self *CounterList) Deviation() (deviation float64) {
	for _, c := range self.List {
		deviation += c.Deviation()
	}

	return
}

func (self *CounterList) Completed() (completed int) {
	for _, c := range self.List {
		switch c.GetProgressType() {
		case SOS:
			completed += 1
		default:
			completed += len(c.Phases)
		}
	}
	return
}

func (self *CounterList) AverageOdds() (odds float64) {
	for _, c := range self.List {
		odds += c.GetOdds() * float64(c.GetRolls())
	}
	return odds / float64(self.TotalRolls())
}

func (self *CounterList) TotalCount() (count int) {
	for _, c := range self.List {
		count += c.GetCount()
	}
	return
}

func (self *CounterList) TotalRolls() (rolls int) {
	for _, c := range self.List {
		rolls += c.GetRolls()
	}
	return
}

func (self *CounterList) Luck() float64 {
	averageOdds := self.AverageOdds()
	rolls := self.TotalRolls()
	completed := self.Completed()

	binomial := distuv.Binomial{
		N:   float64(rolls),
		P:   1 / averageOdds,
		Src: nil,
	}

	return binomial.CDF(float64(completed - 1))
}
