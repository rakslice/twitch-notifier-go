#!/bin/bash
set -e -x

function ifrm() {
	if [ -f "$1" ]; then
		rm "$1"
	fi
}

ifrm $GOPATH/bin/twitchnotifier

rm -rf twitch-notifier-go.app

(cd src/twitchnotifier; go install -ldflags=-s -x)

bash create_macos_bundle.sh

codesign -s V6TRUAB28A twitch-notifier-go.app

ifrm ~/Library/Logs/twitch-notifier-go.log

rm -rf twitch-notifier-go-mac
mkdir twitch-notifier-go-mac
cp -R twitch-notifier-go.app twitch-notifier-go-mac/
cp LICENSE README.md twitch-notifier-go-mac/

ifrm twitch-notifier-go-mac.zip
zip -r twitch-notifier-go-mac.zip twitch-notifier-go-mac
rm -rf twitch-notifier-go-mac
