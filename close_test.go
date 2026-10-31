package main

import "testing"

type fakeShell struct {
	dirty   bool
	answer  string
	asked   int
	emitted []string
}

func (f *fakeShell) FrontendDirty() bool { return f.dirty }
func (f *fakeShell) AskSaveDiscardCancel(string) string {
	f.asked++
	return f.answer
}
func (f *fakeShell) Emit(e string) { f.emitted = append(f.emitted, e) }

func TestStayOpen(t *testing.T) {
	cases := []struct {
		name      string
		dirty     bool
		answer    string
		stay      bool
		asked     int
		saveEvent bool
	}{
		{name: "clean closes without asking"},
		{name: "cancel stays", dirty: true, answer: answerCancel, stay: true, asked: 1},
		{name: "discard closes", dirty: true, answer: answerDiscard, asked: 1},
		{name: "save stays until the frontend finishes", dirty: true, answer: answerSave, stay: true, asked: 1, saveEvent: true},
		{name: "unknown answer stays", dirty: true, answer: "", stay: true, asked: 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a := NewApp()
			sh := &fakeShell{dirty: c.dirty, answer: c.answer}
			if got := a.stayOpen(sh); got != c.stay {
				t.Errorf("stayOpen = %v", got)
			}
			if sh.asked != c.asked {
				t.Errorf("asked %d times", sh.asked)
			}
			if got := len(sh.emitted) == 1; got != c.saveEvent {
				t.Errorf("emitted %v", sh.emitted)
			}
		})
	}
}

// After Discard, or ForceQuit following a save, the re-entrant close must not ask again.
func TestForcedCloseDoesNotAskAgain(t *testing.T) {
	a := NewApp()
	discard := &fakeShell{dirty: true, answer: answerDiscard}
	a.stayOpen(discard)
	again := &fakeShell{dirty: true, answer: answerCancel}
	if a.stayOpen(again) || again.asked != 0 {
		t.Fatalf("asked %d times after Discard", again.asked)
	}
}
