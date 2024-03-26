package generater

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
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

	defaultSurname := "XXXXXXXXXX"

	wcif, err := os.Open("./generater/file/wcif.json")
	if err != nil {
		fmt.Printf("error when read wcif file due to: %v\n", err)
		return err
	}

	defer wcif.Close()

	byteValue, _ := io.ReadAll(wcif)

	var Competition WCACompetition

	json.Unmarshal(byteValue, &Competition)

	fmt.Println("this competition is " + Competition.ID)

	sort.Slice(Competition.Persons, func(i, j int) bool {
		return Competition.Persons[i].PersonName < Competition.Persons[j].PersonName
	})

	registrationDeskFirstTimerFileName := Competition.ID + "-registration-desk-first-timer.csv"
	registrationDeskReturnerFileName := Competition.ID + "-registration-desk-returner.csv"
	registrationDeskIncorrectFormatFileName := Competition.ID + "-registration-desk-incorrect.csv"

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

	regisFirstTimerArray := [][]string{{"ID", "WCA ID", "Name", "Gender", "Country", "Sign", "Remark"}}
	regisReturnerArray := [][]string{{"ID", "WCA ID", "Name", "Gender", "Country", "Sign", "Remark"}}
	regisIncorrectFormatArray := [][]string{{"ID", "WCA ID", "Name", "Gender", "Country", "Sign", "Remark"}}

	for _, person := range Competition.Persons {
		if person.Registration.Status != "accepted" {
			continue
		}

		hasSurname := true
		isIncorrectName := false
		gender := "other"

		personNameWithoutLocal := strings.Split(person.PersonName, " (")
		personNameForBadge := strings.SplitN(personNameWithoutLocal[0], " ", 2)

		if person.Gender == "m" {
			gender = "Male"
		} else if person.Gender == "f" {
			gender = "Female"
		}

		if len(personNameForBadge) != 2 {
			fmt.Println("++++++++ [error] competitor has wrong name: " + person.PersonName + ". No surname")
			hasSurname = false
			isIncorrectName = true
		} else if strings.Contains(personNameForBadge[1], "(") {
			fmt.Println("++++++++ [error] competitor has wrong name: " + person.PersonName + ". No space between English and local")
		} else if strings.Contains(person.PersonName, "  ") {
			fmt.Println("++++++++ [error] competitor has wrong name: " + person.PersonName + ". there are double space in their name")
		} else if !validateCapitalization(personNameWithoutLocal[0]) {
			fmt.Println("++++++++ [error] competitor has wrong name: " + person.PersonName + ". Name is not in correct format")
		} else if isLetter(person.PersonName) {
			fmt.Println("++++++++ [error] competitor has wrong name: " + person.PersonName + ". No English name")
			isIncorrectName = true
		}

		if strings.HasPrefix(person.Registration.AdminNote, "***") {
			isIncorrectName = true
		}

		CompIdString := strconv.Itoa(person.RegistrationID)

		if isIncorrectName {
			name := person.PersonName
			if !hasSurname {
				name = name + defaultSurname
			}

			regisRow := []string{CompIdString, person.WCAID, name, gender, person.ConrtyISO2, "", "Please confirm your full name in English"}
			regisIncorrectFormatArray = append(regisIncorrectFormatArray, regisRow)
		} else if person.WCAID == "" {
			regisRow := []string{CompIdString, "", person.PersonName, gender, person.ConrtyISO2, "", person.Registration.AdminNote}
			regisFirstTimerArray = append(regisFirstTimerArray, regisRow)
		} else {
			regisRow := []string{CompIdString, person.WCAID, person.PersonName, gender, person.ConrtyISO2, "", ""}
			regisReturnerArray = append(regisReturnerArray, regisRow)
		}

	}

	wfRegis.WriteAll(regisFirstTimerArray)
	wrRegis.WriteAll(regisReturnerArray)
	wiRegis.WriteAll(regisIncorrectFormatArray)

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

func isLetter(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) {
			return false
		}
	}
	return true
}
