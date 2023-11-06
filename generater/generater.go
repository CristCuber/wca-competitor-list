package generater

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

type Gen struct{}

type IGenerater interface {
	GenerateNameList()
}

func NewGenerater() *Gen {
	return &Gen{}
}

func (g *Gen) GenerateNameList() error {
	// wcif url: "https://www.worldcubeassociation.org/api/v0/competitions/YourCompetition2022/wcif"

	wcif, err := os.Open("./generater/file/wcif.json")
	if err != nil {
		fmt.Printf("error when read wcif file due to: %v\n", err)
		return err
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
	registrationDeskIncorrectFormatFileName := Competition.ID + "-registration-desk-incorrect.csv"
	badgeListFileName := Competition.ID + "-badge-list.csv"
	certificateListFileName := Competition.ID + "-participants-certificate-list.csv"

	registrationDeskFirstTimerFile, err := os.Create("./generater/file/" + registrationDeskFirstTimerFileName)
	if err != nil {
		fmt.Printf("error when create registration desk file due to: %v", err)
	}
	defer registrationDeskFirstTimerFile.Close()
	wfRegis := csv.NewWriter(registrationDeskFirstTimerFile)

	registrationDeskReturnerFile, err := os.Create("./generater/file/" + registrationDeskReturnerFileName)
	if err != nil {
		fmt.Printf("error when create registration desk file due to: %v", err)
		return err
	}
	defer registrationDeskReturnerFile.Close()
	wrRegis := csv.NewWriter(registrationDeskReturnerFile)

	registrationDeskIncorrectFile, err := os.Create("./generater/file/" + registrationDeskIncorrectFormatFileName)
	if err != nil {
		fmt.Printf("error when create registration desk file due to: %v", err)
		return err
	}
	defer registrationDeskReturnerFile.Close()
	wiRegis := csv.NewWriter(registrationDeskIncorrectFile)

	badgeListFile, err := os.Create("./generater/file/" + badgeListFileName)
	if err != nil {
		fmt.Printf("error when create badge list file due to: %v", err)
		return err
	}
	defer badgeListFile.Close()
	wBadge := csv.NewWriter(badgeListFile)

	certificateListFile, err := os.Create("./generater/file/" + certificateListFileName)
	if err != nil {
		fmt.Printf("error when create certification list file due to: %v", err)
		return err
	}
	defer certificateListFile.Close()
	wPartiCert := csv.NewWriter(certificateListFile)

	regisFirstTimerArray := [][]string{{"ID", "WCA ID", "Name", "Name-Checked", "Birth Date", "BirthDate-Checked", "Country", "Remark"}}
	regisReturnerArray := [][]string{{"ID", "WCA ID", "Name", "Name-Checked", "Birth Date", "BirthDate-Checked", "Country", "Remark"}}
	regisIncorrectFormatArray := [][]string{{"ID", "WCA ID", "Name", "Name-Checked", "Birth Date", "BirthDate-Checked", "Country", "Remark"}}
	badgeArray := [][]string{{"ID", "Name", "Surname", "WCA ID"}}
	certArray := [][]string{{"Name"}}
	for _, person := range Competition.Persons {
		if person.Registration.Status != "accepted" {
			continue
		}

		hasSurname := true
		personNameWithoutLocal := strings.Split(person.PersonName, " (")
		personNameForBadge := strings.SplitN(personNameWithoutLocal[0], " ", 2)
		if len(personNameForBadge) != 2 {
			fmt.Println("++++++++ [error] competitor has wrong name: " + person.PersonName + ". No surname")
			hasSurname = false
		} else if strings.Contains(personNameForBadge[1], "(") {
			fmt.Println("++++++++ [error] competitor has wrong name: " + person.PersonName + ". No space between English and local")
		} else if strings.Contains(person.PersonName, "  ") {
			fmt.Println("++++++++ [error] competitor has wrong name: " + person.PersonName + ". there are double space in their name")
		} else if !validateCapitalization(personNameWithoutLocal[0]) {
			fmt.Println("++++++++ [error] competitor has wrong name: " + person.PersonName + ". Name is not in correct format")
		}

		CompIdString := strconv.Itoa(person.RegistrationID)
		wcaIdForBadge := person.WCAID
		if wcaIdForBadge == "" {
			wcaIdForBadge = "First-timer"
		}

		if person.Registration.AdminNote == "incorrectName" {
			regisRow := []string{CompIdString, person.WCAID, person.PersonName, "", person.Birthdate, "", person.ConrtyISO2, "*** Please confirm your full name in English to a staff ***"}
			regisIncorrectFormatArray = append(regisIncorrectFormatArray, regisRow)
		} else if person.WCAID == "" {
			regisRow := []string{CompIdString, "First-timer", person.PersonName, "", person.Birthdate, "", person.ConrtyISO2, ""}
			regisFirstTimerArray = append(regisFirstTimerArray, regisRow)
		} else {
			regisRow := []string{CompIdString, person.WCAID, person.PersonName, "", person.Birthdate, "", person.ConrtyISO2, ""}
			regisReturnerArray = append(regisReturnerArray, regisRow)
		}

		badgeRow := []string{}
		if hasSurname {
			badgeRow = []string{CompIdString, personNameForBadge[0], personNameForBadge[1], wcaIdForBadge}
		} else {
			badgeRow = []string{CompIdString, personNameForBadge[0], "", wcaIdForBadge}
		}

		badgeArray = append(badgeArray, badgeRow)

		certRow := []string{personNameWithoutLocal[0]}
		certArray = append(certArray, certRow)
	}

	wfRegis.WriteAll(regisFirstTimerArray)
	wrRegis.WriteAll(regisReturnerArray)
	wiRegis.WriteAll(regisIncorrectFormatArray)
	wBadge.WriteAll(badgeArray)
	wPartiCert.WriteAll(certArray)

	return nil
}

func validateCapitalization(fullname string) bool {
	words := strings.Fields(fullname)
	for _, word := range words {
		if !unicode.IsUpper(rune(word[0])) {
			return false
		}
	}
	return true
}
