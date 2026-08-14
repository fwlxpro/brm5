package main

import (
	"fmt"
	"path/filepath"
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

	// Čteme surový kanál z hook.Start() přímo — gohookův Process/Register
	// má bug: při prvním uvolnění klávesy vynuluje celý stav a UPs dalších
	// kláves pak nevyvolá (případně je přiřadí špatné klávese), čímž by
	// nahrávka obsahovala klávesu "stisknutou" až do konce a playback by
	// vrtulník převracel do země.
	keyByCode := make(map[uint16]string)
	for hookName, displayName := range flightKeys {
		if code, ok := hook.Keycode[hookName]; ok {
			keyByCode[code] = displayName
		}
	}
	stopCode := hook.Keycode["o"]

	record := func(when time.Time, displayKey, eventType string) {
		mu.Lock()
		defer mu.Unlock()

		if stopped {
			return
		}

		// Filtruj auto-repeat: DOWN ukládej jen při přechodu nahoru->dolu.
		if eventType == "DOWN" {
			if pressed[displayKey] {
				return
			}
			pressed[displayKey] = true
		} else {
			// UP nikdy nezahazuj — kdyby se ztratil odpovídající DOWN
			// (např. klávesa držená už před startem nahrávky), klávesa by
			// v záznamu zůstala "stisknutá" a playback by ji držel celou dobu,
			// což vrtulník neovladatelně převrací.
			// delete udrží v mapě jen klávesy skutečně držené (viz doplnění
			// UP na konci) — klávesa s false by dostala falešný UP.
			delete(pressed, displayKey)
		}

		// At vychází z času, kdy OS událost odbavil (e.When), ne z času,
		// kdy náš callback stihl proběhnout — to je přesnější.
		at := when.Sub(start).Milliseconds()
		if at < 0 {
			at = 0
		}
		event := Event{
			Key:  displayKey,
			Type: eventType,
			At:   at,
		}
		recorder.Events = append(recorder.Events, event)

		if eventCh != nil {
			select {
			case eventCh <- event:
			default:
			}
		}
	}

	stop := make(chan struct{})
	// Start(tm) určuje periodu pollingu C bufferu. Výchozí 50 ms kvantuje
	// časové značky na ~50 ms kroky a sub-50ms ťuky úplně slévá — s 2 ms
	// zůstává nahrávka věrná v řádu milisekund.
	s := hook.Start(2)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for e := range s {
			mu.Lock()
			stoppedNow := stopped
			mu.Unlock()
			if stoppedNow {
				return
			}

			switch e.Kind {
			case hook.KeyDown, hook.KeyHold:
				if name, ok := keyByCode[e.Keycode]; ok {
					record(e.When, name, "DOWN")
				} else if e.Keycode == stopCode {
					select {
					case <-stop:
					default:
						close(stop)
					}
				}
			case hook.KeyUp:
				if name, ok := keyByCode[e.Keycode]; ok {
					record(e.When, name, "UP")
				}
			}
		}
	}()

	<-stop
	mu.Lock()
	stopped = true
	mu.Unlock()
	hook.End()
	<-done

	// Doplň UP eventy pro klávesy, které byly v momentě zastavení stále stisknuté,
	// aby playback nikdy neskončil se zaseknutou klávesou.
	mu.Lock()
	heldKeys := make([]string, 0, len(pressed))
	for key := range pressed {
		heldKeys = append(heldKeys, key)
	}
	mu.Unlock()
	at := time.Since(start).Milliseconds()
	for _, key := range heldKeys {
		event := Event{Key: key, Type: "UP", At: at}
		recorder.Events = append(recorder.Events, event)
		if eventCh != nil {
			select {
			case eventCh <- event:
			default:
			}
		}
	}

	dir := filepath.Dir(savePath)
	if err := EnsureDir(dir); err != nil {
		return fmt.Errorf("vytvoření složky: %w", err)
	}
	if err := SaveRecording(savePath, recorder); err != nil {
		return fmt.Errorf("uložení nahrávky: %w", err)
	}
	return nil
}
