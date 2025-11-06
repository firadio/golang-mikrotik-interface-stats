// +build !windows

package main

// enableANSI is a no-op on Unix-like systems (Linux, macOS, etc.)
// as they natively support ANSI escape codes
func enableANSI() error {
	return nil
}
