
// +build linux darwin

package main

import "github.com/dontpanic92/wxGo/wx"

func (win *MainStatusWindowImpl) additionalBindings() {
	// last param should be a specific object id if we have one e.g. out.toolbar_icon.GetId()?
	wx.Bind(win.toolbar_icon, wx.EVT_TASKBAR_CLICK, win._on_toolbar_icon_left_dclick, wx.ID_ANY)
	// FIXME the event constants for these appear to be missing
	//wx.Bind(win.toolbar_icon, wx.EVT_NOTIFICATION_MESSAGE_CLICK, win._on_toolbar_balloon_click, wx.ID_ANY)
	//wx.Bind(win.toolbar_icon, wx.EVT_NOTIFICATION_MESSAGE_DISMISSED, win._on_toolbar_balloon_timeout, wx.ID_ANY)
}
