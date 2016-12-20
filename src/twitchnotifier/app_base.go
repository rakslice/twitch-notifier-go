package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"
)

// BASE APP CLASS

type TwitchNotifierMain struct {
	need_channels_refresh   bool
	_auth_oauth             string
	krakenInstance          *Kraken
	options                 *Options
	windows_balloon_tip_obj WindowsBalloonTipInterface
	mainEventsInterface     MainEventsInterface
	queryPageSize           uint
}

func InitTwitchNotifierMain() *TwitchNotifierMain {
	out := &TwitchNotifierMain{}

	msg("init kraken")
	out.krakenInstance = InitKraken()

	out.krakenInstance.addHeader("Accept", "application/vnd.twitchtv.v3+json")

	out.need_channels_refresh = true
	out._auth_oauth = ""
	out.queryPageSize = 25

	return out
}

func (app *TwitchNotifierMain) need_browser_auth() bool {
	msg("options.no_browser_auth %s", app.options.no_browser_auth)
	if app.options.no_browser_auth != nil {
		msg("options.no_browser_auth %s", *app.options.no_browser_auth)
	}
	msg("app._auth_oauth %s", app._auth_oauth)
	return (app.options.no_browser_auth == nil || !*app.options.no_browser_auth) && app._auth_oauth == ""
}

func (app *TwitchNotifierMain) log(msg string) {
	if app.options.debug_output != nil && *app.options.debug_output {
		log.Printf("%s TwitchNotifierMain: %s\n", time.Now(), msg)
	}
}

// API RESPONSE DATA STRUCTURES

type ChannelID float64
type StreamID float64

// Info about a channel
type ChannelInfo struct {
	Id           ChannelID `json:"_id"`
	Display_Name string
	Url          string
	Status       string
	// URL of the channel logo, if any
	Logo *string
}

// Info about a stream, which is a current live video session happening on a channel
type StreamInfo struct {
	Channel     *ChannelInfo
	Is_playlist bool
	Id          StreamID `json:"_id"`
	Created_at  string
	Game        *string
}

// Pair of stream and channel for maps
type StreamChannel struct {
	stream  *StreamInfo
	channel *ChannelInfo
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

// MainEventsInterface stubs
func (app *TwitchNotifierMain) init_channel_display(followed_channel_entries []*ChannelInfo) {

}

func (app *TwitchNotifierMain) stream_state_change(channel_id ChannelID, stream_we_consider_online bool, stream *StreamInfo) {

}

func (app *TwitchNotifierMain) assume_all_streams_offline() {

}

func (app *TwitchNotifierMain) done_state_changes() {

}

func (app *TwitchNotifierMain) _channels_reload_complete() {

}

func (app *TwitchNotifierMain) getEventsInterface() MainEventsInterface {
	// Get the MainEventsInterface registered in app or fall back to its own
	if app.mainEventsInterface != nil {
		return app.mainEventsInterface
	} else {
		return app
	}
}

func (app *TwitchNotifierMain) channel_display_name(channel *ChannelInfo) string {
	return fmt.Sprintf("%s (%v)", channel.Display_Name, uint32(channel.Id))
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
	if app.options.no_popups != nil {
		popupsEnabled = !*app.options.no_popups
	}

	if popupsEnabled && app.windows_balloon_tip_obj != nil {
		app.windows_balloon_tip_obj.balloon_tip("twitch-notifier-go", message, callback, stream_browser_link)
	}
}

// Interface for a desktop notification provider
type WindowsBalloonTipInterface interface {
	balloon_tip(title string, message string, callback NotificationCallback, url string)
}

type OurWindowsBalloonTip struct {
	main_window *MainStatusWindowImpl
}

func NewOurWindowsBalloonTip(main_window *MainStatusWindowImpl) *OurWindowsBalloonTip {
	return &OurWindowsBalloonTip{main_window}
}

func (tip *OurWindowsBalloonTip) balloon_tip(title string, msg string, callback NotificationCallback, url string) {
	tip.main_window.enqueue_notification(title, msg, callback, url)
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

func (app *TwitchNotifierMain) diag_request(parts ...string) {
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

func (app *TwitchNotifierMain) nextWithRetry(pager *KrakenPager, val interface{}, httpErrorTries uint) error {
	var err error
	for httpErrorTries > 0 {
		err = pager.Next(val)
		if err != nil {
			krakenError, wasKrakenError := err.(*KrakenError)
			if wasKrakenError && krakenError != nil {
				if krakenError.statusCode != 200 {
					httpErrorTries -= 1
					app.getEventsInterface().log(fmt.Sprintf("Got HTTP error %v while loading item; tries left %v", krakenError.statusCode, httpErrorTries))
					continue
				}
			}
		}
		break
	}
	return err
}

func (app *TwitchNotifierMain) PagedKrakenWithRetry(httpErrorTries uint, resultsListKey string, pageSize uint, addParams *url.Values, path ...string) (*KrakenPager, error) {
	var err error
	var pager *KrakenPager
	for httpErrorTries > 0 {
		pager, err = app.krakenInstance.PagedKraken(resultsListKey, pageSize, addParams, path...)
		if err != nil {
			krakenError, wasKrakenError := err.(*KrakenError)
			if wasKrakenError && krakenError != nil {
				if krakenError.statusCode != 200 {
					httpErrorTries -= 1
					app.getEventsInterface().log(fmt.Sprintf("Got HTTP error %v while doing initial pager request; tries left %v", krakenError.statusCode, httpErrorTries))
					continue
				}
			}
		}
		break
	}
	return pager, err
}

func (app *TwitchNotifierMain) get_streams_channels_following(followed_channels map[ChannelID]bool) (map[ChannelID]StreamChannel, error) {
	out := map[ChannelID]StreamChannel{}

	initialRequestHTTPTries := uint(2)

	additionalParams := make(url.Values)
	additionalParams.Add("stream_type", "live")
	pager, err := app.PagedKrakenWithRetry(initialRequestHTTPTries, "streams", app.queryPageSize, &additionalParams, "streams", "followed")

	if err != nil {
		return out, err
	}

	for pager.More() {
		var stream *StreamInfo
		httpErrorTries := uint(2)
		err := app.nextWithRetry(pager, &stream, httpErrorTries)
		if err != nil {
			return out, err
		}

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

	return out, nil
}

func (app *TwitchNotifierMain) main_loop() {

}
