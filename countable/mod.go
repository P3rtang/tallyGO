package countable

import (
	EventBus "tallyGo/eventBus"
	"time"
)

type Countable interface {
	GetName() (name string)
	SetName(name string)

	GetCount() int
	SetCount(num int)
	IncreaseBy(add int)

	GetTime() time.Duration
	SetTime(time.Duration)
	AddTime(time.Duration)

	HasCharm() bool
	SetCharm(bool)

	GetProgress() float64
	GetProgressType() ProgressType
}

const (
	// callback arguments (Countable)
	NameChanged EventBus.Signal = "NameChanged"
	// callback arguments (Countable)
	CountChanged = "CountChanged"
	// callback arguments (Countable)
	TimeChanged = "TimeChanged"
	// callback arguments (Countable)
	CompletedStatus = "CompletedStatus"

	// callback arguments ([]Countable)
	ListActiveChanged EventBus.Signal = "ListActiveChanged"

	// callback arguments (*Counter)
	CounterAdded = "CounterAdded"
	// callback arguments (*Counter)
	RemoveCounter = "RemoveCounter"
	// callback arguments (*Counter)
	CounterRemoved = "CounterRemoved"

	// callback arguments (*Counter, newPhase)
	PhaseAdded = "PhaseAdded"
	// callback arguments (*Phase)
	RemovePhase = "RemovePhase"
	// callback arguments (*Counter, *Phase)
	PhaseRemoved = "PhaseRemoved"
)

type ProgressType int

const (
	OldOdds ProgressType = iota
	NewOdds
	SOS
	DexNav
)

type OldProgress struct {
	Type     ProgressType
	Progress float64
}

func newProgress(type_ ProgressType) *OldProgress {
	return &OldProgress{
		type_,
		0.0,
	}
}

func (self ProgressType) HasPhases() bool {
	switch self {
	case OldOdds | NewOdds:
		return true
	default:
		return false
	}
}

func (self ProgressType) IsChain() bool {
	switch self {
	case SOS | DexNav:
		return true
	default:
		return false
	}
}

type callBackType int

const (
	Count callBackType = iota
	Time
)
