package main

import (
	"crypto/rand"
	"errors"
	"slices"
	"strings"
	"unicode"
)

const folderNameLimit = 50

type StudyFolder struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Lessons []string `json:"lessons,omitempty"`
}

func validFolderName(name string) bool {
	return validText(name, folderNameLimit) && strings.IndexFunc(name, unicode.IsControl) == -1
}

func (l *Library) restoreFolders(saved []StudyFolder) error {
	ids, names, lessons := map[string]bool{}, map[string]bool{}, map[string]bool{}
	invalid := errors.New("pastas de estudo inválidas na biblioteca local")
	for _, folder := range saved {
		name := strings.ToLower(folder.Name)
		if !validText(folder.ID, 64) || ids[folder.ID] || names[name] ||
			folder.Name != strings.TrimSpace(folder.Name) || !validFolderName(folder.Name) {
			return invalid
		}
		ids[folder.ID], names[name] = true, true
		for _, id := range folder.Lessons {
			if id == "" || lessons[id] {
				return invalid
			}
			lessons[id] = true
		}
	}
	l.Folders = saved
	return nil
}

func (l *Library) folder(id string) *StudyFolder {
	for i := range l.Folders {
		if l.Folders[i].ID == id {
			return &l.Folders[i]
		}
	}
	return nil
}

func (l *Library) nameFolder(id, value string) (string, error) {
	name := strings.TrimSpace(value)
	if !validFolderName(name) {
		return "", errors.New("Digite um nome com até 50 caracteres.")
	}
	folded := strings.ToLower(name)
	for _, folder := range l.Folders {
		if folder.ID != id && strings.ToLower(folder.Name) == folded {
			return "", errors.New("Já existe uma pasta com esse nome.")
		}
	}
	if id == "" {
		id = "folder-" + rand.Text()
		l.Folders = append(l.Folders, StudyFolder{ID: id, Name: name})
		return id, nil
	}
	folder := l.folder(id)
	if folder == nil {
		return "", errors.New("Pasta de estudo indisponível.")
	}
	folder.Name = name
	return id, nil
}

func (l *Library) folderFor(lessonID string) *StudyFolder {
	for i := range l.Folders {
		if slices.Contains(l.Folders[i].Lessons, lessonID) {
			return &l.Folders[i]
		}
	}
	return nil
}

func (l *Library) moveLesson(lessonID, folderID string) error {
	if lessonID == "" {
		return errors.New("Lição indisponível.")
	}
	destination := l.folder(folderID)
	if folderID != "" && destination == nil {
		return errors.New("Pasta de destino indisponível.")
	}
	previous := l.folderFor(lessonID)
	if previous == destination {
		return nil
	}
	if previous != nil {
		index := slices.Index(previous.Lessons, lessonID)
		previous.Lessons = slices.Delete(previous.Lessons, index, index+1)
	}
	if destination != nil {
		destination.Lessons = append(destination.Lessons, lessonID)
	}
	return nil
}
