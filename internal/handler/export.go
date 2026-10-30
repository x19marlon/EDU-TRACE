package handler

import (
	"encoding/csv"
	"fmt"
	"mime"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/marlon/edu-trace/internal/auth"
)

var fileSafe = regexp.MustCompile(`[^\p{L}\p{N}]+`)

func csvCell(s string) string {
	if s == "" {
		return s
	}
	if _, err := strconv.ParseFloat(s, 64); err == nil {
		return s
	}
	if strings.ContainsRune("=+-@\t\r", rune(s[0])) {
		return "'" + s
	}
	return s
}

func gradeCell(g *float64) string {
	if g == nil {
		return ""
	}
	return strconv.FormatFloat(*g, 'f', -1, 64)
}

func timeCell(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.In(time.Local).Format("2006-01-02 15:04")
}

func writeCSV(w http.ResponseWriter, name string, rows [][]string) {
	filename := strings.Trim(fileSafe.ReplaceAllString(strings.ToLower(name), "-"), "-") + ".csv"
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
	w.Header().Set("Cache-Control", "no-store")
	w.Write([]byte("\xEF\xBB\xBF"))
	cw := csv.NewWriter(w)
	cw.UseCRLF = true
	for _, row := range rows {
		for i := range row {
			row[i] = csvCell(row[i])
		}
		cw.Write(row)
	}
	cw.Flush()
}

func (h *ClassesHandler) ExportGrades(w http.ResponseWriter, r *http.Request) {
	sheet, err := h.store.GradeSheet(auth.UserFrom(r.Context()).ID, r.PathValue("id"))
	if err != nil {
		writeStoreErr(w, err, "exportar las notas")
		return
	}
	header := []string{"Nombre", "Dirección de correo"}
	for _, a := range sheet.Assignments {
		header = append(header, a.Title)
	}
	rows := [][]string{append(header, "Promedio")}
	for _, row := range sheet.Rows {
		line := []string{row.Name, row.Email}
		for _, a := range sheet.Assignments {
			line = append(line, gradeCell(row.Grades[a.ID]))
		}
		rows = append(rows, append(line, gradeCell(row.Average)))
	}
	writeCSV(w, "notas "+sheet.CourseName+" "+sheet.GroupName, rows)
}

var originLabel = map[string]string{auth.OriginAI: "IA aprobada sin cambios", auth.OriginAIEdited: "IA editada por el profesor", auth.OriginTeacher: "Profesor"}

func (h *ClassesHandler) ExportReviews(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFrom(r.Context())
	group, err := h.store.GroupForTeacher(user.ID, r.PathValue("id"))
	if err != nil {
		writeStoreErr(w, err, "exportar las revisiones")
		return
	}
	list, err := h.store.ReviewsExport(user.ID, group.ID)
	if err != nil {
		writeStoreErr(w, err, "exportar las revisiones")
		return
	}
	rows := [][]string{{"Nombre", "Dirección de correo", "Taller", "Entregada", "Retraso (horas)", "Casos que pasa", "Nota",
		"Nota propuesta por la IA", "Devuelta", "Origen del texto", "Minutos de revisión", "Retroalimentación"}}
	for _, e := range list {
		late, tests := "", ""
		if e.LateSeconds > 0 {
			late = strconv.FormatFloat(float64(e.LateSeconds)/3600, 'f', 1, 64)
		}
		if e.TestsTotal > 0 {
			tests = fmt.Sprintf("%d/%d", e.TestsPassed, e.TestsTotal)
		}
		submitted := e.SubmittedAt
		rows = append(rows, []string{e.StudentName, e.Email, e.Assignment, timeCell(&submitted), late, tests, gradeCell(e.Grade),
			gradeCell(e.AIGrade), timeCell(e.ReturnedAt), originLabel[e.Origin], strconv.FormatFloat(float64(e.ReviewMs)/60000, 'f', 1, 64), e.Feedback})
	}
	writeCSV(w, "revisiones "+group.CourseName+" "+group.Name, rows)
}
