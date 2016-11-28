package main

import (
	"fmt"
	"github.com/dontpanic92/wxGo/wx"
	"path"
	"path/filepath"
	"log"
	"flag"
	"sort"
	"net/url"
	"time"
	"runtime"
	"os/exec"
)

func (win MainStatusWindowImpl) _on_list_online_gen(e wx.Event) {
	win.app.log("_on_list_online_gen")
}

func (win MainStatusWindowImpl) _on_list_online_dclick(e wx.Event) {
	win.app.log("_on_list_online_dclick")
}

func (win MainStatusWindowImpl) _on_list_offline_gen(e wx.Event) {
	win.app.log("_on_list_offline_gen")
}

func (win MainStatusWindowImpl) _on_list_offline_dclick(e wx.Event) {
	win.app.log("_on_list_offline_dclick")
}

func (win MainStatusWindowImpl) _on_options_button_click(e wx.Event) {
	win.app.log("_on_options_button_click")
}

func (win MainStatusWindowImpl) _on_button_reload_channels_click(e wx.Event) {
	win.app.log("_on_button_reload_channels_click")
}

func (win MainStatusWindowImpl) _on_button_quit(e wx.Event) {
	win.app.log("_on_button_quit")
}

type MainStatusWindowImpl struct {
	MainStatusWindow
	app *OurTwitchNotifierMain
}

func InitMainStatusWindowImpl() *MainStatusWindowImpl {
	out := &MainStatusWindowImpl{}
	out.MainStatusWindow = *initMainStatusWindow(out)
	return out
}

func _get_asset_icon() {
	subpath := "icon.ico"
	the_icon := wx.NullIcon

	script_path, err := filepath.Abs(".")
	if err != nil {
		log.Fatal(err)
	}
	assets_path := path.Join(script_path, "..", "assets")
	loaded_bitmap := wx.NewBitmap()
	loaded_bitmap.LoadFile(path.Join(assets_path, subpath))
	the_icon.CopyFromBitmap(loaded_bitmap)
}

type TwitchNotifierMain struct {
	need_channels_refresh bool
	_auth_oauth string
	krakenInstance *Kraken
	options *Options
	windows_balloon_tip_obj WindowsBalloonTipInterface
	mainEventsInterface MainEventsInterface
}

