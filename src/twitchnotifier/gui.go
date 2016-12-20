package main

import (
	"flag"
	"fmt"
	"github.com/dontpanic92/wxGo/wx"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"time"
)

// Holds a notification entry that is queued up to go out order behind any other notifications.
// This is intended to be used if the underlying desktop notification system doesn't have its
// own notification queue.
type NotificationQueueEntry struct {
	callback func() error
	title    string
	msg      string
	url      string
}

// CONCRETE WINDOW DEFINITION AND CONSTRUCTOR

/**
MainStatusWindowImpl embeds MainStatusWindow, the GUI skeleton that comes from code generation,
and implements the GUI

*/
type MainStatusWindowImpl struct {
	MainStatusWindow
	app                             wx.App
	main_obj                        *OurTwitchNotifierMain
	toolbar_icon                    wx.TaskBarIcon
	balloon_click_callback          func() error

	// notifications waiting to go on the screen behind the currently shown notification
	notifications_queue             []NotificationQueueEntry
	// whether there is currently a batch of notifications being shown
	notifications_queue_in_progress bool

	timer                           *TimerWrapper
	timer_callback                  func()

	timeHelper                      *WxTimeHelper

	copySelectedUrlMenuItem         wx.MenuItem
}

//func getTokenFilename() string {
//	newParts := append(prefsRelativePath(), "twitchnotifier.token")
//	return userRelativePath(newParts...)
//}

func InitMainStatusWindowImpl(testMode bool, replacementOptionsFunc func() *Options) *MainStatusWindowImpl {
	out := &MainStatusWindowImpl{}
	out.MainStatusWindow = *initMainStatusWindow(out)

	out.timeHelper = NewWxTimeHelper(out)

	out.timer = nil
	out.timer_callback = nil

	out.balloon_click_callback = nil
	out.app = nil

	out.copySelectedUrlMenuItem = nil

	out.notifications_queue_in_progress = false
	out.notifications_queue = make([]NotificationQueueEntry, 0)

	out.toolbar_icon = wx.NewTaskBarIcon()

	the_icon := out._get_asset_icon()
	assert(the_icon.IsOk(), "asset icon was not ok")
	out.toolbar_icon.SetIcon(the_icon)

	out.clearLogo()

	out.additionalBindings()

	wx.Bind(out, wx.EVT_CLOSE_WINDOW, out._on_close, out.GetId())

	twitch_notifier_main := InitOurTwitchNotifierMain()
	if replacementOptionsFunc == nil {
		twitch_notifier_main.options = parse_args()
	} else {
		twitch_notifier_main.options = replacementOptionsFunc()
	}
	twitch_notifier_main.window_impl = out
	oauth_option := twitch_notifier_main.options.authorization_oauth
	msg("oauth option is %s", oauth_option)
	if oauth_option != nil {
		twitch_notifier_main._auth_oauth = *oauth_option
	}

	//tokenFilename := getTokenFilename()
	//
	//if twitch_notifier_main._auth_oauth == "" && !testMode && fileExists(tokenFilename) {
	//	buf, err := ioutil.ReadFile(tokenFilename)
	//	assert(err == nil, "error reading token file '%s': %s", tokenFilename, err)
	//	twitch_notifier_main._auth_oauth = string(buf)
	//}
	//
	//if twitch_notifier_main._auth_oauth == "" && !testMode {
	//	// TODO remove this once we support web OAuth internally
	//	oauthTokenSite := "http://twitchapps.com/tmi"
	//	friendlyMessage := "Missing -auth-oauth\n" +
	//		"\n" +
	//		"Howdy! Twitch login within twitch-notifier isn't supported yet.\n" +
	//		"\n" +
	//		"You'll have to login and get an OAuth token in your browser (e.g. at %s), \n" +
	//		"and then enter the token at the next prompt, or run:\n" +
	//	        "\n" +
	//		"  twitchnotifier -auth-oauth YOUR_TOKEN_HERE\n" +
	//		"\n" +
	//		"putting your token in for the placeholder.\n" +
	//		"Sorry for the inconvenience!\n" +
	//		"\n" +
	//		"Click Ok to open %s"
	//	result := wx.MessageBox(fmt.Sprintf(friendlyMessage, oauthTokenSite, oauthTokenSite), "twitch-notifier-go",
	//	              wx.OK | wx.CANCEL)
	//
	//	enteredToken := false
	//
	//	if result == wx.OK {
	//		webbrowser_open(oauthTokenSite)
	//
	//		// let's attempt to get a token from the user
	//		msg("creating text entry dialog")
	//		dlg := wx.NewTextEntryDialog(out, "Enter your Twitch OAuth token", fmt.Sprintf("This will be saved in '%s'", tokenFilename), "", wx.OK | wx.CANCEL)
	//		if dlg.ShowModal() == wx.ID_OK {
	//			newToken := strings.TrimSpace(dlg.GetValue());
	//			if newToken != "" {
	//				buf := []byte(newToken)
	//				writeErr := ioutil.WriteFile(tokenFilename, buf, 0600)
	//				assert(writeErr == nil, "Error writing token file '%s': %s", tokenFilename, writeErr)
	//
	//				twitch_notifier_main._auth_oauth = newToken
	//				enteredToken = true
	//			}
	//		}
	//
	//	}
	//	if !enteredToken {
	//		log.Fatal("Exiting due to no auth")
	//	}
	//}
	msg("after oauth setting check")
	out.main_obj = twitch_notifier_main

	if twitch_notifier_main.options.help != nil && *twitch_notifier_main.options.help {
		flag.Usage()
		log.Fatal("Showing usage")
	}

	// get rid of the options button for now
	out.button_options.Hide()

	return out
}

