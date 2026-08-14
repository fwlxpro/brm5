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
	"E":     "e",
	"Q":     "q",
}

// playback přehraje eventy podle jejich časových údajů At (ms od startu nahrávky).
func playback(rec Recorder) error {
	rec.Events = normalizeEvents(rec.Events)
	if len(rec.Events) == 0 {
		return fmt.Errorf("prázdný záznam")
	}

	restore := beginHighResTimer()
	defer restore()

	held := make(map[string]bool)
	start := time.Now()

	for _, ev := range rec.Events {
		target := time.Duration(ev.At) * time.Millisecond
		if wait := target - time.Since(start); wait > 0 {
			preciseSleep(wait)
		}

		if err := applyKey(ev, held); err != nil {
			releaseAllKeys(held)
			return err
		}
	}

	releaseAllKeys(held)
	return nil
}

// preciseSleep čeká přesněji než samotný time.Sleep (Windows má běžně
// rozlišení ~15 ms, čímž by se prodloužily všechny držení kláves a vrtulník
// by dostával jiný input, než byl nahrán). Hrubou část přeskočíme Sleepem
// a posledních pár ms dočekáme aktivním čekáním.
func preciseSleep(d time.Duration) {
	if d <= 0 {
		return
	}
	const margin = 2 * time.Millisecond
	end := time.Now().Add(d)
	if d > margin {
		time.Sleep(d - margin)
	}
	for time.Now().Before(end) {
	}
}

// normalizeEvents opraví starší/rozbité nahrávky: zahodí duplicitní DOWN
// (auto-repeat) a osamocené UP a ke každému DOWNu bez UP doplní na konci
// uvolnění, aby playback nikdy nedržel klávesu celou dobu.
func normalizeEvents(events []Event) []Event {
	held := make(map[string]bool)
	lastAt := int64(0)
	out := make([]Event, 0, len(events))
	for _, ev := range events {
		if ev.At > lastAt {
			lastAt = ev.At
		}
		switch ev.Type {
		case "DOWN":
			if held[ev.Key] {
				continue
			}
			held[ev.Key] = true
			out = append(out, ev)
		case "UP":
			if !held[ev.Key] {
				continue
			}
			held[ev.Key] = false
			out = append(out, ev)
		}
	}
	for key := range held {
		out = append(out, Event{Key: key, Type: "UP", At: lastAt})
	}
	return out
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
