package mm_test

import (
	"testing"

	"github.com/inkel/mm"
)

// https://github.com/soveran/micromachine/blob/master/test/transitions.rb

func TestNew(t *testing.T) {
	initialState := "pending"
	m := mm.New(initialState)
	if initialState != m.State() {
		t.Fatalf("expecting initial state of %q, got %q", initialState, m.State())
	}
}

func TestMachine_When(t *testing.T) {
	var err error

	m := mm.New("pending")

	err = m.When("confirm", mm.Transitions{"pending": "confirmed"})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}

	err = m.When("confirm", mm.Transitions{"foo": "bar"})
	if err != mm.ErrEventExists {
		t.Fatalf("expecting error mm.ErrEventExists, got %v", err)
	}
}

func newMachine() *mm.Machine {
	m := mm.New("pending")
	m.When("confirm", mm.Transitions{"pending": "confirmed"})
	m.When("ignore", mm.Transitions{"pending": "ignored"})
	m.When("reset", mm.Transitions{"confirmed": "pending", "ignored": "pending"})
	return m
}

func TestMachine_Events(t *testing.T) {
	m := newMachine()

	events := []string{"confirm", "ignore", "reset"}
	if !equal(events, m.Events()) {
		t.Errorf("expecting %v events, got %v", events, m.Events())
	}
}

func TestMachine_TriggerableEvents(t *testing.T) {
	m := mm.New("pending")

	if e := m.TriggerableEvents(); e != nil {
		t.Fatalf("expecting no events, got %v", e)
	}

	m = newMachine()

	triggerableEvents := []string{"confirm", "ignore"}
	if !equal(triggerableEvents, m.TriggerableEvents()) {
		t.Errorf("expecting %v triggerable events, got %v", triggerableEvents, m.TriggerableEvents())
	}
}

func TestMachine_States(t *testing.T) {
	m := mm.New("pending")

	if s := m.States(); s == nil || len(s) != 1 || s[0] != "pending" {
		t.Fatalf("expecting states with at least initial state, got %v", s)
	}

	m = newMachine()

	states := []string{"pending", "confirmed", "ignored"}
	if !equal(states, m.States()) {
		t.Errorf("expecting %v states, got %v", states, m.States())
	}
}

func TestMachine_Trigger(t *testing.T) {
	t.Run("error on invalid event", func(t *testing.T) {
		m := newMachine()
		if err := m.Trigger("random_event"); err != mm.ErrInvalidEvent {
			t.Fatalf("expecting mm.ErrInvalidEvent, got %v", err)
		}
	})

	t.Run("change state if transition is possible", func(t *testing.T) {
		m := newMachine()
		if err := m.Trigger("confirm"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if "confirmed" != m.State() {
			t.Fatalf("expecting \"confirmed\" state, got: %v", m.State())
		}
	})

	t.Run("fail and preserve state if transition is not possible", func(t *testing.T) {
		m := newMachine()
		if err := m.Trigger("reset"); err != mm.ErrInvalidState {
			t.Fatalf("expecting mm.ErrInvalidState, got %v", err)
		}
		if "pending" != m.State() {
			t.Fatalf("state changed when it shouldn't: %v", m.State())
		}
	})

	t.Run("multiple transitions", func(t *testing.T) {
		trs := []struct{ event, state string }{
			{"confirm", "confirmed"},
			{"reset", "pending"},
			{"ignore", "ignored"},
			{"reset", "pending"},
		}
		m := newMachine()

		for _, tr := range trs {
			if err := m.Trigger(tr.event); err != nil {
				t.Fatalf("unexpected %v when triggering %v", err, tr.event)
			}
			if tr.state != m.State() {
				t.Fatalf("expecting state %v, got %v", tr.state, m.State())
			}
		}
	})
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for _, ea := range a {
		found := false
		for _, eb := range b {
			if ea == eb {
				found = true
				break
			}
		}
		if found == false {
			return false
		}
	}

	return true
}

func TestMachine_On(t *testing.T) {
	m := newMachine()

	var state, stateAny, event, eventAny string

	m.On("pending", func(e string) { state, event = "Pending", e })
	m.On("confirmed", func(e string) { state, event = "Confirmed", e })
	m.OnAny(func(e string) { stateAny, eventAny = state, e })

	cases := []struct{ event, state string }{
		{"confirm", "Confirmed"},
		{"reset", "Pending"},
	}

	for _, c := range cases {
		if err := m.Trigger(c.event); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if state != c.state {
			t.Fatalf("expecting state %v, got %v", c.state, state)
		}
		if stateAny != c.state {
			t.Fatalf("expecting current %v, got %v", c.state, stateAny)
		}
		if event != c.event {
			t.Fatalf("expecting event %v, got %v", c.event, event)
		}
		if eventAny != c.event {
			t.Fatalf("expecting eventAny %v, got %v", c.event, event)
		}
	}

	// OnAny is called even if no callback is defined for that event
	state, stateAny, event, eventAny = "", "", "", ""
	if err := m.Trigger("ignore"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state != "" {
		t.Fatalf("expecting empty state, got %v", state)
	}
	if stateAny != "" {
		t.Fatalf("expecting empty current, got %v", stateAny)
	}
	if event != "" {
		t.Fatalf("expecting empty event, got %v", event)
	}
	if eventAny != "ignore" {
		t.Fatalf("expecting eventAny ignore, got %v", event)
	}

	// Multiple callbacks
	if err := m.Trigger("reset"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var cb1, cb2, cb3 bool
	m.On("ignored", func(e string) { cb1 = true })
	m.On("ignored", func(e string) { cb2 = true })
	m.On("ignored", func(e string) { cb3 = true })
	if err := m.Trigger("ignore"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cb1 || !cb2 || !cb3 {
		t.Fatalf("not all callbacks were executed: %v, %v, %v", cb1, cb2, cb3)
	}
}
