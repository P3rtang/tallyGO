package countable

import (
	"fmt"

	"gonum.org/v1/gonum/stat/distuv"
)

type CounterList struct {
	List []*Counter
}

func NewCounterList(list []*Counter) *CounterList {
	return &CounterList{list}
}

func (self *CounterList) Remove(counter *Counter) {
	if idx, ok := self.GetIdx(counter); ok {
		if idx < len(self.List)-1 {
			self.List = append(self.List[:idx], self.List[idx+1:len(self.List)]...)

		} else {
			self.List = self.List[:idx]
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
		completed += len(c.Phases)
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

	fmt.Println(averageOdds, rolls, completed)

	binomial := distuv.Binomial{
		N:   float64(rolls),
		P:   1 / averageOdds,
		Src: nil,
	}

	return binomial.CDF(float64(completed - 1))
}
