
package wxshowballoon

import (
	"github.com/dontpanic92/wxGo/wx"
	"unsafe"
)


/*
wxWidgets TaskbarIcon implementation on Windows uses the native "notification area" (taskbar icon) support, which also supports notification messages.
wxGo doesn't expose a ShowBalloon call, but exposes the events for interacting with the balloon.

So I've created this additional wrapper for wxTaskbarIcon::ShowBalloon
*/


/*
#cgo CPPFLAGS: -I${SRCDIR}/../../../twitch-notifier-gopath/src/github.com/dontpanic92/wxGo/wx/windows_amd64/ -I${SRCDIR}/../../../twitch-notifier-gopath/src/github.com/dontpanic92/wxGo/wxWidgets/wxWidgets-3.1.0/include -D_FILE_OFFSET_BITS=64 -D__WXMSW__
#cgo LDFLAGS: -L${SRCDIR}/../../../twitch-notifier-gopath/src/github.com/dontpanic92/wxGo/wx/windows_amd64/lib -Wl,--subsystem,windows -mwindows -lwxmsw31u -lwxmsw31u_gl -lwxscintilla -lopengl32 -lglu32 -lwxregexu -lwxexpat -lwxtiff -lwxjpeg -lwxpng -lwxzlib -lrpcrt4 -loleaut32 -lole32 -luuid -lwinspool -lwinmm -lshell32 -lshlwapi -lcomctl32 -lcomdlg32 -ladvapi32 -lversion -lwsock32 -lgdi32 -lntdll -lmsvcrt
#cgo CXXFLAGS: -fpermissive

#include <stdint.h>
#include <stddef.h>

typedef ptrdiff_t intgo;

typedef struct { char *p; intgo n; } _gostring_;

extern _Bool _wrap_TaskBarIcon_ShowBalloon_wx(uintptr_t taskbar_icon, _gostring_ title, _gostring_ text, unsigned msec, int flags,
        uintptr_t icon);
*/
import "C"



func ShowBalloon(p wx.TaskBarIcon, title string, text string, msec uint, flags int, icon wx.Icon) bool {

	// How to deal with different
	// wx.TaskBarIcon
	// argument SwigcptrTaskBarIcon
	// param wxTaskBarIcon *_swig_go_0
	// usage
	//   goes through a arg1 = *(wxTaskBarIcon **)&_swig_go_0;

	// wx.Icon
	// argument C.uintptr_t(icon.Swigcptr())
	// param wxIcon * _swig_go_1
	// usage
	//   goes through a *(wxIcon **)&_swig_go_1

	// see SetTooltip for string
	// argument *(*C.swig_type_32)(unsafe.Pointer(&_swig_i_1))
	// param _gostring_ _swig_go_2
	// usage
	//   wxString arg3_str(_swig_go_2.p, wxConvUTF8, _swig_go_2.n);

	return (bool)(C._wrap_TaskBarIcon_ShowBalloon_wx(C.uintptr_t(p.Swigcptr()),
		*(*C._gostring_)(unsafe.Pointer(&title)),
		*(*C._gostring_)(unsafe.Pointer(&text)),
		C.unsigned(msec),
		C.int(flags),
		C.uintptr_t(icon.Swigcptr())))
}
