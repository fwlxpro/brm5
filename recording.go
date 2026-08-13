package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	hook "github.com/robotn/gohook"
)

var flightKeys = map[string]string{
	"w":     "W",
	"a":     "A",
	"s":     "S",
	"d":     "D",
	"shift": "Shift",
	"space": "Space",
        "e":     "E",
        "q":     "Q",
}

// --- Původní logika (zakomentováno) ---
//
// func recording() {
// 	fmt.Println("Stiskni klávesu...") // test
//
// 	recorder := Recorder{}
// 	start := time.Now()
//
// 	hook.Register(hook.KeyDown, []string{"w"}, func(e hook.Event) {
// 		fmt.Println("W DOWN")
// 		event := Event{
// 			Key:  "W",
// 			Type: "DOWN",
// 			At:   time.Since(start).Milliseconds(),
// 		}
// 		recorder.Events = append(recorder.Events, event)
// 	})
//
// 	hook.Register(hook.KeyUp, []string{"w"}, func(e hook.Event) {
// 		fmt.Println("W UP")
// 		event := Event{
// 			Key:  "W",
// 			Type: "UP",
// 			At:   time.Since(start).Milliseconds(),
// 		}
// 		recorder.Events = append(recorder.Events, event)
// 	})
//
// 	stop := make(chan bool)
//
// 	hook.Register(hook.KeyDown, []string{"o"}, func(e hook.Event) {
// 		fmt.Println("vypínám recording")
// 		stop <- true
// 	})
//
// 	s := hook.Start()
// 	go func() {
// 		<-hook.Process(s)
// 	}()
//
// 	<-stop
// 	SaveRecording(recorder)
// }
//
// func SaveRecording(recorder Recorder) {
// 	data, err := json.Marshal(recorder)
// 	if err != nil {
// 		fmt.Printf("could not marshal json: %s\n", err)
// 		return
// 	}
// 	err = os.WriteFile("data.json", data, 0644)
// 	if err != nil {
// 		fmt.Println("Nemůže data vypsat do souboru data.json")
// 		return
// 	}
// }

// recording zachytává letové klávesy přes gohook a ukládá je do savePath.
// Eventy posílá do eventCh pro live obrazovku v TUI. Stop klávesou O.
func recording(savePath string, eventCh chan<- Event) error {
	recorder := Recorder{}
	start := time.Now()
	var mu sync.Mutex
	pressed := make(map[string]bool)
	stopped := false

	record := func(displayKey, eventType string) {
		mu.Lock()
		defer mu.Unlock()

		if stopped {
			return
		}

		// Filtruj auto-repeat: DOWN ukládej jen při přechodu nahoru->dolu,
		// UP jen při přechodu dolu->nahoru.
		if eventType == "DOWN" {
			if pressed[displayKey] {
				return
			}
			pressed[displayKey] = true
		} else {
			if !pressed[displayKey] {
				return
			}
			delete(pressed, displayKey)
		}

		event := Event{
			Key:  displayKey,
			Type: eventType,
			At:   time.Since(start).Milliseconds(),
		}
		recorder.Events = append(recorder.Events, event)

		if eventCh != nil {
			select {
			case eventCh <- event:
			default:
			}
		}
	}

	registerKey := func(hookKey, displayKey, eventType string) {
		hook.Register(eventTypeToHook(eventType), []string{hookKey}, func(e hook.Event) {
			record(displayKey, eventType)
		})
	}

	for hookKey, displayKey := range flightKeys {
		registerKey(hookKey, displayKey, "DOWN")
		registerKey(hookKey, displayKey, "UP")
	}

	stop := make(chan struct{})
	hook.Register(hook.KeyDown, []string{"o"}, func(e hook.Event) {
		select {
		case <-stop:
		default:
			close(stop)
		}
	})

	s := hook.Start()
	go func() {
		<-hook.Process(s)
	}()

	<-stop
	hook.End()

	// Doplň UP eventy pro klávesy, které byly v momentě zastavení stále stisknuté,
	// aby playback nikdy neskončil se zaseknutou klávesou.
	mu.Lock()
	stopped = true
	at := time.Since(start).Milliseconds()
	for key := range pressed {
		event := Event{Key: key, Type: "UP", At: at}
		recorder.Events = append(recorder.Events, event)
		if eventCh != nil {
			select {
			case eventCh <- event:
			default:
			}
		}
	}
	mu.Unlock()

	dir := filepath.Dir(savePath)
	if err := EnsureDir(dir); err != nil {
		return fmt.Errorf("vytvoření složky: %w", err)
	}
	if err := SaveRecording(savePath, recorder); err != nil {
		return fmt.Errorf("uložení nahrávky: %w", err)
	}
	return nil
}

func eventTypeToHook(eventType string) uint8 {
	if strings.ToUpper(eventType) == "UP" {
		return hook.KeyUp
	}
	return hook.KeyDown
}
