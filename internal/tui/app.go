package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/danjdewhurst/jot-cli/internal/context"
	"github.com/danjdewhurst/jot-cli/internal/model"
	"github.com/danjdewhurst/jot-cli/internal/store"
	"github.com/danjdewhurst/jot-cli/internal/tui/views"
)

type viewID int

const (
	viewList viewID = iota
	viewDetail
	viewCompose
	viewHelp
)

type App struct {
	store         *store.Store
	width         int
	height        int
	view          viewID
	viewStack     []viewID
	contextFilter bool

	list    views.ListView
	detail  views.DetailView
	compose views.ComposeView
	help    views.HelpView

	statusMsg string
}

type notesLoadedMsg struct {
	notes []model.Note
}

type noteCreatedMsg struct {
	note model.Note
}

type noteUpdatedMsg struct {
	note model.Note
}

type noteArchivedMsg struct {
	id string
}

type bulkArchivedMsg struct {
	count int
}

type bulkDeletedMsg struct {
	count int
}

type bulkPinnedMsg struct {
	count  int
	pinned bool
}

type notePinnedMsg struct {
	id     string
	pinned bool
}

type backlinksLoadedMsg struct {
	backlinks []model.Note
}

type searchResultsMsg struct {
	results []store.SearchResult
}

type statusMsg string

type clearStatusMsg struct{}

// clearStatusAfter returns a tea.Cmd that clears the status message after a delay.
func clearStatusAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(_ time.Time) tea.Msg {
		return clearStatusMsg{}
	})
}

func loadNotes(s *store.Store, contextFilter bool) tea.Cmd {
	return func() tea.Msg {
		filter := model.NoteFilter{}
		if contextFilter {
			if folder, err := context.DetectFolder(); err == nil && folder != "" {
				filter.Tags = append(filter.Tags, model.Tag{Key: "folder", Value: folder})
			}
			if repo, err := context.DetectRepo(); err == nil && repo != "" {
				filter.Tags = append(filter.Tags, model.Tag{Key: "git_repo", Value: repo})
			}
			if branch, err := context.DetectBranch(); err == nil && branch != "" {
				filter.Tags = append(filter.Tags, model.Tag{Key: "git_branch", Value: branch})
			}
		}
		notes, err := s.ListNotes(filter)
		if err != nil {
			return statusMsg(fmt.Sprintf("Error: %v", err))
		}
		return notesLoadedMsg{notes: notes}
	}
}

func newApp(s *store.Store) App {
	return App{
		store:   s,
		view:    viewList,
		list:    views.NewListView(),
		detail:  views.NewDetailView(),
		compose: views.NewComposeView(),
		help:    views.NewHelpView(),
	}
}

func (a App) Init() tea.Cmd {
	return loadNotes(a.store, a.contextFilter)
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.list.SetSize(msg.Width, msg.Height-2)
		a.detail.SetSize(msg.Width, msg.Height-2)
		a.compose.SetSize(msg.Width, msg.Height-2)
		return a, nil

	case tea.KeyMsg:
		return a.handleGlobalKeys(msg)

	case notesLoadedMsg:
		a.list.SetNotes(msg.notes)
		return a, nil

	case noteCreatedMsg:
		a.statusMsg = fmt.Sprintf("Created: %s", msg.note.Title)
		a.popView()
		return a, tea.Batch(loadNotes(a.store, a.contextFilter), clearStatusAfter(3*time.Second))

	case noteUpdatedMsg:
		a.statusMsg = fmt.Sprintf("Updated: %s", msg.note.Title)
		a.popView()
		return a, tea.Batch(loadNotes(a.store, a.contextFilter), clearStatusAfter(3*time.Second))

	case noteArchivedMsg:
		a.statusMsg = "Note archived"
		return a, tea.Batch(loadNotes(a.store, a.contextFilter), clearStatusAfter(3*time.Second))

	case bulkArchivedMsg:
		a.list.ClearSelection()
		a.statusMsg = fmt.Sprintf("Archived %d notes", msg.count)
		return a, tea.Batch(loadNotes(a.store, a.contextFilter), clearStatusAfter(3*time.Second))

	case bulkDeletedMsg:
		a.list.ClearSelection()
		a.statusMsg = fmt.Sprintf("Deleted %d notes", msg.count)
		return a, tea.Batch(loadNotes(a.store, a.contextFilter), clearStatusAfter(3*time.Second))

	case bulkPinnedMsg:
		a.list.ClearSelection()
		if msg.pinned {
			a.statusMsg = fmt.Sprintf("Pinned %d notes", msg.count)
		} else {
			a.statusMsg = fmt.Sprintf("Unpinned %d notes", msg.count)
		}
		return a, tea.Batch(loadNotes(a.store, a.contextFilter), clearStatusAfter(3*time.Second))

	case notePinnedMsg:
		if msg.pinned {
			a.statusMsg = "Note pinned"
		} else {
			a.statusMsg = "Note unpinned"
		}
		return a, tea.Batch(loadNotes(a.store, a.contextFilter), clearStatusAfter(3*time.Second))

	case backlinksLoadedMsg:
		a.detail.SetBacklinks(msg.backlinks)
		return a, nil

	case views.SearchTickMsg:
		return a.handleSearchTick(msg)

	case searchResultsMsg:
		var notes []model.Note
		for _, r := range msg.results {
			notes = append(notes, r.Note)
		}
		a.list.SetSearchResults(notes)
		return a, nil

	case statusMsg:
		a.statusMsg = string(msg)
		return a, clearStatusAfter(3 * time.Second)

	case clearStatusMsg:
		a.statusMsg = ""
		return a, nil
	}

	// Delegate to current view
	return a.updateCurrentView(msg)
}