// CONCRETE WINDOW METHODS

func (win *MainStatusWindowImpl) _on_list_gen(e wx.Event, wasOnlineList bool) {
	commandEvent := wx.ToCommandEvent(e)

	idx := commandEvent.GetInt()
	if idx >= 0 {
		otherList := win.main_obj._list_for_is_online(!wasOnlineList)

		otherList.SetSelection(-1)
		channel, stream := win.main_obj.getChannelAndStreamForListEntry(wasOnlineList, idx)
		win.showInfo(channel, stream)
		win.copySelectedUrlMenuItem.Enable(true)
	} else {
		win.clearInfo()
		win.copySelectedUrlMenuItem.Enable(false)
	}
}

func (win *MainStatusWindowImpl) _on_list_online_gen(e wx.Event) {
	win._on_list_gen(e, true)
}

func (win *MainStatusWindowImpl) _on_list_online_dclick(e wx.Event) {
	win.main_obj.openSiteForListEntry(true, e)
}

func (win *MainStatusWindowImpl) _on_list_offline_gen(e wx.Event) {
	win._on_list_gen(e, false)
}

func (win *MainStatusWindowImpl) _on_list_offline_dclick(e wx.Event) {
	win.main_obj.openSiteForListEntry(false, e)
}

func (win *MainStatusWindowImpl) _on_options_button_click(e wx.Event) {
	win.main_obj.log("_on_options_button_click")
}

func (win *MainStatusWindowImpl) _on_button_reload_channels_click(e wx.Event) {
	win.main_obj.doChannelsReload()
}

func (win *MainStatusWindowImpl) setChannelRefreshInProgress(value bool) {
	if value {
		win.button_reload_channels.Disable()
	} else {
		win.button_reload_channels.Enable()
	}
}

func (win *MainStatusWindowImpl) Shutdown() {
	// stop the current batch of notifications if any
	win.notifications_queue_in_progress = false

	// cancel any timers that are already in flight
	win.timeHelper.shutdown()

	// shutdown
	win.toolbar_icon.RemoveIcon()
	win.toolbar_icon.Destroy()
	win.toolbar_icon = nil
	win.Close()
	win.app.ExitMainLoop()
	win.app = nil
}

func (win *MainStatusWindowImpl) _on_button_quit(e wx.Event) {
	win.main_obj.log("_on_button_quit")
	win.Shutdown()
}

func (win *MainStatusWindowImpl) setStreamInfo(stream *StreamInfo) {
	for _, label := range []wx.StaticText{win.label_head_game, win.label_head_started, win.label_head_up} {
		label.Show()
		label.Refresh()
	}

	game := stream.Game
	if game != nil {
		win.label_game.SetLabel(*game)
	} else {
		win.label_game.SetLabel("")
	}

	createdAtStr := stream.Created_at
	startTime, err := convert_rfc3339_time(createdAtStr)
	if err != nil {
		win.main_obj.log(fmt.Sprintf("Error parsing time '%s': %s", createdAtStr, err))
		win.label_start_time.SetLabel(createdAtStr)
		win.label_uptime.SetLabel("")
	} else {
		localStartTime := startTime.Local()
		win.label_start_time.SetLabel(localStartTime.Format(time.RFC1123))
		win.label_uptime.SetLabel(time_desc(time.Now().Sub(startTime)))
	}
}

