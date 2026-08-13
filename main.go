package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var ascii = `
    ____  ____  __  _________   ____  ______    ____  ______
   / __ )/ __ \/  |/  / ____/  / __ \/  _/ /   / __ \/_  __/
  / __  / /_/ / /|_/ /___ \   / /_/ // // /   / / / / / /   
 / /_/ / _, _/ /  / /___/ /  / ____// // /___/ /_/ / / /    
/_____/_/ |_/_/  /_/_____/  /_/   /___/_____/\____/ /_/     
                                                            
`

// blackhawk rescue mission 5 pilot

var version = "1.0.0"

var titleStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.Color("205"))

// --- Původní logika (zakomentováno) ---
//
// func main() {
// 	fmt.Println(titleStyle.Render(ascii))
// 	fmt.Println()
//
// 	_ = tea.NewProgram
//
// 	reader := bufio.NewReader(os.Stdin)
// 	input, _ := reader.ReadString('\n')
//
// 	switch input[0] {
// 	case '1':
// 		recording()
// 	case '2':
// 		fmt.Println("Playback")
// 	case '3':
// 		fmt.Println("goodbye!")
// 		os.Exit(3)
// 	}
// }

func main() {
	p := tea.NewProgram(newModel())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Chyba: %v\n", err)
		os.Exit(1)
	}
}
