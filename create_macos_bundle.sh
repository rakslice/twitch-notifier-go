#!/bin/bash
set -e -x

# set up paths
mkdir -p twitch-notifier-go.app/Contents/MacOS
mkdir -p twitch-notifier-go.app/Contents/Resources
mkdir -p twitch-notifier-go.app/Contents/Resources/English.lproj

# binary
cp ../../../../bin/twitchnotifier twitch-notifier-go.app/Contents/MacOS/
SetFile -t APPL twitch-notifier-go.app/Contents/MacOS/twitchnotifier

# info.plist
cp Info.plist twitch-notifier-go.app/Contents/

# PkgInfo
echo -n 'APPL????' > twitch-notifier-go.app/Contents/PkgInfo

# Icon
# - convert icon to icns
sips -s format icns src/twitchnotifier/assets/icon.ico --out twitch-notifier-go.app/Contents/Resources/icon.icns
# - also copy the windows icon file as wxWidgets can open that
cp src/twitchnotifier/assets/icon.ico twitch-notifier-go.app/Contents/Resources/
