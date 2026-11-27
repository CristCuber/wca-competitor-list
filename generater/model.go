package generater

// Only the fields of the public WCIF that this program reads are declared,
// so a change in an unused field cannot break parsing.
type Competition struct {
	ID      string   `json:"id"`
	Persons []Person `json:"persons"`
}

type Person struct {
	Name         string       `json:"name"`
	WCAID        string       `json:"wcaId"`
	RegistrantID int          `json:"registrantId"`
	CountryISO2  string       `json:"countryIso2"`
	Gender       string       `json:"gender"`
	Registration Registration `json:"registration"`
}

type Registration struct {
	Status    string `json:"status"`
	AdminNote string `json:"administrativeNotes"`
}
