package main

import (
	"fmt"
	"image/color"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type screen int

const (
	screenMain screen = iota
	screenHeli
	screenLocation
	screenHeliport
	screenRecordingPick
	screenRecordingLive
	screenCountdown
	screenManage
	screenInput
	screenConfirm
	screenRecBrowser
	screenRecDetail
)

type flowMode int

const (
	flowNone flowMode = iota
	flowRecording
	flowPlayback
)

type manageMode int

const (
	manageNone manageMode = iota
	manageAddHeli
	manageAddLoc
	manageAddHP
	manageDelHeli
	manageDelLoc
	manageDelHP
	manageDelRecording
)

type recEntry struct {
	Path     string
	HeliName string
	LocName  string
	HpName   string
}

const recDetailVisible = 15

var (
	breadcrumbStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00c8ff"))
	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8b94a3"))
	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00ff9d")).
			Bold(true)
	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ff2d55")).
			Bold(true)
	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6b7280"))
	keyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00ff9d")).
			Bold(true)
	githubStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00e5ff"))
	versionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8b94a3"))
	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00ff9d"))
	neonPink = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ff6ec7"))
)

type neonRGB struct{ r, g, b int }

var neonStops = []neonRGB{
	{255, 45, 149}, // pink
	{196, 85, 255}, // purple
}

const (
	headerWidth = 60
	boxWidth    = headerWidth + 4
)

const heliArt = `
                                        ,dMb,
                                      ,dMMMMb,          ,,
                                   ,dMMMMMMMMMb, eeee8888"
                                ,mMMm!!!!XXXXMMMMM"""
                              ,d!!XXMMXX88888888W"
                             'MX88dMM8888WWWMMMMMMb,
                                '"'"+'MMMMMMMMMMMMMMMMb
                                  MMMMMMMMMMMMMMMMMMb,
                                 dMMMMMMMMMMMMMMMMMMMMb,,
                     _,dMMMMMMMMMMXXXX!!!!!!!!!!!!!!XXXXXMP
                _,dMMXX!!!!!!!!!!!!!!!!!!XXXXX888888888WWC
            _,dMMX!!!MMMM!!!!!!!!XXXXXX888888888888WWMMMMMb,
           dMMX!!!!!MMM!XXXXXX88888888888888888WWMMMMMMMMMMMb
          dMMXXXXXX8MMMM88888888888888888WWWMMMMMMMMMMMMMMMMMb    ,d8
          MMMMWW888888MMMMM8888888WWMMMMMMMMMMMMMMMMMMMMMMMMMMM,d88P'
           YMMMMMWW888888WWMMMMMMMMMMP"""'    '"YMMMMMMMMMMMXMMMMMP
              '"YMMMMMMMMMMMMMP""'            mMMMm!XXXXX8888888e,
                                             ,d!!XXMM888888888888WW
                                            "MX88dMM888888WWWMMMMMMb
                                                 """''"'YMMMMMMMYMMM
                                                                    '"YMMMMM
                                                                       '"YMP`

func neonArt(art string) string {
	lines := strings.Split(strings.Trim(art, "\n"), "\n")

	minLead := -1
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		lead := len(line) - len(strings.TrimLeft(line, " "))
		if minLead == -1 || lead < minLead {
			minLead = lead
		}
	}
	if minLead < 0 {
		minLead = 0
	}

	var b strings.Builder
	for _, line := range lines {
		trimmed := strings.TrimRight(line, " ")
		if len(trimmed) >= minLead {
			trimmed = trimmed[minLead:]
		}
		b.WriteString(neonPink.Render(trimmed))
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}
func neonColor(c neonRGB) color.Color {
	return lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", c.r, c.g, c.b))
}

func lerpRGB(a, b neonRGB, t float64) neonRGB {
	return neonRGB{
		int(float64(a.r) + (float64(b.r)-float64(a.r))*t),
		int(float64(a.g) + (float64(b.g)-float64(a.g))*t),
		int(float64(a.b) + (float64(b.b)-float64(a.b))*t),
	}
}

// neonColorAt vrací barvu duhového spektra v pozici t (0..1).
func neonColorAt(t float64) color.Color {
	if t <= 0 {
		return neonColor(neonStops[0])
	}
	if t >= 1 {
		return neonColor(neonStops[len(neonStops)-1])
	}
	seg := t * float64(len(neonStops)-1)
	i := int(seg)
	if i >= len(neonStops)-1 {
		i = len(neonStops) - 2
	}
	return neonColor(lerpRGB(neonStops[i], neonStops[i+1], seg-float64(i)))
}

func gradStyle(t float64) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(neonColorAt(t))
}

