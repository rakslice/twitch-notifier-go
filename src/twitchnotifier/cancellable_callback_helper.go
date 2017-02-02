package main

import (
	"sync"
	"time"
)

// This is a helper for wrapping two callbacks, a "cancellable" callback and another callback,
// where we want calls to the "cancellable" callback to be held for a certain amount of time,
// and if the other callback is called within that time, to ignore that call to the "cancellable"
// callback by not calling it.  Optionally you can provide another callback to call with
// the cancellable callback args instead if the cancellation happens.
type CallbackCanceller struct {
	sync.Mutex
	cancellationTimeout      time.Duration
	prevCallArgs             *[]interface{}
	cancellableEventCallback func(...interface{})
	cancelledAltCallback     *func(...interface{})
	otherEventCallback       func(...interface{})
	timeHelper               *WxTimeHelper
	prevCallTimer            *TimerWrapper
}

func (win *MainStatusWindowImpl) NewCallbackCanceller(cancellation_timeout time.Duration, cancellableEventCallback func(...interface{}),
otherEventCallback func(...interface{}), cancelledAltCallback *func(...interface{})) *CallbackCanceller {

	out := &CallbackCanceller{sync.Mutex{}, cancellation_timeout, nil, cancellableEventCallback, cancelledAltCallback, otherEventCallback, win.timeHelper, nil}
	return out
}

// Fire the wrapped cancellable callback (if not cancelled)
func (canceller *CallbackCanceller) OnCancellableEvent(args ...interface{}) {
	canceller.Lock()
	defer canceller.Unlock()
	// If we have been called when there was a previous call queued already,
	// we will consider the previous call uncancelled and fire it immediately
	canceller.doPrevCancellableCallIfAny()
	// store the details of this call until the cancel timeout expires
	canceller.prevCallArgs = &args
	canceller.prevCallTimer = canceller.timeHelper.AfterFunc(canceller.cancellationTimeout, canceller.onCancelTimeout)
}

// Fire the wrapped other callback (and cancel the pending cancellable callback if any)
func (canceller *CallbackCanceller) OnOtherEvent(args ...interface{}) {
	canceller.Lock()
	defer canceller.Unlock()
	canceller.cancelPrevCancellableCallIfAny()
	canceller.otherEventCallback(args...)
}

func (canceller *CallbackCanceller) cancelPrevCancellableCallIfAny() {
	if canceller.prevCallTimer != nil {
		canceller.prevCallTimer.Stop()
		if canceller.cancelledAltCallback != nil {
			(*canceller.cancelledAltCallback)((*canceller.prevCallArgs)...)
		}
		canceller.prevCallArgs = nil
		canceller.prevCallTimer = nil
	}
}

func (canceller *CallbackCanceller) doPrevCancellableCallIfAny() {
	if canceller.prevCallTimer != nil {
		canceller.prevCallTimer.Stop()
		canceller.cancellableEventCallback((*canceller.prevCallArgs)...)
		canceller.prevCallArgs = nil
		canceller.prevCallTimer = nil
	}
}

func (canceller *CallbackCanceller) onCancelTimeout() {
	canceller.Lock()
	defer canceller.Unlock()
	// there's a cancellable callback pending that reached the timeout without being cancelled, so do it
	canceller.doPrevCancellableCallIfAny()
}
