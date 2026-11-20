package main

import (
	"context"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestProviderEnvironmentAllowsSessionDirectoriesOnly(t *testing.T) {
	env := []string{
		"APPDATA=C:\\Users\\Test\\AppData", "path=C:\\Tools", "CODEX_HOME=C:\\Codex", "CLAUDE_CONFIG_DIR=C:\\Claude",
		"OPENAI_API_KEY=secret", "CODEX_API_KEY=secret", "CODEX_CI=1", "OPENAI_BASE_URL=https://example.net",
		"ANTHROPIC_API_KEY=secret", "ANTHROPIC_AUTH_TOKEN=secret", "CLAUDE_CODE_USE_BEDROCK=1",
		"ANTHROPIC_BASE_URL=https://example.net", "NODE_OPTIONS=--require custom.js",
	}
	if got := providerEnvironment(env); !reflect.DeepEqual(got, env[:4]) {
		t.Fatal("environment should keep installation and session paths only")
	}
}

func TestCodexLessonPayload(t *testing.T) {
	stream := []byte(`{"type":"thread.started"}
{"type":"item.completed","item":{"type":"reasoning","text":"Planejar a lição"}}
{"type":"item.completed","item":{"type":"agent_message","text":"{\"title\":\"Lição\"}"}}
{"type":"turn.completed"}`)
	got, err := codexLessonPayload(stream)
	if err != nil || string(got) != `{"title":"Lição"}` {
		t.Fatalf("got %q, %v", got, err)
	}
	stream = append(stream, []byte("\n"+`{"type":"item.started","item":{"type":"command_execution","command":"echo unsafe"}}`)...)
	if _, err := codexLessonPayload(stream); err == nil {
		t.Fatal("generation should reject output containing tool calls")
	}
}

func TestRestrictCodexCatalog(t *testing.T) {
	data, err := restrictCodexCatalog([]byte(`{"models":[{
  "slug":"sample","shell_type":"unified_exec","apply_patch_tool_type":"freeform","experimental_supported_tools":["example"]
}]}`))
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{`"slug":"sample"`, `"shell_type":"disabled"`, `"apply_patch_tool_type":null`, `"experimental_supported_tools":[]`} {
		if !strings.Contains(string(data), field) {
			t.Errorf("restricted catalog needs %s", field)
		}
	}
}

func TestClaudeLessonPayload(t *testing.T) {
	result := []byte(`{"is_error":false,"structured_output":{
  "title":"Lição","cards":[{"title":"Leitura"}],
  "exercise1":{"title":"Prática 1","question":"Primeira pergunta?"},
  "exercise2":{"title":"Prática 2","question":"Segunda pergunta?"}
}}`)
	lesson, isError, err := claudeLessonPayload(result)
	if isError || err != nil || len(lesson.Cards) != 3 || lesson.Cards[1].Question != "Primeira pergunta?" {
		t.Fatalf("did not join readings and exercises in order: %+v, %v", lesson, err)
	}
}

func TestProviderLive(t *testing.T) {
	provider := os.Getenv("OPENLOGOS_TEST_AI")
	if provider == "" {
		t.Skip("OPENLOGOS_TEST_AI enables one real request with the selected subscription")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if status, err := providerStatus(ctx, provider); err != nil {
		t.Fatal(err)
	} else {
		t.Log(status)
	}
	lesson, err := generateLesson(ctx, provider, "Por que ocorrem as fases da Lua?", "# Anotação\nA metade iluminada aponta para o Sol.")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%s gerou %d cards e %d referências em %q.", provider, len(lesson.Cards), len(lesson.Sources), lesson.Title)
}