// gradientText obarví znaky textu plynulým gradientem přes celé spektrum.
func gradientText(s string, bold bool) string {
	runes := []rune(s)
	n := len(runes)
	var b strings.Builder
	for i, r := range runes {
		t := 0.0
		if n > 1 {
			t = float64(i) / float64(n-1)
		}
		st := gradStyle(t)
		if bold {
			st = st.Bold(true)
		}
		b.WriteString(st.Render(string(r)))
	}
	return b.String()
}

func gradientDivider(width int) string {
	var b strings.Builder
	for i := 0; i < width; i++ {
		t := 0.0
		if width > 1 {
			t = float64(i) / float64(width-1)
		}
		b.WriteString(gradStyle(t).Render("─"))
	}
	return b.String()
}

type recordingEventMsg struct {
	event Event
}

type recordingDoneMsg struct {
	err error
}

type playbackDoneMsg struct {
	err error
}

type countdownMsg struct {
	remaining int
}

type recordingTickMsg struct{}

type model struct {
	lib    Library
	screen screen
	flow   flowMode
	cursor int
	status string
	errMsg string

	selHeli     int
	selLocation int
	selHeliport int

	heliID       string
	heliName     string
	locID        string
	locName      string
	heliportID   string
	heliportName string

	savePath         string
	recordingStarted bool
	eventCh          chan Event
	doneCh           chan error
	liveEvents       []Event
	recStart         time.Time

	recordings   []string
	selRecording int
	playbackPath string
	countdown    int

	manageMode manageMode
	inputLabel string
	inputValue string

	recBrowser      []recEntry
	recDetailEvents []Event
	recDetailOffset int
	recDetailPath   string
}

func newModel() model {
	lib, err := LoadLibrary()
	m := model{
		lib:    lib,
		screen: screenMain,
		cursor: 0,
	}
	if err != nil {
		m.errMsg = err.Error()
	}
	return m
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if m.screen == screenRecordingLive || m.screen == screenCountdown {
			break
		}

		if m.screen == screenRecDetail {
			return m.handleRecDetailKey(msg)
		}

		switch m.screen {
		case screenInput:
			return m.handleInputKey(msg)
		case screenConfirm:
			return m.handleConfirmKey(msg)
		}

		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < m.listLen()-1 {
				m.cursor++
			}
		case "enter":
			return m.handleEnter()
		case "esc", "q":
			return m.goBack()
		}

	case recordingEventMsg:
		m.liveEvents = append(m.liveEvents, msg.event)
		if len(m.liveEvents) > 10 {
			m.liveEvents = m.liveEvents[len(m.liveEvents)-10:]
		}
		return m, waitForRecordingEvent(m.eventCh)

	case recordingDoneMsg:
		if !m.recordingStarted {
			return m, nil
		}
		m.recordingStarted = false
		if msg.err != nil {
			m.errMsg = msg.err.Error()
			m.status = ""
		} else {
			m.status = fmt.Sprintf("Nahrávka uložena: %s", recordingDisplayName(m.savePath))
			m.errMsg = ""
		}
		m.screen = screenMain
		m.cursor = 0
		m.flow = flowNone
		return m, nil

	case countdownMsg:
		m.countdown = msg.remaining
		if msg.remaining <= 0 {
			return m, runPlaybackCmd(m.playbackPath)
		}
		return m, tickCountdown(msg.remaining)

	case playbackDoneMsg:
		if msg.err != nil {
			m.errMsg = msg.err.Error()
			m.status = ""
		} else {
			m.status = "Playback hotov"
			m.errMsg = ""
		}
		m.screen = screenMain
		m.cursor = 0
		m.flow = flowNone
		return m, nil

	case recordingTickMsg:
		if m.screen == screenRecordingLive && m.recordingStarted {
			return m, tickRecording()
		}
	}

	return m, nil
}

func (m model) handleInputKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "enter":
		return m.submitInput()
	case "esc":
		m.manageMode = manageNone
		m.screen = screenManage
		m.cursor = 0
		return m, nil
	case "backspace":
		if len(m.inputValue) > 0 {
			m.inputValue = m.inputValue[:len(m.inputValue)-1]
		}
	case "space":
		m.inputValue += " "
	default:
		if len(msg.String()) == 1 {
			m.inputValue += msg.String()
		}
	}
	return m, nil
}