func (win *MainStatusWindowImpl) clearStreamInfo() {
	for _, variableLabel := range []wx.StaticText{win.label_game, win.label_uptime, win.label_start_time} {
		variableLabel.SetLabel("")
	}

	for _, fixedLabel := range []wx.StaticText{win.label_head_game, win.label_head_started, win.label_head_up} {
		fixedLabel.Hide()
	}
}

func (win *MainStatusWindowImpl) clearInfo() {
	win.clearStreamInfo()
}

func (win *MainStatusWindowImpl) showImageInWxImage(control wx.StaticBitmap, readCloser io.ReadCloser) {
	// copy the file to a tempfile
	tempfileName, err := readToTempFile(readCloser)
	assert(err == nil, "Error creating temp file: %s", err)

	// Bounce through an event so the GUI interaction happens in the main thread
	win.timeHelper.AfterFunc(0, func() {
		msg("Opening image")
		image := wx.NewImage(tempfileName)
		msg("Deleting temp file %s", tempfileName)
		os.Remove(tempfileName)

		height := control.GetMinHeight()
		width := control.GetMinWidth()
		if width > 0 && height > 0 {
			msg("Scaling image to %vx%v", width, height)
			image = image.Scale(width, height)
		}
		msg("Displaying")
		bitmap := wx.NewBitmap(image)
		control.SetBitmap(bitmap)
	})
}

func (win *MainStatusWindowImpl) emptyBitmap(size wx.Size, colour wx.Colour) wx.Bitmap {
	bmp := wx.NewBitmap(size)
	dc := wx.NewMemoryDC(bmp)
	dc.SetBackground(wx.NewBrush(colour))
	dc.Clear()
	wx.DeleteDC(dc)
	return bmp
}

func (win *MainStatusWindowImpl) clearLogo() {
	win.bitmap_channel_logo.SetBitmap(win.emptyBitmap(win.bitmap_channel_logo.GetSize(), win.GetBackgroundColour()))
}

func (win *MainStatusWindowImpl) showInfo(channel *ChannelInfo, stream *StreamInfo) {
	if channel == nil {
		win.label_channel_status.SetLabel("")
	} else {
		win.label_channel_status.SetLabel(channel.Status)
	}

	if stream != nil {
		win.setStreamInfo(stream)
	} else {
		win.clearStreamInfo()
	}

	win.main_obj.cancelDelayedUrlLoadsForContext("channel")

	// set the logo to our default image pending the load of the channel image
	win.clearLogo()

	logoUrl := channel.Logo
	if logoUrl != nil && *logoUrl != "" {

		staticBitmapToSet := win.bitmap_channel_logo

		win.main_obj.log(fmt.Sprintf("Showing logo %s", *logoUrl))
		win.main_obj.doDelayedUrlLoad("channel", *logoUrl, func(rs *http.Response) {
			if rs == nil {
				return
			}
			defer rs.Body.Close()

			if rs.StatusCode != 200 {
				win.main_obj.log(fmt.Sprintf("Got HTTP error %v %s retrieving %s", rs.StatusCode, rs.Status, *logoUrl))
				return
			}

			win.main_obj.log("Logo loaded")
			//contentType := rs.Header.Get("Content-type")
			// TODO verify content type corresponds to a supported image format

			win.showImageInWxImage(staticBitmapToSet, rs.Body)
		})
	}
}

// ICON HELPER FUNCTIONS

func (win *MainStatusWindowImpl) _get_asset_icon() wx.Icon {
	icon := wx.NullIcon
	bitmap_path, bitmap_type := _get_asset_icon_info()

	var exists bool
	if bitmap_type == wx.BITMAP_TYPE_ICO_RESOURCE {
		exists = true
	} else {
		exists = fileExists(bitmap_path)
	}

	if exists {
		icon = wx.NewIcon(bitmap_path, bitmap_type)
	}

	if !icon.IsOk() {
		msg("Error loading asset icon from %s", bitmap_path)

		// just set the icon to a solid color
		solidBitmap := win.emptyBitmap(wx.NewSize(16, 16), wx.NewColour(byte(72), byte(0), byte(0)))
		assert(solidBitmap.IsOk(), "error creating solid bitmap")
		icon.CopyFromBitmap(solidBitmap)
		assert(icon.IsOk(), "icon copied from bitmap is not ok")
	}

	return icon
}

