package main

import (
	"context"
	"time"

	wailsrt "github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	answerSave    = "save"
	answerDiscard = "discard"
	answerCancel  = "cancel"
)

// shell is what the close flow needs from the desktop runtime, so that a decision with
// data-loss consequences can be tested without a window.
type shell interface {
	// FrontendDirty asks the frontend, which alone knows, whether edits are unsaved.
	FrontendDirty() bool
	// AskSaveDiscardCancel returns an answer* constant.
	AskSaveDiscardCancel(message string) string
	Emit(event string)
}

// stayOpen decides a close request: true keeps the window. The document lives in the editor, so
// "Save" hands off to the frontend, which saves and calls ForceQuit once that has worked.
func (a *App) stayOpen(sh shell) bool {
	if a.forceClose.Load() || !sh.FrontendDirty() {
		return false
	}
	switch sh.AskSaveDiscardCancel("You have unsaved changes. Save them before closing?") {
	case answerDiscard:
		a.forceClose.Store(true)
		return false
	case answerSave:
		sh.Emit("app:save-and-quit")
		return true
	default:
		return true
	}
}

// wailsShell is the real shell.
type wailsShell struct{ ctx context.Context }

// dirtyTimeout bounds the wait for an unresponsive webview.
const dirtyTimeout = 2 * time.Second

// FrontendDirty round-trips through events. No answer counts as dirty, so a hung webview
// cannot silently drop the user's data on quit.
func (s wailsShell) FrontendDirty() bool {
	ch := make(chan bool, 1)
	wailsrt.EventsOnce(s.ctx, "app:dirty-response", func(args ...any) {
		dirty, ok := true, false
		if len(args) > 0 {
			dirty, ok = args[0].(bool)
		}
		ch <- dirty || !ok
	})
	wailsrt.EventsEmit(s.ctx, "app:check-dirty")
	select {
	case d := <-ch:
		return d
	case <-time.After(dirtyTimeout):
		return true
	}
}

// AskSaveDiscardCancel shows the native dialog. A dialog failure counts as Cancel: staying
// open is the safe outcome when the alternative may lose data.
func (s wailsShell) AskSaveDiscardCancel(message string) string {
	resp, err := wailsrt.MessageDialog(s.ctx, wailsrt.MessageDialogOptions{
		Type:          wailsrt.QuestionDialog,
		Title:         "Unsaved changes",
		Message:       message,
		Buttons:       []string{"Save", "Discard", "Cancel"},
		DefaultButton: "Save",
		CancelButton:  "Cancel",
	})
	if err != nil {
		return answerCancel
	}
	switch resp {
	case "Save":
		return answerSave
	case "Discard":
		return answerDiscard
	default:
		return answerCancel
	}
}

// Emit sends an event to the frontend.
func (s wailsShell) Emit(event string) { wailsrt.EventsEmit(s.ctx, event) }