func (m model) handleConfirmKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "enter":
		return m.doDelete()
	case "esc", "q":
		if m.manageMode == manageDelRecording {
			m.manageMode = manageNone
			m.screen = screenRecDetail
			m.cursor = m.selRecording
			return m, nil
		}
		m.manageMode = manageNone
		m.screen = screenManage
		m.cursor = 0
		return m, nil
	}
	return m, nil
}

func (m model) handleRecDetailKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		if m.recDetailOffset > 0 {
			m.recDetailOffset--
		}
	case "down", "j":
		maxOffset := len(m.recDetailEvents) - recDetailVisible
		if maxOffset < 0 {
			maxOffset = 0
		}
		if m.recDetailOffset < maxOffset {
			m.recDetailOffset++
		}
	case "d", "delete":
		m.manageMode = manageDelRecording
		m.screen = screenConfirm
	case "esc", "q":
		return m.goBack()
	}
	return m, nil
}

func (m model) submitInput() (tea.Model, tea.Cmd) {
	name := strings.TrimSpace(m.inputValue)
	if name == "" {
		m.errMsg = "Název nesmí být prázdný"
		return m, nil
	}

	switch m.manageMode {
	case manageAddHeli:
		m.lib.Helicopters = append(m.lib.Helicopters, Helicopter{
			ID:   slugify(name),
			Name: name,
		})
	case manageAddLoc:
		heli := &m.lib.Helicopters[m.selHeli]
		heli.Locations = append(heli.Locations, Location{
			ID:   slugify(name),
			Name: name,
		})
	case manageAddHP:
		heli := &m.lib.Helicopters[m.selHeli]
		loc := &heli.Locations[m.selLocation]
		loc.Heliports = append(loc.Heliports, Heliport{
			ID:   slugify(name),
			Name: name,
		})
	default:
		return m, nil
	}

	if err := SaveLibrary(m.lib); err != nil {
		m.errMsg = err.Error()
		return m, nil
	}

	m.status = "Knihovna uložena"
	m.errMsg = ""
	m.manageMode = manageNone
	m.screen = screenManage
	m.cursor = 0
	m.inputValue = ""
	return m, nil
}

func (m model) doDelete() (tea.Model, tea.Cmd) {
	switch m.manageMode {
	case manageDelHeli:
		m.lib.Helicopters = append(m.lib.Helicopters[:m.selHeli], m.lib.Helicopters[m.selHeli+1:]...)
	case manageDelLoc:
		heli := &m.lib.Helicopters[m.selHeli]
		heli.Locations = append(heli.Locations[:m.selLocation], heli.Locations[m.selLocation+1:]...)
	case manageDelHP:
		heli := &m.lib.Helicopters[m.selHeli]
		loc := &heli.Locations[m.selLocation]
		loc.Heliports = append(loc.Heliports[:m.selHeliport], loc.Heliports[m.selHeliport+1:]...)
	case manageDelRecording:
		if err := os.Remove(m.recDetailPath); err != nil {
			m.errMsg = err.Error()
			m.manageMode = manageNone
			m.screen = screenRecDetail
			return m, nil
		}
		m.status = "Nahrávka smazána"
		m.errMsg = ""
		m.manageMode = manageNone
		m.recBrowser = m.buildRecBrowser()
		if m.selRecording >= len(m.recBrowser) {
			m.selRecording = len(m.recBrowser) - 1
		}
		if m.selRecording < 0 {
			m.selRecording = 0
		}
		m.screen = screenRecBrowser
		m.cursor = m.selRecording
		return m, nil
	}

	if err := SaveLibrary(m.lib); err != nil {
		m.errMsg = err.Error()
		return m, nil
	}

	m.status = "Smazáno"
	m.errMsg = ""
	m.manageMode = manageNone
	m.screen = screenManage
	m.cursor = 0
	return m, nil
}

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ' || r == '-' || r == '_':
			b.WriteByte('_')
		}
	}
	return strings.Trim(b.String(), "_")
}

