package events

import (
	"context"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Emitter abstracts Wails event emission so services can be tested without Wails.
type Emitter interface {
	Emit(event string, data any)
}

// WailsEmitter implements Emitter using the live Wails runtime context.
type WailsEmitter struct {
	ctx context.Context
}

// NewWailsEmitter creates an emitter backed by the Wails app context.
func NewWailsEmitter(ctx context.Context) *WailsEmitter {
	return &WailsEmitter{ctx: ctx}
}

func (e *WailsEmitter) Emit(event string, data any) {
	runtime.EventsEmit(e.ctx, event, data)
}

// Noop is a no-op emitter suitable for unit tests.
type Noop struct{}

func (Noop) Emit(_ string, _ any) {}
