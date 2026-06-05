//go:build darwin

package main

/*
#cgo CFLAGS: -mmacosx-version-min=10.14 -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Foundation -framework UserNotifications -mmacosx-version-min=10.14

#import <Foundation/Foundation.h>
#import <UserNotifications/UserNotifications.h>

static void eyefooDeliverNotification(UNUserNotificationCenter* center, NSString* title, NSString* body);

void eyefooNotify(const char* title, const char* message) {
    NSString* titleStr = [NSString stringWithUTF8String:title];
    NSString* messageStr = [NSString stringWithUTF8String:message];

    UNUserNotificationCenter* center = [UNUserNotificationCenter currentNotificationCenter];

    [center getNotificationSettingsWithCompletionHandler:^(UNNotificationSettings * _Nonnull settings) {
        if (settings.authorizationStatus == UNAuthorizationStatusNotDetermined) {
            UNAuthorizationOptions options = UNAuthorizationOptionAlert | UNAuthorizationOptionSound;
            [center requestAuthorizationWithOptions:options
                                  completionHandler:^(BOOL granted, NSError * _Nullable error) {
                if (granted) {
                    eyefooDeliverNotification(center, titleStr, messageStr);
                }
            }];
        } else if (settings.authorizationStatus == UNAuthorizationStatusAuthorized) {
            eyefooDeliverNotification(center, titleStr, messageStr);
        }
    }];
}

static void eyefooDeliverNotification(UNUserNotificationCenter* center, NSString* title, NSString* body) {
    UNMutableNotificationContent* content = [[UNMutableNotificationContent alloc] init];
    content.title = title;
    content.body = body;
    content.sound = [UNNotificationSound defaultSound];

    NSString* identifier = [[NSUUID UUID] UUIDString];
    UNNotificationRequest* request = [UNNotificationRequest
        requestWithIdentifier:identifier
                      content:content
                      trigger:nil];

    [center addNotificationRequest:request withCompletionHandler:^(NSError * _Nullable error) {
        if (error) {
            NSLog(@"eyefoo notification error: %@", error);
        }
    }];
}
*/
import "C"

import "unsafe"

func platformNotify(title, message string) {
	cTitle := C.CString(title)
	defer C.free(unsafe.Pointer(cTitle))
	cMessage := C.CString(message)
	defer C.free(unsafe.Pointer(cMessage))
	C.eyefooNotify(cTitle, cMessage)
}
