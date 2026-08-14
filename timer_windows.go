//go:build windows

package main

import "golang.org/x/sys/windows"

// beginHighResTimer zvedne přesnost systémového časovače na 1 ms, aby
// time.Sleep odpovídal časům v nahrávce. Vrací funkci, která obnoví
// původní rozlišení.
func beginHighResTimer() func() {
	if err := windows.TimeBeginPeriod(1); err != nil {
		return func() {}
	}
	return func() { _ = windows.TimeEndPeriod(1) }
}
