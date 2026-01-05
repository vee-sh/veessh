package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/vee-sh/veessh/internal/config"
)

// View modes
type viewMode int

const (
	viewProfiles viewMode = iota
	viewGroups
	viewFavorites
	viewRecent
	viewSessions
	viewSettings
	viewEdit
	viewAdd
	viewSearch
)

// Model represents the main TUI application state
type Model struct {
	// Configuration
	config     config.Config
	configPath string

	// View state
	mode          viewMode
	width         int
	height        int
	ready         bool
	quitting      bool
	showHelp      bool
	statusMessage string

	// Profile management
	profiles       []config.Profile // Filtered/sorted profiles
	allProfiles    []config.Profile // All profiles
	groups         []string
	selectedGroup  string
	selectedIndex  int
	selectedProfile *config.Profile
	multiSelect    map[string]bool

	// UI components
	searchInput   textinput.Model
	help          help.Model
	keys          keyMap
	editForm      *editForm
	
	// Search and filter
	searchActive  bool
	searchQuery   string
	filterTags    []string
	favoritesOnly bool

	// Style
	styles *styles
}

// editForm represents the profile edit form
type editForm struct {
	profile       config.Profile
	inputs        []textinput.Model
	focusIndex    int
	errorMessage  string
	testingConn   bool
}

// keyMap defines all keyboard shortcuts
type keyMap struct {
	Up         key.Binding
	Down       key.Binding
	Left       key.Binding
	Right      key.Binding
	Enter      key.Binding
	Space      key.Binding
	Tab        key.Binding
	ShiftTab   key.Binding
	Search     key.Binding
	Add        key.Binding
	Edit       key.Binding
	Delete     key.Binding
	Favorite   key.Binding
	Clone      key.Binding
	Export     key.Binding
	Import     key.Binding
	Connect    key.Binding
	SFTP       key.Binding
	Test       key.Binding
	Help       key.Binding
	Quit       key.Binding
	Cancel     key.Binding
	Save       key.Binding
	PageUp     key.Binding
	PageDown   key.Binding
}

// styles holds all the styling for the TUI
type styles struct {
	Base           lipgloss.Style
	Header         lipgloss.Style
	Footer         lipgloss.Style
	Tabs           lipgloss.Style
	ActiveTab      lipgloss.Style
	InactiveTab    lipgloss.Style
	GroupPane      lipgloss.Style
	ProfilePane    lipgloss.Style
	DetailPane     lipgloss.Style
	SelectedItem   lipgloss.Style
	UnselectedItem lipgloss.Style
	FavoriteIcon   lipgloss.Style
	ConnectedIcon  lipgloss.Style
	Title          lipgloss.Style
	Subtitle       lipgloss.Style
	Label          lipgloss.Style
	Value          lipgloss.Style
	Error          lipgloss.Style
	Success        lipgloss.Style
	Warning        lipgloss.Style
	SearchBox      lipgloss.Style
	Button         lipgloss.Style
	ActiveButton   lipgloss.Style
	Border         lipgloss.Border
}

func defaultKeyMap() keyMap {
	return keyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "left"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "right"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "connect"),
		),
		Space: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("space", "select"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next field"),
		),
		ShiftTab: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "prev field"),
		),
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
		),
		Add: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "add"),
		),
		Edit: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "edit"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete"),
		),
		Favorite: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "favorite"),
		),
		Clone: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "clone"),
		),
		Export: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "export"),
		),
		Import: key.NewBinding(
			key.WithKeys("i"),
			key.WithHelp("i", "import"),
		),
		Connect: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "connect"),
		),
		SFTP: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "SFTP"),
		),
		Test: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "test"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
		Save: key.NewBinding(
			key.WithKeys("ctrl+s"),
			key.WithHelp("ctrl+s", "save"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup"),
			key.WithHelp("pgup", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown"),
			key.WithHelp("pgdn", "page down"),
		),
	}
}

