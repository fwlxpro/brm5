//go:build !windows

package main

// beginHighResTimer je no-op na platformách, kde je rozlišení časovače
// už z výroby přesné (Linux/macOS).
func beginHighResTimer() func() {
	return func() {}
}
