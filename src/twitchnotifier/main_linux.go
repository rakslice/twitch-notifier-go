// +build linux

package main

import "github.com/dontpanic92/wxGo/wx"

func main() {
	commonMain(nil)
}

func prefsRelativePath() []string {
	return []string{}
}

func (win *MainStatusWindowImpl) osNotification(notification *NotificationQueueEntry) {
	nm := wx.NewNotificationMessage()
	nm.SetParent(win)
	icon := win._get_asset_icon()
	assert(icon.IsOk(), "asset icon was not ok")
	nm.SetIcon(icon)
	nm.SetTitle(notification.title)
	nm.SetMessage(notification.msg)
	nm.SetFlags(wx.ICON_INFORMATION)
	result := nm.Show(1)
	if !result {
		msg("wx.NotificationMessage.Show() indicated that the notification for '%s' was not shown", notification.msg)
	}

	// there are no callback timeout semantics so call for the next notification right away
	win.notificationTimeout()
}

func (win *MainStatusWindowImpl) additionalBindings() {
	// last param should be a specific object id if we have one e.g. out.toolbar_icon.GetId()?
	wx.Bind(win.toolbar_icon, wx.EVT_TASKBAR_CLICK, win._on_toolbar_icon_left_dclick, wx.ID_ANY)
	// FIXME the event constants for these appear to be missing
	//wx.Bind(win.toolbar_icon, wx.EVT_NOTIFICATION_MESSAGE_CLICK, win._on_toolbar_balloon_click, wx.ID_ANY)
	//wx.Bind(win.toolbar_icon, wx.EVT_NOTIFICATION_MESSAGE_DISMISSED, win._on_toolbar_balloon_timeout, wx.ID_ANY)

	win.createMenuBar(true)
}

func _get_asset_icon_info() (string, int) {
	return _get_asset_icon_info_common()
}