func (a *App) handleSearchTick(msg views.SearchTickMsg) (tea.Model, tea.Cmd) {
	if !a.list.IsSearching() {
		return a, nil
	}
	// Only search if query hasn't changed since the tick was emitted
	if msg.Query != a.list.SearchQuery() {
		return a, nil
	}
	if msg.Query == "" {
		return a, nil
	}
	s := a.store
	query := msg.Query
	return a, func() tea.Msg {
		results, err := s.Search(query, nil)
		if err != nil {
			return statusMsg(fmt.Sprintf("Search error: %v", err))
		}
		return searchResultsMsg{results: results}
	}
}

func (a App) handleGlobalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// When searching, only Escape is handled globally (by ListView itself)
	if a.view == viewList && a.list.IsSearching() {
		return a.updateCurrentView(msg)
	}

	switch {
	case key.Matches(msg, keys.Quit) && a.view == viewList:
		return a, tea.Quit

	case key.Matches(msg, keys.Back):
		if a.view != viewList {
			a.popView()
			return a, nil
		}
		return a, tea.Quit

	case key.Matches(msg, keys.Help) && a.view == viewList:
		a.pushView(viewHelp)
		return a, nil
	}

	return a.updateCurrentView(msg)
}

func (a *App) updateCurrentView(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch a.view {
	case viewList:
		return a.updateList(msg)
	case viewDetail:
		return a.updateDetail(msg)
	case viewCompose:
		return a.updateCompose(msg)
	}
	return a, nil
}

func (a *App) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	// When searching, delegate all keys to the list's search handler
	if a.list.IsSearching() {
		if kmsg, ok := msg.(tea.KeyMsg); ok {
			if key.Matches(kmsg, keys.Enter) {
				if note, ok := a.list.SelectedNote(); ok {
					a.list.ExitSearch()
					a.detail.SetNote(note)
					a.pushView(viewDetail)
					s := a.store
					noteID := note.ID
					return a, func() tea.Msg {
						backlinks, _ := s.ReferencesTo(noteID)
						return backlinksLoadedMsg{backlinks: backlinks}
					}
				}
				return a, nil
			}
		}
		cmd := a.list.Update(msg)
		return a, cmd
	}

	if kmsg, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(kmsg, keys.Select):
			a.list.ToggleSelection()
			return a, nil

		case key.Matches(kmsg, keys.SelectAll):
			a.list.SelectAll()
			return a, nil

		case key.Matches(kmsg, keys.Back) && a.list.HasSelection():
			a.list.ClearSelection()
			return a, nil

		case key.Matches(kmsg, keys.Archive):
			if a.list.HasSelection() {
				s := a.store
				ids := a.list.SelectedIDs()
				return a, func() tea.Msg {
					count, err := s.ArchiveNotes(ids)
					if err != nil {
						return statusMsg(fmt.Sprintf("Error: %v", err))
					}
					return bulkArchivedMsg{count: count}
				}
			}
			return a, nil

		case key.Matches(kmsg, keys.Enter):
			if note, ok := a.list.SelectedNote(); ok {
				a.detail.SetNote(note)
				a.pushView(viewDetail)
				s := a.store
				noteID := note.ID
				return a, func() tea.Msg {
					backlinks, _ := s.ReferencesTo(noteID)
					return backlinksLoadedMsg{backlinks: backlinks}
				}
			}
			return a, nil
		case key.Matches(kmsg, keys.New):
			a.compose.Reset()
			a.pushView(viewCompose)
			return a, nil
		case key.Matches(kmsg, keys.Delete):
			if a.list.HasSelection() {
				s := a.store
				ids := a.list.SelectedIDs()
				return a, func() tea.Msg {
					count, err := s.ArchiveNotes(ids)
					if err != nil {
						return statusMsg(fmt.Sprintf("Error: %v", err))
					}
					return bulkArchivedMsg{count: count}
				}
			}
			if note, ok := a.list.SelectedNote(); ok {
				s := a.store
				return a, func() tea.Msg {
					if err := s.ArchiveNote(note.ID); err != nil {
						return statusMsg(fmt.Sprintf("Error: %v", err))
					}
					return noteArchivedMsg{id: note.ID}
				}
			}
			return a, nil
		case key.Matches(kmsg, keys.Search):
			a.list.EnterSearch()
			return a, nil
		case key.Matches(kmsg, keys.Pin):
			if a.list.HasSelection() {
				s := a.store
				ids := a.list.SelectedIDs()
				return a, func() tea.Msg {
					count, err := s.PinNotes(ids)
					if err != nil {
						return statusMsg(fmt.Sprintf("Error: %v", err))
					}
					return bulkPinnedMsg{count: count, pinned: true}
				}
			}
			if note, ok := a.list.SelectedNote(); ok {
				s := a.store
				noteID := note.ID
				return a, func() tea.Msg {
					pinned, err := s.TogglePin(noteID)
					if err != nil {
						return statusMsg(fmt.Sprintf("Error: %v", err))
					}
					return notePinnedMsg{id: noteID, pinned: pinned}
				}
			}
			return a, nil
		case key.Matches(kmsg, keys.ContextFilter):
			a.contextFilter = !a.contextFilter
			a.list.SetContextFilter(a.contextFilter)
			if a.contextFilter {
				a.statusMsg = "Context filter: on"
			} else {
				a.statusMsg = "Context filter: off"
			}
			return a, tea.Batch(loadNotes(a.store, a.contextFilter), clearStatusAfter(3*time.Second))
		}
	}
	cmd := a.list.Update(msg)
	return a, cmd
}

