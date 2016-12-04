
// +build windows

package main

import "github.com/dontpanic92/wxGo/wx"

func (win *MainStatusWindowImpl) additionalBindings() {
	wx.Bind(win.toolbar_icon, wx.EVT_TASKBAR_BALLOON_CLICK, win._on_toolbar_balloon_click, wx.ID_ANY)
	wx.Bind(win.toolbar_icon, wx.EVT_TASKBAR_BALLOON_TIMEOUT, win._on_toolbar_balloon_timeout, wx.ID_ANY)
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
