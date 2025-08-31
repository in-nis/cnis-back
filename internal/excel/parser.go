package excel

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
	"github.com/in-nis/cnis-back/internal/db"
	lesson "github.com/in-nis/cnis-back/internal/models"
)

var baseUrl = "https://docs.google.com/spreadsheets/d/1KbzUHfsSwywWOzswZzdtIcdWzeZUNsI1/export?format=xlsx&id=1KbzUHfsSwywWOzswZzdtIcdWzeZUNsI1"

// -------------------- DOWNLOAD --------------------

func GetExcel() (string, error) {
	log.Println("📥 Downloading Excel from:", baseUrl)

	resp, err := http.Get(baseUrl)
	if err != nil {
		return "", fmt.Errorf("failed to fetch excel: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status: %s", resp.Status)
	}

	filePath := "sheet.xlsx"
	out, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	if _, err = io.Copy(out, resp.Body); err != nil {
		return "", fmt.Errorf("failed to save file: %w", err)
	}

	log.Println("✅ Excel saved to", filePath)
	return filePath, nil
}

// -------------------- PARSING --------------------

func ParseExcel(path string) ([]lesson.Lesson, error) {
	log.Println("📖 Opening Excel file:", path)

	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var lessons []lesson.Lesson

	for _, sheetName := range f.GetSheetList() {
		if strings.HasPrefix(sheetName, "12") {
			log.Println("➡️ Parsing sheet:", sheetName)

			sheetLessons, err := parseSheet(f, sheetName)
			if err != nil {
				return nil, fmt.Errorf("error parsing sheet %s: %w", sheetName, err)
			}

			log.Printf("✅ Parsed %d lessons from sheet %s\n", len(sheetLessons), sheetName)

			// Print each lesson (excluding time)
			for _, l := range sheetLessons {
				log.Printf("   ➡️ Grade: %d%s | Day: %d | Name: %s | Group: %s | Teacher: %s | Class: %s",
					l.Grade,
					l.GradeLetter,
					l.LessonDay,
					l.LessonName,
					l.LessonGroup,
					l.LessonTeacher,
					l.LessonClass,
				)
			}
			if err := db.SaveLessons(context.Background(), sheetLessons); err != nil {
				return nil, fmt.Errorf("error saving lessons from sheet %s: %w", sheetName, err)
			}

			lessons = append(lessons, sheetLessons...)
		}
	}

	log.Printf("🎉 Finished parsing. Total lessons: %d\n", len(lessons))
	return lessons, nil
}

func parseSheet(f *excelize.File, sheetName string) ([]lesson.Lesson, error) {
	var lessons []lesson.Lesson

	lessonDay, err := f.GetCellValue(sheetName, "A1")
	if err != nil {
		return nil, err
	}
	log.Println("📅 Lesson day (A1):", lessonDay)

	colToGrade := make(map[string][2]string)

	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, err
	}

	for rowIndex, row := range rows {
		for colIndex, cellValue := range row {
			colName, _ := excelize.ColumnNumberToName(colIndex + 1)

			// First row → header
			if rowIndex == 0 {
				if colName == "A" || colName == "B" || cellValue == "" {
					continue
				}
				runes := []rune(cellValue)
				gradeLetter := string(runes[len(runes)-1])
				gradeNumber := string(runes[:len(runes)-1])
				colToGrade[colName] = [2]string{gradeNumber, gradeLetter}
				log.Printf("📌 Found grade header: %s%s at col %s\n", gradeNumber, gradeLetter, colName)
				continue
			}

			if colName == "A" || colName == "B" {
				continue
			}

			lesson, ok := parseRowLesson(f, sheetName, rowIndex, colName, cellValue, colToGrade, lessonDay)
			if ok {
				lessons = append(lessons, lesson)
				log.Printf("✅ Parsed lesson: %s (%s) at row %d col %s\n", lesson.LessonName, lesson.LessonGroup, rowIndex+1, colName)
			} else {
				if strings.TrimSpace(cellValue) != "" {
					log.Printf("⚠️ Skipped cell at row %d col %s (value: %q)\n", rowIndex+1, colName, cellValue)
				}
			}
		}
	}

	mergedLessons, err := parseMergedLessons(f, sheetName, colToGrade, lessonDay)
	if err != nil {
		return nil, err
	}
	lessons = append(lessons, mergedLessons...)

	return lessons, nil
}

