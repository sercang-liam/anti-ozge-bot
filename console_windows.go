//go:build windows

package main

import "syscall"

// prepareConsole switches the Windows console to UTF-8 so Turkish letters,
// emoji and the QR code display correctly.
func prepareConsole() {
	k := syscall.NewLazyDLL("kernel32.dll")
	k.NewProc("SetConsoleOutputCP").Call(65001)
	k.NewProc("SetConsoleCP").Call(65001)
}