func (a *App) updateDetail(msg tea.Msg) (tea.Model, tea.Cmd) {
	if kmsg, ok := msg.(tea.KeyMsg); ok {
		if key.Matches(kmsg, keys.Edit) {
			note := a.detail.Note()
			a.compose.SetNote(note)
			a.pushView(viewCompose)
			return a, nil
		}
	}
	a.detail.Update(msg)
	return a, nil
}

func (a *App) updateCompose(msg tea.Msg) (tea.Model, tea.Cmd) {
	if kmsg, ok := msg.(tea.KeyMsg); ok {
		if kmsg.String() == "ctrl+s" {
			title, body := a.compose.Content()
			s := a.store
			if id := a.compose.NoteID(); id != "" {
				return a, func() tea.Msg {
					note, err := s.UpdateNote(id, title, body)
					if err != nil {
						return statusMsg(fmt.Sprintf("Error: %v", err))
					}
					return noteUpdatedMsg{note: note}
				}
			}
			return a, func() tea.Msg {
				note, err := s.CreateNote(title, body, context.AutoTags())
				if err != nil {
					return statusMsg(fmt.Sprintf("Error: %v", err))
				}
				return noteCreatedMsg{note: note}
			}
		}
	}
	cmd := a.compose.Update(msg)
	return a, cmd
}

func (a App) View() string {
	var content string
	switch a.view {
	case viewList:
		content = a.list.View()
	case viewDetail:
		content = a.detail.View()
	case viewCompose:
		content = a.compose.View()
	case viewHelp:
		content = a.help.View()
	}

	status := a.renderStatusBar()
	return content + "\n" + status
}

func (a App) renderStatusBar() string {
	left := a.statusMsg
	right := ""
	switch a.view {
	case viewList:
		if a.list.IsSearching() {
			count, query := a.list.ResultCount()
			if query != "" {
				left = fmt.Sprintf("%d results for '%s'", count, query)
			}
			right = "esc:clear search  enter:open"
		} else if a.list.HasSelection() {
			right = "a:archive  d:archive  p:pin  space:toggle  ctrl+a:all  esc:clear"
		} else {
			right = "n:new  p:pin  /:filter  space:select  c:context  ?:help  q:quit"
		}
	case viewDetail:
		right = "e:edit  esc:back"
	case viewCompose:
		right = "ctrl+s:save  esc:cancel"
	case viewHelp:
		right = "esc:back"
	}

	gap := a.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}

	return statusBarStyle.Width(a.width).Render(
		left + fmt.Sprintf("%*s", gap, right),
	)
}

func (a *App) pushView(v viewID) {
	a.viewStack = append(a.viewStack, a.view)
	a.view = v
}

func (a *App) popView() {
	if len(a.viewStack) > 0 {
		a.view = a.viewStack[len(a.viewStack)-1]
		a.viewStack = a.viewStack[:len(a.viewStack)-1]
	}
}

func Run(s *store.Store) error {
	p := tea.NewProgram(newApp(s), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
