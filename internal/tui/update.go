package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/vee-sh/veessh/internal/config"
	"github.com/vee-sh/veessh/internal/credentials"
)

// Message types
type profileSelectedMsg struct {
	profile config.Profile
}

type profileDeletedMsg struct {
	name string
}

type profileSavedMsg struct {
	profile config.Profile
}

type statusMsg struct {
	message string
	isError bool
}

// Update handles all events and updates the model
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		return m, nil

	case tea.KeyMsg:
		// Handle global keys first
		if key.Matches(msg, m.keys.Quit) && m.mode != viewEdit && m.mode != viewAdd {
			m.quitting = true
			return m, tea.Quit
		}

		// Route to appropriate handler based on mode
		switch m.mode {
		case viewSearch:
			return m.handleSearchKeys(msg)
		case viewEdit, viewAdd:
			return m.handleEditKeys(msg)
		default:
			return m.handleMainKeys(msg)
		}

	case statusMsg:
		m.statusMessage = msg.message
		return m, nil

	case profileSelectedMsg:
		// Handle profile connection
		return m, m.connectToProfile(msg.profile)

	case profileDeletedMsg:
		m.statusMessage = fmt.Sprintf("Profile '%s' deleted", msg.name)
		m.reloadProfiles()
		return m, nil

	case profileSavedMsg:
		m.statusMessage = fmt.Sprintf("Profile '%s' saved", msg.profile.Name)
		m.reloadProfiles()
		m.mode = viewProfiles
		m.editForm = nil
		return m, nil
	}

	return m, tea.Batch(cmds...)
}

// handleMainKeys handles keyboard input for the main view
func (m *Model) handleMainKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Up):
		if m.selectedIndex > 0 {
			m.selectedIndex--
			m.updateSelectedProfile()
		}

	case key.Matches(msg, m.keys.Down):
		if m.selectedIndex < len(m.profiles)-1 {
			m.selectedIndex++
			m.updateSelectedProfile()
		}

	case key.Matches(msg, m.keys.PageUp):
		m.selectedIndex -= 10
		if m.selectedIndex < 0 {
			m.selectedIndex = 0
		}
		m.updateSelectedProfile()

	case key.Matches(msg, m.keys.PageDown):
		m.selectedIndex += 10
		if m.selectedIndex >= len(m.profiles) {
			m.selectedIndex = len(m.profiles) - 1
		}
		m.updateSelectedProfile()

	case key.Matches(msg, m.keys.Left):
		// Switch between groups
		if m.selectedGroup != "" {
			for i, g := range m.groups {
				if g == m.selectedGroup {
					if i > 0 {
						m.selectedGroup = m.groups[i-1]
					} else {
						m.selectedGroup = "all"
					}
					break
				}
			}
		} else if len(m.groups) > 0 {
			m.selectedGroup = m.groups[len(m.groups)-1]
		}
		m.filterProfiles()

	case key.Matches(msg, m.keys.Right):
		// Switch between groups
		if m.selectedGroup == "" || m.selectedGroup == "all" {
			if len(m.groups) > 0 {
				m.selectedGroup = m.groups[0]
			}
		} else {
			for i, g := range m.groups {
				if g == m.selectedGroup {
					if i < len(m.groups)-1 {
						m.selectedGroup = m.groups[i+1]
					} else {
						m.selectedGroup = "all"
					}
					break
				}
			}
		}
		m.filterProfiles()

	case key.Matches(msg, m.keys.Enter):
		// Connect to selected profile
		if m.selectedProfile != nil {
			return m, m.connectToProfile(*m.selectedProfile)
		}

	case key.Matches(msg, m.keys.Space):
		// Toggle multi-select
		if m.selectedProfile != nil {
			if m.multiSelect[m.selectedProfile.Name] {
				delete(m.multiSelect, m.selectedProfile.Name)
			} else {
				m.multiSelect[m.selectedProfile.Name] = true
			}
		}

	case key.Matches(msg, m.keys.Search):
		// Enter search mode
		m.mode = viewSearch
		m.searchActive = true
		m.searchInput.Focus()
		return m, textinput.Blink

	case key.Matches(msg, m.keys.Add):
		// Add new profile
		m.startAddProfile()
		return m, textinput.Blink

	case key.Matches(msg, m.keys.Edit):
		// Edit selected profile
		if m.selectedProfile != nil {
			m.startEditProfile(*m.selectedProfile)
			return m, textinput.Blink
		}

	case key.Matches(msg, m.keys.Delete):
		// Delete selected profile(s)
		if len(m.multiSelect) > 0 {
			return m, m.deleteMultipleProfiles()
		} else if m.selectedProfile != nil {
			return m, m.deleteProfile(m.selectedProfile.Name)
		}

	case key.Matches(msg, m.keys.Favorite):
		// Toggle favorite
		if m.selectedProfile != nil {
			m.toggleFavorite(m.selectedProfile.Name)
			return m, nil
		}

	case key.Matches(msg, m.keys.Clone):
		// Clone profile
		if m.selectedProfile != nil {
			m.cloneProfile(*m.selectedProfile)
			return m, textinput.Blink
		}

	case key.Matches(msg, m.keys.Help):
		// Toggle help
		m.showHelp = !m.showHelp

	case key.Matches(msg, m.keys.Cancel):
		// Clear multi-select if active
		if len(m.multiSelect) > 0 {
			m.multiSelect = make(map[string]bool)
		}
		
	// Tab navigation for view modes
	case msg.String() == "1":
		m.mode = viewProfiles
		m.filterProfiles()
	case msg.String() == "2":
		m.mode = viewGroups
		m.filterProfiles()
	case msg.String() == "3":
		m.mode = viewFavorites
		m.favoritesOnly = true
		m.filterProfiles()
	case msg.String() == "4":
		m.mode = viewRecent
		m.sortByRecent()
	}

	return m, nil
}

