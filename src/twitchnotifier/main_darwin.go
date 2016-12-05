
// +build darwin

package main

import (
	"os"
	"log"
	"io"
	"github.com/deckarep/gosx-notifier"
)

func main() {
	logFilename := userRelativePath("Library", "Logs", "twitch-notifier-go.log")
	logFileHandle, err := os.OpenFile(logFilename, os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
	assert(err == nil, "Error opening log file: %s", err)
	defer logFileHandle.Close()
	log.SetOutput(io.MultiWriter(os.Stderr, logFileHandle))

	commonMain()
}

func prefsRelativePath() []string {
	return []string{"Library", "Preferences"}
}

const shouldDoParse = false

func (win *MainStatusWindowImpl) osNotification(notification *NotificationQueueEntry) {

	assert(notification != nil, "called with null notification queue entry")

	note := gosxnotifier.NewNotification(notification.msg)
	
	note.Title = notification.title

	note.Sound = gosxnotifier.Basso

	note.Sender = "twitch-notifier-go"

	note.AppIcon = _get_asset_icon_filename()

	note.Link = notification.url

	err := note.Push()
	if err != nil {
		msg("notification implementation indicated that the notification for '%s' was not shown: %s", notification.msg, err)
	}

}

