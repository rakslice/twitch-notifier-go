package main

import (
	"fmt"
	"github.com/rakslice/wxGo/wx"
	"github.com/tomcatzh/asynchttpclient"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"
)

// EXTENDED APP CLASS

// Extends TwitchNotifierMain to provide channel lists and stream info needed for the GUI
type OurTwitchNotifierMain struct {
	TwitchNotifierMain
	window_impl               *MainStatusWindowImpl
	main_loop_iter            *ChannelWatcher
	followed_channel_entries  []*ChannelInfo
	channel_status_by_id      map[ChannelID]*ChannelStatus
	previously_online_streams map[ChannelID]bool
	stream_by_channel_id      map[ChannelID]*StreamInfo
	follow_notification       map[ChannelID]bool
	asynchttpclient           *asynchttpclient.Client
	need_relayout             bool
	lastReloadTime            time.Time
}

func InitOurTwitchNotifierMain() *OurTwitchNotifierMain {
	out := &OurTwitchNotifierMain{}
	out.TwitchNotifierMain = *InitTwitchNotifierMain()
	out.mainEventsInterface = out
	out.channel_status_by_id = make(map[ChannelID]*ChannelStatus)
	out.previously_online_streams = make(map[ChannelID]bool)
	out.stream_by_channel_id = make(map[ChannelID]*StreamInfo)
	out.follow_notification = make(map[ChannelID]bool)
	msg("before http client")
	out.asynchttpclient = &asynchttpclient.Client{}
	out.asynchttpclient.Concurrency = 3
	out.need_relayout = false
	out.lastReloadTime = time.Now()
	return out
}

// STRUCTS USED IN OurTwitchNotifierMain

type ChannelStatus struct {
	online bool
	idx    uint
}

type WaitItem struct {
	length time.Duration
	reason string
}

// METHODS TO IMPLEMENT MainEventsInterface

// These are "virtual" methods called from the enclosed TwitchNotifierMain

func (app *OurTwitchNotifierMain) init_channel_display(followed_channel_entries []*ChannelInfo) {
	app.TwitchNotifierMain.init_channel_display(followed_channel_entries)

	msg("** init channel display with %v entries", len(followed_channel_entries))

	app.followed_channel_entries = followed_channel_entries
	app.reset_lists()
}

/**
This is called when a channel has gone online or offline
*/
func (app *OurTwitchNotifierMain) stream_state_change(channel_id ChannelID, new_online bool, stream *StreamInfo) {
	msg("stream state change for channel %v", uint64(channel_id))

	app._store_updated_channel_info(stream.Channel)

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

		app.need_relayout = true
	}
}

func (app *OurTwitchNotifierMain) assume_all_streams_offline() {
	app.previously_online_streams = make(map[ChannelID]bool)
	for channel_id, channel_status := range app.channel_status_by_id {
		if channel_status.online {
			app.previously_online_streams[channel_id] = true
		}
	}
}

func (app *OurTwitchNotifierMain) done_state_changes() {
	msg("done state_change - figure out follows")
	// when we're only using the follows API, we we won't see another channel info when a stream goes offline
	// but we've keep track of which streams we saw in the previous update that we haven't seen again
	for channel_id, val := range app.previously_online_streams {
		msg("stream %v was online and now is %v", channel_id, val)
		// stream went offline
		new_online := false
		var no_stream *StreamInfo = nil
		if val {
			app.stream_state_change(channel_id, new_online, no_stream)
		}
	}
	app.previously_online_streams = make(map[ChannelID]bool)

	if app.need_relayout {
		app.window_impl.Frame.Layout()
		app.need_relayout = false
	}
}

func (app *OurTwitchNotifierMain) _channels_reload_complete() {
	app.window_impl.setChannelRefreshInProgress(false)
}

