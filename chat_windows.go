package main

import (
	"fmt"
	"strings"
	"unicode"

	"gioui.org/font"
	"gioui.org/io/key"
	"gioui.org/layout"
)

type studyAttachment struct {
	label, text string
}

type generationDraft struct {
	prompt, file, folder string
	card                 studyAttachment
}

type studyTurn struct {
	prompt  string
	lesson  Lesson
	boardID string
}

func (s *studio) currentDraft() generationDraft {
	return generationDraft{
		prompt: strings.TrimSpace(s.editor.Text()), file: s.note,
		folder: s.foldersUI.destination, card: s.cardSource,
	}
}

func (s *studio) studyFromCard(l Lesson, i int, command string) {
	s.flushReaderNote()
	card := l.Cards[i]
	attachment := studyAttachment{
		label: fmt.Sprintf("Card %d · %s", i+1, card.Title),
		text:  fmt.Sprintf("Lição: %s\nCard: %s\n%s\n%s", l.Title, card.Title, card.Body, card.Question),
	}
	if note := s.lib.progress(l).Notes[i]; note != "" {
		attachment.label += " + anotação"
		attachment.text += "\n\nAnotação pessoal:\n" + note
	}
	s.note, s.cardSource = "", attachment
	s.generationError, s.showCommands = "", false
	s.editor.SetText(command + " " + l.Title + " · " + card.Title)
	s.editor.SetCaret(s.editor.Len(), s.editor.Len())
	s.goTo("home")
}

func (s *studio) commandMenu(g ctx) dim {
	if s.stop != nil {
		return dim{}
	}
	draft := strings.TrimLeftFunc(s.editor.Text(), unicode.IsSpace)
	prefix := ""
	if strings.HasPrefix(draft, "/") && strings.IndexFunc(draft, unicode.IsSpace) < 0 {
		prefix = strings.ToLower(draft)
	} else if !s.showCommands {
		return dim{}
	}
	var options []view
	for _, command := range studyCommands {
		if prefix != "" && !s.showCommands && !strings.HasPrefix(command.name, prefix) {
			continue
		}
		options = append(options, wide(s.commandOption(command)))
	}
	if len(options) == 0 {
		return padding(8, 0, 0, s.label("Comando desconhecido. Clique em / para ver os disponíveis.", 13, muted, font.Normal))(g)
	}
	return padding(8, 0, 0, func(g ctx) dim {
		return box(g, panel, line, 11, wide(inset(4, func(g ctx) dim { return column(g, options...) })))
	})(g)
}

func (s *studio) commandOption(command studyCommand) view {
	return func(g ctx) dim {
		return s.control(g, "command:"+command.name, command.name+" · "+command.description, quietButton, func() {
			subject := strings.TrimSpace(s.editor.Text())
			if strings.HasPrefix(subject, "/") {
				if space := strings.IndexFunc(subject, unicode.IsSpace); space >= 0 {
					subject = strings.TrimSpace(subject[space:])
				} else {
					subject = ""
				}
			}
			s.editor.SetText(command.name + " " + subject)
			s.editor.SetCaret(s.editor.Len(), s.editor.Len())
			s.generationError, s.showCommands = "", false
			g.Execute(key.FocusCmd{Tag: &s.editor})
		}, wide(padding(7, 9, 7, func(g ctx) dim {
			return (layout.Flex{Alignment: layout.Middle}).Layout(g,
				layout.Rigid(fixedWidth(135, s.label(command.name, 14, accent, font.SemiBold))),
				layout.Flexed(1, s.label(command.description, 13, muted, font.Normal)),
			)
		})))
	}
}

func (s *studio) chatHistory() []view {
	if len(s.chat) == 0 {
		return nil
	}
	items := []view{s.label("Nesta conversa", 20, ink, font.SemiBold), gap(10)}
	for _, turn := range s.chat {
		if turn.boardID != "" {
			if b := s.lib.board(turn.boardID); b != nil {
				items = append(items, s.label(turn.prompt, 14, muted, font.SemiBold), gap(5), s.boardTile(*b), gap(15))
			}
			continue
		}
		items = append(items, s.label(turn.prompt, 14, muted, font.SemiBold), gap(5),
			s.lessonTile(turn.lesson, false), gap(15))
	}
	return append(items, gap(10))
}
