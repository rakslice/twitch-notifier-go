//go:generate goversioninfo  -64
//go:generate windres -i icon.rc -O coff -o icon.syso

// +build windows

// To install goversioninfo, do:
//   go get github.com/josephspurrier/goversioninfo/cmd/goversioninfo

package main

import (
	"github.com/rakslice/wxGo/wx"
	"time"
)

func (win *MainStatusWindowImpl) additionalBindings() {
	// FIX If the taskbar icon is double-clicked when there is a toolbar balloon notification up, we get a spurious toolbar balloon click event
	// and the normal toolbar balloon timeout event doesn't happen.
	// The spurious toolbar balloon click event comes in shortly before the double-click event.
	// As a workaround, we will hold processing of the toolbar balloon click event for a short time and cancel it if the double click event comes in
	// during that time.

	toolbar_balloon_click_wrapper := func(args ...interface{}) {
		win._on_toolbar_balloon_click(args[0].(wx.Event))
	}
	toolbar_icon_dclick_wrapper := func(args ...interface{}) {
		win._on_toolbar_icon_left_dclick(args[0].(wx.Event))
	}
	toolbar_balloon_timeout_wrapper := func(args ...interface{}) {
		win._on_toolbar_balloon_timeout(args[0].(wx.Event))
	}
	canceller := win.NewCallbackCanceller(100*time.Millisecond, toolbar_balloon_click_wrapper, toolbar_icon_dclick_wrapper, &toolbar_balloon_timeout_wrapper)
	on_other_event_wrapper := func(e wx.Event) {
		canceller.OnOtherEvent(e)
	}
	on_cancellable_event_wrapper := func(e wx.Event) {
		canceller.OnCancellableEvent(e)
	}

	// last param should be a specific object id if we have one e.g. out.toolbar_icon.GetId()?
	//wx.Bind(win.toolbar_icon, wx.EVT_TASKBAR_LEFT_DCLICK, win._on_toolbar_icon_left_dclick, wx.ID_ANY)
	wx.Bind(win.toolbar_icon, wx.EVT_TASKBAR_LEFT_DCLICK, on_other_event_wrapper, wx.ID_ANY)
	//wx.Bind(win.toolbar_icon, wx.EVT_TASKBAR_BALLOON_CLICK, win._on_toolbar_balloon_click, wx.ID_ANY)
	wx.Bind(win.toolbar_icon, wx.EVT_TASKBAR_BALLOON_CLICK, on_cancellable_event_wrapper, wx.ID_ANY)
	wx.Bind(win.toolbar_icon, wx.EVT_TASKBAR_BALLOON_TIMEOUT, win._on_toolbar_balloon_timeout, wx.ID_ANY)

	win.button_open_channel.SetDefault()

	win.createMenuBar(true)
}

func (win *MainStatusWindowImpl) osNotification(notification *NotificationQueueEntry) {
	var delay_ms uint = 200
	var flags int = 0
	icon := win._get_asset_icon()
	assert(icon.IsOk(), "asset icon was not ok")
	var result bool
	result = win.toolbar_icon.ShowBalloon(notification.title, notification.msg, delay_ms, flags, icon)
	assert(result, "error showing balloon")
}

func main() {
	commonMain(nil)
}

func prefsRelativePath() []string {
	return []string{}
}

func _get_asset_icon_info() (string, int) {
	subpath := "IDI_ICON_ICO"
	bitmap_type := wx.BITMAP_TYPE_ICO_RESOURCE
	return subpath, bitmap_type
}
