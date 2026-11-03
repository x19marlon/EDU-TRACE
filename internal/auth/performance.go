package auth

import (
	"database/sql"
	"math"
)

// Cortes de las bandas del gráfico nota vs. esfuerzo. La nota aprueba desde 3.0 y el
// esfuerzo se mide en intentos de compilación por taller trabajado.
var (
	PerformanceGradeCuts  = [2]float64{3, 4}
	PerformanceEffortCuts = [2]float64{3, 6}
)

// Qué tan pronto conviene actuar con los estudiantes de una categoría.
const (
	AttentionHigh   = "prioritaria"
	AttentionMedium = "seguimiento"
	AttentionLow    = "va_bien"
	AttentionReview = "pendiente"
)

// PerformanceCategory describe la situación de un estudiante para orientar al docente.
// EffortBand y GradeBand ubican la categoría en el gráfico, de 0 (bajo) a 2 (alto), y
// valen -1 si la categoría es de estudiantes que aún no tienen nota.
type PerformanceCategory struct {
	Key        string `json:"key"`
	Title      string `json:"title"`
	Status     string `json:"status"` // cómo va
	Needs      string `json:"needs"`  // qué necesita
	Attention  string `json:"attention"`
	EffortBand int    `json:"effort_band"`
	GradeBand  int    `json:"grade_band"`
}

// PerformanceCategories está en el orden en que conviene atender a los estudiantes.
var PerformanceCategories = []PerformanceCategory{
	{"esfuerzo_sin_resultado", "Se esfuerza sin lograrlo", "Hace muchos intentos, pero su nota sigue por debajo de 3,0. Es probable que esté probando cambios sin entender la causa de los errores.",
		"Apoyo prioritario: sentarse con sus intentos y enseñarle a leer los errores y a planear antes de programar. También le sirve una asesoría o un monitor.", AttentionHigh, 2, 0},
	{"poca_practica_nota_baja", "Poca práctica y nota baja", "Compila pocas veces antes de entregar y su nota está por debajo de 3,0. Puede que le falte tiempo o que no sepa por dónde empezar.",
		"Una conversación pronto para saber qué le impide avanzar, metas pequeñas por semana y ejercicios básicos de Práctica.", AttentionHigh, 0, 0},
	{"sin_actividad", "Sin actividad", "No tiene compilaciones ni entregas en lo que se está mirando, así que no hay evidencia de cómo va.",
		"Contactar a la persona para saber si sigue en el curso y si pudo empezar, y recordarle cómo usar la plataforma.", AttentionHigh, -1, -1},
	{"practica_sin_comprension", "Practica, falta comprensión", "Trabaja con constancia, pero su nota sigue por debajo de 3,0. Parece que hay un concepto que todavía no tiene claro.",
		"Revisar sus intentos para ubicar el tema que falla y explicarlo con un ejemplo. Puedes agregar en su clase los ejemplos de ese tema.", AttentionHigh, 1, 0},
	{"sin_entregar", "Trabaja sin entregar", "Está compilando, pero todavía no ha enviado una entrega oficial.",
		"Recordarle que solo cuenta lo que envía con el botón Enviar, y preguntarle si hay algo que no le deja avanzar.", AttentionMedium, -1, -1},
	{"por_calificar", "Falta tu calificación", "Ya entregó, pero todavía no tiene una nota tuya, así que no se puede ubicar en el gráfico.",
		"Revisar y calificar su entrega. Las propuestas de la IA no cuentan hasta que las guardes o las devuelvas.", AttentionReview, -1, -1},
	{"persistente", "Persistente, va mejorando", "Insiste hasta lograr una nota aprobatoria, aunque le cuesta más intentos de lo normal.",
		"Reconocerle el esfuerzo y enseñarle estrategias de depuración, para que dependa menos del ensayo y error.", AttentionMedium, 2, 1},
	{"aprueba_poca_practica", "Aprueba con poca práctica", "Alcanza una nota media con pocos intentos. Entiende lo básico, pero probablemente tiene vacíos que no ha probado.",
		"Animar a probar más casos antes de entregar, incluidos los casos borde, y a hacer ejercicios de Práctica.", AttentionMedium, 0, 1},
	{"en_progreso", "En progreso", "Lleva un ritmo adecuado y una nota media. Va por buen camino, con espacio para mejorar.",
		"Retroalimentación concreta sobre lo que le falta para superar 4,0, y ejercicios de nivel medio.", AttentionMedium, 1, 1},
	{"logra_con_esfuerzo", "Lo logra con esfuerzo", "Obtiene una nota alta, pero con muchos intentos. El resultado es muy bueno y conviene comprobar que no le esté costando demasiado.",
		"Reconocer su constancia y verificar que entiende sus soluciones, por ejemplo pidiéndole que explique una. Puede aprender a probar con menos intentos.", AttentionLow, 2, 2},
	{"buen_ritmo", "Buen ritmo", "Obtiene una nota alta con una práctica constante. Su forma de trabajar le está funcionando.",
		"Mantener lo que hace y darle retos opcionales. Su retroalimentación puede enfocarse en legibilidad y buenas prácticas.", AttentionLow, 1, 2},
	{"domina_el_tema", "Domina el tema", "Obtiene una nota alta con pocos intentos. Por ahora domina lo que se está evaluando.",
		"Retos de mayor nivel para que siga aprendiendo. Si quiere, puede apoyar a sus compañeros.", AttentionLow, 0, 2},
}

