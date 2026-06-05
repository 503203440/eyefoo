//go:build darwin

package main

/*
#cgo CFLAGS: -mmacosx-version-min=10.13 -x objective-c
#cgo LDFLAGS: -framework CoreGraphics -mmacosx-version-min=10.13

#import <CoreGraphics/CoreGraphics.h>

#define GAMMA_DISPLAYS 4
#define GAMMA_SAMPLES 256

static bool g_gamma_saved[GAMMA_DISPLAYS] = {false};
static CGGammaValue g_orig_red[GAMMA_DISPLAYS][GAMMA_SAMPLES];
static CGGammaValue g_orig_green[GAMMA_DISPLAYS][GAMMA_SAMPLES];
static CGGammaValue g_orig_blue[GAMMA_DISPLAYS][GAMMA_SAMPLES];
static CGDirectDisplayID g_saved_displays[GAMMA_DISPLAYS];
static int g_saved_count = 0;

void eyefooSetWarmth(int warmth) {
    uint32_t displayCount;
    CGDirectDisplayID displays[32];
    CGGetOnlineDisplayList(32, displays, &displayCount);

    for (uint32_t d = 0; d < displayCount; d++) {
        CGDirectDisplayID display = displays[d];

        if (warmth <= 0) {
            for (int i = 0; i < g_saved_count; i++) {
                if (g_saved_displays[i] == display && g_gamma_saved[i]) {
                    CGSetDisplayTransferByTable(display, GAMMA_SAMPLES, g_orig_red[i], g_orig_green[i], g_orig_blue[i]);
                    g_gamma_saved[i] = false;
                    break;
                }
            }
            continue;
        }

        bool found = false;
        for (int i = 0; i < g_saved_count; i++) {
            if (g_saved_displays[i] == display && g_gamma_saved[i]) {
                found = true;
                break;
            }
        }
        if (!found && g_saved_count < GAMMA_DISPLAYS) {
            uint32_t sampleCount = GAMMA_SAMPLES;
            CGGetDisplayTransferByTable(display, GAMMA_SAMPLES, g_orig_red[g_saved_count], g_orig_green[g_saved_count], g_orig_blue[g_saved_count], &sampleCount);
            g_saved_displays[g_saved_count] = display;
            g_gamma_saved[g_saved_count] = true;
            g_saved_count++;
        }

        float warmthFactor = (float)warmth / 100.0f;
        CGGammaValue red[GAMMA_SAMPLES], green[GAMMA_SAMPLES], blue[GAMMA_SAMPLES];
        for (int i = 0; i < GAMMA_SAMPLES; i++) {
            float v = (float)i / 255.0f;
            red[i] = v;
            green[i] = v * (1.0f - warmthFactor * 0.3f);
            blue[i] = v * (1.0f - warmthFactor * 0.5f);
        }
        CGSetDisplayTransferByTable(display, GAMMA_SAMPLES, red, green, blue);
    }
}

void eyefooResetDisplay(void) {
    for (int i = 0; i < g_saved_count; i++) {
        if (g_gamma_saved[i]) {
            CGSetDisplayTransferByTable(g_saved_displays[i], GAMMA_SAMPLES, g_orig_red[i], g_orig_green[i], g_orig_blue[i]);
            g_gamma_saved[i] = false;
        }
    }
    g_saved_count = 0;
}
*/
import "C"

func ApplyColorTemperature(warmth int) {
	if warmth < 0 {
		warmth = 0
	}
	if warmth > 100 {
		warmth = 100
	}
	C.eyefooSetWarmth(C.int(warmth))
}

func ResetDisplaySettings() {
	C.eyefooResetDisplay()
}
