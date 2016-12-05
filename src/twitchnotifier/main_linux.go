
// +build linux

package main

func main() {
	commonMain()
}

func prefsRelativePath() []string {
	return []string{}
}

const shouldDoParse = true

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
}
