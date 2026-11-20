package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func fakeGoogleQuestionSearch(t *testing.T, handle http.HandlerFunc) googleQuestionSearch {
	t.Helper()
	t.Setenv(googleSearchProject, "openlogos-test")
	t.Setenv(googleSearchEngine, "questoes_123")
	t.Setenv(googleSearchKey, "fake-test-key")
	search, err := configuredGoogleQuestionSearch()
	if err != nil {
		t.Fatal(err)
	}
	if search.endpoint != "https://discoveryengine.googleapis.com/v1/" {
		t.Fatal("production search endpoint must be Google")
	}
	server := httptest.NewServer(handle)
	t.Cleanup(server.Close)
	search.endpoint = server.URL + "/v1/"
	return search
}

func TestGoogleQuestionSearchContract(t *testing.T) {
	search := fakeGoogleQuestionSearch(t, func(w http.ResponseWriter, r *http.Request) {
		want := "/v1/projects/openlogos-test/locations/global/collections/default_collection/engines/questoes_123/servingConfigs/default_search:searchLite"
		if r.Method != http.MethodPost || r.URL.Path != want || r.URL.RawQuery != "" ||
			r.Header.Get("X-Goog-Api-Key") != "fake-test-key" || r.Header.Get("Content-Type") != "application/json" {
			t.Error("unexpected Google URL, method or headers")
		}
		var body struct {
			Query             string `json:"query"`
			LanguageCode      string `json:"languageCode"`
			PageSize          int    `json:"pageSize"`
			ContentSearchSpec struct {
				SnippetSpec struct {
					ReturnSnippet bool `json:"returnSnippet"`
				} `json:"snippetSpec"`
			} `json:"contentSearchSpec"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Query != "ENEM frações" || body.LanguageCode != "pt-BR" ||
			body.PageSize != googleResultLimit || !body.ContentSearchSpec.SnippetSpec.ReturnSnippet {
			t.Errorf("invalid search payload: %+v (%v)", body, err)
		}
		fmt.Fprint(w, `{"results":[
			{"document":{"derivedStructData":{"link":"javascript:alert(1)","snippets":[{"snippet":"bad","snippet_status":"SUCCESS"}]}}},
			{"document":{"derivedStructData":{"link":"https://example.edu/prova?id=7&ano=2024","title":"Prova &amp; gabarito","snippets":[{"snippet":"<b>Frações</b>   equivalentes.","snippet_status":"SUCCESS"}]}}},
			{"document":{"derivedStructData":{"link":"https://example.edu/prova?id=7&ano=2024","snippets":[{"snippet":"duplicate","snippet_status":"SUCCESS"}]}}},
			{"document":{"derivedStructData":{"link":"https://example.edu/sem-trecho","snippets":[{"snippet":"none","snippet_status":"NO_SNIPPET_AVAILABLE"}]}}}
		]}`)
	})
	results, err := search.results(context.Background(), " ENEM   frações ")
	if err != nil || len(results) != 1 {
		t.Fatalf("got %d results, %v", len(results), err)
	}
	result := results[0]
	if result.URL != "https://example.edu/prova?id=7&ano=2024" || result.Title != "Prova & gabarito" ||
		len(result.snippets) != 1 || result.snippets[0] != "Frações equivalentes." {
		t.Fatalf("unexpected evidence: %+v", result)
	}
}

func TestGoogleQuestionSearchErrors(t *testing.T) {
	for _, tc := range []struct {
		status int
		body   string
		want   string
	}{
		{http.StatusOK, `{"results":[]}`, ""},
		{http.StatusOK, `null`, "resposta inválida"},
		{http.StatusOK, `{"results":[]}` + strings.Repeat(" ", googleResponseLimit), "resposta inválida"},
		{http.StatusBadRequest, "fake-test-key", "projeto e o ID"},
		{http.StatusForbidden, "fake-test-key", "recusou o acesso"},
		{http.StatusTooManyRequests, "fake-test-key", "cota"},
		{http.StatusBadGateway, "fake-test-key", "HTTP 502"},
		{http.StatusTemporaryRedirect, "", "HTTP 307"},
	} {
		t.Run(tc.want, func(t *testing.T) {
			search := fakeGoogleQuestionSearch(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/secret" {
					t.Error("Google redirect must not be followed")
				}
				w.Header().Set("Location", "/secret")
				w.WriteHeader(tc.status)
				fmt.Fprint(w, tc.body)
			})
			results, err := search.results(context.Background(), "frações")
			if tc.want == "" {
				if err != nil || len(results) != 0 {
					t.Fatalf("empty search: %v / %v", results, err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.want) || strings.Contains(err.Error(), "fake-test-key") {
				t.Fatalf("error does not match %q or leaks key: %v", tc.want, err)
			}
		})
	}
}

func TestValidQuestionURL(t *testing.T) {
	for _, raw := range []string{
		"https://www.gov.br/inep/pt-br/areas-de-atuacao/avaliacao-e-exames-educacionais/enem",
		"https://www.fuvest.br/provas/?ano=2020#resposta",
		"http://acervo.example.edu/documento",
	} {
		if !validQuestionURL(raw) {
			t.Errorf("rejected source URL %q", raw)
		}
	}
	for _, raw := range []string{
		"javascript:alert(1)", "file:///C:/arquivo", "//example.edu/provas", "https://user:secret@example.edu",
		"https://127.0.0.1", "https://127.0.0.01", "https://localhost", "https://host.local",
		"https://example.edu:8080", "https://example.edu/<img>", "https://example.edu/x\nhttps://evil.example",
	} {
		if validQuestionURL(raw) {
			t.Errorf("accepted unsafe source URL %q", raw)
		}
	}
}