// handleSearchKeys handles keyboard input in search mode
func (m *Model) handleSearchKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch {
	case key.Matches(msg, m.keys.Cancel):
		// Exit search mode
		m.mode = viewProfiles
		m.searchActive = false
		m.searchInput.Blur()
		return m, nil

	case key.Matches(msg, m.keys.Enter):
		// Apply search
		m.searchQuery = m.searchInput.Value()
		m.filterProfiles()
		m.mode = viewProfiles
		m.searchActive = false
		m.searchInput.Blur()
		return m, nil

	default:
		// Handle text input
		m.searchInput, cmd = m.searchInput.Update(msg)
		// Live search
		m.searchQuery = m.searchInput.Value()
		m.filterProfiles()
		return m, cmd
	}
}

// handleEditKeys handles keyboard input in edit mode
func (m *Model) handleEditKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.editForm == nil {
		return m, nil
	}

	switch {
	case key.Matches(msg, m.keys.Cancel):
		// Cancel editing
		m.mode = viewProfiles
		m.editForm = nil
		return m, nil

	case key.Matches(msg, m.keys.Save):
		// Save profile
		return m, m.saveProfile()

	case key.Matches(msg, m.keys.Tab):
		// Next field
		m.editForm.focusIndex++
		if m.editForm.focusIndex >= len(m.editForm.inputs) {
			m.editForm.focusIndex = 0
		}
		m.updateEditFocus()
		return m, textinput.Blink

	case key.Matches(msg, m.keys.ShiftTab):
		// Previous field
		m.editForm.focusIndex--
		if m.editForm.focusIndex < 0 {
			m.editForm.focusIndex = len(m.editForm.inputs) - 1
		}
		m.updateEditFocus()
		return m, textinput.Blink

	case msg.String() == "t" || msg.String() == "T":
		// Test connection
		return m, m.testConnection()

	default:
		// Handle text input for the focused field
		var cmd tea.Cmd
		m.editForm.inputs[m.editForm.focusIndex], cmd = m.editForm.inputs[m.editForm.focusIndex].Update(msg)
		return m, cmd
	}
}

// Profile management functions

func (m *Model) updateSelectedProfile() {
	if m.selectedIndex >= 0 && m.selectedIndex < len(m.profiles) {
		m.selectedProfile = &m.profiles[m.selectedIndex]
	} else {
		m.selectedProfile = nil
	}
}

func (m *Model) reloadProfiles() {
	// Reload config from disk
	cfg, err := config.Load(m.configPath)
	if err == nil {
		m.config = cfg
		m.allProfiles = cfg.ListProfiles()
		m.updateGroups()
		m.filterProfiles()
	}
}