func (m model) handleEnter() (tea.Model, tea.Cmd) {
	switch m.screen {
	case screenMain:
		switch m.cursor {
		case 0:
			m.flow = flowRecording
			m.screen = screenHeli
			m.cursor = 0
		case 1:
			m.flow = flowPlayback
			m.screen = screenHeli
			m.cursor = 0
		case 2:
			m.recBrowser = m.buildRecBrowser()
			m.heliName = ""
			m.locName = ""
			m.heliportName = ""
			m.errMsg = ""
			m.screen = screenRecBrowser
			m.cursor = 0
		case 3:
			m.screen = screenManage
			m.cursor = 0
		case 4:
			return m, tea.Quit
		}

	case screenManage:
		switch m.cursor {
		case 0:
			m.manageMode = manageAddHeli
			return m.startInput("Název nové helikoptéry:")
		case 1:
			m.manageMode = manageAddLoc
			m.screen = screenHeli
			m.cursor = 0
		case 2:
			m.manageMode = manageAddHP
			m.screen = screenHeli
			m.cursor = 0
		case 3:
			m.manageMode = manageDelHeli
			m.screen = screenHeli
			m.cursor = 0
		case 4:
			m.manageMode = manageDelLoc
			m.screen = screenHeli
			m.cursor = 0
		case 5:
			m.manageMode = manageDelHP
			m.screen = screenHeli
			m.cursor = 0
		}

	case screenHeli:
		if len(m.lib.Helicopters) == 0 {
			return m, nil
		}
		heli := m.lib.Helicopters[m.cursor]
		m.selHeli = m.cursor
		m.heliID = heli.ID
		m.heliName = heli.Name

		switch m.manageMode {
		case manageAddLoc:
			return m.startInput("Název nové lokace:")
		case manageAddHP, manageDelLoc, manageDelHP:
			m.screen = screenLocation
			m.cursor = 0
			return m, nil
		case manageDelHeli:
			m.screen = screenConfirm
			return m, nil
		default:
			m.screen = screenLocation
			m.cursor = 0
		}

	case screenLocation:
		heli := m.lib.Helicopters[m.selHeli]
		switch m.manageMode {
		case manageAddLoc:
			return m.startInput("Název nové lokace:")
		case manageAddHP:
			return m.startInput("Název nového heliportu:")
		case manageDelLoc, manageDelHP:
			if len(heli.Locations) == 0 {
				return m, nil
			}
			loc := heli.Locations[m.cursor]
			m.selLocation = m.cursor
			m.locID = loc.ID
			m.locName = loc.Name
			if m.manageMode == manageDelLoc {
				m.screen = screenConfirm
			} else {
				m.screen = screenHeliport
				m.cursor = 0
			}
			return m, nil
		}
		if len(heli.Locations) == 0 {
			return m, nil
		}
		loc := heli.Locations[m.cursor]
		m.selLocation = m.cursor
		m.locID = loc.ID
		m.locName = loc.Name
		m.screen = screenHeliport
		m.cursor = 0

	case screenHeliport:
		heli := m.lib.Helicopters[m.selHeli]
		loc := heli.Locations[m.selLocation]
		if len(loc.Heliports) == 0 {
			return m, nil
		}
		hp := loc.Heliports[m.cursor]
		m.selHeliport = m.cursor
		m.heliportID = hp.ID
		m.heliportName = hp.Name

		if m.manageMode == manageDelHP {
			m.screen = screenConfirm
			return m, nil
		}

		dir := RecordingDir(m.heliID, m.locID, m.heliportID)

		if m.flow == flowRecording {
			m.savePath = NewRecordingPath(dir)
			m.screen = screenRecordingLive
			m.liveEvents = nil
			m.recStart = time.Now()
			m.recordingStarted = false
			return m.startRecordingSession()
		}

		recs, err := ListRecordings(dir)
		if err != nil {
			m.errMsg = err.Error()
			return m, nil
		}
		if len(recs) == 0 {
			m.errMsg = "V tomto heliportu nejsou žádné nahrávky"
			return m, nil
		}
		m.recordings = recs
		m.screen = screenRecordingPick
		m.cursor = 0
		m.errMsg = ""

	case screenRecordingPick:
		if len(m.recordings) == 0 {
			return m, nil
		}
		m.playbackPath = m.recordings[m.cursor]
		m.screen = screenCountdown
		m.countdown = 3
		return m, tickCountdown(3)

	case screenRecBrowser:
		if len(m.recBrowser) == 0 {
			m.errMsg = "Žádné nahrávky"
			return m, nil
		}
		entry := m.recBrowser[m.cursor]
		m.selRecording = m.cursor
		m.recDetailPath = entry.Path
		m.heliName = entry.HeliName
		m.locName = entry.LocName
		m.heliportName = entry.HpName

		rec, err := LoadRecording(entry.Path)
		if err != nil {
			m.errMsg = err.Error()
			return m, nil
		}
		m.recDetailEvents = rec.Events
		m.recDetailOffset = 0
		m.errMsg = ""
		m.screen = screenRecDetail
	}

	return m, nil
}

