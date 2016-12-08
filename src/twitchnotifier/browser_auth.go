package main

/**
Use a wx.Dialog with a wx.WebView to do an OAuth login to twitch.tv
 */

import (
	"net/url"
	"github.com/dontpanic92/wxGo/wx"
	"strings"
)

type BrowserAuthDialogCallback func(string, *url.URL)

type BrowserAuthDialog struct {
	wx.Dialog
	sizer               wx.BoxSizer
	browser             wx.WebView
	debug               bool

	callbacksForSchemes map[string]BrowserAuthDialogCallback

	lastCallbackURL     string
}

func InitBrowserAuthDialog(debug bool) *BrowserAuthDialog {
	out := &BrowserAuthDialog{}
	msg("before newdialog")
	out.Dialog = wx.NewDialog(wx.NullWindow, wx.ID_ANY, "twitch-notifier")
	msg("after newdialog")
	out.debug = debug
	out.lastCallbackURL = ""
	out.sizer = wx.NewBoxSizer(wx.VERTICAL)
	out.browser = wx.WebViewNew(out, wx.ID_ANY)

	out.callbacksForSchemes = make(map[string]BrowserAuthDialogCallback)
	out.lastCallbackURL = ""

	wx.Bind(out, wx.EVT_WEBVIEW_NAVIGATING, out.onNavigating, out.browser.GetId())
	wx.Bind(out, wx.EVT_WEBVIEW_NAVIGATED, out.onNavigated, out.browser.GetId())

	out.sizer.Add(out.browser, 1, wx.EXPAND, 10)
	out.SetSizer(out.sizer)
	out.SetSize(wx.NewSize(700, 700))
	return out
}

func (browserDialog *BrowserAuthDialog) setSchemeCallback(scheme string, callback BrowserAuthDialogCallback) {
	browserDialog.callbacksForSchemes[scheme] = callback
}

func (browserDialog *BrowserAuthDialog) onNavigating(e wx.Event) {
	msg("_on_navigating")
	event := wx.ToWebViewEvent(e)
	toUrl := event.GetURL()
	if browserDialog.debug {
		msg("NAVIGATING %s", toUrl)
	}
	browserDialog.onNewURLOpen(toUrl)
}

func (browserDialog *BrowserAuthDialog) onNavigated(e wx.Event) {
	msg("_on_navigated")
	event := wx.ToWebViewEvent(e)
	toUrl := event.GetURL()
	if browserDialog.debug {
		msg("NAVIGATED %s", toUrl)
	}
	browserDialog.onNewURLOpen(toUrl)
}

func (browserDialog *BrowserAuthDialog) onNewURLOpen(urlToParse string) {
	parsed, err := url.Parse(urlToParse)
	assert(err == nil, "Error parsing url '%s'", urlToParse)

	scheme := parsed.Scheme
	callback, gotCallback := browserDialog.callbacksForSchemes[scheme]
	if gotCallback && urlToParse != browserDialog.lastCallbackURL {
		browserDialog.lastCallbackURL = urlToParse
		callback(urlToParse, parsed)
	}
}

/**
This runs the browser auth in a standalone wx.App, shutting it down and running the given
 callback when the auth is done.
 */
func doBrowserAuth(tokenCallback func(string), scopes []string, debug bool) {
	msg("do_browsser newapp")
	app := wx.NewApp()
	msg("init browser dialog")
	dialog := InitBrowserAuthDialog(debug)

	dialog.setSchemeCallback("notifier", func(urlFromCall string, parsed *url.URL) {
		assert(parsed != nil, "no parsed url in callback param")

		fragment := parsed.Fragment
		qs, err := url.ParseQuery(fragment)
		assert(err == nil, "Error parsing fragment %s", err)

		tokens, gotTokens := qs["access_token"]
		assert(gotTokens, "No access_token param found in fragment")
		assert(len(tokens) == 1, "Expected 1 access_token in fragment")
		token := tokens[0]

		if debug {
			msg("done - we visisted %s", urlFromCall)
		}

		dialog.Close()
		app.ExitMainLoop()

		if tokenCallback != nil {
			tokenCallback(token)
		}
	})

	redirectURI := "notifier://main"

	msg("getting auth url")
	authURL := getAuthURL(CLIENT_ID, redirectURI, scopes, nil)
	msg("loading auth url %s", authURL)

	dialog.browser.LoadURL(authURL)
	msg("dialog show")
	dialog.Show()
	msg("dialog mainloop()")
	app.MainLoop()
}

func getAuthURL(clientId string, redirectURI string, scopes []string, state *string) string {
	baseURL := "https://api.twitch.tv/kraken/oauth2/authorize"

	params := make(url.Values)
	params.Add("response_type", "token")
	params.Add("client_id", clientId)
	params.Add("redirect_uri", redirectURI)
	params.Add("scope", strings.Join(scopes, " "))
	if state != nil {
		params.Add("state", *state)
	}
	return baseURL + "?" + params.Encode()
}
