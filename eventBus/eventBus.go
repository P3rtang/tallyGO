package EventBus

var GlobalEvents *EventBus

func InitBus() {
	GlobalEvents = NewEventBus()
}

func GetGlobalBus() *EventBus {
	return GlobalEvents
}

type Signal string

type Event struct {
	kind Signal
	data []interface{}
}

func NewEvent(kind Signal, data ...interface{}) Event {
	return Event{kind, data}
}

type EventBus struct {
	callbacks map[Signal][]func(args ...interface{})
}

func NewEventBus() (self *EventBus) {
	return &EventBus{
		callbacks: map[Signal][]func(args ...interface{}){},
	}
}

func (self *EventBus) SendSignal(signal Signal, args ...interface{}) {
	for _, f := range self.callbacks[signal] {
		f(args...)
	}
}

func (self *EventBus) Send(event Event) {
	for _, f := range self.callbacks[event.kind] {
		f(event.data...)
	}
}

func (self *EventBus) Subscribe(kind Signal, callback func(...interface{})) {
	self.callbacks[kind] = append(self.callbacks[kind], callback)
}
