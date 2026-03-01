# Plan: Live incremental search in TUI

**Priority:** Medium effort, high value

## Summary

Replace the current explicit search command in the TUI with a filter-as-you-type experience. Results update live as the user types.

## Behaviour

1. User presses `/` to enter search mode
2. A text input appears at the top/bottom of the list view
3. As they type, the note list filters in real-time using FTS5
4. Pressing `Enter` selects the highlighted result
5. Pressing `Escape` clears the search and returns to the full list
6. Debounce queries (e.g. 150ms after last keystroke) to avoid hammering SQLite

## Implementation

### TUI changes

- **SearchView refactor**: instead of a separate view, embed search as a mode within `ListView`
- Add `filterQuery` field to `ListView` state
- On each debounced keystroke, call `store.Search()` and replace the displayed notes
- Highlight matching terms in results (optional, defer if complex)

### Debouncing

- Use a `time.Timer` in the Bubbletea update loop
- On keypress: reset timer to 150ms
- On timer fire: send a custom `searchMsg` with the query
- On `searchMsg`: execute search and update list

### Store layer

No changes — `Search()` already returns ranked results.

### Considerations

- Empty query returns to normal `ListNotes()` view
- Maintain cursor position sensibly when results change
- Show result count in status bar ("12 results for 'golang'")
- FTS5 queries are fast enough for interactive use (sub-millisecond on typical note volumes)

## Complexity

~200 lines. Refactor within TUI layer only, no store or model changes.
