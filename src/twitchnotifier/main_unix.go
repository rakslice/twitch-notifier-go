
// +build linux darwin

package main

import "github.com/dontpanic92/wxGo/wx"

func (win *MainStatusWindowImpl) additionalBindings() {
	// FIXME the event constants for these appear to be missing
	//wx.Bind(win.toolbar_icon, wx.EVT_NOTIFICATION_MESSAGE_CLICK, win._on_toolbar_balloon_click, wx.ID_ANY)
	//wx.Bind(win.toolbar_icon, wx.EVT_NOTIFICATION_MESSAGE_DISMISSED, win._on_toolbar_balloon_timeout, wx.ID_ANY)
}

func (win *MainStatusWindowImpl) osNotification(notification *NotificationQueueEntry) {
	nm := wx.NewNotificationMessage()
	nm.SetParent(win)
	icon := win._get_asset_icon()
	assert(icon.IsOk(), "asset icon was not ok")
	nm.SetIcon(icon)
	nm.SetTitle(notification.title)
	nm.SetMessage(notification.msg)
	nm.Show(1)
}
