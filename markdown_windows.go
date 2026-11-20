package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/sys/windows"
)

const maxStudyMarkdown = 20_000

func studiesDir() (string, error) {
	dir := strings.TrimSpace(os.Getenv("OPENLOGOS_STUDIES_DIR"))
	if dir == "" {
		documents, err := windows.KnownFolderPath(windows.FOLDERID_Documents, windows.KF_FLAG_CREATE)
		if err != nil {
			return "", fmt.Errorf("Não foi possível localizar a pasta Documentos: %w", err)
		}
		dir = filepath.Join(documents, "OpenLogos", "Estudos")
	}
	dir, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("Caminho inválido para a pasta Estudos: %w", err)
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("Não foi possível criar a pasta Estudos: %w", err)
	}
	return dir, nil
}

func exportLessonMarkdown(dir string, l Lesson) (string, error) {
	if err := validateLesson(l); err != nil {
		return "", err
	}

	var markdown, answers strings.Builder
	fmt.Fprintf(&markdown, "# %s\n\n%s\n\n%s\n", l.Title, l.Topic, l.Summary)
	for index, card := range l.Cards {
		fmt.Fprintf(&markdown, "\n## %d. %s\n\n", index+1, card.Title)
		if card.Body != "" {
			fmt.Fprintf(&markdown, "%s\n\n", card.Body)
		}
		if card.Question == "" {
			continue
		}
		fmt.Fprintf(&markdown, "%s\n\n", card.Question)
		for i, option := range card.Options {
			fmt.Fprintf(&markdown, "- %c. %s\n", 'A'+i, option)
		}
		fmt.Fprintf(&answers, "### %d. %s\n\n**Resposta %c.** %s\n\n%s\n\n", index+1, card.Title, 'A'+card.Correct, card.Options[card.Correct], card.Explanation)
	}
	fmt.Fprintf(&markdown, "\n## Gabarito\n\n%s## Bibliografia\n\n", answers.String())
	for _, source := range l.Sources {
		fmt.Fprintf(&markdown, "- %s\n", source)
	}

	stem := "Estudo-" + strings.Join(strings.FieldsFunc(l.Title, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}), "-")
	if chars := []rune(stem); len(chars) > 65 {
		stem = string(chars[:65])
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return "", fmt.Errorf("Não foi possível abrir a pasta Estudos: %w", err)
	}
	defer root.Close()
	name := stem + ".md"
	for copy := 2; ; copy++ {
		file, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if errors.Is(err, os.ErrExist) {
			name = fmt.Sprintf("%s-%d.md", stem, copy)
			continue
		}
		if err != nil {
			return "", fmt.Errorf("Não foi possível criar o Markdown: %w", err)
		}
		_, writeErr := file.WriteString(markdown.String())
		if err := errors.Join(writeErr, file.Close()); err != nil {
			return "", fmt.Errorf("Não foi possível salvar o Markdown: %w", errors.Join(err, root.Remove(name)))
		}
		return filepath.Join(dir, name), nil
	}
}

func listStudyMarkdown(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("Não foi possível listar a pasta Estudos: %w", err)
	}
	var names []string
	for _, entry := range entries {
		if entry.Type().IsRegular() && isStudyMarkdownName(entry.Name()) {
			names = append(names, entry.Name())
		}
	}
	return names, nil
}

func readStudyMarkdown(dir, name string) (string, error) {
	if !isStudyMarkdownName(name) {
		return "", errors.New("Escolha um arquivo .md diretamente na pasta Estudos.")
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return "", fmt.Errorf("Não foi possível abrir a pasta Estudos: %w", err)
	}
	defer root.Close()
	info, err := root.Lstat(name)
	if err != nil {
		return "", fmt.Errorf("Não foi possível localizar o Markdown: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", errors.New("Escolha um arquivo Markdown comum na pasta Estudos.")
	}
	file, err := root.Open(name)
	if err != nil {
		return "", fmt.Errorf("Não foi possível abrir o Markdown: %w", err)
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !opened.Mode().IsRegular() || !os.SameFile(info, opened) {
		return "", errors.New("O Markdown mudou enquanto era aberto. Selecione o arquivo novamente.")
	}
	data, err := io.ReadAll(io.LimitReader(file, maxStudyMarkdown*utf8.UTFMax+4))
	if err != nil {
		return "", fmt.Errorf("Não foi possível ler o Markdown: %w", err)
	}
	if !utf8.Valid(data) {
		return "", errors.New("Salve o Markdown com codificação UTF-8 e tente novamente.")
	}
	content := strings.TrimPrefix(string(data), "\uFEFF")
	if utf8.RuneCountInString(content) > maxStudyMarkdown {
		return "", errors.New("O Markdown pode ter até 20.000 caracteres.")
	}
	if strings.TrimSpace(content) == "" {
		return "", errors.New("Adicione conteúdo ao Markdown antes de usá-lo como referência.")
	}
	return content, nil
}

func isStudyMarkdownName(name string) bool {
	return filepath.IsLocal(name) && name == filepath.Base(name) && strings.EqualFold(filepath.Ext(name), ".md")
}
