
// +build darwin

package main

import (
	"os"
	"log"
	"io"
	"github.com/deckarep/gosx-notifier"
	"github.com/dontpanic92/wxGo/wx"
)

func main() {
	// open a log file
	// TODO write this same log file on all platforms, using an appropriate log file location
	logFilename := userRelativePath("Library", "Logs", "twitch-notifier-go.log")
	logFileHandle, err := os.OpenFile(logFilename, os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
	assert(err == nil, "Error opening log file: %s", err)
	defer logFileHandle.Close()
	log.SetOutput(io.MultiWriter(os.Stderr, logFileHandle))

	commonMain(func() *Options {
		return &Options{}
	})
}

func prefsRelativePath() []string {
	return []string{"Library", "Preferences"}
}

func (win *MainStatusWindowImpl) osNotification(notification *NotificationQueueEntry) {

	assert(notification != nil, "called with null notification queue entry")

	note := gosxnotifier.NewNotification(notification.msg)
	
	note.Title = notification.title

	note.Sound = gosxnotifier.Basso

	note.Sender = "twitch-notifier-go"

	note.AppIcon = _get_asset_icon_filename()

	note.Link = notification.url

	err := note.Push()
	if err != nil {
		msg("notification implementation indicated that the notification for '%s' was not shown: %s", notification.msg, err)
	}


	// there are no callback timeout semantics so call for the next notification right away
	win.notificationTimeout()
}

func (win *MainStatusWindowImpl) additionalBindings() {
	// last param should be a specific object id if we have one e.g. out.toolbar_icon.GetId()?
	wx.Bind(win, wx.EVT_TASKBAR_CLICK, win._on_toolbar_icon_left_dclick, wx.ID_ANY)
	// FIXME the event constants for these appear to be missing
	//wx.Bind(win.toolbar_icon, wx.EVT_NOTIFICATION_MESSAGE_CLICK, win._on_toolbar_balloon_click, wx.ID_ANY)
	//wx.Bind(win.toolbar_icon, wx.EVT_NOTIFICATION_MESSAGE_DISMISSED, win._on_toolbar_balloon_timeout, wx.ID_ANY)

	menuBar := wx.NewMenuBar()
	menu := wx.NewMenu()

	// NB: we bind these items to the menubar as platform implementations will move them to different menus

	showGuiItem := menu.Append(wx.ID_ANY, "Show GUI\tCtrl-G")
	wx.Bind(menuBar, wx.EVT_MENU, win.onMenuShowGUI, showGuiItem.GetId())

	hideGuiItem := menu.Append(wx.ID_ANY, "Hide GUI\tCtrl-W")
	wx.Bind(menuBar, wx.EVT_MENU, win.onMenuHideGUI, hideGuiItem.GetId())

	aboutItem := menu.Append(wx.ID_ABOUT, "About twitch-notifier-go")
	assert(aboutItem.GetId() == wx.ID_ABOUT, "expected about item to have GetId() of ID_ABOUT")
	wx.Bind(menuBar, wx.EVT_MENU, win.onMenuAbout, aboutItem.GetId())

	reloadChannelsItem := menu.Append(wx.ID_ANY, "Reload Channels\tCtrl-R")
	wx.Bind(menuBar, wx.EVT_MENU, win.onMenuReloadChannels, reloadChannelsItem.GetId())

	// for future use:
	// menu.Append(wx.ID_HELP, "Help")
	// menu.Append(wx.ID_PREFERENCES, "Preferences")
	// wx.Bind(menuBar, wx.EVT_MENU, win.onMenuPrefs, wx.ID_PREFERENCES)
	menu.Append(wx.ID_EXIT, "Quit twitch-notifier-go")

	menuBar.Append(menu, "Info")

	// On other platforms:
	// win.SetMenuBar(menuBar)

	wx.MenuBarMacSetCommonMenuBar(menuBar)

}

func (win *MainStatusWindowImpl) onMenuShowGUI(e wx.Event) {
	msg("onMenuShowGUI")
	win._on_toolbar_icon_left_dclick(e)
}

func (win *MainStatusWindowImpl) onMenuHideGUI(e wx.Event) {
	msg("onMenuHideGUI")
	win.Hide()
}

func (win *MainStatusWindowImpl) onMenuReloadChannels(e wx.Event) {
	msg("onMenuReloadChannels")
	win.main_obj.doChannelsReload()
}

func (win *MainStatusWindowImpl) onMenuAbout(e wx.Event) {
	msg("onMenuAbout")
	showAboutBox()
}

// For future use:
// func (win *MainStatusWindowImpl) onMenuPrefs(e wx.Event) {
// 	msg("onMenuPrefs")
// 	wx.MessageBox("not implemented yet")
// }

