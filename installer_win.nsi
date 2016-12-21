# define name of installer
OutFile "twitchnotifier64.exe"

# define installation directory
InstallDir $PROGRAMFILES64\twitch-notifier-go

# For removing Start Menu shortcut in Windows 7
RequestExecutionLevel admin

Page components
Page instfiles

# start default section
Section
    # set the installation directory as the destination for the following actions
    SetOutPath $INSTDIR

    File src\twitchnotifier\twitch-notifier-go.exe

    # create the uninstaller
    WriteUninstaller "$INSTDIR\uninstall.exe"

    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\twitch-notifier-go" \
                     "DisplayName" "twitch-notifier-go"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\twitch-notifier-go" \
                     "UninstallString" "$\"$INSTDIR\uninstall.exe$\""
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\twitch-notifier-go" \
                     "DisplayIcon" "$\"$INSTDIR\twitch-notifier-go.exe$\""

SectionEnd

Section "Add to Start Menu"
    # create a shortcut in the start menu programs directory
    # point the new shortcut at the program uninstaller
    CreateShortCut "$SMPROGRAMS\twitch-notifier-go.lnk" "$INSTDIR\twitch-notifier-go.exe"
SectionEnd

Section "Run at Startup"
    CreateShortCut "$SMPROGRAMS\Startup\Run twitch-notifier-go on Startup.lnk" "$INSTDIR\twitch-notifier-go.exe" -hide
SectionEnd

# uninstaller section start
Section "uninstall"

    # first, delete the uninstaller (AT: ?)
    Delete "$INSTDIR\uninstall.exe"

    # clean up uninstaller registry key
    DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\twitch-notifier-go"

    # second, remove the optional main link from the start menu
    Delete "$SMPROGRAMS\twitch-notifier-go.lnk"

    # remove the optional startup link from the start menu
    Delete "$SMPROGRAMS\Startup\Run twitch-notifier-go on Startup.lnk"

    Delete "$INSTDIR\twitch-notifier-go.exe"

# uninstaller section end
SectionEnd