// performanceCategoryKeys ubica las categorías con nota por [esfuerzo][nota].
var performanceCategoryKeys = func() (m [3][3]string) {
	for _, c := range PerformanceCategories {
		if c.GradeBand >= 0 {
			m[c.EffortBand][c.GradeBand] = c.Key
		}
	}
	return m
}()

func performanceBand(v float64, cuts [2]float64) int {
	switch {
	case v < cuts[0]:
		return 0
	case v < cuts[1]:
		return 1
	}
	return 2
}

// StudentPerformance resume el resultado y el trabajo de un estudiante en una clase.
type StudentPerformance struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Grade       *float64 `json:"grade"`        // promedio de las notas del docente; nil si aún no tiene
	GradedCount int      `json:"graded_count"` // talleres con nota del docente
	Attempts    int      `json:"attempts"`     // intentos de compilación
	Submissions int      `json:"submissions"`  // entregas oficiales
	Assignments int      `json:"assignments"`  // talleres en los que trabajó
	Effort      float64  `json:"effort"`       // intentos por taller trabajado
	Category    string   `json:"category"`
}

func (p *StudentPerformance) categorize() {
	switch {
	case p.Grade != nil:
		p.Category = performanceCategoryKeys[performanceBand(p.Effort, PerformanceEffortCuts)][performanceBand(*p.Grade, PerformanceGradeCuts)]
	case p.Submissions > 0:
		p.Category = "por_calificar"
	case p.Attempts > 0:
		p.Category = "sin_entregar"
	default:
		p.Category = "sin_actividad"
	}
}

// GroupPerformance calcula el rendimiento de cada estudiante de una clase. Con
// assignmentID solo cuenta ese taller. La nota de cada taller es la de su entrega
// calificada más reciente, y solo cuentan las notas puestas por el docente: una
// devuelta, o un borrador que no sea la propuesta de la IA sin revisar.
func (s *Store) GroupPerformance(teacherID, groupID, assignmentID string) ([]StudentPerformance, error) {
	if _, err := s.ownedGroup(s.db, teacherID, groupID); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(`SELECT u.id, u.name FROM group_students gs JOIN users u ON u.id = gs.student_id
		WHERE gs.group_id = ? ORDER BY u.name`, groupID)
	if err != nil {
		return nil, err
	}
	out := []StudentPerformance{}
	index := map[string]int{}
	for rows.Next() {
		var p StudentPerformance
		if err := rows.Scan(&p.ID, &p.Name); err != nil {
			rows.Close()
			return nil, err
		}
		index[p.ID] = len(out)
		out = append(out, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Más reciente primero: así el primer valor de cada taller es el que cuenta.
	rows, err = s.db.Query(`SELECT s.student_id, s.kind, COALESCE(s.assignment_id, ''), s.id,
			CASE WHEN r.status = 'returned' OR r.ai_at IS NULL OR r.updated_at > r.ai_at OR r.ai_grade IS NULL OR r.grade != r.ai_grade
				THEN r.grade END
		FROM submissions s LEFT JOIN reviews r ON r.submission_id = s.id
		WHERE s.group_id = ? AND (? = '' OR s.assignment_id = ?)
		ORDER BY s.created_at DESC`, groupID, assignmentID, assignmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	worked := map[string]map[string]bool{}
	graded := map[string]map[string]float64{}
	for rows.Next() {
		var student, kind, assignment, id string
		var grade sql.NullFloat64
		if err := rows.Scan(&student, &kind, &assignment, &id, &grade); err != nil {
			return nil, err
		}
		i, ok := index[student]
		if !ok {
			continue // ya salió de la clase
		}
		if worked[student] == nil {
			worked[student], graded[student] = map[string]bool{}, map[string]float64{}
		}
		worked[student][assignment] = true // sin taller: todo cuenta como práctica libre
		if kind == KindAttempt {
			out[i].Attempts++
			continue
		}
		out[i].Submissions++
		key := assignment
		if key == "" {
			key = id // una entrega sin taller es una nota aparte
		}
		if _, done := graded[student][key]; !done && grade.Valid {
			graded[student][key] = grade.Float64
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range out {
		p := &out[i]
		p.Assignments = len(worked[p.ID])
		if p.Assignments > 0 {
			p.Effort = float64(p.Attempts) / float64(p.Assignments)
		}
		if g := graded[p.ID]; len(g) > 0 {
			sum := 0.0
			for _, v := range g {
				sum += v
			}
			// A centésimas: 4.3 y 4.6 deben dar 4.45, no 4.4499999.
			avg := math.Round(sum/float64(len(g))*100) / 100
			p.Grade, p.GradedCount = &avg, len(g)
		}
		p.categorize()
	}
	return out, nil
}
