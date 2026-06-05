//go:build !darwin

package main

func platformNotify(title, message string) {
	println(title + ": " + message)
}
