//go:build darwin

package main

/*
#cgo CFLAGS: -framework ApplicationServices -framework CoreGraphics
#include <ApplicationServices/ApplicationServices.h>

extern void goActivityCallback(int eventType);

static CGEventMask activityEventMask(void) {
    return (CGEventMaskBit(kCGEventKeyDown)
         | CGEventMaskBit(kCGEventLeftMouseDown)
         | CGEventMaskBit(kCGEventRightMouseDown)
         | CGEventMaskBit(kCGEventOtherMouseDown)
         | CGEventMaskBit(kCGEventMouseMoved));
}

static CGEventRef eyefooTapCallback(CGEventTapProxy proxy, CGEventType type, CGEventRef event, void *refcon) {
    if (type != kCGEventTapDisabledByUserInput
     && type != kCGEventTapDisabledByTimeout) {
        goActivityCallback((int)type);
    }
    return event;
}

static CFMachPortRef g_eyefooTap = NULL;
static CFRunLoopRef g_eyefooRunLoop = NULL;

static int eyefooStartTap(void) {
    if (g_eyefooTap != NULL) return 0;
    g_eyefooTap = CGEventTapCreate(
        kCGSessionEventTap,
        kCGHeadInsertEventTap,
        kCGEventTapOptionListenOnly,
        activityEventMask(),
        eyefooTapCallback,
        NULL
    );
    if (g_eyefooTap == NULL) return -1;
    CFRunLoopSourceRef src = CFMachPortCreateRunLoopSource(NULL, g_eyefooTap, 0);
    if (src == NULL) {
        CFRelease(g_eyefooTap);
        g_eyefooTap = NULL;
        return -2;
    }
    g_eyefooRunLoop = CFRunLoopGetCurrent();
    CFRunLoopAddSource(g_eyefooRunLoop, src, kCFRunLoopCommonModes);
    CGEventTapEnable(g_eyefooTap, true);
    return 0;
}

static void eyefooStopTap(void) {
    if (g_eyefooTap != NULL) {
        CGEventTapEnable(g_eyefooTap, false);
        CFMachPortInvalidate(g_eyefooTap);
        CFRelease(g_eyefooTap);
        g_eyefooTap = NULL;
    }
    if (g_eyefooRunLoop != NULL) {
        CFRunLoopStop(g_eyefooRunLoop);
        g_eyefooRunLoop = NULL;
    }
}
*/
import "C"

import (
	"errors"
	"runtime"
	"sync/atomic"
	"time"
)

var (
	lastActivityUnix  atomic.Int64
	activityTapActive atomic.Bool
)

//export goActivityCallback
func goActivityCallback(_ C.int) {
	lastActivityUnix.Store(time.Now().UnixNano())
}

// StartActivityTap spawns an OS thread that runs a CFRunLoop listening for
// keyboard and mouse events system-wide. It returns an error if the event
// tap could not be created (e.g. accessibility permission not granted).
func StartActivityTap() error {
	ready := make(chan error, 1)
	go func() {
		runtime.LockOSThread()
		rc := C.eyefooStartTap()
		if rc != 0 {
			ready <- errors.New("activity tap: create failed (accessibility permission required)")
			return
		}
		activityTapActive.Store(true)
		ready <- nil
		C.CFRunLoopRun()
		activityTapActive.Store(false)
	}()
	select {
	case err := <-ready:
		return err
	case <-time.After(2 * time.Second):
		return errors.New("activity tap: start timeout")
	}
}

func StopActivityTap() {
	C.eyefooStopTap()
}

func IsActivityTapEnabled() bool {
	return activityTapActive.Load()
}

func HasActivity() bool {
	return lastActivityUnix.Load() != 0
}

func LastActivity() time.Time {
	n := lastActivityUnix.Load()
	if n == 0 {
		return time.Time{}
	}
	return time.Unix(0, n)
}