//type StreamInfo map[string]string
type StreamInfo struct {
	channel *ChannelInfo
	is_playlist bool
	_id string
	created_at string
	game *string
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

type ChannelInfo struct {
	_id string
	display_name string
	url string
}

type StreamChannel struct {
	stream *StreamInfo
	channel *ChannelInfo
}

func (app TwitchNotifierMain) log(msg string) {
	if *app.options.debug_output {
		log.Printf("%s TwitchNotifierMain: %s\n", time.Now(), msg)
	}
}

type MainEventsInterface interface {
	init_channel_display(followed_channel_entries []ChannelInfo)
	stream_state_change(channel_id string, stream_we_consider_online bool, stream *StreamInfo)
	assume_all_streams_offline()
	done_state_changes()
	_channels_reload_complete()
	log(msg string)
}

func (app TwitchNotifierMain) init_channel_display(followed_channel_entries []ChannelInfo) {

}

func (app TwitchNotifierMain) stream_state_change(channel_id string, stream_we_consider_online bool, stream *StreamInfo) {

}

func (app TwitchNotifierMain) assume_all_streams_offline() {

}

func (app TwitchNotifierMain) done_state_changes() {

}

func (app TwitchNotifierMain) _channels_reload_complete() {

}

func (app TwitchNotifierMain) getEventsInterface() MainEventsInterface {
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
	layout := "2006-01-02T15:04:05.000Z"
	t, err := time.Parse(layout, iso_time)

	return t, err
}

func time_desc(elapsed_s time.Duration) string {
	return elapsed_s.String()
}

func (app TwitchNotifierMain) notify_for_stream(channel_name string, stream *StreamInfo) {
	created_at := stream.created_at
	start_time, err := convert_iso_time(created_at)
	if err != nil {
		log.Fatal(fmt.Sprintf("Error converting time %s", created_at))
	}
	elapsed_s := time.Now().Sub(start_time)


	stream_browser_link := stream.channel.url
	game := stream.game

	var show_info string
	if game == nil {
		show_info = ""
	} else {
		show_info = fmt.Sprintf("with %s", stream.game)
	}

	message := fmt.Sprintf("%s is now live %s(up %s)", channel_name, show_info, time_desc(elapsed_s))

	app.getEventsInterface().log(fmt.Sprintf("Showing message: '%s'", message))

	callback := NotificationCallback{channel_name, stream_browser_link}
	if *app.options.popups && app.windows_balloon_tip_obj != nil {
		app.windows_balloon_tip_obj.balloon_tip("twitch-notifier", message, callback)
	}
}

type WindowsBalloonTipInterface interface {
	balloon_tip(title string, message string, callback NotificationCallback)
}



type NotificationCallback struct {
	channel_name        string
	stream_browser_link string
}

func (callback NotificationCallback) callback() {
	fmt.Printf("notification for %s clicked\n", callback.channel_name)
	webbrowser_open(callback.stream_browser_link)
}

func (app TwitchNotifierMain) get_streams_channels_following(followed_channels map[string]bool) map[string]StreamChannel {
	out := map[string]StreamChannel{}

	params := url.Values{}
	params.Add("limit", "25")
	params.Add("offset", "0")
	params.Add("stream_type", "live")

	var followed_response struct {
		streams *[]StreamInfo
	}

	err := app.krakenInstance.kraken(followed_response, "streams", "followed?" + params.Encode())
	if err != nil {
		log.Fatal("followed request failed with ", err)
	}
	if followed_response.streams == nil {
		log.Fatal("followed request did not return a streams list")
	}

	for _, stream := range *followed_response.streams {
		channel := stream.channel
		if channel == nil {
			log.Fatal("stream has no channel")
		}
		channel_id := stream.channel._id
		val, ok := followed_channels[channel_id]
		if val && ok {
			out[channel_id] = StreamChannel{&stream, channel}
		} else {
			app.getEventsInterface().log(fmt.Sprintf("skipping channel_id %s because it's not a followed channel", channel_id))
		}
	}

	return out
}

func (app TwitchNotifierMain) _init_notifier() {
	// FIXME notifications implementation
}

func (app TwitchNotifierMain) _notifier_fini() {
	// FIXME shut down notifications implementation
}

func (app TwitchNotifierMain) main_loop_yielder() {

	channels_followed := make(map[string]bool)
	channel_info := make(map[string]ChannelInfo)
	last_streams := make(map[string]string)

	channels_followed_names := []string{}

	app._init_notifier()

	app.krakenInstance = InitKraken()

	for {
		if app.need_channels_refresh {
			app.need_channels_refresh = false
			channels_followed = make(map[string]bool)
			channel_info = make(map[string]ChannelInfo)
			last_streams = make(map[string]string)
			channels_followed_names = []string{}

			// first time querying

			if app._auth_oauth != "" {
				authorization := "OAuth " + app._auth_oauth
				app.krakenInstance.addHeader("Authorization", authorization)

				// FIXME set fast query mode (support for slow query later)

				if app.options.username == nil {
					var root_response struct {
						token struct {
							      user_name string
						      }
					}
					err := app.krakenInstance.kraken(&root_response)
					if err != nil {
						log.Fatal("root request error", err)
					}
					app.options.username = &root_response.token.user_name

				}
			}

			notificationsDisabledFor := []string{}

			// FIXME this will wnat to change to something that will support the paged query

			// twitch.api.v3.follows.by_user
			type FollowEntry struct {
				channel *ChannelInfo
				notifications bool
			}

			var follows_by_user_response struct {
				follows *[]FollowEntry
			}

			if app.options.username == nil {
				log.Fatal("username was nil during request")
			}
			err := app.krakenInstance.kraken(&follows_by_user_response, "users", *app.options.username, "follows", "channels")
			if err != nil {
				log.Fatal("follows request error", err)
			}
			if follows_by_user_response.follows == nil {
				log.Fatal("follows request did not have a follows list")
			}
			for _, follow := range *follows_by_user_response.follows {
				channel := follow.channel
				channel_id := channel._id
				channel_name := channel.display_name
				notifications_enabled := follow.notifications
				if *app.options.all || notifications_enabled {
					channels_followed[channel_id] = true
					channels_followed_names = append(channels_followed_names, channel_name)
					channel_info[channel_id] = *channel
				} else {
					notificationsDisabledFor = append(notificationsDisabledFor, channel_name)
				}
			}

			followed_channel_entries := ChannelSlice{}

			for channel_id, present := range channels_followed {
				if !present {
					continue
				}
				followed_channel_entries = append(followed_channel_entries, channel_info[channel_id])
			}

			sort.Sort(followed_channel_entries)

			app.getEventsInterface().init_channel_display(followed_channel_entries)

			app.getEventsInterface()._channels_reload_complete()
		} // done channels refresh

		// regular status change checks time
		log.Println("STUB: lock and idle check implementation")

		// FIXME just fast query implemented for now
		channel_stream_iterator := app.get_streams_channels_following(channels_followed)
		for channel_id, channel_stream := range channel_stream_iterator {
			var channel *ChannelInfo = channel_stream.channel
			var stream *StreamInfo = channel_stream.stream
			if channel == nil {
				log.Fatal("channel_stream had no channel")
			}
			channel_name := channel.display_name

			stream_we_consider_online := stream != nil && !stream.is_playlist

			app.getEventsInterface().stream_state_change(channel_id, stream_we_consider_online, stream)

			if stream_we_consider_online {
				stream_id := stream._id
				val, ok := last_streams[channel_id]
				if ok && val != stream_id {
					app.notify_for_stream(channel_name, stream)
				}
				last_streams[channel_id] = stream_id
			} else {
				if stream == nil {

					app.getEventsInterface().log(fmt.Sprintf("channel_id %s had stream null", channel_id))
				} else {
					app.getEventsInterface().log(fmt.Sprintf("channel_id  %s is_playlist %s", channel_id, stream.is_playlist))
				}
				_, ok := last_streams[channel_id]
				if ok {
					delete(last_streams, channel_id)
				}
			}

		}

		app.getEventsInterface().done_state_changes()

		sleep_until_next_poll_s := *app.options.poll
		if sleep_until_next_poll_s < 60 {
			sleep_until_next_poll_s = 60
		}
		app.getEventsInterface().log(fmt.Sprintf("Waiting %s s for next poll", sleep_until_next_poll_s))
	}
}

type ChannelSlice []ChannelInfo

func (l ChannelSlice) Len() int {
	return len(l)
}

func (l ChannelSlice) Less(i, j int) bool {
	return l[i].display_name < l[j].display_name
}

func (l ChannelSlice) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}

