//go:build windows

package main

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modGdi32  = windows.NewLazySystemDLL("gdi32.dll")
	modUser32 = windows.NewLazySystemDLL("user32.dll")

	procGetDC              = modUser32.NewProc("GetDC")
	procReleaseDC          = modUser32.NewProc("ReleaseDC")
	procGetDeviceGammaRamp = modGdi32.NewProc("GetDeviceGammaRamp")
	procSetDeviceGammaRamp = modGdi32.NewProc("SetDeviceGammaRamp")
)

const gammaSamples = 256

type gammaRamp struct {
	Red   [gammaSamples]uint16
	Green [gammaSamples]uint16
	Blue  [gammaSamples]uint16
}

var (
	savedGamma gammaRamp
	gammaSaved bool
)

func ApplyColorTemperature(warmth int) {
	if warmth < 0 {
		warmth = 0
	}
	if warmth > 100 {
		warmth = 100
	}

	hdc, _, _ := procGetDC.Call(0)
	if hdc == 0 {
		return
	}
	defer procReleaseDC.Call(0, hdc)

	if warmth == 0 {
		if gammaSaved {
			procSetDeviceGammaRamp.Call(hdc, uintptr(unsafe.Pointer(&savedGamma)))
			gammaSaved = false
		}
		return
	}

	// Save original gamma once on first non-zero warmth
	if !gammaSaved {
		procGetDeviceGammaRamp.Call(hdc, uintptr(unsafe.Pointer(&savedGamma)))
		gammaSaved = true
	}

	// Always compute from saved original gamma — never from the current
	// (already-modified) display gamma, to avoid cumulative compounding.
	ramp := savedGamma

	warmthFactor := float64(warmth) / 100.0
	for i := 0; i < gammaSamples; i++ {
		ramp.Green[i] = uint16(float64(savedGamma.Green[i]) * (1.0 - warmthFactor*0.3))
		ramp.Blue[i] = uint16(float64(savedGamma.Blue[i]) * (1.0 - warmthFactor*0.5))
	}

	procSetDeviceGammaRamp.Call(hdc, uintptr(unsafe.Pointer(&ramp)))
}

func ResetDisplaySettings() {
	if gammaSaved {
		hdc, _, _ := procGetDC.Call(0)
		if hdc != 0 {
			procSetDeviceGammaRamp.Call(hdc, uintptr(unsafe.Pointer(&savedGamma)))
			procReleaseDC.Call(0, hdc)
		}
		gammaSaved = false
	}
}
