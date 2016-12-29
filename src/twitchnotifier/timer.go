package main

import (
	"github.com/rakslice/wxGo/wx"
	"sync"
	"time"
	//"runtime/debug"
)

/**
dontpanic92's wxGo doesn't have a wx.Timer analogous to the wxPython one.
That may be because go's built-in time.AfterFunc() provides similar functionality,
running a callback in a goroutine after a delay.

However wx GUI methods don't support calls outside the main thread, so can't be
called from an arbitrary goroutine. So this file provides a GUI-safe AfterFunc.

The implementation wraps time.AfterFunc, and ships its callbacks to the wx main
thread by way of a wx.ThreadingEvent. This approach is based on the
wxGo threadevent example:

https://github.com/rakslice/wxGo/blob/master/examples/src/threadevent/main.go
*/

// TIME HELPER STUFF

type WxTimeHelper struct {
	hostFrame                wx.Frame
	wx_event_id              int
	next_callback_num        int
	next_callback_num_mutex  *sync.Mutex
	callbacks_map            map[int]func()
	callbacks_map_mutex      *sync.Mutex
	timer_wrappers_map       map[int]*TimerWrapper
	timer_wrappers_map_mutex *sync.Mutex
	hostFrameMutex           *sync.Mutex
}

var next_wx_event_id int = wx.ID_HIGHEST + 1

func NewWxTimeHelper(hostFrame wx.Frame) *WxTimeHelper {
	out := &WxTimeHelper{}

	out.next_callback_num_mutex = &sync.Mutex{}
	out.callbacks_map_mutex = &sync.Mutex{}
	out.timer_wrappers_map_mutex = &sync.Mutex{}
	out.hostFrameMutex = &sync.Mutex{}
	out.callbacks_map = make(map[int]func())
	out.timer_wrappers_map = make(map[int]*TimerWrapper)
	// get an event id for this particular WxTimeHelper
	out.wx_event_id = next_wx_event_id
	next_wx_event_id += 1
	out.hostFrame = hostFrame
	out.next_callback_num = 1

	// Set up an event handler on the host frame that we will use to bring execution into
	// the GUI thread
	wx.Bind(out.hostFrame, wx.EVT_THREAD, out.on_thread_event, out.wx_event_id)

	return out
}

type TimerWrapper struct {
	timer        *time.Timer
	callback_num int
	helper       *WxTimeHelper
}

func (wrapper *TimerWrapper) Stop() {
	wrapper.helper.pop_timer_wrapper(wrapper.callback_num)
	wrapper.helper.pop_callback(wrapper.callback_num)
	wrapper.timer.Stop()
}

func (wrapper *TimerWrapper) Reset(d time.Duration) {
	wrapper.timer.Reset(d)
}

func (helper *WxTimeHelper) pop_callback(callback_num int) func() {
	helper.callbacks_map_mutex.Lock()
	callback, ok := helper.callbacks_map[callback_num]
	if ok {
		delete(helper.callbacks_map, callback_num)
	}
	helper.callbacks_map_mutex.Unlock()
	if !ok {
		msg("error retrieving callback for WxTimeWrapper callback num %s", callback_num)
		return nil
	}
	return callback
}

func (helper *WxTimeHelper) pop_timer_wrapper(callback_num int) *TimerWrapper {
	helper.timer_wrappers_map_mutex.Lock()
	timerWrapper, ok := helper.timer_wrappers_map[callback_num]
	if ok {
		delete(helper.timer_wrappers_map, callback_num)
	}
	helper.timer_wrappers_map_mutex.Unlock()
	if ok {
		return timerWrapper
	} else {
		return nil
	}
}

func (helper *WxTimeHelper) on_thread_event(e wx.Event) {
	msg("on_thread_event")
	// get the callback num from the thread event
	threadEvent := wx.ToThreadEvent(e)
	callback_num := threadEvent.GetInt()

	// pop the callback out of the callbacks file
	callback := helper.pop_callback(callback_num)

	// call the callback
	callback()
}