func _get_asset_icon_info_common() (string, int) {
	subpath := "icon.ico"
	bitmap_type := wx.BITMAP_TYPE_ICO
	script_path, err := filepath.Abs(".")
	assert(err == nil, "icon path failed: %s", err)
	assets_path := path.Join(script_path, "assets")
	bitmap_path := path.Join(assets_path, subpath)
	return bitmap_path, bitmap_type
}

// TIME RELATED STUFF IN GUI FRAME

func (win *MainStatusWindowImpl) set_timer_with_callback(length time.Duration, callback func()) {
	assert(win.timer == nil, "there is already a win.timer")
	assert(win.timer_callback == nil, "there is already a win.timer_callback")

	win.timer_callback = callback
	//msg("before set_timer_with_callback AfterFunc call")

	win.timer = win.timeHelper.AfterFunc(length, win._timer_internal_callback)
	//msg("after set_timer_with_callback AfterFunc call")
}

func (win *MainStatusWindowImpl) cancel_timer_callback_immediate() bool {
	if win.timer == nil {
		return false
	}
	win.timer.Stop()
	win.timer = nil
	cur_callback := win.timer_callback
	win.timer_callback = nil
	cur_callback()
	return true
}

func (win *MainStatusWindowImpl) _timer_internal_callback() {
	win.timer = nil
	cur_callback := win.timer_callback
	win.timer_callback = nil
	cur_callback()
}

// EVENT HANDLERS AND OTHER CONCRETE WINDOW METHODS

func (win *MainStatusWindowImpl) _on_toolbar_icon_left_dclick(e wx.Event) {
	win.Show()
	win.Raise()
}

func (win *MainStatusWindowImpl) set_timeout(length time.Duration, callback func()) {
	win.timeHelper.AfterFunc(length, callback)
}

func (win *MainStatusWindowImpl) set_balloon_click_callback(callback func() error) {
	win.balloon_click_callback = callback
}

func (win *MainStatusWindowImpl) enqueue_notification(title string, msg string, callback NotificationCallback, url string) {
	notification := NotificationQueueEntry{callback.callback, title, msg, url}
	win.notifications_queue = append(win.notifications_queue, notification)
	if !win.notifications_queue_in_progress {
		// kick off the notification cycle
		win._dispense_remaining_notifications()
	}
}

func (win *MainStatusWindowImpl) _dispense_remaining_notifications() {
	// hack to avoid double-triggering of events that happens
	if len(win.notifications_queue) == 0 {
		win.notifications_queue_in_progress = false
		return
	}

	win.notifications_queue_in_progress = true
	// pop a notification off the queue
	var notification NotificationQueueEntry = win.notifications_queue[0]
	win.notifications_queue = append(win.notifications_queue[:0], win.notifications_queue[1:]...)

	// show the notification
	win.set_balloon_click_callback(notification.callback)

	win.main_obj.log(fmt.Sprintf("Showing notification '%s'", notification.msg))
	win.osNotification(&notification)
}

func (win *MainStatusWindowImpl) _on_toolbar_balloon_click(e wx.Event) {
	win.main_obj.log("notification clicked")
	if win.balloon_click_callback != nil {
		err := win.balloon_click_callback()
		if err != nil {
			win.main_obj.log(fmt.Sprintf("Balloon click callback returned error: %s", err))
		}
	}
	win.notificationFinished()
}

func (win *MainStatusWindowImpl) notificationFinished() {
	if win.notifications_queue_in_progress {
		// ok, on to the next
		win.set_timeout(250*time.Millisecond, win._dispense_remaining_notifications)
	}
}

func (win *MainStatusWindowImpl) notificationTimeout() {
	win.main_obj.log("notification timeout")
	win.notificationFinished()
}

func (win *MainStatusWindowImpl) _on_toolbar_balloon_timeout(e wx.Event) {
	win.notificationTimeout()
}

