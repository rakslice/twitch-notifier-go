
// +build darwin

package main

import (
	"os"
	"log"
	"io"
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
