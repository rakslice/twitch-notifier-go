package main

// Workaround for both wxGo and wxshowballoon both importing the headers

//#cgo LDFLAGS: -Wl,--allow-multiple-definition
import "C"

import (
	"fmt"
	"log"
	"github.com/dontpanic92/wxGo/wx"
	"flag"
	"sync"
	"time"
	"path/filepath"
	"path"
	"runtime"
	"os/exec"
	"strings"
	"encoding/json"
	"net/url"
	"sort"
	"wxshowballoon"
)


func assert(condition bool, message string, a... interface{}) {
	if !condition {
		formatted := fmt.Sprintf(message, a...)
		msg("assertion failure: %s", formatted)
		log.Fatal(formatted)
	}
}

func (win *MainStatusWindowImpl) _on_list_online_gen(e wx.Event) {
	win.main_obj.log("_on_list_online_gen")
}

func (win *MainStatusWindowImpl) _on_list_online_dclick(e wx.Event) {
	win.main_obj.log("_on_list_online_dclick")
}

func (win *MainStatusWindowImpl) _on_list_offline_gen(e wx.Event) {
	win.main_obj.log("_on_list_offline_gen")
}

func (win *MainStatusWindowImpl) _on_list_offline_dclick(e wx.Event) {
	win.main_obj.log("_on_list_offline_dclick")
}

func (win *MainStatusWindowImpl) _on_options_button_click(e wx.Event) {
	win.main_obj.log("_on_options_button_click")
}

