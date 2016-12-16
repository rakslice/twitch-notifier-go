//go:generate goversioninfo  -64
//go:generate windres -i icon.rc -O coff -o icon.syso

// +build windows

// To install goversioninfo, do:
//   go get github.com/josephspurrier/goversioninfo/cmd/goversioninfo

package main

import "github.com/dontpanic92/wxGo/wx"

func (win *MainStatusWindowImpl) additionalBindings() {
	// last param should be a specific object id if we have one e.g. out.toolbar_icon.GetId()?
	wx.Bind(win.toolbar_icon, wx.EVT_TASKBAR_LEFT_DCLICK, win._on_toolbar_icon_left_dclick, wx.ID_ANY)
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
