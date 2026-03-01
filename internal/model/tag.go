package model

import (
	"fmt"
	"strings"
)

type Tag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (t Tag) String() string {
	return t.Key + ":" + t.Value
}

func ParseTag(s string) (Tag, error) {
	key, value, ok := strings.Cut(s, ":")
	if !ok || key == "" || value == "" {
		return Tag{}, fmt.Errorf("invalid tag format %q, expected key:value", s)
	}
	return Tag{Key: key, Value: value}, nil
}