func (win *MainStatusWindowImpl) _on_button_reload_channels_click(e wx.Event) {
	win.main_obj.log("_on_button_reload_channels_click")
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

type NotificationQueueEntry struct {
	callback func() error
	title string
	msg string
}

type MainStatusWindowImpl struct {
	MainStatusWindow
	app wx.App
	main_obj *OurTwitchNotifierMain
	toolbar_icon wx.TaskBarIcon
	cur_timeout_timer *TimerWrapper
	cur_timeout_callback func() error
	balloon_click_callback func() error

	notifications_queue []NotificationQueueEntry
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

	// last param should be a specific object id if we have one e.g. out.toolbar_icon.GetId()?
	wx.Bind(out.toolbar_icon, wx.EVT_TASKBAR_LEFT_DCLICK, out._on_toolbar_icon_left_dclick, wx.ID_ANY)
	wx.Bind(out.toolbar_icon, wx.EVT_TASKBAR_BALLOON_CLICK, out._on_toolbar_balloon_click, wx.ID_ANY)
	wx.Bind(out.toolbar_icon, wx.EVT_TASKBAR_BALLOON_TIMEOUT, out._on_toolbar_balloon_timeout, wx.ID_ANY)

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

	// init things in the struct
	out.next_callback_num_mutex = &sync.Mutex{}
	out.callbacks_map_mutex = &sync.Mutex{}
	out.callbacks_map = make(map[int]func())
	out.wx_event_id = next_wx_event_id
	next_wx_event_id += 1
	out.hostFrame = hostFrame
	out.next_callback_num = 1

	// set up an event handler on the host frame that we will use to bring execution into
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
time.AfterFunc() runs its callback in a goroutine. However we can't interact with wx GUI stuff outside the main thread.
So here's a wrapper for it that runs the callback in the wx main thread by passing a wx.ThreadingEvent
 */
func (helper *WxTimeHelper) AfterFunc(duration time.Duration, callback func()) *TimerWrapper {
	// get a callback num and file the callback

	// TODO wraparound collision safeguard
	helper.next_callback_num_mutex.Lock()
	callback_num := helper.next_callback_num
	helper.next_callback_num += 1
	helper.next_callback_num_mutex.Unlock()

	helper.callbacks_map_mutex.Lock()
	helper.callbacks_map[callback_num] = callback
	helper.callbacks_map_mutex.Unlock()

	// do real AfterFunc call with a callback that sets up an event to do the wrapper callback

	//msg("before delay for callback %s", callback_num)
	timer := time.AfterFunc(duration, func() {
		//msg("after delay for callback %s", callback_num)
		helper.on_call_complete(callback_num)
	})
	//msg("after calling real AfterFunc")

	return &TimerWrapper{timer, callback_num, helper}
}

func (helper *WxTimeHelper) on_call_complete(callback_num int) {
	// this gets called in a thread other than the wx main thread, so we must only set up some thread events and cannot call into the GUI directly
	//msg("timer for callback %s complete", callback_num)
	threadEvent := wx.NewThreadEvent(wx.EVT_THREAD, helper.wx_event_id)
	threadEvent.SetInt(callback_num)
	helper.hostFrame.QueueEvent(threadEvent)
}

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
		// kick off the progress
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
	result := wxshowballoon.ShowBalloon(win.toolbar_icon, notification.title, notification.msg, 200, 0, _get_asset_icon())
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

type StreamInfo struct {
	Channel     *ChannelInfo
	Is_playlist bool
	Id          StreamID	`json:"_id"`
	Created_at  string
	Game        *string
}

const CLIENT_ID = "pkvo0qdzjzxeapwpf8bfogx050n4bn8"

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
	popups *bool
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
	options.popups = flag.Bool("no-popups", true, "Don't do popups, for when using just the UI")
	options.help = flag.Bool("help", false, "Show usage")
	flag.Parse()
	return options
}

type ChannelID float64
type StreamID float64

type ChannelInfo struct {
	Id           ChannelID	`json:"_id"`
	Display_Name string
	Url          string
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

type MainEventsInterface interface {
	init_channel_display(followed_channel_entries []*ChannelInfo)
	stream_state_change(channel_id ChannelID, stream_we_consider_online bool, stream *StreamInfo)
	assume_all_streams_offline()
	done_state_changes()
	_channels_reload_complete()
	log(msg string)
}

func (app *TwitchNotifierMain) init_channel_display(followed_channel_entries []*ChannelInfo) {

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

type ChannelStatus struct {
	online bool
	idx uint
}

func (app *TwitchNotifierMain) channel_display_name(channel *ChannelInfo) string {
	return fmt.Sprintf("%s (%v)", channel.Display_Name, uint32(channel.Id))
}

func (app *OurTwitchNotifierMain) reset_lists() {
	msg("resetting lists")
	msg("app %s", app)
	msg("window_impl %s", app.window_impl)
	msg("list_online %s", app.window_impl.list_online)
	app.window_impl.list_online.Clear()
	//msg("here we are where we would have cleared lists...")
	//debug.PrintStack()
	msg("online clear")
	app.window_impl.list_offline.Clear()
	msg("offline clear")
	app.channel_status_by_id = make(map[ChannelID]*ChannelStatus)
	msg("done list clears")

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

func convert_iso_time(iso_time string) (time.Time, error) {
	layout := "2006-01-02T15:04:05Z"
	t, err := time.Parse(layout, iso_time)

	return t, err
}

func time_desc(elapsed_s time.Duration) string {
	return elapsed_s.String()
}

func (app *TwitchNotifierMain) notify_for_stream(channel_name string, stream *StreamInfo) {
	created_at := stream.Created_at
	start_time, err := convert_iso_time(created_at)
	assert(err == nil, "Error converting time %s", created_at)
	elapsed_s := time.Now().Round(time.Second).Sub(start_time.Round(time.Second))

	stream_browser_link := stream.Channel.Url
	game := stream.Game

	msg("got the info")

	var show_info string
	if game == nil {
		show_info = ""
	} else {
		show_info = fmt.Sprintf("with %s ", *stream.Game)
	}

	message := fmt.Sprintf("%s is now live %s(up %s)", channel_name, show_info, time_desc(elapsed_s))

	msg("built the message, events interface is %s", app.getEventsInterface())

	//app.getEventsInterface().log(fmt.Sprintf("Showing message: '%s'", message))

	msg("going to pass the message to balloon tip obj %s", app.windows_balloon_tip_obj)

	callback := NotificationCallback{channel_name, stream_browser_link}
	if *app.options.popups && app.windows_balloon_tip_obj != nil {
		app.windows_balloon_tip_obj.balloon_tip("twitch-notifier", message, callback)
	}
}

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

	params := url.Values{}
	params.Add("limit", "25")
	params.Add("offset", "0")
	params.Add("stream_type", "live")

	var followed_response struct {
		Streams *[]StreamInfo
	}

	//app.diag_request("streams", "followed?" + params.Encode())

	err := app.krakenInstance.kraken(&followed_response, "streams", "followed?" + params.Encode())
	assert(err == nil, "followed request failed with %s", err)
	assert(followed_response.Streams != nil, "followed request did not return a streams list")

	for _, stream := range *followed_response.Streams {
		channel := stream.Channel
		assert(channel != nil, "stream has no channel")
		channel_id := stream.Channel.Id
		val, ok := followed_channels[channel_id]
		if val && ok {
			out[channel_id] = StreamChannel{&stream, channel}
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

		var follows_by_user_response struct {
			Follows *[]FollowEntry
		}

		assert(app.options.username != nil, "username was nil during request")
		assert(*app.options.username != "", "username was empty during request")

		msg("got username")

		params := url.Values{}
		params.Add("limit", "25")
		params.Add("offset", "0")

		msg("before kraken call for follows by user response")
		err := app.krakenInstance.kraken(&follows_by_user_response,
			"users", *app.options.username, "follows",
			"channels?" + params.Encode())
		msg("after kraken call for follows by user response")
		assert(err == nil, "follows request error: %s", err)
		assert(follows_by_user_response.Follows != nil, "follows request did not have a follows list")
		for _, follow := range *follows_by_user_response.Follows {
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
			msg("stream fetch output: %v, %v", uint64(val), ok)
			if !ok || val != stream_id {
				msg("before stream notify for %s", channel_name)
				app.notify_for_stream(channel_name, stream)
				msg("after stream notify for %s", channel_name)
			}
			watcher.last_streams[channel_id] = stream_id
		} else {
			msg("channel %s is offline", channel_name)
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

type OurTwitchNotifierMain struct {
	TwitchNotifierMain
	window_impl *MainStatusWindowImpl
	main_loop_iter *ChannelWatcher
	followed_channel_entries []*ChannelInfo
	channel_status_by_id map[ChannelID]*ChannelStatus
	previously_online_streams map[ChannelID]bool
	stream_by_channel_id map[ChannelID]*StreamInfo
}

func InitOurTwitchNotifierMain() *OurTwitchNotifierMain {
	out := &OurTwitchNotifierMain{}
	out.TwitchNotifierMain = *InitTwitchNotifierMain()
	out.mainEventsInterface = out
	out.channel_status_by_id = make(map[ChannelID]*ChannelStatus)
	out.previously_online_streams = make(map[ChannelID]bool)
	out.stream_by_channel_id = make(map[ChannelID]*StreamInfo)
	return out
}

func (app *OurTwitchNotifierMain) main_loop_main_window_timer() error {
	if app.need_browser_auth() {
		assert(false, "need browser auth")
	} else {
		app.main_loop_main_window_timer_with_auth()
	}
	return nil
}

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

func (app *OurTwitchNotifierMain) log(message string) {
	line_item := fmt.Sprintf("%v: %s", time.Now(), message)
	msg("In log function, appending: %s", line_item)
	//debug.PrintStack()
	//msg("window %s", app.window_impl)
	//msg("list_log %s", app.window_impl.list_log)
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
