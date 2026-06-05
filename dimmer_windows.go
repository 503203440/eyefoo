//go:build windows

package main

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modDxva2  = windows.NewLazySystemDLL("dxva2.dll")
	modGdi32  = windows.NewLazySystemDLL("gdi32.dll")
	modUser32 = windows.NewLazySystemDLL("user32.dll")

	procGetNumberOfPhysicalMonitorsFromHMONITOR = modDxva2.NewProc("GetNumberOfPhysicalMonitorsFromHMONITOR")
	procGetPhysicalMonitorsFromHMONITOR         = modDxva2.NewProc("GetPhysicalMonitorsFromHMONITOR")
	procGetMonitorBrightness                    = modDxva2.NewProc("GetMonitorBrightness")
	procSetMonitorBrightness                    = modDxva2.NewProc("SetMonitorBrightness")
	procDestroyPhysicalMonitor                  = modDxva2.NewProc("DestroyPhysicalMonitor")
	procMonitorFromWindow                       = modUser32.NewProc("MonitorFromWindow")
	procGetDC                                   = modUser32.NewProc("GetDC")
	procReleaseDC                               = modUser32.NewProc("ReleaseDC")
	procGetDeviceGammaRamp                      = modGdi32.NewProc("GetDeviceGammaRamp")
	procSetDeviceGammaRamp                      = modGdi32.NewProc("SetDeviceGammaRamp")
)

const (
	monitorDefaultToPrimary = 1
	gammaSamples            = 256
)

type physicalMonitor struct {
	hPhysicalMonitor uintptr
	szDescription    [128]uint16
}

type gammaRamp struct {
	Red   [gammaSamples]uint16
	Green [gammaSamples]uint16
	Blue  [gammaSamples]uint16
}

var (
	savedGamma   gammaRamp
	gammaSaved   bool
	gammaRampHDC uintptr
)

func ApplyDisplayBrightness(level int) {
	if level < 0 {
		level = 0
	}
	if level > 100 {
		level = 100
	}

	hMonitor, _, _ := procMonitorFromWindow.Call(0, monitorDefaultToPrimary)
	if hMonitor == 0 {
		return
	}

	var numMonitors uint32
	ret, _, _ := procGetNumberOfPhysicalMonitorsFromHMONITOR.Call(hMonitor, uintptr(unsafe.Pointer(&numMonitors)))
	if ret == 0 || numMonitors == 0 {
		return
	}

	monitors := make([]physicalMonitor, numMonitors)
	ret, _, _ = procGetPhysicalMonitorsFromHMONITOR.Call(hMonitor, uintptr(numMonitors), uintptr(unsafe.Pointer(&monitors[0])))
	if ret == 0 {
		return
	}

	newBrightness := uint32(level)
	for _, m := range monitors {
		procSetMonitorBrightness.Call(uintptr(m.hPhysicalMonitor), uintptr(newBrightness))
		procDestroyPhysicalMonitor.Call(uintptr(m.hPhysicalMonitor))
	}
}

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
			gammaRampHDC = 0
		}
		return
	}

	if !gammaSaved {
		procGetDeviceGammaRamp.Call(hdc, uintptr(unsafe.Pointer(&savedGamma)))
		gammaSaved = true
		gammaRampHDC = hdc

	}

	var ramp gammaRamp
	procGetDeviceGammaRamp.Call(hdc, uintptr(unsafe.Pointer(&ramp)))

	warmthFactor := float64(warmth) / 100.0
	for i := 0; i < gammaSamples; i++ {
		ramp.Red[i] = uint16(float64(ramp.Red[i]))
		ramp.Green[i] = uint16(float64(ramp.Green[i]) * (1.0 - warmthFactor*0.3))
		ramp.Blue[i] = uint16(float64(ramp.Blue[i]) * (1.0 - warmthFactor*0.5))
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

	ApplyDisplayBrightness(100)
}
