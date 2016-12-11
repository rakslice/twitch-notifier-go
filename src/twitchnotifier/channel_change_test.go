package main

import (
	"testing"
	"time"

	"github.com/dontpanic92/wxGo/wx"
	"github.com/jarcoal/httpmock"
)

// Until we get the issues with app deletion sorted out, let's just reuse
// the same app instance for all the tests
var commonAppForTests wx.App
var appInitialized bool

type guiTestFuncType func(*testing.T, *MainStatusWindowImpl, func())

func commonGuiTestAsync(t *testing.T, testFunc guiTestFuncType) {
	frame := commonTestStart()

	// set up the test func to run, and then run main loop until the
	// done callback happens
	app := frame.app

	frame.set_timeout(0, func() {
		// run the test fun...
		testFunc(t, frame, func() {
			// and when it's done stop the main loop
			app.ExitMainLoop()
		})

	})

	msg("starting test mainloop")
	app.MainLoop()
	msg("ending test mainloop")

	msg("frame Shutdown")
	frame.Shutdown()
	msg("frame Destroy")
	frame.Destroy()
}

func commonTestStart() *MainStatusWindowImpl {
	preApp()

	if !appInitialized {
		appInitialized = true
		msg("initializing app")
		commonAppForTests = wx.NewApp()
		msg("app initialized")
	} else {
		msg("app already initialized")
	}

	app := commonAppForTests

	frame := InitMainStatusWindowImpl(true, func() *Options {
		return &Options{}
	})
	frame.app = app
	app.SetTopWindow(frame)
	msg("showing frame")
	frame.Show()
	return frame
}

func commonTestEnd(frame *MainStatusWindowImpl) {
	// let's try running the main loop for a little bit
	app := frame.app

	frame.set_timeout(500*time.Millisecond, func() {
		app.ExitMainLoop()
	})

	msg("starting test mainloop")
	app.MainLoop()
	msg("ending test mainloop")

	msg("frame Shutdown")
	frame.Shutdown()
	msg("frame Destroy")
	frame.Destroy()
}

func TestExactLastPage(t *testing.T) {
	commonGuiTestAsync(t, guiTestExactLastPage)
}

func guiTestExactLastPage(t *testing.T, frame *MainStatusWindowImpl, testDoneCallback func()) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	fake_oauth_token := "fakeoauth123"
	msg("setting mock oauth token: %s", fake_oauth_token)
	frame.main_obj.options.authorization_oauth = &fake_oauth_token
	frame.main_obj._auth_oauth = fake_oauth_token

	msg("set small pages")
	frame.main_obj.queryPageSize = 1

	msg("creating channel watcher")
	frame.main_obj.main_loop_iter = frame.main_obj.NewChannelWatcher()

	msg("mocking username call")
	httpmock.RegisterResponder("GET", "https://api.twitch.tv/kraken",
		httpmock.NewStringResponder(200, `{"token": {"user_name": "fakeusername"}}`))

	msg("mocking call to get followed channels")

	httpmock.RegisterResponder("GET", "https://api.twitch.tv/kraken/users/fakeusername/follows/channels?limit=1&offset=0",
		httpmock.NewStringResponder(200, `{"_total": 1, "follows": [{"notifications": true, "channel": {
		  "id": 123,
		  "display_name": "FakeChannel",
		  "url": "https://twitch.tv/fakechannel",
		  "status": "somestatus",
		  "logo": null
		}}]}`))

	msg("mocking first call to see what streams are up")

	httpmock.RegisterResponder("GET", "https://api.twitch.tv/kraken/streams/followed?limit=1&offset=0&stream_type=live",
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

	testDoneCallback()
}

func TestStreamsGoOffline(t *testing.T) {
	commonGuiTestAsync(t, guiTestStreamsGoOffline)
}

func guiTestStreamsGoOffline(t *testing.T, frame *MainStatusWindowImpl, testDoneCallback func()) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

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

	httpmock.RegisterResponder("GET", "https://api.twitch.tv/kraken/streams/followed?limit=25&offset=0&stream_type=live",
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

	httpmock.RegisterResponder("GET", "https://api.twitch.tv/kraken/streams/followed?limit=25&offset=0&stream_type=live",
		httpmock.NewStringResponder(200, `{"_total": 0, "streams": []}`))

	msg("do the next poll right away")
	next_wait = frame.main_obj.main_loop_iter.next()
	frame.main_obj.log(next_wait.reason)

	msg("checks")
	streams_online = frame.list_online.GetCount()
	streams_offline = frame.list_offline.GetCount()
	assertEqual(0, streams_online, "streams online")
	assertEqual(1, streams_offline, "streams offline")

	testDoneCallback()
}

func assertEqual(expectedValue uint, actualValue uint, desc string) {
	assert(expectedValue == actualValue, "%s expected %v, got %v", desc, expectedValue, actualValue)
}
