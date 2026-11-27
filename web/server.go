// Package web serves a local page where you enter a competition id, preview
// its check-in PDFs and download them through the browser.
package web

import (
	"archive/zip"
	"bytes"
	_ "embed"
	"fmt"
	"html/template"
	"log"
	"mime"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"wca-competitor-list/generater"
)

//go:embed index.html
var indexHTML string

var page = template.Must(template.New("index").Parse(indexHTML))

// cacheTTL is how long a fetched competition is reused for previews and
// downloads. Submitting the form always fetches it again.
const cacheTTL = 10 * time.Minute

type cached struct {
	res *generater.Result
	at  time.Time
}

// Generator builds the check-in files of a competition (generater.Gen).
type Generator interface {
	Generate(id string) (*generater.Result, error)
}

type server struct {
	gen   Generator
	mu    sync.Mutex
	cache map[string]cached
}

// Handler returns the web page's HTTP handler.
func Handler(gen Generator) http.Handler {
	s := &server{gen: gen, cache: map[string]cached{}}
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.index)
	mux.HandleFunc("/pdf", s.pdf)
	mux.HandleFunc("/zip", s.zip)
	return mux
}

// Run serves the page on addr (falling back to a free port if addr is taken)
// and, if openBrowser is set, opens it in the default browser.
func Run(gen Generator, addr string, openBrowser bool) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Printf("%s is not available (%v), using a free port", addr, err)
		if ln, err = net.Listen("tcp", "127.0.0.1:0"); err != nil {
			return err
		}
	}
	url := "http://" + ln.Addr().String()
	fmt.Printf("WCA competitor list is running at %s\nPress Ctrl+C to stop.\n", url)
	if openBrowser {
		open(url)
	}

	srv := &http.Server{Handler: Handler(gen), ReadHeaderTimeout: 10 * time.Second}
	return srv.Serve(ln)
}

// result returns the competition's files, from the cache unless fresh is set.
func (s *server) result(id string, fresh bool) (*generater.Result, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := strings.ToLower(id)
	if c, ok := s.cache[key]; ok && !fresh && time.Since(c.at) < cacheTTL {
		return c.res, nil
	}
	res, err := s.gen.Generate(id)
	if err != nil {
		return nil, err
	}
	s.cache[key] = cached{res: res, at: time.Now()}
	return res, nil
}

type pageData struct {
	ID      string
	Err     string
	Result  *generater.Result
	Current generater.File
	Columns []string
}

func (s *server) index(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	q := r.URL.Query()
	data := pageData{ID: strings.TrimSpace(q.Get("id")), Columns: generater.Columns}
	if data.ID != "" {
		res, err := s.result(data.ID, q.Get("refresh") != "")
		if err != nil {
			data.Err = err.Error()
		} else {
			data.Result = res
			data.ID = res.CompetitionID
			var ok bool
			if data.Current, ok = res.File(q.Get("file")); !ok {
				data.Current = res.Files[0]
			}
		}
	}

	var buf bytes.Buffer
	if err := page.Execute(&buf, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	buf.WriteTo(w)
}

// pdf serves one check-in PDF: inline for the preview, or as an attachment
// (download=1) so the browser saves it with its own download manager.
func (s *server) pdf(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	res, err := s.result(strings.TrimSpace(q.Get("id")), false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	f, ok := res.File(q.Get("file"))
	if !ok {
		http.Error(w, "unknown file", http.StatusNotFound)
		return
	}

	var buf bytes.Buffer
	if err := generater.WritePDF(&buf, f); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	disposition := "inline"
	if q.Get("download") != "" {
		disposition = "attachment"
	}
	send(w, "application/pdf", disposition, f.Name, &buf)
}

// zip serves all check-in PDFs of a competition in one zip file.
func (s *server) zip(w http.ResponseWriter, r *http.Request) {
	res, err := s.result(strings.TrimSpace(r.URL.Query().Get("id")), false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, f := range res.Files {
		fw, err := zw.CreateHeader(&zip.FileHeader{Name: f.Name, Method: zip.Deflate, Modified: time.Now()})
		if err == nil {
			err = generater.WritePDF(fw, f)
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if err := zw.Close(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	send(w, "application/zip", "attachment", res.CompetitionID+"-check-in.zip", &buf)
}

func send(w http.ResponseWriter, contentType, disposition, filename string, body *bytes.Buffer) {
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", mime.FormatMediaType(disposition, map[string]string{"filename": filename}))
	w.Header().Set("Cache-Control", "no-store")
	body.WriteTo(w)
}

func open(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		log.Printf("cannot open the browser (%v): open %s yourself", err, url)
	}
}
