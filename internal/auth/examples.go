package auth

import "time"

// GroupExampleIDs devuelve los ejemplos de sintaxis que el docente activó en su clase.
func (s *Store) GroupExampleIDs(teacherID, groupID string) ([]string, error) {
	if _, err := s.ownedGroup(s.db, teacherID, groupID); err != nil {
		return nil, err
	}
	return s.groupExampleIDs(groupID)
}

func (s *Store) groupExampleIDs(groupID string) ([]string, error) {
	rows, err := s.db.Query(`SELECT example_id FROM group_examples WHERE group_id = ?`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// SetGroupExamples agrega o quita ejemplos de la clase. Es idempotente, para que dos
// docentes de la misma materia no se pisen al cambiar ejemplos distintos.
func (s *Store) SetGroupExamples(teacherID, groupID string, ids []string, enabled bool) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := s.ownedGroup(tx, teacherID, groupID); err != nil {
		return err
	}
	now := fmtTime(time.Now())
	for _, id := range ids {
		if enabled {
			_, err = tx.Exec(`INSERT OR IGNORE INTO group_examples (group_id, example_id, added_at) VALUES (?, ?, ?)`, groupID, id, now)
		} else {
			_, err = tx.Exec(`DELETE FROM group_examples WHERE group_id = ? AND example_id = ?`, groupID, id)
		}
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

// LearnerGroupExamples son los ejemplos que el docente dejó en una clase del estudiante.
type LearnerGroupExamples struct {
	ID         string
	Name       string
	CourseName string
	ExampleIDs []string
}

// LearnerExamples devuelve las clases del estudiante, o las que acompaña como monitor,
// con los ejemplos que su docente activó. Incluye las clases que aún no tienen ninguno.
func (s *Store) LearnerExamples(learnerID string) ([]LearnerGroupExamples, error) {
	rows, err := s.db.Query(`SELECT g.id, g.name, c.name FROM class_groups g
		JOIN courses c ON c.id = g.course_id
		WHERE g.id IN (SELECT group_id FROM group_students WHERE student_id = ?)
			OR g.id IN (SELECT group_id FROM group_monitors WHERE monitor_id = ?)
		ORDER BY c.name, g.name`, learnerID, learnerID)
	if err != nil {
		return nil, err
	}
	out := []LearnerGroupExamples{}
	for rows.Next() {
		var g LearnerGroupExamples
		if err := rows.Scan(&g.ID, &g.Name, &g.CourseName); err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, g)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		if out[i].ExampleIDs, err = s.groupExampleIDs(out[i].ID); err != nil {
			return nil, err
		}
	}
	return out, nil
}