func defaultStyles() *styles {
	// Define colors
	primary := lipgloss.Color("#7571F9")
	success := lipgloss.Color("#71F9A5")
	warning := lipgloss.Color("#F9E871")
	danger := lipgloss.Color("#F97171")
	muted := lipgloss.Color("#6B7280")
	bg := lipgloss.Color("#1F2937")
	bgLight := lipgloss.Color("#374151")
	text := lipgloss.Color("#F3F4F6")
	textDim := lipgloss.Color("#9CA3AF")

	border := lipgloss.RoundedBorder()

	return &styles{
		Base: lipgloss.NewStyle().
			Foreground(text).
			Background(bg),
		Header: lipgloss.NewStyle().
			Foreground(text).
			Background(bgLight).
			Bold(true).
			Padding(0, 1),
		Footer: lipgloss.NewStyle().
			Foreground(textDim).
			Background(bgLight).
			Padding(0, 1),
		Tabs: lipgloss.NewStyle().
			Padding(0, 1),
		ActiveTab: lipgloss.NewStyle().
			Foreground(primary).
			Background(bgLight).
			Bold(true).
			Padding(0, 2),
		InactiveTab: lipgloss.NewStyle().
			Foreground(textDim).
			Padding(0, 2),
		GroupPane: lipgloss.NewStyle().
			Border(border, true).
			BorderForeground(muted).
			Padding(0, 1),
		ProfilePane: lipgloss.NewStyle().
			Border(border, true).
			BorderForeground(muted).
			Padding(0, 1),
		DetailPane: lipgloss.NewStyle().
			Border(border, true).
			BorderForeground(muted).
			Padding(0, 1),
		SelectedItem: lipgloss.NewStyle().
			Foreground(text).
			Background(primary).
			Bold(true).
			Padding(0, 1),
		UnselectedItem: lipgloss.NewStyle().
			Foreground(text).
			Padding(0, 1),
		FavoriteIcon: lipgloss.NewStyle().
			Foreground(warning).
			SetString("★"),
		ConnectedIcon: lipgloss.NewStyle().
			Foreground(success).
			SetString("●"),
		Title: lipgloss.NewStyle().
			Foreground(primary).
			Bold(true),
		Subtitle: lipgloss.NewStyle().
			Foreground(textDim),
		Label: lipgloss.NewStyle().
			Foreground(textDim).
			Width(12),
		Value: lipgloss.NewStyle().
			Foreground(text),
		Error: lipgloss.NewStyle().
			Foreground(danger).
			Bold(true),
		Success: lipgloss.NewStyle().
			Foreground(success).
			Bold(true),
		Warning: lipgloss.NewStyle().
			Foreground(warning).
			Bold(true),
		SearchBox: lipgloss.NewStyle().
			Border(border, true).
			BorderForeground(primary).
			Padding(0, 1),
		Button: lipgloss.NewStyle().
			Foreground(text).
			Background(bgLight).
			Padding(0, 2).
			MarginRight(1),
		ActiveButton: lipgloss.NewStyle().
			Foreground(bg).
			Background(primary).
			Bold(true).
			Padding(0, 2).
			MarginRight(1),
		Border: border,
	}
}

// New creates a new TUI model
func New(cfgPath string) (*Model, error) {
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize search input
	searchInput := textinput.New()
	searchInput.Placeholder = "Search profiles..."
	searchInput.CharLimit = 100

	// Initialize the model
	m := &Model{
		config:        cfg,
		configPath:    cfgPath,
		allProfiles:   cfg.ListProfiles(),
		profiles:      cfg.ListProfiles(),
		multiSelect:   make(map[string]bool),
		searchInput:   searchInput,
		help:          help.New(),
		keys:          defaultKeyMap(),
		styles:        defaultStyles(),
		mode:          viewProfiles,
	}

	// Extract unique groups
	m.updateGroups()

	return m, nil
}

// Init initializes the model
func (m *Model) Init() tea.Cmd {
	return tea.EnterAltScreen
}

// updateGroups extracts unique groups from profiles
func (m *Model) updateGroups() {
	groupMap := make(map[string]bool)
	for _, p := range m.allProfiles {
		group := p.Group
		if group == "" {
			group = "default"
		}
		groupMap[group] = true
	}
	
	m.groups = make([]string, 0, len(groupMap))
	for g := range groupMap {
		m.groups = append(m.groups, g)
	}
	
	// Sort groups
	for i := 0; i < len(m.groups)-1; i++ {
		for j := i + 1; j < len(m.groups); j++ {
			if strings.ToLower(m.groups[i]) > strings.ToLower(m.groups[j]) {
				m.groups[i], m.groups[j] = m.groups[j], m.groups[i]
			}
		}
	}
}

// filterProfiles applies current filters to the profile list
func (m *Model) filterProfiles() {
	m.profiles = make([]config.Profile, 0)
	
	for _, p := range m.allProfiles {
		// Apply group filter
		if m.selectedGroup != "" && m.selectedGroup != "all" {
			group := p.Group
			if group == "" {
				group = "default"
			}
			if group != m.selectedGroup {
				continue
			}
		}
		
		// Apply favorites filter
		if m.favoritesOnly && !p.Favorite {
			continue
		}
		
		// Apply search filter
		if m.searchQuery != "" {
			query := strings.ToLower(m.searchQuery)
			profileStr := strings.ToLower(fmt.Sprintf("%s %s %s %s %s",
				p.Name, p.Host, p.Username, p.Group, p.Description))
			
			// Also search in tags
			for _, tag := range p.Tags {
				profileStr += " " + strings.ToLower(tag)
			}
			
			if !strings.Contains(profileStr, query) {
				continue
			}
		}
		
		// Apply tag filters
		if len(m.filterTags) > 0 {
			hasAllTags := true
			for _, filterTag := range m.filterTags {
				found := false
				for _, profileTag := range p.Tags {
					if strings.EqualFold(profileTag, filterTag) {
						found = true
						break
					}
				}
				if !found {
					hasAllTags = false
					break
				}
			}
			if !hasAllTags {
				continue
			}
		}
		
		m.profiles = append(m.profiles, p)
	}
	
	// Reset selection if needed
	if m.selectedIndex >= len(m.profiles) {
		m.selectedIndex = len(m.profiles) - 1
	}
	if m.selectedIndex < 0 && len(m.profiles) > 0 {
		m.selectedIndex = 0
	}
	
	// Update selected profile
	if m.selectedIndex >= 0 && m.selectedIndex < len(m.profiles) {
		m.selectedProfile = &m.profiles[m.selectedIndex]
	} else {
		m.selectedProfile = nil
	}
}