func parseRowLesson(
	f *excelize.File,
	sheetName string,
	rowIndex int,
	colName string,
	cellValue string,
	colToGrade map[string][2]string,
	lessonDay string,
) (lesson.Lesson, bool) {
	lessonDetails := strings.Split(cellValue, "\n")
	if len(lessonDetails) == 0 || strings.TrimSpace(lessonDetails[0]) == "" {
		return lesson.Lesson{}, false
	}

	timeCell, _ := f.GetCellValue(sheetName, fmt.Sprintf("B%d", rowIndex+1), excelize.Options{RawCellValue: true})
	timeParts := strings.Split(timeCell, "-")
	start, end := "", ""
	if len(timeParts) == 2 {
		start = strings.TrimSpace(timeParts[0])
		end = strings.TrimSpace(timeParts[1])
	}

	if start == "" || end == "" {
		log.Printf("⏭️ Skipping row %d col %s: missing time (%q)\n", rowIndex+1, colName, timeCell)
		return lesson.Lesson{}, false
	}

	gradeParts, ok := colToGrade[colName]
	if !ok {
		log.Printf("⏭️ Skipping row %d col %s: no grade mapping\n", rowIndex+1, colName)
		return lesson.Lesson{}, false
	}

	gradeNumber, err := strconv.Atoi(gradeParts[0])
	if err != nil {
		log.Printf("❌ Invalid grade number %q at row %d col %s\n", gradeParts[0], rowIndex+1, colName)
		return lesson.Lesson{}, false
	}
	gradeLetter := gradeParts[1]

	lessonName := strings.TrimSpace(lessonDetails[0])
	lessonGroup := ""
	if idx := strings.Index(lessonName, "№"); idx != -1 {
		lessonGroup = strings.TrimSpace(lessonName[idx:])
		lessonName = strings.TrimSpace(lessonName[:idx])
		gradeLetter = ""
	}

	startTime, err := time.Parse("15:04", start)
	if err != nil {
		log.Printf("❌ Invalid start time %q at row %d col %s\n", start, rowIndex+1, colName)
		return lesson.Lesson{}, false
	}
	endTime, err := time.Parse("15:04", end)
	if err != nil {
		log.Printf("❌ Invalid end time %q at row %d col %s\n", end, rowIndex+1, colName)
		return lesson.Lesson{}, false
	}
	startTime = time.Date(2000, 1, 1, startTime.Hour(), startTime.Minute(), 0, 0, time.UTC)
	endTime   = time.Date(2000, 1, 1, endTime.Hour(), endTime.Minute(), 0, 0, time.UTC)

	log.Printf("From %s:%s to %s:%s", start, end, startTime, endTime)

	lessonObj := lesson.Lesson{
		Grade:         gradeNumber,
		GradeLetter:   gradeLetter,
		LessonDay:     parseDayToIndex(lessonDay),
		LessonStart:   startTime,
		LessonEnd:     endTime,
		LessonName:    lessonName,
		LessonTeacher: "",
		LessonClass:   "",
		LessonGroup:   lessonGroup,
	}

	if len(lessonDetails) >= 2 {
		lessonObj.LessonTeacher = strings.TrimSpace(lessonDetails[1])
	}
	if len(lessonDetails) >= 3 {
		lessonObj.LessonClass = strings.TrimSpace(lessonDetails[2])
	}

	return lessonObj, true
}

