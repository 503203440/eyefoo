//go:build darwin

package main

/*
#cgo CFLAGS: -mmacosx-version-min=10.13 -x objective-c
#cgo LDFLAGS: -framework Cocoa -mmacosx-version-min=10.13

#import <Cocoa/Cocoa.h>

static NSApplicationPresentationOptions eyefooSavedOptions = 0;
static bool eyefooKioskActive = false;

void eyefooEnterKiosk(void) {
    if (eyefooKioskActive) return;
    eyefooSavedOptions = [NSApp presentationOptions];
    NSApplicationPresentationOptions opts =
          NSApplicationPresentationFullScreen
        | NSApplicationPresentationHideDock
        | NSApplicationPresentationHideMenuBar
        | NSApplicationPresentationDisableProcessSwitching
        | NSApplicationPresentationDisableForceQuit
        | NSApplicationPresentationDisableSessionTermination;
    [NSApp setPresentationOptions:opts];
    eyefooKioskActive = true;
}

void eyefooExitKiosk(void) {
    if (!eyefooKioskActive) return;
    [NSApp setPresentationOptions:eyefooSavedOptions];
    eyefooSavedOptions = 0;
    eyefooKioskActive = false;
}
*/
import "C"

func EnterKiosk() {
	C.eyefooEnterKiosk()
}

func ExitKiosk() {
	C.eyefooExitKiosk()
}
