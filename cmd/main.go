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
	// wcif url: "https://www.worldcubeassociation.org/api/v0/competitions/YourCompetition2022/wcif"

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

	registrationDeskFirstTimerFileName := Competition.ID + "-registration-desk-first-timer.csv"
	registrationDeskReturnerFileName := Competition.ID + "-registration-desk-returner.csv"
	badgeListFileName := Competition.ID + "-badge-list.csv"
	certificateListFileName := Competition.ID + "-participants-certificate-list.csv"

	registrationDeskFirstTimerFile, err := os.Create("./file/" + registrationDeskFirstTimerFileName)
	if err != nil {
		fmt.Printf("error when create registration desk file due to: %v", err)
	}
	defer registrationDeskFirstTimerFile.Close()
	wfRegis := csv.NewWriter(registrationDeskFirstTimerFile)

	registrationDeskReturnerFile, err := os.Create("./file/" + registrationDeskReturnerFileName)
	if err != nil {
		fmt.Printf("error when create registration desk file due to: %v", err)
	}
	defer registrationDeskReturnerFile.Close()
	wrRegis := csv.NewWriter(registrationDeskReturnerFile)

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

	regisFirstTimerArray := [][]string{{"ID", "Name", "Country", "WCA ID", "Birth Date", "Remark"}}
	regisReturnerArray := [][]string{{"ID", "Name", "Country", "WCA ID", "Birth Date", "Remark"}}
	badgeArray := [][]string{{"ID", "Name", "Surname", "WCA ID"}}
	certArray := [][]string{{"Name"}}
	for _, person := range Competition.Persons {
		if person.Registration.Status != "accepted" {
			continue
		}

		personNameWithoutLocal := strings.Split(person.PersonName, " (")
		personNameForBadge := strings.SplitN(personNameWithoutLocal[0], " ", 2)
		if len(personNameForBadge) != 2 {
			fmt.Println("++++++++ [error] competitor has wrong name: " + person.PersonName + ". No surname")
			break
		} else if strings.Contains(personNameForBadge[1], "(") {
			fmt.Println("++++++++ [error] competitor has wrong name: " + person.PersonName + ". No space between English and local")
			break
		}

		CompIdString := strconv.Itoa(person.RegistrationID)
		wcaIdForBadge := person.WCAID
		if wcaIdForBadge == "" {
			wcaIdForBadge = "First-timer"
		}

		regisRow := []string{CompIdString, person.PersonName, person.ConrtyISO2, person.WCAID, person.Birthdate, ""}
		if person.WCAID == "" {
			regisFirstTimerArray = append(regisFirstTimerArray, regisRow)
		} else {
			regisReturnerArray = append(regisReturnerArray, regisRow)
		}

		badgeRow := []string{CompIdString, personNameForBadge[0], personNameForBadge[1], wcaIdForBadge}
		badgeArray = append(badgeArray, badgeRow)

		certRow := []string{personNameWithoutLocal[0]}
		certArray = append(certArray, certRow)
	}

	wfRegis.WriteAll(regisFirstTimerArray)
	wrRegis.WriteAll(regisReturnerArray)
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