func (m model) startInput(label string) (tea.Model, tea.Cmd) {
	m.inputLabel = label
	m.inputValue = ""
	m.screen = screenInput
	return m, nil
}

func (m model) goBack() (tea.Model, tea.Cmd) {
	switch m.screen {
	case screenMain:
		return m, tea.Quit
	case screenManage:
		m.screen = screenMain
		m.cursor = 0
	case screenHeli:
		if m.manageMode != manageNone {
			m.screen = screenManage
			m.cursor = 0
			m.manageMode = manageNone
			return m, nil
		}
		m.screen = screenMain
		m.cursor = 0
		m.flow = flowNone
	case screenLocation:
		m.screen = screenHeli
		m.cursor = m.selHeli
	case screenHeliport:
		m.screen = screenLocation
		m.cursor = m.selLocation
	case screenRecordingPick:
		m.screen = screenHeliport
		m.cursor = m.selHeliport
	case screenRecBrowser:
		m.screen = screenMain
		m.cursor = 0
		m.flow = flowNone
	case screenRecDetail:
		m.screen = screenRecBrowser
		m.cursor = m.selRecording
	case screenInput:
		m.screen = screenManage
		m.cursor = 0
		m.manageMode = manageNone
	case screenConfirm:
		switch m.manageMode {
		case manageDelHeli:
			m.screen = screenManage
		case manageDelLoc:
			m.screen = screenHeli
			m.cursor = m.selHeli
		case manageDelHP:
			m.screen = screenLocation
			m.cursor = m.selLocation
		case manageDelRecording:
			m.screen = screenRecDetail
			m.cursor = m.selRecording
		default:
			m.screen = screenManage
		}
		m.manageMode = manageNone
		return m, nil
	case screenRecordingLive, screenCountdown:
		return m, nil
	}
	m.errMsg = ""
	return m, nil
}

func (m model) startRecordingSession() (tea.Model, tea.Cmd) {
	if m.recordingStarted {
		return m, waitForRecordingEvent(m.eventCh)
	}
	m.recordingStarted = true
	m.eventCh = make(chan Event, 64)
	m.doneCh = make(chan error, 1)

	go func() {
		m.doneCh <- recording(m.savePath, m.eventCh)
		close(m.eventCh)
	}()

	return m, tea.Batch(
		waitForRecordingEvent(m.eventCh),
		waitForRecordingDone(m.doneCh),
		tickRecording(),
	)
}

func tickRecording() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return recordingTickMsg{}
	})
}

func waitForRecordingEvent(ch <-chan Event) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-ch
		if !ok {
			return nil
		}
		return recordingEventMsg{event: ev}
	}
}

func waitForRecordingDone(ch <-chan error) tea.Cmd {
	return func() tea.Msg {
		err := <-ch
		return recordingDoneMsg{err: err}
	}
}

func tickCountdown(n int) tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return countdownMsg{remaining: n - 1}
	})
}

func runPlaybackCmd(path string) tea.Cmd {
	return func() tea.Msg {
		err := PlaybackFile(path)
		return playbackDoneMsg{err: err}
	}
}

func (m model) listLen() int {
	switch m.screen {
	case screenMain:
		return 5
	case screenManage:
		return 6
	case screenHeli:
		return len(m.lib.Helicopters)
	case screenLocation:
		if m.selHeli >= len(m.lib.Helicopters) {
			return 0
		}
		return len(m.lib.Helicopters[m.selHeli].Locations)
	case screenHeliport:
		if m.selHeli >= len(m.lib.Helicopters) {
			return 0
		}
		heli := m.lib.Helicopters[m.selHeli]
		if m.selLocation >= len(heli.Locations) {
			return 0
		}
		return len(heli.Locations[m.selLocation].Heliports)
	case screenRecordingPick:
		return len(m.recordings)
	case screenRecBrowser:
		return len(m.recBrowser)
	default:
		return 0
	}
}