func (win *MainStatusWindowImpl) _on_close(e wx.Event) {
	win.Hide()
}

// MENU

func (win *MainStatusWindowImpl) createMenuBar(menuInAppWindow bool) wx.MenuBar {
	menuBar := wx.NewMenuBar()
	menu := wx.NewMenu()

	// NB: we bind these items to the menubar as platform implementations will move them to different menus

	if !menuInAppWindow {
		showGuiItem := menu.Append(wx.ID_ANY, "Show GUI\tCtrl-G")
		wx.Bind(menuBar, wx.EVT_MENU, win.onMenuShowGUI, showGuiItem.GetId())
	}

	hideGuiItem := menu.Append(wx.ID_ANY, "Hide GUI\tCtrl-W")
	wx.Bind(menuBar, wx.EVT_MENU, win.onMenuHideGUI, hideGuiItem.GetId())

	copyURLItem := menu.Append(wx.ID_ANY, "Copy URL\tCtrl-C")
	copyURLItem.Enable(false)
	wx.Bind(menuBar, wx.EVT_MENU, win.onMenuCopySelectedURL, copyURLItem.GetId())
	win.copySelectedUrlMenuItem = copyURLItem

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

	if menuInAppWindow {
		win.SetMenuBar(menuBar)
	}

	return menuBar
}

func (win *MainStatusWindowImpl) onMenuShowGUI(e wx.Event) {
	msg("onMenuShowGUI")
	win._on_toolbar_icon_left_dclick(e)
}

func (win *MainStatusWindowImpl) onMenuHideGUI(e wx.Event) {
	msg("onMenuHideGUI")
	win.Hide()
}

func (win *MainStatusWindowImpl) getSelectedItemURL() (string, bool) {
	for _, online := range []bool {true, false} {
		list := win.main_obj._list_for_is_online(online)
		idx := list.GetSelection()
		if idx >= 0 {
			url, found := win.main_obj.getUrlForListEntry(online, idx)
			if found {
				return url, true
			}
		}
	}
	return "", false
}

func (win *MainStatusWindowImpl) onMenuCopySelectedURL(e wx.Event) {
	url, found := win.getSelectedItemURL()

	if found {
		clipboard := wx.NewClipboard()
		if !clipboard.IsOpened() {
			clipboard.Open()
			defer clipboard.Close()
			clipData := wx.NewTextDataObject()
			clipData.SetText(url)
			clipboard.SetData(clipData)
		}
	}
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

// ABOUT BOX

func showAboutBox() {
	// 	title := "About twitch-notifier-go"
	// 	message := `twitch-notifier-go pre-beta 0.01
	// https://github.com/rakslice/twitch-notifier-go

	// A desktop notifier for twitch.tv streams.

	// See the accompanying LICENSE file for terms
	// `
	// wx.MessageBox(message, title)

	info := wx.NewAboutDialogInfo()
	info.SetName("twitch-notifier-go")
	info.SetWebSite("https://github.com/rakslice/twitch-notifier-go")

	info.SetDescription(`A desktop notifier for twitch.tv streams.

See the accompanying LICENSE file for terms

Source code and more information available at the project github site`)
	info.SetVersion("pre-beta 0.01")

	wx.AboutBox(info)
}

// MAIN FUNCTION UTILS

var imageHandlersInitialized bool = false

func preApp() {
	if !imageHandlersInitialized {
		imageHandlersInitialized = true
		// initialize the handlers for all image formats so that wx.Bitmap routines can load all
		// supported image formats from disk
		wx.InitAllImageHandlers()
	}
}

func commonMain(replacementOptionsFunc func() *Options) {
	preApp()

	app := wx.NewApp()

	msg("init top")
	frame := InitMainStatusWindowImpl(false, replacementOptionsFunc)

	frame.app = app
	app.SetTopWindow(frame)
	msg("showing frame")
	frame.Show()

	// we're doing this in set_timeout so that it happens inside app.MainLoop() -- otherwise
	// the wx thread safeguard gets confused
	frame.set_timeout(0, frame.main_obj.main_loop_main_window_timer)

	msg("starting main loop")
	app.MainLoop()
	msg("main loop stopped")
	msg("destroying window")
	frame.Destroy()
	//msg("deleting app")
	//wx.DeleteApp(app)
	//msg("wx app deleted")
}
