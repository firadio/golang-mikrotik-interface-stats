// +build windows

package main

import (
	"os"
	"syscall"
	"unsafe"
)

var (
	kernel32           = syscall.NewLazyDLL("kernel32.dll")
	procSetConsoleMode = kernel32.NewProc("SetConsoleMode")
	procGetConsoleMode = kernel32.NewProc("GetConsoleMode")
)

const (
	enableVirtualTerminalProcessing = 0x0004
)

// enableANSI enables ANSI escape sequence processing on Windows
func enableANSI() error {
	stdout := syscall.Handle(os.Stdout.Fd())

	var mode uint32
	if _, _, err := procGetConsoleMode.Call(uintptr(stdout), uintptr(unsafe.Pointer(&mode))); err != nil && err != syscall.Errno(0) {
		return err
	}

	mode |= enableVirtualTerminalProcessing
	if _, _, err := procSetConsoleMode.Call(uintptr(stdout), uintptr(mode)); err != nil && err != syscall.Errno(0) {
		return err
	}

	return nil
}