func (app TwitchNotifierMain) main_loop() {

}

func InitTwitchNotifierMain() *TwitchNotifierMain {
	out := &TwitchNotifierMain{}

	out.need_channels_refresh = true
	out._auth_oauth = ""
	return out
}

type OurTwitchNotifierMain struct {
	TwitchNotifierMain
	window_impl *MainStatusWindowImpl
}

func InitOurTwitchNotifierMain() *OurTwitchNotifierMain {
	out := &OurTwitchNotifierMain{}
	out.TwitchNotifierMain = *InitTwitchNotifierMain()
	out.mainEventsInterface = out
	return out
}

func (app *OurTwitchNotifierMain) log(msg string) {
	line_item := fmt.Sprintf("%s: %s", time.Now(), msg)
	app.window_impl.list_log.Append(line_item)
}

func main() {

	// FIXME figure out where this part needs to fit:
	twitch_notifier_main := InitOurTwitchNotifierMain()
	twitch_notifier_main.options = parse_args()

	if *twitch_notifier_main.options.help {
		flag.Usage()
		return
	}

	// The original main code I was using to test the generated wx skeleton
	app := wx.NewApp()
	frame := InitMainStatusWindowImpl()
	frame.app = twitch_notifier_main
	twitch_notifier_main.window_impl = frame
	frame.Show()
	app.MainLoop()
	return
}
