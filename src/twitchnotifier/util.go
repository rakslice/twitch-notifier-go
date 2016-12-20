package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path"
	"runtime"
	"time"
)

// BASIC UTILS

func assert(condition bool, message string, a ...interface{}) {
	if !condition {
		formatted := fmt.Sprintf(message, a...)
		msg("assertion failure: %s", formatted)
		log.Fatal(formatted)
	}
}

func msg(format string, a ...interface{}) {
	message := fmt.Sprintf(format, a...)
	log.Println(message)
	//wx.MessageBox(message)
}

// FILE/PATH STUFF

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

func userRelativePath(parts ...string) string {
	curUser, userErr := user.Current()
	assert(userErr == nil, "error getting current user: %s", userErr)
	assert(curUser != nil, "current user was nil")

	newParts := append([]string{curUser.HomeDir}, parts...)

	logFilename := path.Join(newParts...)
	return logFilename
}

func readToTempFile(readCloser io.ReadCloser) (string, error) {
	tempfile, err := ioutil.TempFile(os.TempDir(), "")
	if err != nil {
		return "", err
	}
	defer tempfile.Close()

	tempfileName := tempfile.Name()

	msg("Saving to %s", tempfileName)

	io.Copy(tempfile, readCloser)
	return tempfileName, nil
}

// WEB UTILS

// Open a URL in the default browser
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

// TIME UTILS

// Convert RFC3339 combined date and time with tz to time.Time
func convert_rfc3339_time(rfc3339_time string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, rfc3339_time)

	return t, err
}

// Convert time.Duration to hours/mins string
func time_desc(elapsed time.Duration) string {
	if elapsed.Hours() >= 1 {
		return fmt.Sprintf("%d h %02d m", elapsed/time.Hour, (elapsed/time.Minute)%60)
	} else {
		return fmt.Sprintf("%d min", elapsed/time.Minute)
	}
}
