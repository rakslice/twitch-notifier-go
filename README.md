# twitch-notifier-go

A quick golang port of my python twitch notifier.  Works on Windows; Linux support in progress.

This is past the proof-of-concept stage, and is usable for getting notifications, but doesn't have all the features of my Python twitch-notifier yet, and is certainly a work in progress.

## Usage

Run the built twitchnotifier.exe file with command line options.

You'll need to pass `-auth-oauth` with a token you can get from [https://twitchapps.com/tmi/](https://twitchapps.com/tmi/), as the wx wrapper I'm using doesn't support the wxHTML stuff that the python version uses to do the the Twitch OAuth login.

Also the slow channel stream checking used by the username-only mode in the python client isn't ported yet so there's no way around passing the OAuth token at the moment. 

## Options

    -auth-oauth TOKEN   - OAuth token to use
        
## Acknowledgments

Uses code for accessing the Twitch API from [Fugi](https://github.com/fugiman)'s [Kaet](https://github.com/fugiman/kaet)

GUI uses [wxWidgets](https://www.wxwidgets.org/) and [Liu Shengqiu (dontpanic92)](https://github.com/dontpanic92)'s [wxGo](https://github.com/dontpanic92/wxGo) wrapper for it.

## Building from source

First, go to [https://golang.org/dl/](https://golang.org/dl/) and grab the Windows install of Go 1.7.x.

Then in theory you can do:

	go get github.com/rakslice/twitch-notifier-go/twitchnotifier

or something like that? But that probably won't work just yet.

So the old fashioned way to get up and running is to:

1. Follow the directions at [https://github.com/dontpanic92/wxGo](https://github.com/dontpanic92/wxGo) to get wxGo installed
2. Download the twitch-notifier-go source
3. Fix those include paths
5. Use the same environment as for wxGo to `go build twitchnotifier` 


