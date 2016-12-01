#include <stdint.h>
#include <stddef.h>

#include "wx/wx.h"
#include "wx/taskbar.h"
#include "wx/msw/taskbar.h"
#include "wx/icon.h"
#include "wx/msw/icon.h"

#ifdef __cplusplus
extern "C" {
#endif

typedef ptrdiff_t intgo;
typedef struct { char *p; intgo n; } _gostring_;

bool _wrap_TaskBarIcon_ShowBalloon_wx(wxTaskBarIcon * taskbar_icon, _gostring_ title, _gostring_ text, unsigned msec, int flags,
        wxIcon * icon) {

    wxTaskBarIcon * processed_taskbar_icon = *(wxTaskBarIcon **)&taskbar_icon;

    wxString utf8_title(title.p, wxConvUTF8, title.n);
    wxString utf8_text(text.p, wxConvUTF8, text.n);

    wxIcon * processed_icon = *(wxIcon **)&icon;

    /*
    We are calling:

    bool
    wxTaskBarIcon::ShowBalloon(const wxString& title,
                               const wxString& text,
                               unsigned msec,
                               int flags,
                               const wxIcon& icon)
    */

    return processed_taskbar_icon->ShowBalloon((wxString const &) utf8_title, (wxString const &) utf8_text,
                                                msec, flags, (wxIcon const &)*processed_icon);

}

#ifdef __cplusplus
}
#endif