func (m model) currentItems() []string {
	switch m.screen {
	case screenMain:
		return []string{"Recording", "Playback", "Browse Recordings", "Knihovna", "Exit"}
	case screenManage:
		return []string{
			"Přidat helikoptéru",
			"Přidat lokaci",
			"Přidat heliport",
			"Smazat helikoptéru",
			"Smazat lokaci",
			"Smazat heliport",
		}
	case screenHeli:
		items := make([]string, len(m.lib.Helicopters))
		for i, h := range m.lib.Helicopters {
			items[i] = h.Name
		}
		return items
	case screenLocation:
		heli := m.lib.Helicopters[m.selHeli]
		items := make([]string, len(heli.Locations))
		for i, l := range heli.Locations {
			items[i] = l.Name
		}
		return items
	case screenHeliport:
		heli := m.lib.Helicopters[m.selHeli]
		loc := heli.Locations[m.selLocation]
		items := make([]string, len(loc.Heliports))
		for i, h := range loc.Heliports {
			items[i] = h.Name
		}
		return items
	case screenRecordingPick:
		items := make([]string, len(m.recordings))
		for i, p := range m.recordings {
			items[i] = recordingDisplayName(p)
		}
		return items
	case screenRecBrowser:
		items := make([]string, len(m.recBrowser))
		for i, e := range m.recBrowser {
			items[i] = fmt.Sprintf("%s — %s", recordingLabel(e), filepath.Base(e.Path))
		}
		return items
	default:
		return nil
	}
}

func recordingLabel(e recEntry) string {
	var parts []string
	for _, s := range []string{e.HeliName, e.LocName, e.HpName} {
		if s != "" {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, " > ")
}

func (m model) buildRecBrowser() []recEntry {
	var entries []recEntry
	_ = filepath.WalkDir(libraryRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(d.Name()), ".json") {
			return nil
		}
		if !strings.HasPrefix(d.Name(), "flight_") {
			return nil
		}

		rel, err := filepath.Rel(libraryRoot, path)
		if err != nil {
			return nil
		}
		parts := strings.Split(rel, string(filepath.Separator))

		var heliName, locName, hpName string
		switch {
		case len(parts) >= 4:
			heliName = m.resolveHeliName(parts[0])
			locName = m.resolveLocName(parts[0], parts[1])
			hpName = m.resolveHpName(parts[0], parts[1], parts[2])
		case len(parts) == 3:
			heliName = m.resolveHeliName(parts[0])
			locName = m.resolveLocName(parts[0], parts[1])
		case len(parts) == 2:
			heliName = m.resolveHeliName(parts[0])
			locName = parts[0]
			hpName = parts[1]
		default:
			return nil
		}

		entries = append(entries, recEntry{
			Path:     path,
			HeliName: heliName,
			LocName:  locName,
			HpName:   hpName,
		})
		return nil
	})
	return entries
}

func (m model) resolveHeliName(id string) string {
	for _, h := range m.lib.Helicopters {
		if h.ID == id {
			return h.Name
		}
	}
	return id
}

func (m model) resolveLocName(heliID, locID string) string {
	for _, h := range m.lib.Helicopters {
		if h.ID == heliID {
			for _, l := range h.Locations {
				if l.ID == locID {
					return l.Name
				}
			}
		}
	}
	return locID
}

func (m model) resolveHpName(heliID, locID, hpID string) string {
	for _, h := range m.lib.Helicopters {
		if h.ID == heliID {
			for _, l := range h.Locations {
				if l.ID == locID {
					for _, hp := range l.Heliports {
						if hp.ID == hpID {
							return hp.Name
						}
					}
				}
			}
		}
	}
	return hpID
}

func (m model) breadcrumb() string {
	parts := []string{}
	if m.heliName != "" {
		parts = append(parts, m.heliName)
	}
	if m.locName != "" {
		parts = append(parts, m.locName)
	}
	if m.heliportName != "" {
		parts = append(parts, m.heliportName)
	}
	if len(parts) == 0 {
		return ""
	}
	return breadcrumbStyle.Render(strings.Join(parts, " > "))
}

func (m model) renderHeader() string {
	title := neonBox(renderRainbowTitle())
	github := lipgloss.PlaceHorizontal(boxWidth, lipgloss.Center, githubStyle.Render("github: flwxpro"))
	verText := lipgloss.PlaceHorizontal(boxWidth, lipgloss.Center, versionStyle.Render(fmt.Sprintf("verze %s · první verze", version)))

	var b strings.Builder
	b.WriteString(title)
	b.WriteString("\n\n")
	b.WriteString(github)
	b.WriteString("\n")
	b.WriteString(verText)
	b.WriteString("\n\n")
	b.WriteString(gradientDivider(boxWidth))
	return b.String()
}

func renderRainbowTitle() string {
	clean := strings.Trim(strings.TrimLeft(ascii, "\n"), "\n")
	lines := strings.Split(clean, "\n")
	var b strings.Builder
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		b.WriteString(gradientText(line, true))
		b.WriteString("\n")
	}
	return b.String()
}

