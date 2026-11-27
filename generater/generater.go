package generater

import (
	"bufio"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/jung-kurt/gofpdf"
)

//go:embed fonts/NotoSans-Regular.ttf
var notoSans []byte

//go:embed fonts/NotoSans-Bold.ttf
var notoSansBold []byte

const (
	wcifURL = "https://www.worldcubeassociation.org/api/v0/competitions/%s/wcif/public"
	font    = "NotoSans"
)

var (
	Columns     = []string{"ID", "WCA ID", "Name", "Gender", "Country", "Sign", "Remark"}
	columnWidth = []float64{10, 25, 65, 15, 15, 15, 45}
)

const nameCol, remarkCol = 2, 6

// File is one check-in PDF: its rows and the name it is saved under.
type File struct {
	Key   string // used in URLs, e.g. "first-timer"
	Label string // shown to people, e.g. "First-timer"
	Name  string // e.g. "MyComp2026-check-in-first-timer.pdf"
	Rows  [][]string
}

// NameIssue is a first-timer whose name is not in the WCA format. They are
// left out of the check-in files until the name is fixed on the WCA website.
type NameIssue struct {
	RegistrantID int
	Name         string
	Problem      string
	Fix          string
}

// Result holds the check-in files of a competition and the first-timers
// whose name has to be fixed first.
type Result struct {
	CompetitionID string
	Files         []File
	NameIssues    []NameIssue
}

// CompetitionURL is the competition's page on the WCA website.
func (r *Result) CompetitionURL() string {
	return "https://www.worldcubeassociation.org/competitions/" + url.PathEscape(r.CompetitionID)
}

// File returns the file with the given key.
func (r *Result) File(key string) (File, bool) {
	for _, f := range r.Files {
		if f.Key == key {
			return f, true
		}
	}
	return File{}, false
}

type Gen struct {
	client  *http.Client
	wcifURL string // with %s for the competition id
}

func NewGenerater() *Gen {
	return &Gen{client: &http.Client{Timeout: 30 * time.Second}, wcifURL: wcifURL}
}

// Generate downloads the public WCIF of a competition and builds its check-in files.
func (g *Gen) Generate(id string) (*Result, error) {
	comp, err := g.fetchCompetition(id)
	if err != nil {
		return nil, err
	}
	return build(comp), nil
}

// GenerateNameList asks for a competition id on the terminal and writes the
// check-in PDFs next to the executable.
func (g *Gen) GenerateNameList() error {
	id, err := readCompetitionID(os.Stdin)
	if err != nil {
		return err
	}
	res, err := g.Generate(id)
	if err != nil {
		return err
	}
	fmt.Println("this competition is " + res.CompetitionID)
	for _, n := range res.NameIssues {
		fmt.Printf("++++++++ [error] competitor %d has wrong name: %s. %s (%s)\n", n.RegistrantID, n.Name, n.Problem, n.Fix)
	}
	if len(res.NameIssues) > 0 {
		fmt.Println("These competitors are left out of the files: fix their names on the WCA website and run again.")
	}

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("find executable folder: %w", err)
	}
	for _, f := range res.Files {
		if err := writePDFFile(filepath.Join(filepath.Dir(exePath), f.Name), f); err != nil {
			return fmt.Errorf("write %s: %w", f.Name, err)
		}
	}
	return nil
}

func readCompetitionID(r io.Reader) (string, error) {
	fmt.Print("Enter competition id: ")
	sc := bufio.NewScanner(r)
	if !sc.Scan() {
		if err := sc.Err(); err != nil {
			return "", fmt.Errorf("read competition id: %w", err)
		}
	}
	id := strings.TrimSpace(sc.Text())
	if id == "" {
		return "", errors.New("no competition id entered")
	}
	return id, nil
}

func (g *Gen) fetchCompetition(id string) (*Competition, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, errors.New("no competition id entered")
	}
	resp, err := g.client.Get(fmt.Sprintf(g.wcifURL, url.PathEscape(id)))
	if err != nil {
		return nil, fmt.Errorf("call wcif: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cannot get wcif of %q (HTTP %d): check the competition id", id, resp.StatusCode)
	}

	var comp Competition
	if err := json.NewDecoder(resp.Body).Decode(&comp); err != nil {
		return nil, fmt.Errorf("parse wcif: %w", err)
	}
	if comp.ID == "" {
		return nil, fmt.Errorf("wcif of %q has no competition id", id)
	}
	return &comp, nil
}

// build sorts the accepted competitors by name and puts each one in one of
// the three check-in lists. Only first-timers' names are validated.
func build(comp *Competition) *Result {
	persons := comp.Persons
	sort.SliceStable(persons, func(i, j int) bool {
		return strings.ToLower(persons[i].Name) < strings.ToLower(persons[j].Name)
	})

	var firstTimers, returners [][]string
	var issues []NameIssue
	for _, p := range persons {
		if p.Registration.Status != "accepted" {
			continue
		}
		row := []string{strconv.Itoa(p.RegistrantID), p.WCAID, englishName(p.Name), genderLabel(p.Gender), p.CountryISO2, "", ""}

		if p.WCAID != "" {
			returners = append(returners, row)
			continue
		}
		if issue := checkName(p.Name); issue != nil {
			issues = append(issues, NameIssue{RegistrantID: p.RegistrantID, Name: p.Name, Problem: issue.problem, Fix: issue.fix})
			continue
		}
		row[remarkCol] = p.Registration.AdminNote
		firstTimers = append(firstTimers, row)
	}

	file := func(key, label string, rows [][]string) File {
		return File{Key: key, Label: label, Name: comp.ID + "-check-in-" + key + ".pdf", Rows: rows}
	}
	return &Result{
		CompetitionID: comp.ID,
		Files: []File{
			file("first-timer", "First-timer", firstTimers),
			file("returner", "Returner", returners),
		},
		NameIssues: issues,
	}
}