/** Call a function after a delay. The function will be called in the GUI thread of the
WxTimeHelper's frame.  Use the returned object to cancel the call or call early.
*/
func (helper *WxTimeHelper) AfterFunc(duration time.Duration, callback func()) *TimerWrapper {
	// get a callback num and file the callback

	// TODO safeguard against id collision when we wrap around the int space
	helper.next_callback_num_mutex.Lock()
	callback_num := helper.next_callback_num
	helper.next_callback_num += 1
	helper.next_callback_num_mutex.Unlock()

	helper.callbacks_map_mutex.Lock()
	helper.callbacks_map[callback_num] = callback
	helper.callbacks_map_mutex.Unlock()

	//msg("timer for callback %v setup %s", callback_num, callback)
	//debug.PrintStack()

	// Do the real AfterFunc call with a callback that sets up an event to do the wrapper callback

	//msg("before delay for callback %s", callback_num)
	timer := time.AfterFunc(duration, func() {
		//msg("after delay for callback %s", callback_num)
		helper.on_call_complete(callback_num)
	})
	//msg("after calling real AfterFunc")

	timerWrapper := &TimerWrapper{timer, callback_num, helper}

	helper.timer_wrappers_map_mutex.Lock()
	helper.timer_wrappers_map[callback_num] = timerWrapper
	helper.timer_wrappers_map_mutex.Unlock()

	return timerWrapper
}

func (helper *WxTimeHelper) shutdown() {
	helper.stopAll()
	helper.hostFrameMutex.Lock()
	helper.hostFrame = nil
	helper.hostFrameMutex.Unlock()
}

func (helper *WxTimeHelper) stopAll() {
	msg("Stopping all timers")
	for {
		gotItem := false
		var curTimerWrapper *TimerWrapper
		helper.timer_wrappers_map_mutex.Lock()
		for _, timerWrapper := range helper.timer_wrappers_map {
			gotItem = true
			curTimerWrapper = timerWrapper
			break
		}
		helper.timer_wrappers_map_mutex.Unlock()
		if gotItem {
			//helper.callbacks_map_mutex.Lock()
			//callback, callbackOk := helper.callbacks_map[curTimerWrapper.callback_num]
			//helper.callbacks_map_mutex.Unlock()
			//
			//if callbackOk {
			//	msg("Stopping timer %d callback %s", curTimerWrapper.callback_num, callback)
			//}
			curTimerWrapper.Stop()
		} else {
			// we're all done
			break
		}
	}

	// We also want to prevent any callbacks for timers that have already made it
	// to the event queue
	helper.hostFrameMutex.Lock()
	hostFrame := helper.hostFrame
	helper.hostFrameMutex.Unlock()

	if hostFrame != nil {
		hostFrame.DeletePendingEvents()
	}
	// TODO if we add support to the timer wrappers for tracking & cancelling the queue events,
	// the DeletePendingEvents() call won't be necessary.
}

func (helper *WxTimeHelper) on_call_complete(callback_num int) {
	// This method gets called in a thread other than the wx main thread, so it must only set up some thread events and cannot call into the GUI directly

	// we're done with this timer wrapper as stopping its timer won't do anything anymore
	helper.pop_timer_wrapper(callback_num)
	// TODO instead of that, have the timerWrapper keep track of the QueueEvent and cancel
	// it if stopped

	msg("timer for callback %v complete", callback_num)
	threadEvent := wx.NewThreadEvent(wx.EVT_THREAD, helper.wx_event_id)
	threadEvent.SetInt(callback_num)

	helper.hostFrameMutex.Lock()
	hostFrame := helper.hostFrame
	helper.hostFrameMutex.Unlock()
	if hostFrame != nil {
		hostFrame.QueueEvent(threadEvent)
	}
}
