//go:build !darwin && !windows

package main

func ApplyDisplayBrightness(level int) {}

func ApplyColorTemperature(warmth int) {}

func ResetDisplaySettings() {}