func neonBox(content string) string {
	lines := strings.Split(strings.TrimRight(content, "\n"), "\n")
	maxW := headerWidth
	for _, line := range lines {
		if w := lipgloss.Width(line); w > maxW {
			maxW = w
		}
	}
	inner := maxW + 2

	var b strings.Builder
	b.WriteString(gradStyle(0).Render("┌" + strings.Repeat("─", inner) + "┐"))
	b.WriteString("\n")
	for i, line := range lines {
		pad := maxW - lipgloss.Width(line)
		b.WriteString(gradStyle(float64(i)/float64(len(lines))).Render("│"))
		b.WriteString(" ")
		b.WriteString(line)
		b.WriteString(strings.Repeat(" ", pad))
		b.WriteString(" ")
		b.WriteString(gradStyle(float64(i)/float64(len(lines))).Render("│"))
		b.WriteString("\n")
	}
	b.WriteString(gradStyle(1).Render("└" + strings.Repeat("─", inner) + "┘"))
	return b.String()
}

func (m model) renderMenu(title string, items []string) string {
	var left strings.Builder
	left.WriteString(m.renderHeader())
	left.WriteString("\n\n")
	if bc := m.breadcrumb(); bc != "" {
		left.WriteString(bc)
		left.WriteString("\n\n")
	}
	left.WriteString(gradientText(title, true))
	left.WriteString("\n\n")

	for i, item := range items {
		cursor := "  "
		line := item
		if i == m.cursor {
			cursor = "> "
			line = selectedStyle.Render(item)
		} else {
			line = dimStyle.Render(item)
		}
		left.WriteString(cursor)
		left.WriteString(line)
		left.WriteString("\n")
	}

	left.WriteString("\n")
	help := keyStyle.Render("↑/↓") + dimStyle.Render(" výběr  •  ") + keyStyle.Render("Enter") + dimStyle.Render(" potvrdit  •  ") + keyStyle.Render("Esc/q") + dimStyle.Render(" zpět")
	left.WriteString(help)

	columns := lipgloss.JoinHorizontal(lipgloss.Top, left.String(), "           ", neonArt(heliArt))
	return columns
}

func (m model) confirmName() string {
	switch m.manageMode {
	case manageDelHeli:
		if m.selHeli < len(m.lib.Helicopters) {
			return m.lib.Helicopters[m.selHeli].Name
		}
	case manageDelLoc:
		if m.selHeli < len(m.lib.Helicopters) {
			heli := m.lib.Helicopters[m.selHeli]
			if m.selLocation < len(heli.Locations) {
				return heli.Locations[m.selLocation].Name
			}
		}
	case manageDelHP:
		if m.selHeli < len(m.lib.Helicopters) {
			heli := m.lib.Helicopters[m.selHeli]
			if m.selLocation < len(heli.Locations) {
				loc := heli.Locations[m.selLocation]
				if m.selHeliport < len(loc.Heliports) {
					return loc.Heliports[m.selHeliport].Name
				}
			}
		}
	case manageDelRecording:
		return filepath.Base(m.recDetailPath)
	}
	return ""
}

