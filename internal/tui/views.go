package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/vee-sh/veessh/internal/config"
)

// effectivePort returns the effective port for a profile
func effectivePort(p config.Profile) int {
	if p.Port > 0 {
		return p.Port
	}
	switch p.Protocol {
	case config.ProtocolSSH, config.ProtocolSFTP:
		return 22
	case config.ProtocolTelnet:
		return 23
	default:
		return 0
	}
}

// View renders the entire TUI
func (m *Model) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	if m.quitting {
		return ""
	}

	// Build the main view based on current mode
	switch m.mode {
	case viewEdit, viewAdd:
		return m.viewEditForm()
	case viewSearch:
		return m.viewSearchMode()
	default:
		return m.viewMainScreen()
	}
}

// viewMainScreen renders the main profile management screen
func (m *Model) viewMainScreen() string {
	var sections []string

	// Header
	sections = append(sections, m.viewHeader())

	// Main content area
	contentHeight := m.height - 4 // Account for header and footer
	
	// Create three-pane layout
	groupWidth := 20
	profileWidth := 30
	detailWidth := m.width - groupWidth - profileWidth - 6 // Account for borders

	// Ensure minimum widths
	if detailWidth < 30 {
		detailWidth = 30
	}

	// Render panes
	groupPane := m.viewGroupPane(groupWidth, contentHeight-2)
	profilePane := m.viewProfilePane(profileWidth, contentHeight-2)
	detailPane := m.viewDetailPane(detailWidth, contentHeight-2)

	// Join panes horizontally
	content := lipgloss.JoinHorizontal(
		lipgloss.Top,
		groupPane,
		profilePane,
		detailPane,
	)

	sections = append(sections, content)

	// Status bar / footer
	sections = append(sections, m.viewFooter())

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// viewHeader renders the header with tabs
func (m *Model) viewHeader() string {
	title := m.styles.Title.Render("veessh v0.6.0")
	
	tabs := []string{"Profiles", "Groups", "Favorites", "Recent", "Sessions", "Settings"}
	tabViews := make([]string, len(tabs))
	
	for i, tab := range tabs {
		style := m.styles.InactiveTab
		if (i == 0 && m.mode == viewProfiles) ||
			(i == 1 && m.mode == viewGroups) ||
			(i == 2 && m.mode == viewFavorites) ||
			(i == 3 && m.mode == viewRecent) ||
			(i == 4 && m.mode == viewSessions) ||
			(i == 5 && m.mode == viewSettings) {
			style = m.styles.ActiveTab
		}
		tabViews[i] = style.Render(tab)
	}
	
	header := lipgloss.JoinHorizontal(
		lipgloss.Left,
		title,
		"  ",
		strings.Join(tabViews, ""),
	)
	
	return m.styles.Header.Width(m.width).Render(header)
}

// viewGroupPane renders the group list
func (m *Model) viewGroupPane(width, height int) string {
	var content strings.Builder
	
	content.WriteString(m.styles.Subtitle.Render("Groups\n"))
	content.WriteString(strings.Repeat("─", width-2) + "\n")
	
	// Count profiles per group
	groupCounts := make(map[string]int)
	for _, p := range m.allProfiles {
		group := p.Group
		if group == "" {
			group = "default"
		}
		groupCounts[group]++
	}
	
	// Add "All" option
	allItem := "All"
	if m.selectedGroup == "" || m.selectedGroup == "all" {
		allItem = m.styles.SelectedItem.Width(width-4).Render(fmt.Sprintf("▶ All (%d)", len(m.allProfiles)))
	} else {
		allItem = m.styles.UnselectedItem.Width(width-4).Render(fmt.Sprintf("  All (%d)", len(m.allProfiles)))
	}
	content.WriteString(allItem + "\n")
	
	// List groups
	for _, group := range m.groups {
		count := groupCounts[group]
		display := fmt.Sprintf("  %s (%d)", group, count)
		
		if m.selectedGroup == group {
			display = fmt.Sprintf("▶ %s (%d)", group, count)
			content.WriteString(m.styles.SelectedItem.Width(width-4).Render(display))
		} else {
			content.WriteString(m.styles.UnselectedItem.Width(width-4).Render(display))
		}
		content.WriteString("\n")
	}
	
	// Apply pane style
	paneContent := content.String()
	return m.styles.GroupPane.
		Width(width).
		Height(height).
		Render(paneContent)
}

// viewProfilePane renders the profile list
func (m *Model) viewProfilePane(width, height int) string {
	var content strings.Builder
	
	// Header
	content.WriteString(m.styles.Subtitle.Render("Profiles\n"))
	content.WriteString(strings.Repeat("─", width-2) + "\n")
	
	// Calculate visible profiles
	visibleHeight := height - 4 // Account for header and padding
	startIdx := 0
	
	// Ensure selected item is visible (simple scrolling)
	if m.selectedIndex >= visibleHeight {
		startIdx = m.selectedIndex - visibleHeight + 1
	}
	
	endIdx := startIdx + visibleHeight
	if endIdx > len(m.profiles) {
		endIdx = len(m.profiles)
	}
	
	// Render visible profiles
	for i := startIdx; i < endIdx; i++ {
		p := m.profiles[i]
		
		// Build profile display
		display := p.Name
		
		// Add indicators
		icons := ""
		if p.Favorite {
			icons += m.styles.FavoriteIcon.Render() + " "
		}
		if m.multiSelect[p.Name] {
			icons = "[✓] " + icons
		} else {
			icons = "[ ] " + icons
		}
		
		display = icons + display
		
		// Apply selection style
		if i == m.selectedIndex {
			content.WriteString(m.styles.SelectedItem.Width(width-4).Render("▶ " + display))
		} else {
			content.WriteString(m.styles.UnselectedItem.Width(width-4).Render("  " + display))
		}
		content.WriteString("\n")
	}
	
	// Show scroll indicators
	if startIdx > 0 {
		scrollUp := m.styles.Subtitle.Render(fmt.Sprintf("  ↑ %d more", startIdx))
		content.WriteString(scrollUp + "\n")
	}
	if endIdx < len(m.profiles) {
		scrollDown := m.styles.Subtitle.Render(fmt.Sprintf("  ↓ %d more", len(m.profiles)-endIdx))
		content.WriteString(scrollDown + "\n")
	}
	
	// Apply pane style
	return m.styles.ProfilePane.
		Width(width).
		Height(height).
		Render(content.String())
}

// viewDetailPane renders the profile details
func (m *Model) viewDetailPane(width, height int) string {
	var content strings.Builder
	
	content.WriteString(m.styles.Subtitle.Render("Details\n"))
	content.WriteString(strings.Repeat("─", width-2) + "\n")
	
	if m.selectedProfile == nil {
		content.WriteString(m.styles.Subtitle.Render("\nNo profile selected\n"))
		return m.styles.DetailPane.
			Width(width).
			Height(height).
			Render(content.String())
	}
	
	p := m.selectedProfile
	
	// Profile details
	details := []struct {
		label string
		value string
	}{
		{"Name", p.Name},
		{"Host", p.Host},
		{"Port", fmt.Sprintf("%d", effectivePort(*p))},
		{"User", p.Username},
		{"Protocol", string(p.Protocol)},
		{"Group", p.Group},
	}
	
	for _, d := range details {
		if d.value == "" && d.label != "Group" {
			continue
		}
		if d.value == "" {
			d.value = "default"
		}
		line := m.styles.Label.Render(d.label+":") + m.styles.Value.Render(d.value)
		content.WriteString(line + "\n")
	}
	
	// Usage stats
	content.WriteString("\n")
	if !p.LastUsed.IsZero() {
		lastUsed := formatDuration(time.Since(p.LastUsed))
		content.WriteString(m.styles.Label.Render("Last Used:") + m.styles.Value.Render(lastUsed) + "\n")
	}
	content.WriteString(m.styles.Label.Render("Use Count:") + m.styles.Value.Render(fmt.Sprintf("%d", p.UseCount)) + "\n")
	
	// Tags
	if len(p.Tags) > 0 {
		content.WriteString("\n")
		tags := make([]string, len(p.Tags))
		for i, tag := range p.Tags {
			tags[i] = "[" + tag + "]"
		}
		content.WriteString(m.styles.Label.Render("Tags:") + m.styles.Value.Render(strings.Join(tags, " ")) + "\n")
	}
	
	// ProxyJump
	if p.ProxyJump != "" {
		content.WriteString(m.styles.Label.Render("ProxyJump:") + m.styles.Value.Render(p.ProxyJump) + "\n")
	}
	
	// Description
	if p.Description != "" {
		content.WriteString("\n")
		content.WriteString(m.styles.Label.Render("Description:") + "\n")
		content.WriteString(m.styles.Value.Render(p.Description) + "\n")
	}
	
	// Action buttons
	content.WriteString("\n\n")
	buttons := []string{
		m.styles.Button.Render("[Enter] Connect"),
		m.styles.Button.Render("[e] Edit"),
		m.styles.Button.Render("[c] Clone"),
		m.styles.Button.Render("[d] Delete"),
	}
	content.WriteString(strings.Join(buttons, ""))
	
	// Apply pane style
	return m.styles.DetailPane.
		Width(width).
		Height(height).
		Render(content.String())
}

// viewFooter renders the footer with key hints
func (m *Model) viewFooter() string {
	var hints []string
	
	if m.searchActive {
		hints = []string{
			"[Enter] Apply",
			"[Esc] Cancel",
		}
	} else if len(m.multiSelect) > 0 {
		count := len(m.multiSelect)
		hints = []string{
			fmt.Sprintf("%d selected", count),
			"[t] Tag all",
			"[g] Move to group",
			"[d] Delete selected",
			"[Esc] Clear selection",
		}
	} else {
		hints = []string{
			"[↑↓] Navigate",
			"[Enter] Connect",
			"[e] Edit",
			"[a] Add",
			"[d] Delete",
			"[/] Search",
			"[Space] Multi-select",
			"[?] Help",
			"[q] Quit",
		}
	}
	
	// Add status message if present
	footer := strings.Join(hints, "  ")
	if m.statusMessage != "" {
		footer = m.styles.Warning.Render(m.statusMessage) + "  " + footer
	}
	
	return m.styles.Footer.Width(m.width).Render(footer)
}

// viewSearchMode renders the search interface
func (m *Model) viewSearchMode() string {
	var content strings.Builder
	
	content.WriteString(m.viewHeader())
	content.WriteString("\n\n")
	
	// Search box
	searchBox := m.styles.SearchBox.Width(60).Render(
		m.styles.Title.Render("Search\n") +
			m.searchInput.View() + "\n\n" +
			m.styles.Subtitle.Render("Enter search terms, use @ for user, # for tags, : for port"),
	)
	
	// Center the search box
	centeredBox := lipgloss.Place(
		m.width,
		m.height-6,
		lipgloss.Center,
		lipgloss.Center,
		searchBox,
	)
	
	content.WriteString(centeredBox)
	content.WriteString("\n")
	content.WriteString(m.viewFooter())
	
	return content.String()
}

// viewEditForm renders the profile edit form
func (m *Model) viewEditForm() string {
	if m.editForm == nil {
		return "Error: No edit form"
	}
	
	var content strings.Builder
	
	title := "Edit Profile"
	if m.mode == viewAdd {
		title = "Add Profile"
	}
	
	content.WriteString(m.styles.Title.Render(title) + "\n\n")
	
	// Form fields
	fields := []string{
		"Name", "Host", "Port", "Username",
		"Identity File", "ProxyJump", "Group", "Tags",
		"Description",
	}
	
	for i, field := range fields {
		label := m.styles.Label.Render(field + ":")
		
		inputStyle := m.styles.Value
		if i == m.editForm.focusIndex {
			inputStyle = m.styles.ActiveButton
		}
		
		input := inputStyle.Render(m.editForm.inputs[i].View())
		content.WriteString(label + " " + input + "\n")
		
		// Add spacing for readability
		if i == 3 || i == 5 || i == 7 {
			content.WriteString("\n")
		}
	}
	
	// Error message
	if m.editForm.errorMessage != "" {
		content.WriteString("\n" + m.styles.Error.Render(m.editForm.errorMessage) + "\n")
	}
	
	// Action buttons
	content.WriteString("\n\n")
	buttons := []string{
		m.styles.Button.Render("[Ctrl+S] Save"),
		m.styles.Button.Render("[T] Test Connection"),
		m.styles.Button.Render("[Esc] Cancel"),
	}
	content.WriteString(strings.Join(buttons, ""))
	
	// Center the form
	form := m.styles.DetailPane.Width(80).Render(content.String())
	centeredForm := lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		form,
	)
	
	return centeredForm
}

// Helper function to format duration
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		return fmt.Sprintf("%d minutes ago", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%d hours ago", int(d.Hours()))
	}
	days := int(d.Hours() / 24)
	if days == 1 {
		return "1 day ago"
	}
	return fmt.Sprintf("%d days ago", days)
}

