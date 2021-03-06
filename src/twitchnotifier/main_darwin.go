// +build darwin

package main

import (
	"io"
	"log"
	"os"
	"path"

	"github.com/deckarep/gosx-notifier"
	"github.com/rakslice/wxGo/wx"
	"github.com/kardianos/osext"
)

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Foundation

void clearOSXWebkitTwitchCookies();
*/
import "C"

func main() {
	// open a log file
	// TODO write this same log file on all platforms, using an appropriate log file location
	logFilename := userRelativePath("Library", "Logs", "twitch-notifier-go.log")
	logFileHandle, err := os.OpenFile(logFilename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
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

	iconFilename, _ := _get_asset_icon_info()

	note.AppIcon = iconFilename

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

	menuBar := win.createMenuBar(false)
	wx.MenuBarMacSetCommonMenuBar(menuBar)

	menu := menuBar.GetMenu(0)
	clearCookiesItem := menu.Append(wx.ID_ANY, "Quit With Logout")
	wx.Bind(menuBar, wx.EVT_MENU, win.onClearCookiesMenu, clearCookiesItem.GetId())
}

func (win *MainStatusWindowImpl) onClearCookiesMenu(e wx.Event) {
	msg("onClearCookiesMenu")
	C.clearOSXWebkitTwitchCookies()
	win.Shutdown()
}

func _get_asset_icon_info() (string, int) {
	subpath := "icon.ico"
	bitmap_type := wx.BITMAP_TYPE_ICO

	// TODO use wx.StandardPaths.GetResourcesDir() once it's available
	exeDir, err := osext.ExecutableFolder()
	assert(err == nil, "ExecutableFolder failed: %s", err)
	assets_path := path.Join(exeDir, "..", "Resources")
	bitmap_path := path.Join(assets_path, subpath)
	return bitmap_path, bitmap_type
}
