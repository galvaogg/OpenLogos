package main

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"image/color"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"gioui.org/app"
	"gioui.org/font/gofont"
	"gioui.org/font/opentype"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"golang.org/x/sys/windows"
)

type result struct {
	provider string
	status   string
	lesson   Lesson
	draft    generationDraft
	board    Board
	boardJob *boardJob
	err      error
}

type connection struct {
	status   string
	checking bool
}

type studio struct {
	window             *app.Window
	theme              *material.Theme
	ctx                context.Context
	results            chan result
	work               sync.WaitGroup
	stop               context.CancelFunc
	path               string
	localWarning       string
	saveBlocked        bool
	lib                Library
	samples            []Lesson
	lesson             *Lesson
	page               string
	editor             widget.Editor
	clicks             map[string]*studioButton
	body, rail, detail layout.List
	resetBody          bool
	connections        map[string]connection
	generationError    string
	activeTopic        string
	studiesDir         string
	studyFiles         []string
	studyError         string
	note               string
	notice             string
	foldersUI          folderState
	readerState
	cardSource   studyAttachment
	showCommands bool
	chat         []studyTurn
	boardUI      boardState
}

func main() {
	go func() {
		if err := run(); err != nil {
			message, _ := syscall.UTF16PtrFromString("OpenLogos: " + err.Error())
			title, _ := syscall.UTF16PtrFromString("OpenLogos")
			syscall.NewLazyDLL("user32.dll").NewProc("MessageBoxW").Call(0, uintptr(unsafe.Pointer(message)), uintptr(unsafe.Pointer(title)), 0x10)
			os.Exit(1)
		}
		os.Exit(0)
	}()
	app.Main()
}

func run() error {
	var animate uint32
	const spiGetClientAreaAnimation = 0x1042
	ok, _, _ := syscall.NewLazyDLL("user32.dll").NewProc("SystemParametersInfoW").Call(spiGetClientAreaAnimation, 0, uintptr(unsafe.Pointer(&animate)), 0)
	reduceMotion = ok != 0 && animate == 0
	w := new(app.Window)
	w.Option(app.Title("OpenLogos"), app.Size(1260, 810), app.MinSize(750, 580))
	ctx, cancel := context.WithCancel(context.Background())
	path, pathErr := libraryPath()
	if pathErr == nil {
		name := fmt.Sprintf("Local\\OpenLogos-%x", sha256.Sum256([]byte(strings.ToLower(path))))
		mutex, err := windows.CreateMutex(nil, false, windows.StringToUTF16Ptr(name))
		if mutex != 0 {
			defer windows.CloseHandle(mutex)
		}
		if errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
			cancel()
			return errors.New("A biblioteca já está aberta em outra janela do OpenLogos.")
		}
		if err != nil {
			pathErr = err
		}
	}
	lib, loadErr := loadLibrary(path)
	s := studio{
		window: w, theme: studioTheme(), ctx: ctx, results: make(chan result, 8),
		path: path, lib: lib, samples: sampleLessons(), page: "home",
		editor: widget.Editor{SingleLine: true, Submit: true, MaxLen: 240},
		clicks: make(map[string]*studioButton), connections: make(map[string]connection),
		body: layout.List{Axis: layout.Vertical}, rail: layout.List{Axis: layout.Vertical},
		detail: layout.List{Axis: layout.Vertical},
	}
	defer func() {
		cancel()
		s.work.Wait()
	}()
	if pathErr != nil || loadErr != nil {
		s.saveBlocked = true
		s.localWarning = "A biblioteca local ficou indisponível. Alterações valem nesta sessão."
	}
	s.refreshStudies()
	s.checkProvider()
	var ops op.Ops
	for {
		switch e := w.Event().(type) {
		case app.Win32ViewEvent:
			if e.HWND != 0 {
				styleWindow(e.HWND)
			}
		case app.DestroyEvent:
			s.flushReaderNote()
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			s.tickReaderNote(gtx)
			s.collect()
			paint.Fill(gtx.Ops, canvas)
			s.layout(gtx)
			e.Frame(gtx.Ops)
		}
	}
}

