!define APPNAME twitch-notifier-go
!define ARP "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}"
!include "FileFunc.nsh"

# This should be updated to agree with versioninfo.json
!define VERSION_MAJOR 0
!define VERSION_MINOR 0
!define POINT_VERSION 1
!define BUILD_VERSION 0
!define DISPLAY_VERSION ${VERSION_MAJOR}.${VERSION_MINOR}.${POINT_VERSION}.${BUILD_VERSION}

Name twitch-notifier-go

OutFile "twitchnotifier64.exe"

LoadLanguageFile "${NSISDIR}\Contrib\Language files\English.nlf"

# version tag info to go into the installer and uninstaller exes
VIProductVersion "${DISPLAY_VERSION}"
VIAddVersionKey /LANG=${LANG_ENGLISH} "ProductName" "twitch-notifier-go"
VIAddVersionKey /LANG=${LANG_ENGLISH} "CompanyName" "rakslice"
VIAddVersionKey /LANG=${LANG_ENGLISH} "LegalCopyright" "©2016 rakslice and others; see LICENSE file for terms"
VIAddVersionKey /LANG=${LANG_ENGLISH} "FileDescription" "twitch-notifier-go installer"
VIAddVersionKey /LANG=${LANG_ENGLISH} "FileVersion" "v${DISPLAY_VERSION}"


# default installation directory
InstallDir $PROGRAMFILES64\twitch-notifier-go

RequestExecutionLevel admin

Page license

LicenseText "License for twitch-notifier-go"
LicenseData LICENSE

Page directory
Page components
Page instfiles

UninstPage uninstConfirm
UninstPage instfiles

# start default section
Section
    # set the installation directory as the destination for the following actions
    SetOutPath $INSTDIR

    File src\twitchnotifier\twitch-notifier-go.exe

    File LICENSE
    File README.md

    # create the uninstaller
    WriteUninstaller "$INSTDIR\uninstall.exe"

    WriteRegStr HKLM "${ARP}" \
                     "DisplayName" "twitch-notifier-go"
    WriteRegStr HKLM "${ARP}" \
                     "InstallLocation" "$INSTDIR"
    WriteRegStr HKLM "${ARP}" \
                     "Publisher" "rakslice"
    WriteRegStr HKLM "${ARP}" \
                     "URLUpdateInfo" "https://twitch-notifier.blogspot.ca/"
    WriteRegStr HKLM "${ARP}" \
                     "URLInfoAbout" "https://twitch-notifier.blogspot.ca/"
    WriteRegStr HKLM "${ARP}" \
                     "UninstallString" "$\"$INSTDIR\uninstall.exe$\""
    WriteRegStr HKLM "${ARP}" \
                     "DisplayIcon" "$\"$INSTDIR\twitch-notifier-go.exe$\""
    WriteRegStr HKLM "${ARP}" \
                     "Readme" "$INSTDIR\README.md"
    WriteRegDWORD HKLM "${ARP}" \
                     "VersionMajor" "${VERSION_MAJOR}"
    WriteRegDWORD HKLM "${ARP}" \
                     "VersionMinor" "${VERSION_MINOR}"
    WriteRegStr HKLM "${ARP}" \
                     "DisplayVersion" "${DISPLAY_VERSION}"

    ${GetSize} "$INSTDIR" "/S=0K" $0 $1 $2
    IntFmt $0 "0x%08X" $0
    WriteRegDWORD HKLM "${ARP}" "EstimatedSize" "$0"
SectionEnd

Section "Add to Start Menu"
    # create a shortcut in the start menu programs directory
    # point the new shortcut at the program uninstaller
    CreateShortCut "$SMPROGRAMS\twitch-notifier-go.lnk" "$INSTDIR\twitch-notifier-go.exe"
SectionEnd

Section "Run at Startup"
    CreateShortCut "$SMPROGRAMS\Startup\Run twitch-notifier-go on Startup.lnk" "$INSTDIR\twitch-notifier-go.exe" -hide
SectionEnd

Section "uninstall"

    # first, delete the uninstaller [rak: ?]
    Delete "$INSTDIR\uninstall.exe"

    # clean up uninstaller registry key
    DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\twitch-notifier-go"

    # remove the optional main link from the start menu
    Delete "$SMPROGRAMS\twitch-notifier-go.lnk"

    # remove the optional startup link from the start menu
    Delete "$SMPROGRAMS\Startup\Run twitch-notifier-go on Startup.lnk"

    Delete "$INSTDIR\twitch-notifier-go.exe"

    Delete "$INSTDIR\LICENSE"
    Delete "$INSTDIR\README.md"

# uninstaller section end
SectionEnd

Section "Run after install"
   SetOutPath $INSTDIR
   Exec '"$WINDIR\explorer.exe" "$INSTDIR\twitch-notifier-go.exe"'
SectionEnd