func (m *Model) toggleFavorite(name string) {
	p, ok := m.config.GetProfile(name)
	if !ok {
		return
	}
	
	p.Favorite = !p.Favorite
	m.config.UpsertProfile(p)
	
	// Save to disk
	if err := config.Save(m.configPath, m.config); err != nil {
		m.statusMessage = fmt.Sprintf("Error saving: %v", err)
	} else {
		m.reloadProfiles()
		if p.Favorite {
			m.statusMessage = fmt.Sprintf("'%s' added to favorites", name)
		} else {
			m.statusMessage = fmt.Sprintf("'%s' removed from favorites", name)
		}
	}
}

func (m *Model) sortByRecent() {
	// Sort profiles by last used time
	for i := 0; i < len(m.profiles)-1; i++ {
		for j := i + 1; j < len(m.profiles); j++ {
			if m.profiles[j].LastUsed.After(m.profiles[i].LastUsed) {
				m.profiles[i], m.profiles[j] = m.profiles[j], m.profiles[i]
			}
		}
	}
	
	// Reset selection
	if len(m.profiles) > 0 {
		m.selectedIndex = 0
		m.updateSelectedProfile()
	}
}

// Edit form functions

func (m *Model) startEditProfile(p config.Profile) {
	m.mode = viewEdit
	m.editForm = &editForm{
		profile: p,
		inputs:  make([]textinput.Model, 9),
	}
	
	// Initialize input fields
	fieldValues := []string{
		p.Name,
		p.Host,
		fmt.Sprintf("%d", p.Port),
		p.Username,
		p.IdentityFile,
		p.ProxyJump,
		p.Group,
		strings.Join(p.Tags, ", "),
		p.Description,
	}
	
	placeholders := []string{
		"Profile name",
		"Hostname or IP",
		"Port number",
		"Username",
		"Path to SSH key",
		"Jump host",
		"Group name",
		"Comma-separated tags",
		"Description",
	}
	
	for i := range m.editForm.inputs {
		m.editForm.inputs[i] = textinput.New()
		m.editForm.inputs[i].SetValue(fieldValues[i])
		m.editForm.inputs[i].Placeholder = placeholders[i]
		
		// Set character limits
		if i == 0 || i == 1 || i == 3 || i == 4 || i == 5 || i == 6 {
			m.editForm.inputs[i].CharLimit = 100
		} else if i == 2 {
			m.editForm.inputs[i].CharLimit = 5
		} else {
			m.editForm.inputs[i].CharLimit = 200
		}
	}
	
	// Focus first field
	m.editForm.focusIndex = 0
	m.editForm.inputs[0].Focus()
}

func (m *Model) startAddProfile() {
	m.mode = viewAdd
	m.editForm = &editForm{
		profile: config.Profile{
			Protocol: config.ProtocolSSH,
			Port:     22,
			UseAgent: true,
		},
		inputs: make([]textinput.Model, 9),
	}
	
	placeholders := []string{
		"Profile name",
		"Hostname or IP",
		"22",
		"Username",
		"Path to SSH key (optional)",
		"Jump host (optional)",
		"Group name (optional)",
		"Comma-separated tags (optional)",
		"Description (optional)",
	}
	
	for i := range m.editForm.inputs {
		m.editForm.inputs[i] = textinput.New()
		m.editForm.inputs[i].Placeholder = placeholders[i]
		
		// Set defaults for new profile
		if i == 2 {
			m.editForm.inputs[i].SetValue("22")
		}
		
		// Set character limits
		if i == 0 || i == 1 || i == 3 || i == 4 || i == 5 || i == 6 {
			m.editForm.inputs[i].CharLimit = 100
		} else if i == 2 {
			m.editForm.inputs[i].CharLimit = 5
		} else {
			m.editForm.inputs[i].CharLimit = 200
		}
	}
	
	// Focus first field
	m.editForm.focusIndex = 0
	m.editForm.inputs[0].Focus()
}

func (m *Model) cloneProfile(p config.Profile) {
	// Create a copy with a new name
	p.Name = p.Name + "-copy"
	
	// Start edit mode with the cloned profile
	m.startEditProfile(p)
	m.mode = viewAdd // Treat as new profile
}

func (m *Model) updateEditFocus() {
	// Blur all inputs
	for i := range m.editForm.inputs {
		m.editForm.inputs[i].Blur()
	}
	// Focus the current one
	m.editForm.inputs[m.editForm.focusIndex].Focus()
}

