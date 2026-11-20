package main

import (
	"strings"
	"unicode/utf8"
)

const maxCardNote = 2400

func (p *Progress) setCardNote(index, cards int, value string) bool {
	if index < 0 || index >= cards || utf8.RuneCountInString(value) > maxCardNote {
		return false
	}
	if strings.TrimSpace(value) == "" {
		value = ""
	}
	if p.Notes[index] == value {
		return false
	}
	if value == "" {
		delete(p.Notes, index)
	} else {
		if p.Notes == nil {
			p.Notes = make(map[int]string)
		}
		p.Notes[index] = value
	}
	return true
}

func notePreview(value string) string {
	remaining := 160
	value = strings.Join(strings.Fields(value), " ")
	for offset := range value {
		if remaining == 0 {
			return strings.TrimSpace(value[:offset]) + "…"
		}
		remaining--
	}
	return value
}
