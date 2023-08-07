package countable

import "time"

type Countable interface {
	GetName() (name string)
	SetName(name string)

	GetCount() int
	SetCount(num int)
	IncreaseBy(add int)

	AddTime(time.Duration)
	SetTime(time.Duration)
	GetTime() time.Duration

	GetProgress() float64
	GetProgressType() ProgressType
	HasCharm() bool
	SetCharm(bool)

	ConnectChanged(field string, f func())
	callback(field string)

	IsNil() bool
}

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

type CounterElement interface {
	SetCompleted(set bool)
	ConnectChanged(field string, callback func())
}
