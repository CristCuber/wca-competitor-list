package generater

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/jung-kurt/gofpdf"
)

type Gen struct{}

type IGenerater interface {
	GenerateNameList()
}

func NewGenerater() *Gen {
	return &Gen{}
}

func (g *Gen) GenerateNameList() error {
	fontSize := 9.0
	columns := []string{"ID", "WCA ID", "Name", "Gender", "Country", "Sign", "Remark"}
	columnWidth := []float64{10.0, 25.0, 65.0, 15.0, 15.0, 15.0, 45.0}

	pdfFirstTimer := gofpdf.New("P", "mm", "A4", "")
	pdfFirstTimer.SetFont("Arial", "", fontSize)

	pdfReturner := gofpdf.New("P", "mm", "A4", "")
	pdfReturner.SetFont("Arial", "", fontSize)

	pdfIncorrect := gofpdf.New("P", "mm", "A4", "")
	pdfIncorrect.SetFont("Arial", "", fontSize)

	defaultSurname := "XXXXXXXXXX"
	wcifURL := "https://www.worldcubeassociation.org/api/v0/competitions/{competitionID}/wcif/public"

	scanner := bufio.NewScanner(os.Stdin)

	fmt.Printf("Enter competition id:")
	scanner.Scan()
	input := scanner.Text()

	thisCompURL := strings.Replace(wcifURL, "{competitionID}", input, -1)

	resp, err := http.Get(thisCompURL)
	if err != nil {
		fmt.Printf("error when call wcif: %v\n", err)
		return err
	}
	defer resp.Body.Close()

	wcifByte, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("error when read body: %v\n", err)
		return err
	}

	var Competition WCACompetition

	json.Unmarshal(wcifByte, &Competition)

	fmt.Println("this competition is " + Competition.ID)

	sort.Slice(Competition.Persons, func(i, j int) bool {
		return Competition.Persons[i].PersonName < Competition.Persons[j].PersonName
	})

	exePath, err := os.Executable()
	if err != nil {
		fmt.Println("Error:", err)
		return err
	}

	exeDir := filepath.Dir(exePath)

	checkInFirstTimerFilePath := exeDir + "/" + Competition.ID + "-check-in-first-timer.pdf"
	checkInReturnerFilePath := exeDir + "/" + Competition.ID + "-check-in-returner.pdf"
	checkInIncorrectFormatFilePath := exeDir + "/" + Competition.ID + "-check-in-incorrect.pdf"

	checkInFirstTimerArray := [][]string{}
	checkInReturnerArray := [][]string{}
	checkInIncorrectFormatArray := [][]string{}

	for _, person := range Competition.Persons {
		if person.Registration.Status != "accepted" {
			continue
		}

		hasSurname := true
		isIncorrectName := false
		gender := "other"

		personNameWithoutLocal := strings.Split(person.PersonName, " (")
		personNameWithoutLocalArray := strings.SplitN(personNameWithoutLocal[0], " ", 2)

		if person.Gender == "m" {
			gender = "Male"
		} else if person.Gender == "f" {
			gender = "Female"
		}

		note := ""

		if len(personNameWithoutLocalArray) != 2 {
			note = "No surname"
			fmt.Println("++++++++ [error] competitor has wrong name: " + person.PersonName + ". No surname")
			hasSurname = false
			isIncorrectName = true
		} else if strings.Contains(personNameWithoutLocalArray[1], "(") {
			note = "No space"
			fmt.Println("++++++++ [error] competitor has wrong name: " + person.PersonName + ". No space between English and local")
			isIncorrectName = true
		} else if strings.Contains(person.PersonName, "  ") {
			note = "Have double space"
			fmt.Println("++++++++ [error] competitor has wrong name: " + person.PersonName + ". There are double space in their name")
			isIncorrectName = true
		} else if !validateCapitalization(personNameWithoutLocal[0]) {
			note = "Incorrect format"
			fmt.Println("++++++++ [error] competitor has wrong name: " + person.PersonName + ". Name is not in correct format")
			isIncorrectName = true
		} else if isLetter(person.PersonName) {
			note = "No English"
			fmt.Println("++++++++ [error] competitor has wrong name: " + person.PersonName + ". No English name")
			isIncorrectName = true
		}

		CompIdString := strconv.Itoa(person.RegistrationID)

		if isIncorrectName {
			name := personNameWithoutLocal[0]
			if !hasSurname {
				name = name + defaultSurname
			}

			regisRow := []string{CompIdString, person.WCAID, name, gender, person.ConrtyISO2, "", note}
			checkInIncorrectFormatArray = append(checkInIncorrectFormatArray, regisRow)
		} else if person.WCAID == "" {
			regisRow := []string{CompIdString, "", personNameWithoutLocal[0], gender, person.ConrtyISO2, "", person.Registration.AdminNote}
			checkInFirstTimerArray = append(checkInFirstTimerArray, regisRow)
		} else {
			regisRow := []string{CompIdString, person.WCAID, personNameWithoutLocal[0], gender, person.ConrtyISO2, "", ""}
			checkInReturnerArray = append(checkInReturnerArray, regisRow)
		}

	}

	err = printPDF(pdfFirstTimer, columns, columnWidth, checkInFirstTimerArray, checkInFirstTimerFilePath)
	if err != nil {
		fmt.Printf("error print first timer file: %v\n", err)
		return err
	}
	err = printPDF(pdfReturner, columns, columnWidth, checkInReturnerArray, checkInReturnerFilePath)
	if err != nil {
		fmt.Printf("error print returner file: %v\n", err)
		return err
	}
	err = printPDF(pdfIncorrect, columns, columnWidth, checkInIncorrectFormatArray, checkInIncorrectFormatFilePath)
	if err != nil {
		fmt.Printf("error print incorrect file: %v\n", err)
		return err
	}

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

func printPDF(pdf *gofpdf.Fpdf, columns []string, columnWidth []float64, data [][]string, filename string) error {
	herder := func() {
		pdf.SetY(10)
		pdf.SetFont("Arial", "", 7)
		pdf.Cell(0, 10, filename)
		pdf.Ln(12)

		pdf.SetFont("Arial", "B", 10)
		pdf.SetFillColor(189, 189, 189)
		for i, colText := range columns {
			pdf.CellFormat(columnWidth[i], 10, colText, "1", 0, "C", true, 0, "")
		}
		pdf.Ln(-1)
	}

	footer := func() {
		pdf.SetY(-15)
		pdf.CellFormat(0, 10, "Page "+strconv.Itoa(pdf.PageNo()), "", 0, "R", false, 0, "")
	}

	pdf.SetHeaderFunc(herder)
	pdf.AddPage()
	pdf.SetFooterFunc(footer)

	prevFirstLetter := ""
	fillColor := true

	for _, row := range data {
		firstLetter := strings.ToUpper(string(row[2][0]))
		if firstLetter != prevFirstLetter {
			pdf.SetFillColor(222, 222, 222)
			pdf.CellFormat(0, 7, firstLetter, "1", 0, "", true, 0, "")
			pdf.Ln(-1)
			prevFirstLetter = firstLetter
		}

		if fillColor {
			pdf.SetFillColor(243, 243, 243)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}
		fillColor = !fillColor

		for i, cell := range row {
			pdf.CellFormat(columnWidth[i], 7, cell, "1", 0, "", true, 0, "")
		}
		pdf.Ln(-1)
	}

	err := pdf.OutputFileAndClose(filename)
	if err != nil {
		fmt.Printf("error when close first timer file: %v\n", err)
		return err
	}

	return nil
}
