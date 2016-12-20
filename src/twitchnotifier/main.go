package main

//#cgo windows LDFLAGS: -static-libgcc -static-libstdc++ -Wl,-Bstatic -lstdc++ -lpthread -Wl,-Bdynamic
import "C"

import (
	"flag"
)

// CONSTANTS AND SIMILAR

const CLIENT_ID = "pkvo0qdzjzxeapwpf8bfogx050n4bn8"

func getNeededTwitchScopes() []string {
	return []string{"user_read"} // required for /streams/followed
}

// COMMAND LINE OPTIONS STUFF

type Options struct {
	username                  *string
	no_browser_auth           *bool
	poll                      *int
	all                       *bool
	idle                      *int
	unlock_notify             *bool
	debug_output              *bool
	authorization_oauth       *string
	ui                        *bool
	no_popups                 *bool
	help                      *bool
	reload_time_interval_mins *uint
}

func parse_args() *Options {
	options := &Options{}
	options.username = flag.String("username", "", "username to use")
	options.no_browser_auth = flag.Bool("no-browser-auth", false, "don't authenticate through twitch website login if token not supplied")
	options.poll = flag.Int("poll", 60, "poll interval")
	options.all = flag.Bool("all", false, "Watch all followed streams, not just ones with notifications enabled")
	options.idle = flag.Int("idle", 300, "idle time threshold to consider locked (seconds)")
	options.unlock_notify = flag.Bool("no-unlock-notify", true, "Don't notify again on unlock")
	options.debug_output = flag.Bool("debug", false, "Debug mode")
	options.authorization_oauth = flag.String("auth-oauth", "", "Authorization OAuth header value to send")
	options.ui = flag.Bool("ui", false, "Use the wxpython UI")
	options.no_popups = flag.Bool("no-popups", false, "Don't do popups, for when using just the UI")
	options.help = flag.Bool("help", false, "Show usage")
	options.reload_time_interval_mins = flag.Uint("reload-time-interval", 60, "Number of minutes between automatic channel reloads")
	msg("before flag parse")
	flag.Parse()
	msg("after flag parse")
	return options
}

// main() functions proper are in main_<platform>.go