func styleWindow(hwnd uintptr) {
	dwm := syscall.NewLazyDLL("dwmapi.dll").NewProc("DwmSetWindowAttribute")
	for attribute, value := range map[uintptr]uint32{
		20: 1,          // DWMWA_USE_IMMERSIVE_DARK_MODE
		34: 0x00292420, // DWMWA_BORDER_COLOR
		35: 0x00161311, // DWMWA_CAPTION_COLOR
		36: 0x00ece9e4, // DWMWA_TEXT_COLOR
	} {
		dwm.Call(hwnd, attribute, uintptr(unsafe.Pointer(&value)), 4)
	}
	user := syscall.NewLazyDLL("user32.dll")
	module, _, _ := syscall.NewLazyDLL("kernel32.dll").NewProc("GetModuleHandleW").Call(0)
	loadImage := user.NewProc("LoadImageW")
	postMessage := user.NewProc("PostMessageW")
	for kind, pixels := range []uintptr{16, 32} {
		const iconGroupID, imageIcon, lrShared, wmSetIcon = 101, 1, 0x8000, 0x80
		icon, _, _ := loadImage.Call(module, iconGroupID, imageIcon, pixels, pixels, lrShared)
		if icon != 0 {
			postMessage.Call(hwnd, wmSetIcon, uintptr(kind), icon, 0)
		}
	}
}

func studioTheme() *material.Theme {
	t := material.NewTheme()
	fonts := gofont.Collection()
	for _, name := range []string{"segoeui.ttf", "seguisb.ttf", "segoeuib.ttf", "georgia.ttf"} {
		data, err := os.ReadFile(filepath.Join(os.Getenv("WINDIR"), "Fonts", name))
		if err != nil {
			continue
		}
		system, err := opentype.ParseCollection(data)
		if err == nil {
			fonts = append(fonts, system...)
		}
	}
	t.Shaper = text.NewShaper(text.NoSystemFonts(), text.WithCollection(fonts))
	t.Fg, t.Bg = ink, canvas
	t.ContrastBg, t.ContrastFg = accent, color.NRGBA{A: 255}
	t.Face = "Segoe UI"
	return t
}

func (s *studio) send(r result) {
	if s.ctx.Err() != nil {
		return
	}
	select {
	case s.results <- r:
		s.window.Invalidate()
	case <-s.ctx.Done():
	}
}

func (s *studio) checkProvider() {
	p := s.lib.Provider
	if s.connections[p].checking {
		return
	}
	s.connections[p] = connection{status: "Verificando sessão…", checking: true}
	s.work.Add(1)
	go func() {
		defer s.work.Done()
		ctx, cancel := context.WithTimeout(s.ctx, 12*time.Second)
		defer cancel()
		status, err := providerStatus(ctx, p)
		s.send(result{provider: p, status: status, err: err})
	}()
}

func (s *studio) generate() {
	if s.stop != nil {
		return
	}
	draft := s.currentDraft()
	request, err := parseStudyRequest(draft.prompt)
	if err != nil {
		s.generationError = err.Error()
		return
	}
	p := s.lib.Provider
	var notes []string
	if draft.file != "" {
		markdown, err := readStudyMarkdown(s.studiesDir, draft.file)
		if err != nil {
			s.generationError = err.Error()
			return
		}
		notes = append(notes, markdown)
	}
	if draft.card.text != "" {
		notes = append(notes, draft.card.text)
	}
	if request.boardKind != "" {
		s.startBoard(request.boardKind, request.topic, strings.Join(notes, "\n\n"), nil, draft)
		return
	}
	s.generationError, s.activeTopic = "", request.topic
	s.showCommands = false
	ctx, cancel := context.WithTimeout(s.ctx, 3*time.Minute)
	s.stop = cancel
	s.work.Add(1)
	go func() {
		defer s.work.Done()
		lesson, err := generateLesson(ctx, p, request, notes...)
		if ctx.Err() != nil {
			err = ctx.Err()
		}
		if errors.Is(err, context.DeadlineExceeded) {
			err = errors.New("O agente demorou demais. Tente novamente.")
		}
		s.send(result{lesson: lesson, draft: draft, err: err})
	}()
}

