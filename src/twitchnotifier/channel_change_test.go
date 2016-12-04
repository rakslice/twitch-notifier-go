
package main

import (
	"testing"
	"github.com/jarcoal/httpmock"
	"github.com/dontpanic92/wxGo/wx"
)

func TestStreamsGoOffline(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	// initialize the handlers for all image formats so that wx.Bitmap routines can load all
	// supported image formats from disk
	wx.InitAllImageHandlers()

	app := wx.NewApp()
	frame := InitMainStatusWindowImpl(true)
	frame.app = app
	app.SetTopWindow(frame)
	msg("showing frame")
	frame.Show()

	fake_oauth_token := "fakeoauth123"
	msg("setting mock oauth token: %s", fake_oauth_token)
	frame.main_obj.options.authorization_oauth = &fake_oauth_token
	frame.main_obj._auth_oauth = fake_oauth_token

	msg("creating channel watcher")
	frame.main_obj.main_loop_iter = frame.main_obj.NewChannelWatcher()

	msg("mocking username call")
	httpmock.RegisterResponder("GET", "https://api.twitch.tv/kraken",
		httpmock.NewStringResponder(200, `{"token": {"user_name": "fakeusername"}}`))

	msg("mocking call to get followed channels")

	httpmock.RegisterResponder("GET", "https://api.twitch.tv/kraken/users/fakeusername/follows/channels?limit=25&offset=0",
		httpmock.NewStringResponder(200, `{"_total": 1, "follows": [{"notifications": true, "channel": {
		  "id": 123,
		  "display_name": "FakeChannel",
		  "url": "https://twitch.tv/fakechannel",
		  "status": "somestatus",
		  "logo": null
		}}]}`))

	msg("mocking first call to see what streams are up")

	httpmock.RegisterResponder("GET", "https://api.twitch.tv/kraken/streams/followed?limit=25&offset=0",
		httpmock.NewStringResponder(200, `{"_total": 1, "streams": [
			{"channel": {
				  "id": 123,
				  "display_name": "FakeChannel",
				  "url": "https://twitch.tv/fakechannel",
				  "status": "somestatus",
				  "logo": null
				},
			 "is_playlist": false,
			 "id": 456,
			 "created_at": "2016-01-01T01:01:01Z",
			 "game": "a vidya game"
			}
		]}`))

	msg("doing iterator call")
	next_wait := frame.main_obj.main_loop_iter.next()
	frame.main_obj.log(next_wait.reason)

	msg("checks")
	streams_online := frame.list_online.GetCount()
	streams_offline := frame.list_offline.GetCount()
	assertEqual(1, streams_online, "streams online")
	assertEqual(0, streams_offline, "streams offline")

	msg("mocking a second follow call where the stream has gone offline")
	httpmock.Reset()

	httpmock.RegisterResponder("GET", "https://api.twitch.tv/kraken/streams/followed?limit=25&offset=0",
		httpmock.NewStringResponder(200, `{"_total": 0, "streams": []}`))

	msg("do the next poll right away")
	next_wait = frame.main_obj.main_loop_iter.next()
	frame.main_obj.log(next_wait.reason)

	msg("checks")
	streams_online = frame.list_online.GetCount()
	streams_offline = frame.list_offline.GetCount()
	assertEqual(0, streams_online, "streams online")
	assertEqual(1, streams_offline, "streams offline")

}

func assertEqual(expectedValue uint, actualValue uint, desc string) {
	assert(expectedValue == actualValue, "%s expected %v, got %v", desc, expectedValue, actualValue)
}

