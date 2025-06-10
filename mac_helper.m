#import <Cocoa/Cocoa.h>

void SetNSWindowTitleBarColor(void *nswindowPtr) {
    NSWindow *window = (__bridge NSWindow *)(nswindowPtr);

    [window setTitleVisibility:NSWindowTitleHidden];
    [window setTitlebarAppearsTransparent:YES];
    [window setBackgroundColor:[NSColor colorWithRed:247.0/255.0
                                        green:247.0/255.0
                                        blue:247.0/255.0
                                        alpha:1.0]];
}
