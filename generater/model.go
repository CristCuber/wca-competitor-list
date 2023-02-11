package generater

type WCACompetition struct {
	Version         string   `json:"formatVersion"`
	ID              string   `json:"id"`
	CompetitionName string   `json:"name"`
	CompShortName   string   `json:"shortName"`
	SeriesComp      string   `json:"series"`
	Persons         []Person `json:"persons"`
	Events          []Event  `json:"events"`
}

type Person struct {
	PersonName     string              `json:"name"`
	WCAUserID      string              `json:"wcaUserId"`
	WCAID          string              `json:"wcaId"`
	RegistrationID int                 `json:"registrantId"`
	ConrtyISO2     string              `json:"countryIso2"`
	Gender         string              `json:"gender"`
	Registration   Registration        `json:"registration"`
	Assignments    []PersonAssignments `json:"assignments"`
	PersonalBests  []PersonalBest      `json:"personalBests"`
	Birthdate      string              `json:"birthdate"`
	Email          string              `json:"email"`
}

type PersonAssignments struct {
	ActivityID     string `json:"activityId"`
	StationNumber  string `json:"stationNumber"`
	AssignmentCode string `json:"assignmentCode"`
}

type Registration struct {
	WCARegistrationID int      `json:"wcaRegistrationId"`
	EventIds          []string `json:"eventIds"`
	Status            string   `json:"status"`
	Guests            int      `json:"guests"`
	Comments          string   `json:"comments"`
}

type PersonalBest struct {
	EventId            string `json:"eventId"`
	Best               int    `json:"best"`
	WorldRanking       int    `json:"worldRanking"`
	ContinentalRanking int    `json:"continentalRanking"`
	NationalRanking    int    `json:"nationalRanking"`
	Type               string `json:"type"`
}

type Event struct {
	ID     string  `json:"id"`
	Rounds []Round `json:"rounds"`
}

type Round struct {
	RoundID              string               `json:"id"`
	Format               string               `json:"format"`
	TimeLimit            TimeLimit            `json:"timeLimit"`
	Cutoff               int                  `json:"cutoff"`
	AdvancementCondition AdvancementCondition `json:"advancementCondition"`
	ScrambleSetCount     int                  `json:"scrambleSetCount"`
}

type TimeLimit struct {
	CentiSeconds       int      `json:"centiseconds"`
	CumulativeRoundIDs []string `json:"cumulativeRoundIds"`
}

type AdvancementCondition struct {
	Type  string `json:"type"`
	Level int    `json:"level"`
}
