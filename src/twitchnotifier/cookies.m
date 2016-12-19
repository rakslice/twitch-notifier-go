
#import <Foundation/Foundation.h>

void clearOSXWebkitTwitchCookies() {
  NSHTTPCookieStorage *cookieJar = [NSHTTPCookieStorage sharedHTTPCookieStorage];

  NSURL *url = [NSURL URLWithString:@"https://api.twitch.tv"];

  for (NSHTTPCookie *cookie in [cookieJar cookiesForURL:url])
  {
    [cookieJar deleteCookie:cookie];
  }
}
