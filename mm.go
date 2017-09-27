package mm

import (
	"errors"
)

// https://github.com/soveran/micromachine/blob/master/lib/micromachine.rb

type Machine struct {
	state       string
	events      map[string]Transitions
	callbacks   map[string][]Callback
	callbackAny Callback
}

func New(state string) *Machine {
	return &Machine{
		state:     state,
		events:    make(map[string]Transitions),
		callbacks: make(map[string][]Callback),
	}
}

func (m *Machine) State() string {
	return m.state
}

type Transitions = map[string]string

var ErrEventExists = errors.New("event exists")

func (m *Machine) When(event string, transitions Transitions) error {
	if _, ok := m.events[event]; ok {
		return ErrEventExists
	}
	m.events[event] = transitions
	return nil
}

func (m *Machine) Events() []string {
	var events []string
	for event := range m.events {
		events = append(events, event)
	}
	return events
}

func (m *Machine) TriggerableEvents() []string {
	var events []string

	for event, transitions := range m.events {
		if _, ok := transitions[m.state]; ok {
			events = append(events, event)
		}
	}

	return events
}

func (m *Machine) States() []string {
	var (
		states = []string{m.state}
		found  = map[string]bool{m.state: true}
	)

	for _, transitions := range m.events {
		for state := range transitions {
			if !found[state] {
				found[state] = true
				states = append(states, state)
			}
		}
	}

	return states
}

var ErrInvalidEvent = errors.New("invalid event")
var ErrInvalidState = errors.New("invalid state")

func (m *Machine) Trigger(event string) error {
	tr, ok := m.events[event]
	if !ok {
		return ErrInvalidEvent
	}

	state, ok := tr[m.state]
	if !ok {
		return ErrInvalidState
	}

	m.state = state

	if cbs, ok := m.callbacks[state]; ok {
		for _, cb := range cbs {
			cb(event)
		}
	}

	if m.callbackAny != nil {
		m.callbackAny(event)
	}

	return nil
}

type Callback = func(string)

func (m *Machine) On(event string, fn Callback) {
	m.callbacks[event] = append(m.callbacks[event], fn)
}

func (m *Machine) OnAny(fn Callback) {
	m.callbackAny = fn
}
