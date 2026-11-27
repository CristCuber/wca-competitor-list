package generater

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

func TestEnglishName(t *testing.T) {
	tests := map[string]string{
		"Somchai Jaidee (สมชาย ใจดี)": "Somchai Jaidee",
		"Anna Smith":     "Anna Smith",
		"  Anna Smith  ": "Anna Smith",
		"Zed Last(ซี)":   "Zed Last(ซี)", // no space before "(": kept, checkName reports it
		"":               "",
	}
	for in, want := range tests {
		if got := englishName(in); got != want {
			t.Errorf("englishName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestGenderLabel(t *testing.T) {
	tests := map[string]string{"m": "Male", "f": "Female", "o": "Other", "": "Other"}
	for in, want := range tests {
		if got := genderLabel(in); got != want {
			t.Errorf("genderLabel(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFirstLetter(t *testing.T) {
	tests := map[string]string{
		"Anna":  "A",
		"de":    "D", // lower case is grouped with upper case
		"Đặng":  "Đ", // whole Vietnamese letter, not its first byte
		"ánh":   "Á",
		"สมชาย": "ส",
		"":      "",
	}
	for in, want := range tests {
		if got := firstLetter(in); got != want {
			t.Errorf("firstLetter(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestIsLatin(t *testing.T) {
	tests := map[string]bool{
		"Anna Smith":      true,
		"Nguyễn Văn Đức":  true,
		"Jean-Luc O'Neil": true,
		"สมชาย ใจดี":      false,
		"Anna สมชาย":      false,
		"李 小龙":            false,
		"123 456":         false, // no letters at all
		"":                false,
	}
	for in, want := range tests {
		if got := isLatin(in); got != want {
			t.Errorf("isLatin(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestIsCapitalized(t *testing.T) {
	tests := map[string]bool{
		"Anna Smith":     true,
		"Đặng Ánh":       true,
		"Jean-luc Smith": true, // only the first letter of each word is checked
		"anna Smith":     false,
		"Anna smith":     false,
		"ánh Nguyen":     false, // lower-case accented letter (was missed before)
		"de Souza":       false,
	}
	for in, want := range tests {
		if got := isCapitalized(in); got != want {
			t.Errorf("isCapitalized(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestCheckName(t *testing.T) {
	tests := []struct {
		name string
		want *nameIssue
	}{
		{"Somchai Jaidee (สมชาย ใจดี)", nil},
		{"Anna Smith", nil},
		{"Nguyễn Văn Đức", nil},
		{"Jean-Luc O'Neil", nil},
		{"Mono", &issueNoSurname},
		{"Mono (โมโน)", &issueNoSurname},
		{"Somchai(สมชาย)", &issueNoSpace},
		{"Zed Last(ซี)", &issueNoSpace},
		{"John  Doe", &issueDoubleSpace},
		{"John Doe  (จอห์น)", &issueDoubleSpace},
		{"สมชาย ใจดี", &issueNoEnglish},
		{"nguyễn ánh", &issueFormat},
		{"Anna smith", &issueFormat},
	}
	for _, tt := range tests {
		if got := checkName(tt.name); got != tt.want {
			t.Errorf("checkName(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestBuild(t *testing.T) {
	accepted := Registration{Status: "accepted"}
	comp := &Competition{ID: "TestOpen2026", Persons: []Person{
		{Name: "Zed Last(ซี)", RegistrantID: 1, Registration: accepted},
		{Name: "Somchai Jaidee (สมชาย ใจดี)", RegistrantID: 2, Gender: "m", CountryISO2: "TH", Registration: Registration{Status: "accepted", AdminNote: "paid"}},
		{Name: "de Souza Ana", WCAID: "2019SOUZ01", RegistrantID: 3, Registration: accepted},
		{Name: "Anna Smith", WCAID: "2015SMIT01", RegistrantID: 4, Gender: "f", CountryISO2: "US", Registration: accepted},
		{Name: "Bob Pending", RegistrantID: 5, Registration: Registration{Status: "pending"}},
		{Name: "Carl Deleted", RegistrantID: 6, Registration: Registration{Status: "deleted"}},
		{Name: "Dan Noregistration", RegistrantID: 7},
		{Name: "Mono", WCAID: "2010MONO01", RegistrantID: 8, Registration: accepted}, // returners' names are not checked
		{Name: "Mono", RegistrantID: 9, Registration: accepted},
		{Name: "Bea Brown", RegistrantID: 10, Gender: "f", Registration: accepted},
	}}
	res := build(comp)

	if res.CompetitionID != "TestOpen2026" {
		t.Errorf("CompetitionID = %q", res.CompetitionID)
	}

	t.Run("files", func(t *testing.T) {
		var keys, names []string
		for _, f := range res.Files {
			keys = append(keys, f.Key)
			names = append(names, f.Name)
		}
		if want := []string{"first-timer", "returner"}; !reflect.DeepEqual(keys, want) {
			t.Errorf("file keys = %v, want %v", keys, want)
		}
		if want := []string{"TestOpen2026-check-in-first-timer.pdf", "TestOpen2026-check-in-returner.pdf"}; !reflect.DeepEqual(names, want) {
			t.Errorf("file names = %v, want %v", names, want)
		}
	})

	t.Run("first-timers have a correct name, are sorted and keep the admin note", func(t *testing.T) {
		f, _ := res.File("first-timer")
		want := [][]string{
			{"10", "", "Bea Brown", "Female", "", "", ""},
			{"2", "", "Somchai Jaidee", "Male", "TH", "", "paid"},
		}
		if !reflect.DeepEqual(f.Rows, want) {
			t.Errorf("rows = %v, want %v", f.Rows, want)
		}
	})

	t.Run("returners are sorted without case", func(t *testing.T) {
		f, _ := res.File("returner")
		want := [][]string{
			{"4", "2015SMIT01", "Anna Smith", "Female", "US", "", ""},
			{"3", "2019SOUZ01", "de Souza Ana", "Other", "", "", ""},
			{"8", "2010MONO01", "Mono", "Other", "", "", ""},
		}
		if !reflect.DeepEqual(f.Rows, want) {
			t.Errorf("rows = %v, want %v", f.Rows, want)
		}
	})

	t.Run("wrong first-timer names are reported, not in any file", func(t *testing.T) {
		want := []NameIssue{
			{RegistrantID: 9, Name: "Mono", Problem: issueNoSurname.problem, Fix: issueNoSurname.fix},
			{RegistrantID: 1, Name: "Zed Last(ซี)", Problem: issueNoSpace.problem, Fix: issueNoSpace.fix},
		}
		if !reflect.DeepEqual(res.NameIssues, want) {
			t.Errorf("NameIssues = %+v, want %+v", res.NameIssues, want)
		}
	})
}

func TestResultFileAndURL(t *testing.T) {
	res := &Result{CompetitionID: "My Comp", Files: []File{{Key: "a"}, {Key: "b"}}}
	if f, ok := res.File("b"); !ok || f.Key != "b" {
		t.Errorf("File(b) = %v, %v", f, ok)
	}
	if _, ok := res.File("c"); ok {
		t.Error("File(c) found a file")
	}
	if got, want := res.CompetitionURL(), "https://www.worldcubeassociation.org/competitions/My%20Comp"; got != want {
		t.Errorf("CompetitionURL() = %q, want %q", got, want)
	}
}

func TestReadCompetitionID(t *testing.T) {
	tests := []struct {
		in, want string
		wantErr  bool
	}{
		{"MyComp2026\n", "MyComp2026", false},
		{"  MyComp2026  \r\n", "MyComp2026", false},
		{"MyComp2026", "MyComp2026", false}, // no newline at the end
		{"\n", "", true},
		{"   \n", "", true},
		{"", "", true}, // end of input
	}
	for _, tt := range tests {
		got, err := readCompetitionID(strings.NewReader(tt.in))
		if got != tt.want || (err != nil) != tt.wantErr {
			t.Errorf("readCompetitionID(%q) = %q, %v; want %q, error %v", tt.in, got, err, tt.want, tt.wantErr)
		}
	}
}

// wcifServer stands in for the WCA API.
func wcifServer(t *testing.T) (*Gen, *[]string) {
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.EscapedPath())
		switch r.URL.Path {
		case "/OkComp2026":
			fmt.Fprint(w, `{"id":"OkComp2026","persons":[
				{"name":"Anna Smith","wcaUserId":1,"wcaId":"2015SMIT01","registrantId":1,"countryIso2":"US","gender":"f",
				 "registration":{"wcaRegistrationId":10,"eventIds":["333"],"status":"accepted","isCompeting":true},
				 "assignments":[{"activityId":1,"stationNumber":null,"assignmentCode":"competitor"}],
				 "personalBests":[{"eventId":"333","best":900,"worldRanking":1,"continentalRanking":1,"nationalRanking":1,"type":"single"}]},
				{"name":"Bea Brown","wcaUserId":2,"wcaId":null,"registrantId":2,"countryIso2":"TH","gender":"f","registration":null}],
				"events":[{"id":"333","rounds":[{"id":"333-r1","format":"a","cutoff":{"numberOfAttempts":2,"attemptResult":3000},"timeLimit":null,"advancementCondition":null}]}]}`)
		case "/Bad Json":
			fmt.Fprint(w, `{"id":`)
		case "/NoID":
			fmt.Fprint(w, `{"persons":[]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return &Gen{client: srv.Client(), wcifURL: srv.URL + "/%s"}, &paths
}

func TestGenerate(t *testing.T) {
	g, paths := wcifServer(t)

	t.Run("real-looking WCIF with fields of other types", func(t *testing.T) {
		res, err := g.Generate("  OkComp2026 ")
		if err != nil {
			t.Fatal(err)
		}
		f, _ := res.File("returner")
		if res.CompetitionID != "OkComp2026" || len(f.Rows) != 1 || f.Rows[0][2] != "Anna Smith" {
			t.Errorf("unexpected result %+v", res)
		}
	})

	errorTests := []struct{ id, errContains string }{
		{"NoSuchComp", "HTTP 404"},
		{"Bad Json", "parse wcif"},
		{"NoID", "no competition id"},
		{"   ", "no competition id entered"},
	}
	for _, tt := range errorTests {
		if _, err := g.Generate(tt.id); err == nil || !strings.Contains(err.Error(), tt.errContains) {
			t.Errorf("Generate(%q) error = %v, want it to contain %q", tt.id, err, tt.errContains)
		}
	}

	want := []string{"/OkComp2026", "/NoSuchComp", "/Bad%20Json", "/NoID"} // the blank id is not sent
	if !reflect.DeepEqual(*paths, want) {
		t.Errorf("requested paths = %v, want %v", *paths, want)
	}
}

func rowsNamed(names ...string) [][]string {
	var rows [][]string
	for _, n := range names {
		rows = append(rows, []string{"", "", n, "", "", "", ""})
	}
	return rows
}

// describe turns pages into e.g. [["[A]" "Anna" "Andy"] ["[A]" "Amy"]].
func describe(pages [][]line) [][]string {
	var out [][]string
	for _, page := range pages {
		var p []string
		for _, l := range page {
			if l.row == nil {
				p = append(p, "["+l.letter+"]")
			} else {
				p = append(p, l.row[nameCol])
			}
		}
		out = append(out, p)
	}
	return out
}

func TestPageLines(t *testing.T) {
	tests := []struct {
		name    string
		rows    [][]string
		perPage int
		want    [][]string
	}{
		{"no competitors gives one empty page", nil, 5, [][]string{nil}},
		{"letter row before each new letter", rowsNamed("Anna", "Andy", "Bea", "de Souza"), 10,
			[][]string{{"[A]", "Anna", "Andy", "[B]", "Bea", "[D]", "de Souza"}}},
		{"letter is repeated on the new page when its names go on", rowsNamed("Anna", "Andy", "Amy", "Abe"), 3,
			[][]string{{"[A]", "Anna", "Andy"}, {"[A]", "Amy", "Abe"}}},
		{"no extra letter row when a new letter starts the page", rowsNamed("Anna", "Andy", "Bea"), 3,
			[][]string{{"[A]", "Anna", "Andy"}, {"[B]", "Bea"}}},
		{"letter row is not left alone at the bottom of a page", rowsNamed("Anna", "Bea"), 3,
			[][]string{{"[A]", "Anna"}, {"[B]", "Bea"}}},
		{"page filled exactly", rowsNamed("Anna", "Andy", "Bea", "Ben"), 3,
			[][]string{{"[A]", "Anna", "Andy"}, {"[B]", "Bea", "Ben"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := describe(pageLines(tt.rows, tt.perPage)); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("pages = %q, want %q", got, tt.want)
			}
		})
	}
}

var pageObject = regexp.MustCompile(`/Type /Page\b[^s]`)

func pdfPages(t *testing.T, rows [][]string) int {
	t.Helper()
	var buf bytes.Buffer
	if err := WritePDF(&buf, File{Name: "Test-check-in-first-timer.pdf", Rows: rows}); err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(buf.Bytes(), []byte("%PDF-")) {
		t.Fatalf("output is not a PDF: %q", buf.Bytes()[:20])
	}
	return len(pageObject.FindAll(buf.Bytes(), -1))
}

func TestWritePDF(t *testing.T) {
	if got := pdfPages(t, nil); got != 1 {
		t.Errorf("empty list: %d pages, want 1", got)
	}
	if got := pdfPages(t, rowsNamed("Anna Smith", "Đặng Ánh", "Somchai Jaidee")); got != 1 {
		t.Errorf("3 competitors: %d pages, want 1", got)
	}
	// An A4 page holds 34 lines under the header: the letter row and 33
	// names. 100 names of one letter fill 3 pages and put 1 name on page 4.
	// More pages would mean gofpdf broke a page by itself in the middle.
	many := make([]string, 100)
	for i := range many {
		many[i] = fmt.Sprintf("Anna Smith%03d", i)
	}
	if got := pdfPages(t, rowsNamed(many...)); got != 4 {
		t.Errorf("100 competitors: %d pages, want 4", got)
	}
	if got := pdfPages(t, rowsNamed(many[:33]...)); got != 1 {
		t.Errorf("33 competitors: %d pages, want 1 (a full page)", got)
	}
	if got := pdfPages(t, rowsNamed(many[:34]...)); got != 2 {
		t.Errorf("34 competitors: %d pages, want 2", got)
	}
}

func TestWritePDFFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.pdf")
	if err := writePDFFile(path, File{Name: "out.pdf", Rows: rowsNamed("Anna Smith")}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil || !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Errorf("written file is not a PDF (err %v)", err)
	}
	if err := writePDFFile(filepath.Join(dir, "missing", "out.pdf"), File{}); err == nil {
		t.Error("writing into a missing folder did not fail")
	}
}