/** Show a message in the normal log that is on-screen in the GUI window */
func (app *OurTwitchNotifierMain) log(message string) {
	line_item := fmt.Sprintf("%v: %s", time.Now(), message)
	msg("In log function, appending: %s", line_item)
	app.window_impl.list_log.Insert(line_item, uint(0))
	//msg("after log")
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

func (app *OurTwitchNotifierMain) main_loop_main_window_timer() {
	//msg("need browser auth")
	if app.need_browser_auth() {
		msg("do browser auth")
		app.do_browser_auth()
	} else {
		msg("do not need browser auth")
		app.main_loop_main_window_timer_with_auth()
	}
}

func (app *OurTwitchNotifierMain) do_browser_auth() {
	//debug := app.options.debug_output == nil || *app.options.debug_output
	debug := true

	scopes := getNeededTwitchScopes()

	doBrowserAuth(app._auth_complete_callback, scopes, debug)
}

func (app *OurTwitchNotifierMain) _auth_complete_callback(token string) {
	assert(token != "", "token was empty")

	app._auth_oauth = token

	app.main_loop_main_window_timer_with_auth()
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

func (app *OurTwitchNotifierMain) _channel_for_id(channel_id ChannelID) *ChannelInfo {
	for _, channel := range app.followed_channel_entries {
		if channel.Id == channel_id {
			return channel
		}
	}
	return nil
}

func (app *OurTwitchNotifierMain) _store_updated_channel_info(updatedChannel *ChannelInfo) {
	if updatedChannel == nil {
		return
	}
	for i, existingChannel := range app.followed_channel_entries {
		if existingChannel.Id == updatedChannel.Id {
			app.followed_channel_entries[i] = updatedChannel
			break
		}
	}
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

func (app *OurTwitchNotifierMain) getUrlForListEntry(isOnline bool, index int) (string, bool) {
	channel, stream := app.getChannelAndStreamForListEntry(isOnline, index)

	var url string
	if stream != nil {
		url = stream.Channel.Url
	} else if channel != nil {
		url = channel.Url
	} else {
		app.log("Channel is none somehow")
		return "", false
	}
	return url, true
}

func (app *OurTwitchNotifierMain) openSiteForListEntry(isOnline bool, e wx.Event) {
	commandEvent := wx.ToCommandEvent(e)

	index := commandEvent.GetInt()

	url, found := app.getUrlForListEntry(isOnline, index)

	if found {
		webbrowser_open(url)
	}
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

func (app *OurTwitchNotifierMain) _init_notifier() {
	assert(app.window_impl != nil, "window_impl not initialized in _init_notifier")
	app.windows_balloon_tip_obj = NewOurWindowsBalloonTip(app.window_impl)
}

func (app *OurTwitchNotifierMain) _notifier_fini() {
	// pass
}

type ChannelWatcher struct {
	app               *OurTwitchNotifierMain
	channels_followed map[ChannelID]bool
	channel_info      map[ChannelID]*ChannelInfo
	last_streams      map[ChannelID]StreamID

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
	/* This method does one API poll, potentially loading the user's list of followed channels
	   first, makes the calls to stuff in watcher.app to update followed stream details, and then
	   returns a token with info about the pause before the next poll so the caller can
	   sleep and/or schedule the next call
	*/

	// check if it's time to do a channel reload
	curTime := time.Now()
	elapsedSinceLastRefresh := curTime.Sub(watcher.app.lastReloadTime)
	msg("%0.2f seconds since last refresh", elapsedSinceLastRefresh.Seconds())
	app := watcher.app

	var reloadTimeInterval time.Duration
	if app.options.reload_time_interval_mins != nil {
		reloadTimeInterval = time.Duration(*app.options.reload_time_interval_mins) * time.Minute
	} else {
		reloadTimeInterval = 10 * time.Minute
	}
	msg("%0.2f seconds between autorefreshes", reloadTimeInterval.Seconds())
	if elapsedSinceLastRefresh >= reloadTimeInterval {
		app.need_channels_refresh = true
		msg("doing scheduled refresh")
	}

	// do channel reload if necessary
	if app.need_channels_refresh {
		msg("doing a refresh")
		watcher.app.lastReloadTime = curTime
		app.need_channels_refresh = false
		watcher.channels_followed = make(map[ChannelID]bool)
		watcher.channel_info = make(map[ChannelID]*ChannelInfo)
		watcher.channels_followed_names = []string{}

		// first time querying

		if app._auth_oauth != "" {
			var authToken string = strings.TrimSpace(app._auth_oauth)

			tmiOauthPrefix := "oauth:"
			if strings.HasPrefix(authToken, tmiOauthPrefix) {
				authToken = authToken[len(tmiOauthPrefix):]
			}

			authorization := "OAuth " + authToken
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

		pager, err := app.krakenInstance.PagedKraken(resultsListKey, app.queryPageSize, nil,
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
			if (app.options.all != nil && *app.options.all) || notifications_enabled {
				watcher.channels_followed[channel_id] = true
				watcher.channels_followed_names = append(watcher.channels_followed_names, channel_name)
				watcher.channel_info[channel_id] = channel
				watcher.app.follow_notification[channel_id] = notifications_enabled
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
	channel_stream_iterator, streamsError := app.get_streams_channels_following(watcher.channels_followed)

	if streamsError != nil {
		app.getEventsInterface().log(fmt.Sprintf("Error during update streams follows request: %s", streamsError))
		app.getEventsInterface().log("Processing any partial update and waiting until the next request time")
	} else {
		app.assume_all_streams_offline()
	}

	for channel_id, channel_stream := range channel_stream_iterator {
		var channel *ChannelInfo = channel_stream.channel
		var stream *StreamInfo = channel_stream.stream
		assert(channel != nil, "channel_stream had no channel")
		channel_name := channel.Display_Name

		// update channel info from the channel object in this stream
		watcher.channel_info[channel_id] = channel

		stream_we_consider_online := stream != nil && !stream.Is_playlist

		app.getEventsInterface().stream_state_change(channel_id, stream_we_consider_online, stream)

		if stream_we_consider_online {
			stream_id := stream.Id
			val, ok := watcher.last_streams[channel_id]
			//msg("stream fetch output: %v, %v", uint64(val), ok)
			if !ok || val != stream_id {
				ok, notifications_enabled := app.follow_notification[channel_id]
				if ok && notifications_enabled {
					app.notify_for_stream(channel_name, stream)
				}
			}
			watcher.last_streams[channel_id] = stream_id
		} else {
			//msg("channel %s is offline", channel_name)
			if stream == nil {

				app.getEventsInterface().log(fmt.Sprintf("channel_id %v had stream null", channel_id))
			} else {
				app.getEventsInterface().log(fmt.Sprintf("channel_id %v is_playlist %v", channel_id, stream.Is_playlist))
			}
			_, ok := watcher.last_streams[channel_id]
			if ok {
				delete(watcher.last_streams, channel_id)
			}
		}

	}

	app.getEventsInterface().done_state_changes()

	var sleep_until_next_poll_s int
	if app.options.poll == nil {
		sleep_until_next_poll_s = 60
	} else {
		sleep_until_next_poll_s = *app.options.poll
	}

	if sleep_until_next_poll_s < 60 {
		sleep_until_next_poll_s = 60
	}
	reason := fmt.Sprintf("Waiting %v s for next poll", sleep_until_next_poll_s)
	return WaitItem{time.Duration(sleep_until_next_poll_s) * time.Second, reason}

}

// SORTABLE LIST OF CHANNELS

type ChannelSlice []*ChannelInfo

func (l ChannelSlice) Len() int {
	return len(l)
}

func (l ChannelSlice) Less(i, j int) bool {
	return strings.ToLower(l[i].Display_Name) < strings.ToLower(l[j].Display_Name)
}

func (l ChannelSlice) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}
