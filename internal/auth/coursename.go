package auth

// GroupCourseName devuelve el nombre del curso de una clase, o vacío si no existe. Es
// para quien ya comprobó el acceso a la clase.
func (s *Store) GroupCourseName(groupID string) string {
	var name string
	if err := s.db.QueryRow(`SELECT c.name FROM class_groups g JOIN courses c ON c.id = g.course_id
		WHERE g.id = ?`, groupID).Scan(&name); err != nil {
		return ""
	}
	return name
}

// CourseName devuelve el nombre del curso de una clase, o del taller si no se indica
// clase, solo si el usuario pertenece a ella: docente de la materia, estudiante inscrito
// o monitor. Vacío si no hay contexto o no tiene acceso. La IA lo usa para elegir los
// documentos de la materia.
func (s *Store) CourseName(u *User, groupID, assignmentID string) string {
	if groupID == "" {
		if assignmentID == "" {
			return ""
		}
		groupID = s.AssignmentGroup(assignmentID)
	}
	var name string
	err := s.db.QueryRow(`SELECT c.name FROM class_groups g JOIN courses c ON c.id = g.course_id
		WHERE g.id = ? AND (
			c.id IN (SELECT course_id FROM course_staff WHERE teacher_id = ?)
			OR g.id IN (SELECT group_id FROM group_students WHERE student_id = ?)
			OR g.id IN (SELECT group_id FROM group_monitors WHERE monitor_id = ?))`, groupID, u.ID, u.ID, u.ID).Scan(&name)
	if err != nil {
		return ""
	}
	return name
}