// parseMergedLessons handles merged cells
func parseMergedLessons(
	f *excelize.File,
	sheetName string,
	colToGrade map[string][2]string,
	lessonDay string,
) ([]lesson.Lesson, error) {
	var lessons []lesson.Lesson
	skipped := 0

	mergedCells, err := f.GetMergeCells(sheetName)
	if err != nil {
		return nil, err
	}

	for _, mc := range mergedCells {
		val := mc.GetCellValue()
		if val == "" {
			skipped++
			log.Printf("⚠️ Skipped merged cell %s-%s: empty value\n", mc.GetStartAxis(), mc.GetEndAxis())
			continue
		}

		startAxis := mc.GetStartAxis()
		endAxis := mc.GetEndAxis()

		// Extract column and row numbers
		startCol := strings.TrimRightFunc(startAxis, func(r rune) bool {
			return r >= '0' && r <= '9'
		})
		endRow := strings.TrimLeftFunc(endAxis, func(r rune) bool {
			return r < '0' || r > '9'
		})

		lessonDetails := strings.Split(val, "\n")
		if len(lessonDetails) == 0 || strings.TrimSpace(lessonDetails[0]) == "" {
			skipped++
			log.Printf("⚠️ Skipped merged cell %s-%s: no lesson details\n", startAxis, endAxis)
			continue
		}

		// Get time from B<endRow>
		timeCell, _ := f.GetCellValue(sheetName, "B"+endRow, excelize.Options{RawCellValue: true})
		timeParts := strings.Split(timeCell, "-")
		start, end := "", ""
		if len(timeParts) == 2 {
			start = strings.TrimSpace(timeParts[0])
			end = strings.TrimSpace(timeParts[1])
		}

		if start == "" || end == "" {
			skipped++
			log.Printf("⚠️ Skipped merged cell %s-%s: missing time (%q)\n", startAxis, endAxis, timeCell)
			continue
		}

		gradeParts, ok := colToGrade[startCol]
		if !ok {
			skipped++
			log.Printf("⚠️ Skipped merged cell %s-%s: no grade mapping for col %s\n", startAxis, endAxis, startCol)
			continue
		}

		gradeNumber, err := strconv.Atoi(gradeParts[0])
		if err != nil {
			skipped++
			log.Printf("❌ Invalid grade number %q at merged cell %s-%s\n", gradeParts[0], startAxis, endAxis)
			continue
		}
		gradeLetter := gradeParts[1]

		lessonName := strings.TrimSpace(lessonDetails[0])
		lessonGroup := ""
		if idx := strings.Index(lessonName, "№"); idx != -1 {
			lessonGroup = strings.TrimSpace(lessonName[idx:])
			lessonName = strings.TrimSpace(lessonName[:idx])
			gradeLetter = "" // group lessons are for the whole parallel
		}

		startTime, err := time.Parse("15:04", start)
		if err != nil {
			skipped++
			log.Printf("❌ Invalid start time %q at merged cell %s-%s\n", start, startAxis, endAxis)
			continue
		}
		endTime, err := time.Parse("15:04", end)
		if err != nil {
			skipped++
			log.Printf("❌ Invalid end time %q at merged cell %s-%s\n", end, startAxis, endAxis)
			continue
		}
		startTime = time.Date(2000, 1, 1, startTime.Hour(), startTime.Minute(), 0, 0, time.UTC)
		endTime   = time.Date(2000, 1, 1, endTime.Hour(), endTime.Minute(), 0, 0, time.UTC)

		lessonObj := lesson.Lesson{
			Grade:         gradeNumber,
			GradeLetter:   gradeLetter,
			LessonDay:     parseDayToIndex(lessonDay),
			LessonStart:   startTime,
			LessonEnd:     endTime,
			LessonName:    lessonName,
			LessonTeacher: "",
			LessonClass:   "",
			LessonGroup:   lessonGroup,
		}

		if len(lessonDetails) >= 2 {
			lessonObj.LessonTeacher = strings.TrimSpace(lessonDetails[1])
		}
		if len(lessonDetails) >= 3 {
			lessonObj.LessonClass = strings.TrimSpace(lessonDetails[2])
		}

		lessons = append(lessons, lessonObj)
		log.Printf("✅ Parsed merged lesson: %s (%s) at %s-%s\n with %s : %s from %s : %s row %s", lessonObj.LessonName, lessonObj.LessonGroup, startAxis, endAxis, start, end, startTime, endTime, endRow)
	}

	log.Printf("📊 Finished merged cells in %s: %d lessons, %d skipped\n", sheetName, len(lessons), skipped)
	return lessons, nil
}

func parseDayToIndex(day string) int {
	day = strings.ToLower(strings.TrimSpace(day))
	switch {
	case strings.Contains(day, "понедельник"), strings.Contains(day, "mon"):
		return 1
	case strings.Contains(day, "вторник"), strings.Contains(day, "tue"):
		return 2
	case strings.Contains(day, "среда"), strings.Contains(day, "wed"):
		return 3
	case strings.Contains(day, "четверг"), strings.Contains(day, "thu"):
		return 4
	case strings.Contains(day, "пятница"), strings.Contains(day, "fri"):
		return 5
	case strings.Contains(day, "суббота"), strings.Contains(day, "sat"):
		return 6
	case strings.Contains(day, "воскресенье"), strings.Contains(day, "sun"):
		return 7
	default:
		log.Printf("⚠️ Unknown day string: %q → returning 0", day)
		return 0
	}
}