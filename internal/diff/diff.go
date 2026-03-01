// Package diff provides simple line-based diff utilities for note history.
package diff

import (
	"fmt"
	"strings"
)

// Op represents a diff operation.
type Op int

const (
	OpEqual  Op = iota
	OpAdd       // line added
	OpRemove    // line removed
)

// Line represents a single line in a diff.
type Line struct {
	Op   Op
	Text string
}

// Lines computes a simple line-based diff between old and new text using the
// longest common subsequence algorithm.
func Lines(old, new string) []Line {
	oldLines := splitLines(old)
	newLines := splitLines(new)

	lcs := longestCommonSubsequence(oldLines, newLines)

	var result []Line
	oi, ni, li := 0, 0, 0

	for li < len(lcs) {
		// Emit removals from old until we reach the LCS line
		for oi < len(oldLines) && oldLines[oi] != lcs[li] {
			result = append(result, Line{Op: OpRemove, Text: oldLines[oi]})
			oi++
		}
		// Emit additions from new until we reach the LCS line
		for ni < len(newLines) && newLines[ni] != lcs[li] {
			result = append(result, Line{Op: OpAdd, Text: newLines[ni]})
			ni++
		}
		// Emit the common line
		result = append(result, Line{Op: OpEqual, Text: lcs[li]})
		oi++
		ni++
		li++
	}

	// Remaining lines
	for oi < len(oldLines) {
		result = append(result, Line{Op: OpRemove, Text: oldLines[oi]})
		oi++
	}
	for ni < len(newLines) {
		result = append(result, Line{Op: OpAdd, Text: newLines[ni]})
		ni++
	}

	return result
}

// Summary returns a short diff summary like "+3 / -1".
func Summary(old, new string) string {
	lines := Lines(old, new)
	var added, removed int
	for _, l := range lines {
		switch l.Op {
		case OpAdd:
			added++
		case OpRemove:
			removed++
		}
	}
	if added == 0 && removed == 0 {
		return "no changes"
	}
	return fmt.Sprintf("+%d / -%d", added, removed)
}

// Format returns a human-readable unified-style diff string.
func Format(old, new string) string {
	lines := Lines(old, new)
	var b strings.Builder
	for _, l := range lines {
		switch l.Op {
		case OpAdd:
			fmt.Fprintf(&b, "+ %s\n", l.Text)
		case OpRemove:
			fmt.Fprintf(&b, "- %s\n", l.Text)
		case OpEqual:
			fmt.Fprintf(&b, "  %s\n", l.Text)
		}
	}
	return b.String()
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

func longestCommonSubsequence(a, b []string) []string {
	m, n := len(a), len(b)
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}

	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			switch {
			case a[i-1] == b[j-1]:
				dp[i][j] = dp[i-1][j-1] + 1
			case dp[i-1][j] >= dp[i][j-1]:
				dp[i][j] = dp[i-1][j]
			default:
				dp[i][j] = dp[i][j-1]
			}
		}
	}

	// Backtrack to find the LCS
	lcs := make([]string, dp[m][n])
	i, j, k := m, n, dp[m][n]-1
	for i > 0 && j > 0 {
		switch {
		case a[i-1] == b[j-1]:
			lcs[k] = a[i-1]
			i--
			j--
			k--
		case dp[i-1][j] >= dp[i][j-1]:
			i--
		default:
			j--
		}
	}

	return lcs
}
