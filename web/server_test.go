package web

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"wca-competitor-list/generater"
)

// fakeGen stands in for generater.Gen and counts the calls per id.
type fakeGen struct {
	mu    sync.Mutex
	calls map[string]int
}

func (f *fakeGen) Generate(id string) (*generater.Result, error) {
	f.mu.Lock()
	f.calls[id]++
	f.mu.Unlock()
	if id != "TestOpen2026" {
		return nil, errors.New(`cannot get wcif of "` + id + `" (HTTP 404): check the competition id`)
	}
	row := func(id, wcaID, name string) []string { return []string{id, wcaID, name, "Male", "TH", "", ""} }
	return &generater.Result{
		CompetitionID: "TestOpen2026",
		Files: []generater.File{
			{Key: "first-timer", Label: "First-timer", Name: "TestOpen2026-check-in-first-timer.pdf",
				Rows: [][]string{row("1", "", "Anna Smith"), row("2", "", "Bea Brown")}},
			{Key: "returner", Label: "Returner", Name: "TestOpen2026-check-in-returner.pdf",
				Rows: [][]string{row("3", "2015SMIT01", "Carl Smith")}},
		},
		NameIssues: []generater.NameIssue{
			{RegistrantID: 4, Name: "Mono", Problem: "No surname", Fix: "Add the surname after the first name"},
			{RegistrantID: 5, Name: "<script>alert(1)</script> X", Problem: "Incorrect capitalization", Fix: "Fix it"},
		},
	}, nil
}

func newTestServer() (http.Handler, *fakeGen) {
	gen := &fakeGen{calls: map[string]int{}}
	return Handler(gen), gen
}

func get(t *testing.T, h http.Handler, url string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, url, nil))
	return rec
}

func assertContains(t *testing.T, body string, parts ...string) {
	t.Helper()
	for _, p := range parts {
		if !strings.Contains(body, p) {
			t.Errorf("page does not contain %q", p)
		}
	}
}

func TestIndexWithoutID(t *testing.T) {
	h, gen := newTestServer()
	rec := get(t, h, "/")
	if rec.Code != http.StatusOK || !strings.HasPrefix(rec.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("status %d, content type %q", rec.Code, rec.Header().Get("Content-Type"))
	}
	body := rec.Body.String()
	assertContains(t, body, `name="id"`, "Load competition", `data-set-theme="light"`, `data-set-theme="dark"`)
	if strings.Contains(body, "Download both") || len(gen.calls) != 0 {
		t.Error("a competition was loaded without an id")
	}
}

func TestIndexWithCompetition(t *testing.T) {
	h, _ := newTestServer()
	body := get(t, h, "/?id=TestOpen2026&refresh=1").Body.String()

	t.Run("both files with their counts and download links", func(t *testing.T) {
		assertContains(t, body,
			"TestOpen2026-check-in-first-timer.pdf", "TestOpen2026-check-in-returner.pdf",
			`2 <span`, `1 <span`,
			`href="/pdf?id=TestOpen2026&file=first-timer&download=1"`,
			`href="/pdf?id=TestOpen2026&file=returner&download=1"`,
			`href="/zip?id=TestOpen2026"`, `download="TestOpen2026-check-in.zip"`, "Download both")
	})

	t.Run("first file is previewed by default", func(t *testing.T) {
		assertContains(t, body, `<iframe src="/pdf?id=TestOpen2026&file=first-timer"`)
	})

	t.Run("wrong names are listed with the fix and the WCA link", func(t *testing.T) {
		assertContains(t, body,
			"2 first-timer name(s) must be fixed on the WCA website first",
			"not in the check-in files", "Mono", "No surname", "Add the surname after the first name",
			`href="https://www.worldcubeassociation.org/competitions/TestOpen2026"`)
	})

	t.Run("names are HTML-escaped", func(t *testing.T) {
		if strings.Contains(body, "<script>alert(1)</script>") {
			t.Error("competitor name was not escaped")
		}
		assertContains(t, body, "&lt;script&gt;alert(1)&lt;/script&gt; X")
	})
}

func TestIndexSelectsPreviewFile(t *testing.T) {
	h, _ := newTestServer()
	body := get(t, h, "/?id=TestOpen2026&file=returner").Body.String()
	assertContains(t, body, `<iframe src="/pdf?id=TestOpen2026&file=returner"`, "Preview: <strong>TestOpen2026-check-in-returner.pdf")

	body = get(t, h, "/?id=TestOpen2026&file=nope").Body.String()
	assertContains(t, body, `<iframe src="/pdf?id=TestOpen2026&file=first-timer"`)
}