func (m model) View() tea.View {
	var content string

	switch m.screen {
	case screenRecordingLive:
		elapsed := time.Since(m.recStart).Round(time.Millisecond)
		var b strings.Builder
		b.WriteString(neonBox(renderRainbowTitle()))
		b.WriteString("\n\n")
		b.WriteString(selectedStyle.Render("● Nahrávání"))
		b.WriteString("\n\n")
		if bc := m.breadcrumb(); bc != "" {
			b.WriteString(bc)
			b.WriteString("\n\n")
		}
		b.WriteString(dimStyle.Render(fmt.Sprintf("Čas: %s\n", elapsed)))
		b.WriteString(dimStyle.Render(fmt.Sprintf("Uloží se do: %s\n\n", recordingDisplayName(m.savePath))))
		b.WriteString(selectedStyle.Render("Poslední eventy:\n"))
		if len(m.liveEvents) == 0 {
			b.WriteString(dimStyle.Render("  (zatím žádné)\n"))
		}
		for _, ev := range m.liveEvents {
			b.WriteString(fmt.Sprintf("  [%4d ms] %s %s\n", ev.At, keyStyle.Render(ev.Key), dimStyle.Render(ev.Type)))
		}
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("Stiskni O v Robloxu pro ukončení nahrávání"))
		content = b.String()

	case screenCountdown:
		var b strings.Builder
		b.WriteString(neonBox(renderRainbowTitle()))
		b.WriteString("\n\n")
		b.WriteString(selectedStyle.Render("Připrav se na playback"))
		b.WriteString("\n\n")
		b.WriteString(dimStyle.Render("Přepni do Robloxu…\n\n"))
		b.WriteString(fmt.Sprintf("Start za: %s\n\n", gradientText(fmt.Sprintf("%d", m.countdown), true)))
		b.WriteString(helpStyle.Render("Soubor: " + recordingDisplayName(m.playbackPath)))
		content = b.String()

	case screenInput:
		var b strings.Builder
		b.WriteString(neonBox(renderRainbowTitle()))
		b.WriteString("\n\n")
		if bc := m.breadcrumb(); bc != "" {
			b.WriteString(bc)
			b.WriteString("\n\n")
		}
		b.WriteString(selectedStyle.Render(m.inputLabel))
		b.WriteString("\n\n")
		b.WriteString("> " + gradientText(m.inputValue, false))
		if m.inputValue == "" {
			b.WriteString(" ")
		}
		b.WriteString("\n\n")
		b.WriteString(keyStyle.Render("Enter") + dimStyle.Render(" uložit  •  ") + keyStyle.Render("Esc") + dimStyle.Render(" zrušit"))
		content = b.String()

	case screenConfirm:
		var b strings.Builder
		b.WriteString(neonBox(renderRainbowTitle()))
		b.WriteString("\n\n")
		b.WriteString(selectedStyle.Render("Opravdu smazat?"))
		b.WriteString("\n\n")
		if bc := m.breadcrumb(); bc != "" {
			b.WriteString(bc)
			b.WriteString("\n\n")
		}
		b.WriteString(gradientText(m.confirmName(), true))
		b.WriteString("\n\n")
		b.WriteString(keyStyle.Render("Enter") + dimStyle.Render(" smazat  •  ") + keyStyle.Render("Esc") + dimStyle.Render(" zrušit"))
		content = b.String()

	case screenRecDetail:
		var b strings.Builder
		b.WriteString(neonBox(renderRainbowTitle()))
		b.WriteString("\n\n")
		b.WriteString(selectedStyle.Render("Prohlížeč nahrávek"))
		b.WriteString("\n\n")
		if bc := m.breadcrumb(); bc != "" {
			b.WriteString(bc)
			b.WriteString("\n")
		}
		b.WriteString(dimStyle.Render(filepath.Base(m.recDetailPath)))
		b.WriteString("\n\n")
		if len(m.recDetailEvents) == 0 {
			b.WriteString(dimStyle.Render("  (žádné eventy)\n"))
		} else {
			total := m.recDetailEvents[len(m.recDetailEvents)-1].At
			b.WriteString(dimStyle.Render(fmt.Sprintf("Délka letu: %.2f s\n", float64(total)/1000)))
		}
		b.WriteString("\n")
		start := m.recDetailOffset
		end := start + recDetailVisible
		if end > len(m.recDetailEvents) {
			end = len(m.recDetailEvents)
		}
		for _, ev := range m.recDetailEvents[start:end] {
			b.WriteString(fmt.Sprintf("  [%4d ms] %s %s\n", ev.At, keyStyle.Render(ev.Key), dimStyle.Render(ev.Type)))
		}
		if end < len(m.recDetailEvents) {
			b.WriteString(dimStyle.Render(fmt.Sprintf("\n  … a %d dalších eventů\n", len(m.recDetailEvents)-end)))
		}
		b.WriteString("\n")
		b.WriteString(keyStyle.Render("↑/↓") + dimStyle.Render(" scroll  •  ") + keyStyle.Render("d") + dimStyle.Render(" smazat  •  ") + keyStyle.Render("Esc") + dimStyle.Render(" zpět"))
		content = b.String()

	default:
		title := ""
		switch m.screen {
		case screenMain:
			title = "Hlavní menu"
		case screenManage:
			title = "Knihovna"
		case screenHeli:
			title = "Vyber helikoptéru"
		case screenLocation:
			title = "Vyber lokaci"
		case screenHeliport:
			title = "Vyber heliport"
		case screenRecordingPick:
			title = "Vyber nahrávku"
		case screenRecBrowser:
			title = "Prohlížeč nahrávek"
		}
		content = m.renderMenu(title, m.currentItems())
	}

	if m.status != "" {
		content += "\n\n" + statusStyle.Render(m.status)
	}
	if m.errMsg != "" {
		content += "\n\n" + errorStyle.Render("Chyba: "+m.errMsg)
	}

	v := tea.NewView(content)
	v.AltScreen = true
	v.BackgroundColor = lipgloss.Color("#0a0a12")
	return v
}
