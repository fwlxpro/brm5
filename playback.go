package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-vgo/robotgo"
)

// robotKeyMap převádí uložené názvy kláves na formát robotgo.
var robotKeyMap = map[string]string{
	"W":     "w",
	"A":     "a",
	"S":     "s",
	"D":     "d",
	"Shift": "shift",
	"Space": "space",
	"Ctrl":  "ctrl",
}

// playback přehraje eventy podle jejich časových údajů At (ms od startu nahrávky).
func playback(rec Recorder) error {
	if len(rec.Events) == 0 {
		return fmt.Errorf("prázdný záznam")
	}

	held := make(map[string]bool)
	start := time.Now()

	for _, ev := range rec.Events {
		target := time.Duration(ev.At) * time.Millisecond
		if wait := target - time.Since(start); wait > 0 {
			time.Sleep(wait)
		}

		if err := applyKey(ev, held); err != nil {
			releaseAllKeys(held)
			return err
		}
	}

	releaseAllKeys(held)
	return nil
}

func applyKey(ev Event, held map[string]bool) error {
	key, ok := robotKeyMap[ev.Key]
	if !ok {
		key = strings.ToLower(ev.Key)
	}

	switch ev.Type {
	case "DOWN":
		if err := robotgo.KeyToggle(key, "down"); err != nil {
			return fmt.Errorf("KeyDown %s: %w", ev.Key, err)
		}
		held[key] = true
	case "UP":
		if err := robotgo.KeyToggle(key, "up"); err != nil {
			return fmt.Errorf("KeyUp %s: %w", ev.Key, err)
		}
		delete(held, key)
	default:
		return fmt.Errorf("neznámý typ eventu: %s", ev.Type)
	}
	return nil
}

func releaseAllKeys(held map[string]bool) {
	for key := range held {
		_ = robotgo.KeyToggle(key, "up")
		delete(held, key)
	}
}

// PlaybackFile načte JSON a spustí playback.
func PlaybackFile(path string) error {
	rec, err := LoadRecording(path)
	if err != nil {
		return err
	}
	return playback(rec)
}
