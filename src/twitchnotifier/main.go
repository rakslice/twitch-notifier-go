package main

import (
	"fmt"
	"log"
	"github.com/dontpanic92/wxGo/wx"
	"github.com/tomcatzh/asynchttpclient"
	"flag"
	"sync"
	"time"
	"path/filepath"
	"path"
	"runtime"
	"os/exec"
	"strings"
	"encoding/json"
	"sort"
	"net/http"
	"io"
	"io/ioutil"
	"os"
)


func assert(condition bool, message string, a... interface{}) {
	if !condition {
		formatted := fmt.Sprintf(message, a...)
		msg("assertion failure: %s", formatted)
		log.Fatal(formatted)
	}
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
	} else {
		win.clearInfo()
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

func (win *MainStatusWindowImpl) _on_button_quit(e wx.Event) {
	win.main_obj.log("_on_button_quit")
	win.toolbar_icon.RemoveIcon()
	win.toolbar_icon.Destroy()
	win.toolbar_icon = nil
	win.Close()
	win.app.ExitMainLoop()
	win.app = nil
}

func (win *MainStatusWindowImpl) setStreamInfo(stream *StreamInfo) {
	for _, label := range []wx.StaticText {win.label_head_game, win.label_head_started, win.label_head_up} {
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
	win.label_start_time.SetLabel(createdAtStr)
	startTime, err := convert_rfc3339_time(createdAtStr)
	if (err != nil) {
		win.main_obj.log(fmt.Sprintf("Error parsing time '%s': %s", createdAtStr, err))
		win.label_uptime.SetLabel("")
	} else {
		win.label_uptime.SetLabel(time_desc(time.Now().Sub(startTime)))
	}
}

func (win *MainStatusWindowImpl) clearStreamInfo() {
	for _, variableLabel := range []wx.StaticText {win.label_game, win.label_uptime, win.label_start_time} {
		variableLabel.SetLabel("")
	}

	for _, fixedLabel := range []wx.StaticText {win.label_head_game, win.label_head_started, win.label_head_up} {
		fixedLabel.Hide()
	}
}

func (win *MainStatusWindowImpl) clearInfo() {
	win.clearStreamInfo()
}

func (win *MainStatusWindowImpl) showImageInWxImage(control wx.StaticBitmap, readCloser io.ReadCloser) {
	// copy the file to a tempfile

	tempfile, err := ioutil.TempFile(os.TempDir(), "")
	assert(err == nil, "Error creating temp file: %s", err)

	tempfileName := tempfile.Name()

	msg("Saving image to %s", tempfileName)

	io.Copy(tempfile, readCloser)
	tempfile.Close()

	// Bounce through an event so the GUI interaction happens in the main thread
	win.set_timeout(0, func() error {
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
		return nil
	})
}

func (win *MainStatusWindowImpl) emptyBitmap(size wx.Size, colour wx.Colour) wx.Bitmap {
	bmp := wx.NewBitmap(size)
	dc := wx.NewMemoryDC(bmp)
	dc.SetBackground(wx.NewBrush(colour))
	dc.Clear()
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

	if stream != nil  {
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

// Holds a notification entry that is queued up to go out order behind any other notifications.
// This is intended to be used if the underlying desktop notification system doesn't have its
// own notification queue.
type NotificationQueueEntry struct {
	callback func() error
	title string
	msg string
}

// CONCRETE WINDOW DEFINITION AND CONSTRUCTOR

/**
MainStatusWindowImpl embeds MainStatusWindow, the GUI skeleton that comes from code generation,
and implements t

 */
type MainStatusWindowImpl struct {
	MainStatusWindow
	app wx.App
	main_obj *OurTwitchNotifierMain
	toolbar_icon wx.TaskBarIcon
	cur_timeout_timer *TimerWrapper
	cur_timeout_callback func() error
	balloon_click_callback func() error

	// notifications waiting to go on the screen behind the currently shown notification
	notifications_queue []NotificationQueueEntry
	// whether there is currently a batch of notifications being shown
	notifications_queue_in_progress bool

	timer *TimerWrapper
	timer_callback func()

	timeHelper *WxTimeHelper
}

func InitMainStatusWindowImpl() *MainStatusWindowImpl {
	out := &MainStatusWindowImpl{}
	out.MainStatusWindow = *initMainStatusWindow(out)

	out.timeHelper = NewWxTimeHelper(out)

	out.timer = nil
	out.timer_callback = nil

	out.cur_timeout_timer = nil
	out.cur_timeout_callback = nil
	out.balloon_click_callback = nil
	out.app = nil

	out.notifications_queue_in_progress = false
	out.notifications_queue = make([]NotificationQueueEntry, 0)

	out.toolbar_icon = wx.NewTaskBarIcon()

	the_icon := _get_asset_icon()
	out.toolbar_icon.SetIcon(the_icon)

	out.clearLogo()

	// last param should be a specific object id if we have one e.g. out.toolbar_icon.GetId()?
	wx.Bind(out.toolbar_icon, wx.EVT_TASKBAR_LEFT_DCLICK, out._on_toolbar_icon_left_dclick, wx.ID_ANY)
	wx.Bind(out.toolbar_icon, wx.EVT_TASKBAR_BALLOON_CLICK, out._on_toolbar_balloon_click, wx.ID_ANY)
	wx.Bind(out.toolbar_icon, wx.EVT_TASKBAR_BALLOON_TIMEOUT, out._on_toolbar_balloon_timeout, wx.ID_ANY)
	wx.Bind(out, wx.EVT_CLOSE_WINDOW, out._on_close, out.GetId())

	twitch_notifier_main := InitOurTwitchNotifierMain()
	twitch_notifier_main.options = parse_args()
	twitch_notifier_main.window_impl = out
	oauth_option := twitch_notifier_main.options.authorization_oauth
	if oauth_option != nil {
		twitch_notifier_main._auth_oauth = *oauth_option
	}
	out.main_obj = twitch_notifier_main

	if *twitch_notifier_main.options.help {
		flag.Usage()
		log.Fatal("Showing usage")
	}
	return out
}

// TIME HELPER STUFF

type WxTimeHelper struct {
	hostFrame               wx.Frame
	wx_event_id             int
	next_callback_num       int
	next_callback_num_mutex *sync.Mutex
	callbacks_map           map[int]func()
	callbacks_map_mutex     *sync.Mutex
}

var next_wx_event_id int = wx.ID_HIGHEST + 1

func NewWxTimeHelper(hostFrame wx.Frame) *WxTimeHelper {
	out := &WxTimeHelper{}

	out.next_callback_num_mutex = &sync.Mutex{}
	out.callbacks_map_mutex = &sync.Mutex{}
	out.callbacks_map = make(map[int]func())
	// get an event id for this particular WxTimeHelper
	out.wx_event_id = next_wx_event_id
	next_wx_event_id += 1
	out.hostFrame = hostFrame
	out.next_callback_num = 1

	// Set up an event handler on the host frame that we will use to bring execution into
	// the GUI thread
	wx.Bind(out.hostFrame, wx.EVT_THREAD, out.on_thread_event, out.wx_event_id)

	return out
}

type TimerWrapper struct {
	timer *time.Timer
	callback_num int
	helper *WxTimeHelper
}

func (wrapper *TimerWrapper) Stop() {
	wrapper.helper.pop_callback(wrapper.callback_num)
	wrapper.timer.Stop()
}

func (wrapper *TimerWrapper) Reset(d time.Duration) {
	wrapper.timer.Reset(d)
}

func (helper *WxTimeHelper) pop_callback(callback_num int) func() {
	helper.callbacks_map_mutex.Lock()
	callback, ok := helper.callbacks_map[callback_num]
	if ok {
		delete(helper.callbacks_map, callback_num)
	}
	helper.callbacks_map_mutex.Unlock()
	assert(ok, "error retrieving callback for WxTimeWrapper callback num %s", callback_num)
	return callback
}

func (helper *WxTimeHelper) on_thread_event(e wx.Event) {
	// get the callback num from the thread event
	threadEvent := wx.ToThreadEvent(e)
	callback_num := threadEvent.GetInt()

	// pop the callback out of the callbacks file
	callback := helper.pop_callback(callback_num)

	// call the callback
	callback()
}

/**
This wxGo doesn't have a wx.Timer analogous to the wxPython one.  That may be because go's built-in time.AfterFunc()
provides similar functionality, running a callback in a goroutine after a delay.
However wx GUI methods don't support calls outside the main thread.
So here's an AfterFunc implementation that wraps time.AfterFunc, and ships its callbacks to the wx main thread by
way of a wx.ThreadingEvent.

This approach is based on the wxGo threadevent example:
https://github.com/dontpanic92/wxGo/blob/master/examples/src/threadevent/main.go
 */
func (helper *WxTimeHelper) AfterFunc(duration time.Duration, callback func()) *TimerWrapper {
	// get a callback num and file the callback

	// TODO safeguard against id collision when we wrap around the int space
	helper.next_callback_num_mutex.Lock()
	callback_num := helper.next_callback_num
	helper.next_callback_num += 1
	helper.next_callback_num_mutex.Unlock()

	helper.callbacks_map_mutex.Lock()
	helper.callbacks_map[callback_num] = callback
	helper.callbacks_map_mutex.Unlock()

	// Do the real AfterFunc call with a callback that sets up an event to do the wrapper callback

	//msg("before delay for callback %s", callback_num)
	timer := time.AfterFunc(duration, func() {
		//msg("after delay for callback %s", callback_num)
		helper.on_call_complete(callback_num)
	})
	//msg("after calling real AfterFunc")

	return &TimerWrapper{timer, callback_num, helper}
}

func (helper *WxTimeHelper) on_call_complete(callback_num int) {
	// This method gets called in a thread other than the wx main thread, so it must only set up some thread events and cannot call into the GUI directly

	//msg("timer for callback %s complete", callback_num)
	threadEvent := wx.NewThreadEvent(wx.EVT_THREAD, helper.wx_event_id)
	threadEvent.SetInt(callback_num)
	helper.hostFrame.QueueEvent(threadEvent)
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
	if (win.timer == nil) {
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

func (win *MainStatusWindowImpl) set_timeout(length time.Duration, callback func() error) {
	assert(win.cur_timeout_timer == nil, "timer already in progress")
	win.cur_timeout_callback = callback
	//msg("before set_timeout AfterFunc")
	win.cur_timeout_timer = win.timeHelper.AfterFunc(length, win._on_timeout_timer_complete)
	//msg("after set_timeout AfterFunc")
}

func (win *MainStatusWindowImpl) set_balloon_click_callback(callback func() error) {
	win.balloon_click_callback = callback
}

func (win *MainStatusWindowImpl) enqueue_notification(title string, msg string, callback NotificationCallback) {
	notification := NotificationQueueEntry{callback.callback, title, msg}
	win.notifications_queue = append(win.notifications_queue, notification)
	if !win.notifications_queue_in_progress {
		// kick off the notification cycle
		win._dispense_remaining_notifications()
	}
}

func (win *MainStatusWindowImpl) _dispense_remaining_notifications() error {
	// hack to avoid double-triggering of events that happens
	if len(win.notifications_queue) == 0 {
		win.notifications_queue_in_progress = false
		return nil
	}

	win.notifications_queue_in_progress = true
	// pop a notification off the queue
	var notification NotificationQueueEntry = win.notifications_queue[0]
	win.notifications_queue = append(win.notifications_queue[:0], win.notifications_queue[1:]...)

	// show the notification
	win.set_balloon_click_callback(notification.callback)
	var delay_ms uint = 200
	var flags int = 0
	result := win.toolbar_icon.ShowBalloon(notification.title, notification.msg, delay_ms, flags, _get_asset_icon())
	assert(result, "error showing balloon")
	return nil
}

func (win *MainStatusWindowImpl) _on_timeout_timer_complete() {
	win.cur_timeout_timer = nil
	callback := win.cur_timeout_callback
	win.cur_timeout_callback = nil
	err := callback()
	if err != nil {
		win.main_obj.log(fmt.Sprintf("set_timeout callback returned error: %s", err))
	}
}

func (win *MainStatusWindowImpl) _on_toolbar_balloon_click(e wx.Event) {
	win.main_obj.log("notification clicked")
	if win.balloon_click_callback != nil {
		err := win.balloon_click_callback()
		if (err != nil) {
			win.main_obj.log(fmt.Sprintf("Balloon click callback returned error: %s", err))
		}
	}
	win.set_timeout(250 * time.Millisecond, win._dispense_remaining_notifications)
}

func (win *MainStatusWindowImpl) _on_toolbar_balloon_timeout(e wx.Event) {
	win.main_obj.log("notification timeout")
	// ok, on to the next
	win.set_timeout(250 * time.Millisecond, win._dispense_remaining_notifications)
}

func (win *MainStatusWindowImpl) _on_close(e wx.Event) {
	win.Hide()
}

// ICON HELPER FUNCTIONS

func _get_asset_icon_filename() string {
	subpath := "icon.ico"
	script_path, err := filepath.Abs(".")
	assert(err == nil, "icon path failed: %s", err)
	assets_path := path.Join(script_path, "..", "assets")
	bitmap_path := path.Join(assets_path, subpath)
	return bitmap_path
}

func _get_asset_icon() wx.Icon {
	the_icon := wx.NullIcon
	bitmap_path := _get_asset_icon_filename()
	loaded_bitmap := wx.NewBitmap(bitmap_path, wx.BITMAP_TYPE_ICO)
	the_icon.CopyFromBitmap(loaded_bitmap)
	return the_icon
}

// BASE APP CLASS

type TwitchNotifierMain struct {
	need_channels_refresh bool
	_auth_oauth string
	krakenInstance *Kraken
	options *Options
	windows_balloon_tip_obj WindowsBalloonTipInterface
	mainEventsInterface MainEventsInterface
}

func InitTwitchNotifierMain() *TwitchNotifierMain {
	out := &TwitchNotifierMain{}

	msg("init kraken")
	out.krakenInstance = InitKraken()

	out.need_channels_refresh = true
	out._auth_oauth = ""
	return out
}


func (app *TwitchNotifierMain) need_browser_auth() bool {
	return (!*app.options.no_browser_auth) && app._auth_oauth == ""
}

func msg(format string, a... interface{}) {
	message := fmt.Sprintf(format, a...)
	log.Println(message)
	//wx.MessageBox(message)
}

// Info about a stream, which is a current live video session happening on a channel
type StreamInfo struct {
	Channel     *ChannelInfo
	Is_playlist bool
	Id          StreamID	`json:"_id"`
	Created_at  string
	Game        *string
}

const CLIENT_ID = "pkvo0qdzjzxeapwpf8bfogx050n4bn8"

// COMMAND LINE OPTIONS STUFF

type Options struct {
	username *string
	no_browser_auth *bool
	poll *int
	all *bool
	idle *int
	unlock_notify *bool
	debug_output *bool
	authorization_oauth *string
	ui *bool
	no_popups *bool
	help *bool
}

func parse_args() *Options {
	options := &Options{}
	options.username = flag.String("username", "", "username to use")
	options.no_browser_auth = flag.Bool("no-browser-auth", true, "don't authenticate through twitch website login if token not supplied")
	options.poll = flag.Int("poll", 60, "poll interval")
	options.all = flag.Bool("all", false, "Watch all followed streams, not just ones with notifications enabled")
	options.idle = flag.Int("idle", 300, "idle time threshold to consider locked (seconds)")
	options.unlock_notify = flag.Bool("no-unlock-notify", true, "Don't notify again on unlock")
	options.debug_output = flag.Bool("debug", false, "Debug mode")
	options.authorization_oauth = flag.String("auth-oauth", "", "Authorization OAuth header value to send")
	options.ui = flag.Bool("ui", false, "Use the wxpython UI")
	options.no_popups = flag.Bool("no-popups", false, "Don't do popups, for when using just the UI")
	options.help = flag.Bool("help", false, "Show usage")
	flag.Parse()
	return options
}

// API RESPONSE DATA STRUCTURES

type ChannelID float64
type StreamID float64

// Info about a channel
type ChannelInfo struct {
	Id           ChannelID	`json:"_id"`
	Display_Name string
	Url          string
	Status       string
	// URL of the channel logo, if any
	Logo         *string
}

type StreamChannel struct {
	stream *StreamInfo
	channel *ChannelInfo
}

func (app *TwitchNotifierMain) log(msg string) {
	if *app.options.debug_output {
		log.Printf("%s TwitchNotifierMain: %s\n", time.Now(), msg)
	}
}

// This is for "virtual functions" in the base app class that should go through the extended app class
type MainEventsInterface interface {
	init_channel_display(followed_channel_entries []*ChannelInfo)
	stream_state_change(channel_id ChannelID, stream_we_consider_online bool, stream *StreamInfo)
	assume_all_streams_offline()
	done_state_changes()
	_channels_reload_complete()
	log(msg string)
}

// MORE APP METHODS

func (app *TwitchNotifierMain) init_channel_display(followed_channel_entries []*ChannelInfo) {
	// pass
}

func (app *OurTwitchNotifierMain) init_channel_display(followed_channel_entries []*ChannelInfo) {
	app.TwitchNotifierMain.init_channel_display(followed_channel_entries)

	msg("** init channel display with %v entries", len(followed_channel_entries))

	app.followed_channel_entries = followed_channel_entries
	app.reset_lists()
}

func (app *OurTwitchNotifierMain) _channel_for_id(channel_id ChannelID) *ChannelInfo {
	for _, channel := range app.followed_channel_entries {
		if channel.Id == channel_id {
			return channel
		}
	}
	return nil
}

func (app *OurTwitchNotifierMain) _list_for_is_online(online bool) wx.ListBox {
	if online {
		return app.window_impl.list_online
	} else {
		return app.window_impl.list_offline
	}
}

func (app *OurTwitchNotifierMain) doChannelsReload() {
	app.need_channels_refresh = true
	app.window_impl.setChannelRefreshInProgress(true)

	// If there is a main loop timer wait in progress we want to cancel it and do the next main
	// loop iteration right away
	app.window_impl.cancel_timer_callback_immediate()
}

func (app *OurTwitchNotifierMain) openSiteForListEntry(isOnline bool, e wx.Event) {
	commandEvent := wx.ToCommandEvent(e)

	index := commandEvent.GetInt()
	channel, stream := app.getChannelAndStreamForListEntry(isOnline, index)

	var url string
	if stream != nil {
		url = stream.Channel.Url
	} else if channel != nil {
		url = channel.Url
	} else {
		app.log("Channel is none somehow")
		return
	}

	webbrowser_open(url)
}

func (app *OurTwitchNotifierMain) _channels_reload_complete() {
	app.window_impl.setChannelRefreshInProgress(false)
}

/**
This is called when a channel has gone online or offline
 */
func (app *OurTwitchNotifierMain) stream_state_change(channel_id ChannelID, new_online bool, stream *StreamInfo) {
	msg("stream state change for channel %v", uint64(channel_id))
	val, ok := app.previously_online_streams[channel_id]
	if ok && val {
		delete(app.previously_online_streams, channel_id)
	}

	app.stream_by_channel_id[channel_id] = stream

	channel_obj := app._channel_for_id(channel_id)
	if channel_obj == nil {
		msg("skipping channel id %s state change check ")
		return
	}

	channel_status, channel_status_ok := app.channel_status_by_id[channel_id]
	assert(channel_status_ok, "channel status for channel %s not found", channel_id)
	assert(channel_status != nil, "nil channel status entry at %s", channel_id)
	old_online := channel_status.online
	if old_online != new_online {
		old_index := channel_status.idx
		out_of_list := app._list_for_is_online(old_online)
		out_of_list.Delete(old_index)

		into_list := app._list_for_is_online(new_online)
		new_index := into_list.GetCount()
		into_list.Append(app.channel_display_name(channel_obj))

		// update the later indexes
		for _, cur_status := range app.channel_status_by_id {
			if cur_status.online == old_online && cur_status.idx > old_index {
				cur_status.idx -= 1
			} else if cur_status.online == new_online && cur_status.idx >= new_index {
				cur_status.idx += 1
			}
		}

		channel_status.online = new_online
		channel_status.idx = new_index
	}
}


func (app *OurTwitchNotifierMain) done_state_changes() {
	msg("done state_change - figure out follows")
	// when we're only using the follows API, we we won't see another channel info when a stream goes offline
	// but we've keep track of which streams we saw in the previous update that we haven't seen again
	for channel_id, val := range app.previously_online_streams {
		msg("stream %s was online and now is %s", channel_id, val)
		// stream went offline
		new_online := false
		var no_stream *StreamInfo = nil
		if val {
			app.stream_state_change(channel_id, new_online, no_stream)
		}
	}
	app.previously_online_streams = make(map[ChannelID]bool)
}

// Note that this implementation will run the callback on another thread, so the callback needs to pass control
// back to the main thread.
func (app *OurTwitchNotifierMain) doDelayedUrlLoad(ctx string, url string, callback func(*http.Response)) {
	app.asynchttpclient.Get(url, func(err error, response *http.Response) {
		if err != nil {
			msg("error requesting %s: %s", url, err)
			callback(nil)
		} else {
			callback(response)
		}
	})
}

type ChannelStatus struct {
	online bool
	idx uint
}

func (app *TwitchNotifierMain) channel_display_name(channel *ChannelInfo) string {
	return fmt.Sprintf("%s (%v)", channel.Display_Name, uint32(channel.Id))
}

func (app *OurTwitchNotifierMain) reset_lists() {
	msg("resetting lists")
	app.window_impl.list_online.Clear()
	app.window_impl.list_offline.Clear()
	app.channel_status_by_id = make(map[ChannelID]*ChannelStatus)

	for i, channel := range app.followed_channel_entries {
		app.window_impl.list_offline.Append(app.channel_display_name(channel))
		channel_id := channel.Id
		app.channel_status_by_id[channel_id] = &ChannelStatus{false, uint(i)}
	}
	msg("done resetting lists")

}

func (app *TwitchNotifierMain) stream_state_change(channel_id ChannelID, stream_we_consider_online bool, stream *StreamInfo) {

}

func (app *TwitchNotifierMain) assume_all_streams_offline() {

}

func (app *OurTwitchNotifierMain) assume_all_streams_offline() {
	app.previously_online_streams = make(map[ChannelID]bool)
	for channel_id, channel_status := range app.channel_status_by_id {
		if channel_status.online {
			app.previously_online_streams[channel_id] = true
		}
	}
}

func (app *TwitchNotifierMain) done_state_changes() {

}

func (app *TwitchNotifierMain) _channels_reload_complete() {

}

func (app *TwitchNotifierMain) getEventsInterface() MainEventsInterface {
	if app.mainEventsInterface != nil {
		return app.mainEventsInterface
	} else {
		return app
	}
}

// Open a URL in the default browser
func webbrowser_open(url string) error {
	// Based on https://stackoverflow.com/questions/39320371/how-start-web-server-to-open-page-in-browser-in-golang

	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}
	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}

// Convert RFC3339 combined date and time with tz to time.Time
func convert_rfc3339_time(rfc3339_time string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, rfc3339_time)

	return t, err
}

// Convert time.Duration to hours/mins string
func time_desc(elapsed time.Duration) string {
	if elapsed.Hours() >= 1 {
		return fmt.Sprintf("%d h %02d m", elapsed / time.Hour, (elapsed / time.Minute) % 60)
	} else {
		return fmt.Sprintf("%d min", elapsed / time.Minute)
	}
}

// Create a notification for the given stream
func (app *TwitchNotifierMain) notify_for_stream(channel_name string, stream *StreamInfo) {
	created_at := stream.Created_at
	start_time, err := convert_rfc3339_time(created_at)
	assert(err == nil, "Error converting time %s", created_at)
	elapsed_s := time.Now().Round(time.Second).Sub(start_time.Round(time.Second))

	stream_browser_link := stream.Channel.Url
	game := stream.Game

	var show_info string
	if game == nil {
		show_info = ""
	} else {
		show_info = fmt.Sprintf("with %s ", *stream.Game)
	}

	message := fmt.Sprintf("%s is now live %s(up %s)", channel_name, show_info, time_desc(elapsed_s))

	msg("Showing message: '%s'", message)

	// Supply a callback to handle the event where the notification was clicked
	callback := NotificationCallback{channel_name, stream_browser_link}

	popupsEnabled := true
	if (app.options.no_popups != nil) {
		popupsEnabled = !*app.options.no_popups
	}

	if popupsEnabled && app.windows_balloon_tip_obj != nil {
		app.windows_balloon_tip_obj.balloon_tip("twitch-notifier", message, callback)
	}
}

// Interface for a desktop notification provider
type WindowsBalloonTipInterface interface {
	balloon_tip(title string, message string, callback NotificationCallback)
}


type OurWindowsBalloonTip struct {
	main_window *MainStatusWindowImpl
}

func NewOurWindowsBalloonTip(main_window *MainStatusWindowImpl) *OurWindowsBalloonTip {
	return &OurWindowsBalloonTip{main_window}
}

func (tip *OurWindowsBalloonTip) balloon_tip(title string, msg string, callback NotificationCallback) {
	tip.main_window.enqueue_notification(title, msg, callback)
}

type NotificationCallback struct {
	channel_name        string
	stream_browser_link string
}

func (callback NotificationCallback) callback() error {
	fmt.Printf("notification for %s clicked\n", callback.channel_name)
	webbrowser_open(callback.stream_browser_link)
	return nil
}

func (app *TwitchNotifierMain) diag_request(parts... string) {
	/** Do an API call and just pretty print the response contents */
	url_parts := strings.Join(parts, "/")
	msg("diag request for %s", url_parts)

	var output *map[string]interface{} = &map[string]interface{}{}
	err := app.krakenInstance.kraken(output, parts...)
	assert(err == nil, "diag request of %s failed with %s", url_parts, err)
	prettyOutput, err2 := json.MarshalIndent(output, "", "  ")
	assert(err == nil, "error remarshalling the json return value: %s", err2)
	msg("Output of request %s was %s", url_parts, prettyOutput)
}

func (app *TwitchNotifierMain) get_streams_channels_following(followed_channels map[ChannelID]bool) map[ChannelID]StreamChannel {
	out := map[ChannelID]StreamChannel{}

	pager, err := app.krakenInstance.PagedKraken("streams", 25, "streams", "followed")
	assert(err == nil, "followed request pager failed with %s", err)

	pager.AddParam("stream_type", "live")

	for pager.More() {
		var stream *StreamInfo
		err = pager.Next(&stream)
		assert(err == nil, "next return error: %s", err)
		assert(stream != nil, "stream was nil")
		channel := stream.Channel
		assert(channel != nil, "stream has no channel")
		channel_id := stream.Channel.Id
		val, ok := followed_channels[channel_id]
		if val && ok {
			out[channel_id] = StreamChannel{stream, channel}
		} else {
			app.getEventsInterface().log(fmt.Sprintf("skipping channel %s because it's not a followed channel", app.channel_display_name(stream.Channel)))
		}
	}

	return out
}

func (app *OurTwitchNotifierMain) _init_notifier() {
	assert(app.window_impl != nil, "window_impl not initialized in _init_notifier")
	app.windows_balloon_tip_obj = NewOurWindowsBalloonTip(app.window_impl)
}

func (app *OurTwitchNotifierMain) _notifier_fini() {
	// pass
}

type ChannelWatcher struct {
	app *OurTwitchNotifierMain
	channels_followed map[ChannelID]bool
	channel_info map[ChannelID]*ChannelInfo
	last_streams map[ChannelID]StreamID

	channels_followed_names []string
}

func (app *OurTwitchNotifierMain) NewChannelWatcher() *ChannelWatcher {
	msg("init notifier")
	app._init_notifier()
	watcher := &ChannelWatcher{}
	watcher.app = app
	watcher.channels_followed = make(map[ChannelID]bool)
	watcher.channel_info = make(map[ChannelID]*ChannelInfo)
	watcher.last_streams = make(map[ChannelID]StreamID)
	return watcher
}

func (watcher *ChannelWatcher) next() WaitItem {

	/* This does one API poll, potentially loading the user's list of followed channels first,
	   makes the calls to stuff in watcher.app to update followed stream details, and then
	   returns a token with info about the pause before the next poll so the caller can
	   sleep and/or schedule the next call
	 */

	app := watcher.app
	if app.need_channels_refresh {

		msg("doing a refresh")
		app.need_channels_refresh = false
		watcher.channels_followed = make(map[ChannelID]bool)
		watcher.channel_info = make(map[ChannelID]*ChannelInfo)
		watcher.last_streams = make(map[ChannelID]StreamID)
		watcher.channels_followed_names = []string{}

		// first time querying

		if app._auth_oauth != "" {
			authorization := "OAuth " + app._auth_oauth
			app.krakenInstance.addHeader("Authorization", authorization)

			// FIXME set fast query mode (support for slow query later)

			if app.options.username == nil || *app.options.username == "" {
				var root_response struct {
					Token struct {
						      User_Name string
					      }
				}
				//app.diag_request()
				msg("before kraken call for username")
				err := app.krakenInstance.kraken(&root_response)
				msg("after kraken call for username")
				assert(err == nil, "root request error: %s", err)
				assert(root_response.Token.User_Name != "", "got empty username from root request")
				app.options.username = &root_response.Token.User_Name

			}
		}

		notificationsDisabledFor := []string{}

		// FIXME this will want to change to something that will support the paged query

		// twitch.api.v3.follows.by_user
		type FollowEntry struct {
			Channel       *ChannelInfo
			Notifications bool
		}

		assert(app.options.username != nil, "username was nil during request")
		assert(*app.options.username != "", "username was empty during request")

		msg("got username")

		msg("before paged kraken call for follows by user response")
		resultsListKey := "follows"
		pageSize := uint(25)

		pager, err := app.krakenInstance.PagedKraken(resultsListKey, pageSize,
			"users", *app.options.username, "follows",
			"channels")
		msg("after paged kraken call for follows by user response")
		assert(err == nil, "follows pager error: %s", err)
		for pager.More() {
			var follow FollowEntry
			err = pager.Next(&follow)
			assert(err == nil, "follows get error: %s", err)

			channel := follow.Channel
			channel_id := channel.Id
			channel_name := channel.Display_Name
			msg("processing channel follow for %s", channel_name)
			notifications_enabled := follow.Notifications
			if *app.options.all || notifications_enabled {
				watcher.channels_followed[channel_id] = true
				watcher.channels_followed_names = append(watcher.channels_followed_names, channel_name)
				watcher.channel_info[channel_id] = channel
			} else {
				notificationsDisabledFor = append(notificationsDisabledFor, channel_name)
			}
		}

		msg("processing followed channels")

		followed_channel_entries := ChannelSlice{}

		for channel_id, present := range watcher.channels_followed {
			if !present {
				continue
			}
			followed_channel_entries = append(followed_channel_entries, watcher.channel_info[channel_id])
		}

		msg("sorting")

		sort.Sort(followed_channel_entries)

		msg("init channel display")

		app.getEventsInterface().init_channel_display(followed_channel_entries)

		msg("channels reload complete")

		app.getEventsInterface()._channels_reload_complete()
	} // done channels refresh

	// regular status change checks time
	log.Println("STUB: lock and idle check implementation")

	// FIXME just fast query implemented for now
	app.assume_all_streams_offline()
	channel_stream_iterator := app.get_streams_channels_following(watcher.channels_followed)
	for channel_id, channel_stream := range channel_stream_iterator {
		var channel *ChannelInfo = channel_stream.channel
		var stream *StreamInfo = channel_stream.stream
		assert(channel != nil, "channel_stream had no channel")
		channel_name := channel.Display_Name

		stream_we_consider_online := stream != nil && !stream.Is_playlist

		app.getEventsInterface().stream_state_change(channel_id, stream_we_consider_online, stream)

		if stream_we_consider_online {
			stream_id := stream.Id
			val, ok := watcher.last_streams[channel_id]
			//msg("stream fetch output: %v, %v", uint64(val), ok)
			if !ok || val != stream_id {
				//msg("before stream notify for %s", channel_name)
				app.notify_for_stream(channel_name, stream)
				//msg("after stream notify for %s", channel_name)
			}
			watcher.last_streams[channel_id] = stream_id
		} else {
			//msg("channel %s is offline", channel_name)
			if stream == nil {

				app.getEventsInterface().log(fmt.Sprintf("channel_id %s had stream null", channel_id))
			} else {
				app.getEventsInterface().log(fmt.Sprintf("channel_id  %s is_playlist %s", channel_id, stream.Is_playlist))
			}
			_, ok := watcher.last_streams[channel_id]
			if ok {
				delete(watcher.last_streams, channel_id)
			}
		}

	}

	app.getEventsInterface().done_state_changes()

	sleep_until_next_poll_s := *app.options.poll
	if sleep_until_next_poll_s < 60 {
		sleep_until_next_poll_s = 60
	}
	reason := fmt.Sprintf("Waiting %v s for next poll", sleep_until_next_poll_s)
	return WaitItem{time.Duration(sleep_until_next_poll_s) * time.Second, reason}

}

type ChannelSlice []*ChannelInfo

func (l ChannelSlice) Len() int {
	return len(l)
}

func (l ChannelSlice) Less(i, j int) bool {
	return l[i].Display_Name < l[j].Display_Name
}

func (l ChannelSlice) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}

func (app *TwitchNotifierMain) main_loop() {

}

type WaitItem struct {
	length time.Duration
	reason string
}

// EXTENDED APP CLASS

// Extends TwitchNotifierMain to provide channel lists and stream info needed for the GUI
type OurTwitchNotifierMain struct {
	TwitchNotifierMain
	window_impl *MainStatusWindowImpl
	main_loop_iter *ChannelWatcher
	followed_channel_entries []*ChannelInfo
	channel_status_by_id map[ChannelID]*ChannelStatus
	previously_online_streams map[ChannelID]bool
	stream_by_channel_id map[ChannelID]*StreamInfo
	asynchttpclient *asynchttpclient.Client
}

func InitOurTwitchNotifierMain() *OurTwitchNotifierMain {
	out := &OurTwitchNotifierMain{}
	out.TwitchNotifierMain = *InitTwitchNotifierMain()
	out.mainEventsInterface = out
	out.channel_status_by_id = make(map[ChannelID]*ChannelStatus)
	out.previously_online_streams = make(map[ChannelID]bool)
	out.stream_by_channel_id = make(map[ChannelID]*StreamInfo)
	out.asynchttpclient = &asynchttpclient.Client{}
	out.asynchttpclient.Concurrency = 3
	return out
}

// EVEN MORE APP METHODS

func (app *OurTwitchNotifierMain) cancelDelayedUrlLoadsForContext(ctx string) {
	msg("STUB: cancelDelayedUrlLoadsForContext('%s') - cancellation of async requests not implemented", ctx)
}

func (app *OurTwitchNotifierMain) getChannelAndStreamForListEntry(isOnline bool, index int) (*ChannelInfo, *StreamInfo) {
	var stream *StreamInfo = nil
	var channel *ChannelInfo = nil

	channelId := app.getChannelIdForListEntry(isOnline, index)
	if channelId != nil {
		stream = app.stream_by_channel_id[*channelId]
		channel = app._channel_for_id(*channelId)
		if channel == nil {
			app.log(fmt.Sprintf("Channel entry not found for id %v", channelId))
		}
	}

	return channel, stream
}

func (app *OurTwitchNotifierMain) getChannelIdForListEntry(isOnline bool, index int) *ChannelID {
	for channelId, curStatus := range app.channel_status_by_id {
		if curStatus.idx == uint(index) && curStatus.online == isOnline {
			return &channelId
		}
	}
	return nil
}

func (app *OurTwitchNotifierMain) main_loop_main_window_timer() error {
	if app.need_browser_auth() {
		assert(false, "need browser auth")
	} else {
		app.main_loop_main_window_timer_with_auth()
	}
	return nil
}

/** Do a poll of the API and set up a timer to get us to the next poll
 */
func (app *OurTwitchNotifierMain) set_next_time() {
	msg("doing iterator call")
	next_wait := app.main_loop_iter.next()
	app.log(next_wait.reason)
	app.window_impl.set_timer_with_callback(next_wait.length, app.set_next_time)
}

func (app *OurTwitchNotifierMain) main_loop_main_window_timer_with_auth() {
	msg("creating channel watcher")
	app.main_loop_iter = app.NewChannelWatcher()
	app.set_next_time()
}

/** Show a message in the normal log that is on-screen in the GUI window */
func (app *OurTwitchNotifierMain) log(message string) {
	line_item := fmt.Sprintf("%v: %s", time.Now(), message)
	msg("In log function, appending: %s", line_item)
	app.window_impl.list_log.Append(line_item)
	//msg("after log")
}


func main() {
	// initialize the handlers for all image formats so that wx.Bitmap routines can load all
	// supported image formats from disk
	wx.InitAllImageHandlers()

	app := wx.NewApp()
	frame := InitMainStatusWindowImpl()
	frame.app = app
	app.SetTopWindow(frame)
	msg("showing frame")
	frame.Show()

	// we're doing this in set_timeout so that it happens inside app.MainLoop() -- otherwise
	// the wx thread safeguard gets confused
	frame.set_timeout(0, frame.main_obj.main_loop_main_window_timer)

	msg("starting main loop")
	app.MainLoop()
	return
}