func (s *studio) collect() {
	for {
		select {
		case r := <-s.results:
			if r.provider != "" {
				status := r.status
				if r.err != nil {
					status = r.err.Error()
				}
				s.connections[r.provider] = connection{status: status}
				continue
			}
			s.stop()
			s.stop, s.activeTopic = nil, ""
			if r.boardJob != nil {
				s.finishBoard(r.board, *r.boardJob, r.err)
				continue
			}
			if r.err == nil {
				r.err = s.lib.add(r.lesson)
			}
			if r.err != nil {
				if !errors.Is(r.err, context.Canceled) {
					s.generationError = r.err.Error()
				}
				continue
			}
			lesson := s.lib.Generated[0]
			if r.draft.folder != "" {
				_ = s.lib.moveLesson(lesson.ID, r.draft.folder)
			}
			s.chat = append([]studyTurn{{prompt: r.draft.prompt, lesson: lesson}}, s.chat...)
			s.chat = s.chat[:min(4, len(s.chat))]
			sameDraft := s.currentDraft() == r.draft
			if sameDraft {
				s.editor.SetText("")
				s.note = ""
				s.cardSource = studyAttachment{}
				s.foldersUI.destination = ""
			}
			if s.page == "home" && sameDraft {
				s.open(lesson)
			} else {
				s.save()
			}
		default:
			return
		}
	}
}

func (s *studio) refreshStudies() {
	folder, err := studiesDir()
	if err == nil {
		s.studiesDir = folder
		s.studyFiles, err = listStudyMarkdown(folder)
	}
	s.studyError = ""
	if err != nil {
		s.studyError = "Não foi possível ler a pasta Estudos."
	}
}

func (s *studio) openStudies() {
	s.refreshStudies()
	if s.studiesDir == "" {
		return
	}
	c := exec.Command(filepath.Join(os.Getenv("WINDIR"), "explorer.exe"), s.studiesDir)
	if err := c.Start(); err != nil {
		s.studyError = "Não foi possível abrir a pasta no Explorador."
		return
	}
	go func() { _ = c.Wait() }()
}

func (s *studio) export(l Lesson) {
	s.refreshStudies()
	if s.studiesDir == "" {
		return
	}
	file, err := exportLessonMarkdown(s.studiesDir, l)
	if err != nil {
		s.notice = err.Error()
		return
	}
	s.notice = "Markdown disponível em Estudos: " + filepath.Base(file)
	s.refreshStudies()
}

func (s *studio) save() {
	if s.saveBlocked {
		return
	}
	if err := saveLibrary(s.path, s.lib); err != nil {
		s.localWarning = "Falha ao salvar no dispositivo. Sua sessão ainda está aberta."
		return
	}
	s.localWarning = ""
}

func (s *studio) goTo(page string) {
	if s.page != page {
		if s.page == "reader" {
			s.flushReaderNote()
		}
		s.resetBody = true
	}
	s.page = page
}

func (s *studio) open(l Lesson) {
	s.notice = ""
	s.detail.Position = layout.Position{}
	s.lesson = &l
	s.goTo("reader")
	s.jump(s.lib.progress(l).Cursor)
}

func (s *studio) jump(index int) {
	l := *s.lesson
	p := s.lib.progress(l)
	p.Cursor = max(0, min(index, len(l.Cards)))
	s.loadReaderCard(l, p)
	if p.Cursor < len(l.Cards) {
		p.Visited[p.Cursor] = true
	} else {
		p.Reviewed = true
	}
	s.resetBody = true
	s.save()
}
