package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strconv"
	"strings"
)

func main() {
	wcif, err := os.Open("./file/wcif.json")
	if err != nil {
		fmt.Printf("error when read wcif file due to: %v\n", err)
	}

	defer wcif.Close()

	byteValue, _ := ioutil.ReadAll(wcif)

	var Competition WCACompetition

	json.Unmarshal(byteValue, &Competition)

	fmt.Println("this competition is " + Competition.ID)

	sort.Slice(Competition.Persons, func(i, j int) bool {
		return Competition.Persons[i].PersonName < Competition.Persons[j].PersonName
	})

	registrationDeskFileName := Competition.ID + "-registration-desk.csv"
	badgeListFileName := Competition.ID + "-badge-list.csv"
	certificateListFileName := Competition.ID + "-participants-certificate-list.csv"

	registrationDeskFile, err := os.Create("./file/" + registrationDeskFileName)
	if err != nil {
		fmt.Printf("error when create registration desk file due to: %v", err)
	}
	defer registrationDeskFile.Close()
	wRegis := csv.NewWriter(registrationDeskFile)

	badgeListFile, err := os.Create("./file/" + badgeListFileName)
	if err != nil {
		fmt.Printf("error when create badge list file due to: %v", err)
	}
	defer badgeListFile.Close()
	wBadge := csv.NewWriter(badgeListFile)

	certificateListFile, err := os.Create("./file/" + certificateListFileName)
	if err != nil {
		fmt.Printf("error when create certification list file due to: %v", err)
	}
	defer certificateListFile.Close()
	wPartiCert := csv.NewWriter(certificateListFile)

	regisArray := [][]string{{"ID", "Name", "Country", "WCA ID", "Birth Date", "Remark"}}
	badgeArray := [][]string{{"ID", "Name", "WCA ID"}}
	certArray := [][]string{{"Name"}}
	for _, person := range Competition.Persons {
		if person.Registration.Status != "accepted" {
			continue
		}

		personNameWithoutLocal := strings.Split(person.PersonName, " (")
		CompIdString := strconv.Itoa(person.RegistrationID)
		wcaIdForBadge := person.WCAID
		if wcaIdForBadge == "" {
			wcaIdForBadge = "First-timer"
		}

		regisRow := []string{CompIdString, person.PersonName, person.ConrtyISO2, person.Birthdate, ""}
		regisArray = append(regisArray, regisRow)

		badgeRow := []string{CompIdString, personNameWithoutLocal[0], wcaIdForBadge}
		badgeArray = append(badgeArray, badgeRow)

		certRow := []string{personNameWithoutLocal[0]}
		certArray = append(certArray, certRow)
	}

	wRegis.WriteAll(regisArray)
	wBadge.WriteAll(badgeArray)
	wPartiCert.WriteAll(certArray)
}

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
