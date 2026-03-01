package model

import "testing"

func TestParseTag(t *testing.T) {
	tests := []struct {
		input   string
		wantKey string
		wantVal string
		wantErr bool
	}{
		{"folder:work", "folder", "work", false},
		{"git_repo:user/project", "git_repo", "user/project", false},
		{"invalid", "", "", true},
		{":value", "", "", true},
		{"key:", "", "", true},
	}

	for _, tt := range tests {
		tag, err := ParseTag(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseTag(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if err == nil {
			if tag.Key != tt.wantKey || tag.Value != tt.wantVal {
				t.Errorf("ParseTag(%q) = {%q, %q}, want {%q, %q}", tt.input, tag.Key, tag.Value, tt.wantKey, tt.wantVal)
			}
		}
	}
}

func TestTagString(t *testing.T) {
	tag := Tag{Key: "folder", Value: "work"}
	if s := tag.String(); s != "folder:work" {
		t.Errorf("String() = %q, want %q", s, "folder:work")
	}
}