func TestIndexShowsError(t *testing.T) {
	h, _ := newTestServer()
	rec := get(t, h, "/?id=NoSuchComp&refresh=1")
	if rec.Code != http.StatusOK {
		t.Errorf("status %d, want 200 with the error on the page", rec.Code)
	}
	body := rec.Body.String()
	assertContains(t, body, "Could not load the competition.", "HTTP 404")
	if strings.Contains(body, "<iframe") {
		t.Error("preview shown for a failed competition")
	}
}

func TestUnknownPath(t *testing.T) {
	h, _ := newTestServer()
	if rec := get(t, h, "/nope"); rec.Code != http.StatusNotFound {
		t.Errorf("status %d, want 404", rec.Code)
	}
}

func TestCache(t *testing.T) {
	h, gen := newTestServer()
	get(t, h, "/?id=TestOpen2026&refresh=1")
	get(t, h, "/?id=TestOpen2026&file=returner")
	get(t, h, "/pdf?id=TestOpen2026&file=returner")
	get(t, h, "/zip?id=TestOpen2026")
	if n := gen.calls["TestOpen2026"]; n != 1 {
		t.Errorf("previews and downloads fetched the WCIF again: %d calls, want 1", n)
	}
	get(t, h, "/?id=TestOpen2026&refresh=1")
	if n := gen.calls["TestOpen2026"]; n != 2 {
		t.Errorf("Load competition did not fetch the WCIF again: %d calls, want 2", n)
	}
	gen.calls["NoSuchComp"] = 0
	get(t, h, "/?id=NoSuchComp")
	get(t, h, "/?id=NoSuchComp")
	if n := gen.calls["NoSuchComp"]; n != 2 {
		t.Errorf("a failed fetch was cached: %d calls, want 2", n)
	}
}

func TestPDF(t *testing.T) {
	h, _ := newTestServer()
	tests := []struct {
		url, disposition string
	}{
		{"/pdf?id=TestOpen2026&file=first-timer", "inline; filename=TestOpen2026-check-in-first-timer.pdf"},
		{"/pdf?id=TestOpen2026&file=returner&download=1", "attachment; filename=TestOpen2026-check-in-returner.pdf"},
	}
	for _, tt := range tests {
		rec := get(t, h, tt.url)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: status %d", tt.url, rec.Code)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "application/pdf" {
			t.Errorf("%s: content type %q", tt.url, ct)
		}
		if cd := rec.Header().Get("Content-Disposition"); cd != tt.disposition {
			t.Errorf("%s: disposition %q, want %q", tt.url, cd, tt.disposition)
		}
		if !bytes.HasPrefix(rec.Body.Bytes(), []byte("%PDF-")) {
			t.Errorf("%s: body is not a PDF", tt.url)
		}
	}

	if rec := get(t, h, "/pdf?id=TestOpen2026&file=nope"); rec.Code != http.StatusNotFound {
		t.Errorf("unknown file: status %d, want 404", rec.Code)
	}
	if rec := get(t, h, "/pdf?id=NoSuchComp&file=returner"); rec.Code != http.StatusBadGateway {
		t.Errorf("unknown competition: status %d, want 502", rec.Code)
	}
}

func TestZip(t *testing.T) {
	h, _ := newTestServer()
	rec := get(t, h, "/zip?id=TestOpen2026")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/zip" {
		t.Errorf("content type %q", ct)
	}
	if cd, want := rec.Header().Get("Content-Disposition"), "attachment; filename=TestOpen2026-check-in.zip"; cd != want {
		t.Errorf("disposition %q, want %q", cd, want)
	}

	zr, err := zip.NewReader(bytes.NewReader(rec.Body.Bytes()), int64(rec.Body.Len()))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, f := range zr.File {
		names = append(names, f.Name)
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, _ := io.ReadAll(rc)
		rc.Close()
		if !bytes.HasPrefix(data, []byte("%PDF-")) {
			t.Errorf("%s in the zip is not a PDF", f.Name)
		}
	}
	if want := "TestOpen2026-check-in-first-timer.pdf,TestOpen2026-check-in-returner.pdf"; strings.Join(names, ",") != want {
		t.Errorf("zip holds %v, want %s", names, want)
	}

	if rec := get(t, h, "/zip?id=NoSuchComp"); rec.Code != http.StatusBadGateway {
		t.Errorf("unknown competition: status %d, want 502", rec.Code)
	}
}