func genderLabel(g string) string {
	switch g {
	case "m":
		return "Male"
	case "f":
		return "Female"
	}
	return "Other"
}

// englishName drops the local name: "Somchai Jaidee (สมชาย ใจดี)" -> "Somchai Jaidee".
func englishName(name string) string {
	english, _, _ := strings.Cut(name, " (")
	return strings.TrimSpace(english)
}

type nameIssue struct {
	problem string
	fix     string
}

var (
	issueNoSpace     = nameIssue{"No space between English and local name", "Put a space before the local name in brackets, e.g. Somchai Jaidee (สมชาย ใจดี)"}
	issueNoSurname   = nameIssue{"No surname", "Add the surname after the first name"}
	issueDoubleSpace = nameIssue{"Double space in the name", "Remove the extra space"}
	issueNoEnglish   = nameIssue{"No English name", "Write the name in English letters, with the local name in brackets after it"}
	issueFormat      = nameIssue{"Incorrect capitalization", "Start every word of the name with a capital letter, e.g. Somchai Jaidee"}
)

// checkName returns what is wrong with a WCA name, or nil if it is fine.
func checkName(name string) *nameIssue {
	english := englishName(name)
	switch {
	case strings.Contains(english, "("):
		return &issueNoSpace
	case !strings.Contains(english, " "):
		return &issueNoSurname
	case strings.Contains(name, "  "):
		return &issueDoubleSpace
	case !isLatin(english):
		return &issueNoEnglish
	case !isCapitalized(english):
		return &issueFormat
	}
	return nil
}

// isLatin reports whether s has letters and all of them are Latin
// (accents such as Vietnamese ones included).
func isLatin(s string) bool {
	hasLetter := false
	for _, r := range s {
		if unicode.IsLetter(r) {
			if !unicode.Is(unicode.Latin, r) {
				return false
			}
			hasLetter = true
		}
	}
	return hasLetter
}

// isCapitalized reports whether every word starts with an upper-case letter.
func isCapitalized(s string) bool {
	for _, word := range strings.Fields(s) {
		if r, _ := utf8.DecodeRuneInString(word); !unicode.IsUpper(r) {
			return false
		}
	}
	return true
}

func firstLetter(s string) string {
	r, size := utf8.DecodeRuneInString(s)
	if size == 0 {
		return ""
	}
	return string(unicode.ToUpper(r))
}

func writePDFFile(path string, f File) error {
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := WritePDF(out, f); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// WritePDF renders a check-in file as an A4 PDF.
func WritePDF(w io.Writer, f File) error {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddUTF8FontFromBytes(font, "", notoSans)
	pdf.AddUTF8FontFromBytes(font, "B", notoSansBold)

	pdf.SetHeaderFunc(func() {
		pdf.SetY(10)
		pdf.SetFont(font, "", 7)
		pdf.Cell(0, 10, f.Name)
		pdf.Ln(12)

		pdf.SetFont(font, "B", 10)
		pdf.SetFillColor(189, 189, 189)
		for i, col := range Columns {
			pdf.CellFormat(columnWidth[i], 10, col, "1", 0, "C", true, 0, "")
		}
		pdf.Ln(-1)
	})
	pdf.SetFooterFunc(func() {
		pdf.SetY(-15)
		pdf.SetFont(font, "", 9)
		pdf.CellFormat(0, 10, "Page "+strconv.Itoa(pdf.PageNo()), "", 0, "R", false, 0, "")
	})
	pdf.AddPage()

	const rowHeight = 7
	_, pageHeight := pdf.GetPageSize()
	_, bottomMargin := pdf.GetAutoPageBreak()
	perPage := int(math.Floor((pageHeight - bottomMargin - pdf.GetY()) / rowHeight))

	pdf.SetFont(font, "", 9)
	nameRow := 0
	for p, page := range pageLines(f.Rows, perPage) {
		if p > 0 {
			pdf.AddPage()
		}
		for _, l := range page {
			if l.row == nil {
				pdf.SetFillColor(222, 222, 222)
				pdf.CellFormat(0, rowHeight, l.letter, "1", 0, "", true, 0, "")
				pdf.Ln(-1)
				continue
			}
			if nameRow%2 == 0 {
				pdf.SetFillColor(243, 243, 243)
			} else {
				pdf.SetFillColor(255, 255, 255)
			}
			nameRow++
			for j, cell := range l.row {
				pdf.CellFormat(columnWidth[j], rowHeight, cell, "1", 0, "", true, 0, "")
			}
			pdf.Ln(-1)
		}
	}

	return pdf.Output(w)
}

// line is one row of a PDF page: a competitor, or (row == nil) the row
// with the first letter of the names below it.
type line struct {
	letter string
	row    []string
}

// pageLines splits the competitors into pages of at most perPage lines.
// A letter line comes before the first name of every letter and again at
// the top of a page when the names of that letter go on from the page
// before; a letter line is never left alone at the bottom of a page.
func pageLines(rows [][]string, perPage int) [][]line {
	pages := [][]line{nil}
	prevLetter := ""
	for _, row := range rows {
		letter := firstLetter(row[nameCol])
		showLetter := letter != prevLetter
		need := 1
		if showLetter {
			need++
		}
		if cur := pages[len(pages)-1]; len(cur) > 0 && len(cur)+need > perPage {
			pages = append(pages, nil)
			showLetter = true
		}
		last := len(pages) - 1
		if showLetter {
			pages[last] = append(pages[last], line{letter: letter})
			prevLetter = letter
		}
		pages[last] = append(pages[last], line{letter: letter, row: row})
	}
	return pages
}