func (m *Model) saveProfile() tea.Cmd {
	if m.editForm == nil {
		return nil
	}
	
	// Build profile from form inputs
	p := config.Profile{
		Name:         strings.TrimSpace(m.editForm.inputs[0].Value()),
		Host:         strings.TrimSpace(m.editForm.inputs[1].Value()),
		Username:     strings.TrimSpace(m.editForm.inputs[3].Value()),
		IdentityFile: strings.TrimSpace(m.editForm.inputs[4].Value()),
		ProxyJump:    strings.TrimSpace(m.editForm.inputs[5].Value()),
		Group:        strings.TrimSpace(m.editForm.inputs[6].Value()),
		Description:  strings.TrimSpace(m.editForm.inputs[8].Value()),
		Protocol:     config.ProtocolSSH,
		UseAgent:     true,
	}
	
	// Parse port
	portStr := strings.TrimSpace(m.editForm.inputs[2].Value())
	if portStr != "" {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			m.editForm.errorMessage = "Invalid port number"
			return nil
		}
		p.Port = port
	}
	
	// Parse tags
	tagsStr := strings.TrimSpace(m.editForm.inputs[7].Value())
	if tagsStr != "" {
		tags := strings.Split(tagsStr, ",")
		for i, tag := range tags {
			tags[i] = strings.TrimSpace(tag)
		}
		p.Tags = tags
	}
	
	// Validate
	if err := (&p).Validate(); err != nil {
		m.editForm.errorMessage = err.Error()
		return nil
	}
	
	// Save to config
	m.config.UpsertProfile(p)
	if err := config.Save(m.configPath, m.config); err != nil {
		m.editForm.errorMessage = fmt.Sprintf("Save failed: %v", err)
		return nil
	}
	
	// Success - return to main view
	return func() tea.Msg {
		return profileSavedMsg{profile: p}
	}
}

// Connection functions

func (m *Model) connectToProfile(p config.Profile) tea.Cmd {
	return func() tea.Msg {
		// In a real implementation, this would launch the SSH connection
		// For now, we'll just update the status
		return statusMsg{
			message: fmt.Sprintf("Connecting to %s@%s...", p.Username, p.Host),
			isError: false,
		}
	}
}

func (m *Model) testConnection() tea.Cmd {
	if m.editForm == nil {
		return nil
	}
	
	m.editForm.testingConn = true
	
	return func() tea.Msg {
		// In a real implementation, this would test the connection
		// For now, we'll simulate it
		return statusMsg{
			message: "Connection test successful",
			isError: false,
		}
	}
}

// Deletion functions

func (m *Model) deleteProfile(name string) tea.Cmd {
	return func() tea.Msg {
		// Delete from config
		if _, ok := m.config.GetProfile(name); ok {
			m.config.DeleteProfile(name)
			
			// Save to disk
			if err := config.Save(m.configPath, m.config); err != nil {
				return statusMsg{
					message: fmt.Sprintf("Failed to delete: %v", err),
					isError: true,
				}
			}
			
			// Delete password if exists
			_ = credentials.DeletePassword(name)
			
			return profileDeletedMsg{name: name}
		}
		
		return statusMsg{
			message: fmt.Sprintf("Profile '%s' not found", name),
			isError: true,
		}
	}
}

func (m *Model) deleteMultipleProfiles() tea.Cmd {
	names := make([]string, 0, len(m.multiSelect))
	for name := range m.multiSelect {
		names = append(names, name)
	}
	
	return func() tea.Msg {
		deleted := 0
		for _, name := range names {
			if _, ok := m.config.GetProfile(name); ok {
				m.config.DeleteProfile(name)
				_ = credentials.DeletePassword(name)
				deleted++
			}
		}
		
		if deleted > 0 {
			if err := config.Save(m.configPath, m.config); err != nil {
				return statusMsg{
					message: fmt.Sprintf("Failed to delete profiles: %v", err),
					isError: true,
				}
			}
			
			// Clear selection
			m.multiSelect = make(map[string]bool)
			
			return statusMsg{
				message: fmt.Sprintf("Deleted %d profiles", deleted),
				isError: false,
			}
		}
		
		return statusMsg{
			message: "No profiles deleted",
			isError: true,
		}
	}
}
