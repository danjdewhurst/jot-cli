package editor

import "testing"

func TestEditorCmd(t *testing.T) {
	t.Run("override takes precedence", func(t *testing.T) {
		t.Setenv("VISUAL", "emacs")
		t.Setenv("EDITOR", "nano")
		got := editorCmd("nvim")
		if got != "nvim" {
			t.Errorf("editorCmd(\"nvim\") = %q, want %q", got, "nvim")
		}
	})

	t.Run("falls back to VISUAL", func(t *testing.T) {
		t.Setenv("VISUAL", "emacs")
		t.Setenv("EDITOR", "nano")
		got := editorCmd("")
		if got != "emacs" {
			t.Errorf("editorCmd(\"\") = %q, want %q", got, "emacs")
		}
	})

	t.Run("falls back to EDITOR", func(t *testing.T) {
		t.Setenv("VISUAL", "")
		t.Setenv("EDITOR", "nano")
		got := editorCmd("")
		if got != "nano" {
			t.Errorf("editorCmd(\"\") = %q, want %q", got, "nano")
		}
	})

	t.Run("falls back to vi", func(t *testing.T) {
		t.Setenv("VISUAL", "")
		t.Setenv("EDITOR", "")
		got := editorCmd("")
		if got != "vi" {
			t.Errorf("editorCmd(\"\") = %q, want %q", got, "vi")
		}
	})
}
