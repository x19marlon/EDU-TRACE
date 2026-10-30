CREATE TABLE IF NOT EXISTS users (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  email TEXT NOT NULL UNIQUE,
  role TEXT NOT NULL CHECK (role IN ('student', 'teacher', 'monitor')),
  password_hash  TEXT NOT NULL,
  created_at TEXT NOT NULL,
  compiles INTEGER NOT NULL DEFAULT 0,
  feedbacks INTEGER NOT NULL DEFAULT 0,
  last_active_at TEXT,
  recovery_hash TEXT NOT NULL DEFAULT '',
  consent_at TEXT,
  consent_version TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS sessions (
  token_hash TEXT PRIMARY KEY, 
  user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  expires_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS courses (
  id TEXT PRIMARY KEY,
  teacher_id TEXT NOT NULL REFERENCES users(id),
  name TEXT NOT NULL,
  created_at TEXT NOT NULL,
  review_minutes_max REAL,
  batch_hours_max REAL,
  feedback_hours_max REAL
);
CREATE INDEX IF NOT EXISTS courses_teacher ON courses(teacher_id);


CREATE TABLE IF NOT EXISTS class_groups (
  id TEXT PRIMARY KEY,
  course_id  TEXT NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  join_code  TEXT NOT NULL UNIQUE,
  created_at TEXT NOT NULL,
  ai_mode TEXT NOT NULL DEFAULT 'supervisada' CHECK (ai_mode IN ('supervisada', 'inmediata'))
);


CREATE TABLE IF NOT EXISTS group_students (
  group_id   TEXT NOT NULL REFERENCES class_groups(id) ON DELETE CASCADE,
  student_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  joined_at  TEXT NOT NULL,
  PRIMARY KEY (group_id, student_id)
);
CREATE INDEX IF NOT EXISTS group_students_student ON group_students(student_id);

CREATE TABLE IF NOT EXISTS assignments (
  id TEXT PRIMARY KEY,
  group_id TEXT NOT NULL REFERENCES class_groups(id) ON DELETE CASCADE,
  title TEXT NOT NULL,
  statement TEXT NOT NULL,
  rubric TEXT NOT NULL DEFAULT '',
  due_at TEXT,
  created_at TEXT NOT NULL,
  allow_late INTEGER NOT NULL DEFAULT 1,
  late_until TEXT,
  late_penalty REAL
);

CREATE TABLE IF NOT EXISTS assignment_files (   
  id TEXT PRIMARY KEY,
  assignment_id TEXT NOT NULL REFERENCES assignments(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  size INTEGER NOT NULL,
  content_type TEXT NOT NULL DEFAULT '',
  position INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS submissions (
  id TEXT PRIMARY KEY,
  kind TEXT NOT NULL CHECK (kind IN ('official', 'attempt')),
  group_id  TEXT NOT NULL REFERENCES class_groups(id) ON DELETE CASCADE,
  assignment_id TEXT REFERENCES assignments(id) ON DELETE SET NULL,
  student_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TEXT NOT NULL,
  success INTEGER NOT NULL,
  run_error TEXT NOT NULL DEFAULT '',
  exit_code INTEGER NOT NULL,
  lines INTEGER NOT NULL,
  has_ai_feedback INTEGER NOT NULL,
  error_summary TEXT NOT NULL DEFAULT '',
  code_hash TEXT NOT NULL DEFAULT '',
  code TEXT NOT NULL,
  stdin TEXT NOT NULL DEFAULT '',
  compiler_output TEXT NOT NULL DEFAULT '',
  program_output TEXT NOT NULL DEFAULT '',
  ai_feedback TEXT NOT NULL DEFAULT '',
  ai_feedback_informal TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS submissions_group ON submissions(group_id, kind, created_at);
CREATE INDEX IF NOT EXISTS submissions_student ON submissions(student_id, group_id, created_at);

CREATE TABLE IF NOT EXISTS reviews (
  submission_id TEXT PRIMARY KEY REFERENCES submissions(id) ON DELETE CASCADE,
  teacher_id TEXT NOT NULL REFERENCES users(id),
  ai_feedback TEXT NOT NULL DEFAULT '',
  ai_grade REAL,
  ai_reason TEXT NOT NULL DEFAULT '',
  ai_confidence TEXT NOT NULL DEFAULT '',
  ai_at TEXT,
  feedback TEXT NOT NULL DEFAULT '',
  grade REAL,
  status TEXT NOT NULL CHECK (status IN ('draft', 'returned')),
  returned_at TEXT,
  updated_at TEXT NOT NULL,
  ai_discarded_at TEXT,
  feedback_origin TEXT NOT NULL DEFAULT '',
  review_ms INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS group_monitors (
  group_id TEXT NOT NULL REFERENCES class_groups(id) ON DELETE CASCADE,
  monitor_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  added_at TEXT NOT NULL,
  PRIMARY KEY (group_id, monitor_id)
);
CREATE INDEX IF NOT EXISTS group_monitors_monitor ON group_monitors(monitor_id);

CREATE TABLE IF NOT EXISTS monitor_feedback (
  id TEXT PRIMARY KEY,
  submission_id TEXT NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
  monitor_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  body TEXT NOT NULL,
  created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS monitor_feedback_submission ON monitor_feedback(submission_id, created_at);

CREATE TABLE IF NOT EXISTS course_raes (
  id TEXT PRIMARY KEY,
  course_id TEXT NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
  code TEXT NOT NULL,
  description TEXT NOT NULL,
  position INTEGER NOT NULL,
  created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS course_raes_course ON course_raes(course_id, position);

CREATE TABLE IF NOT EXISTS review_raes (
  submission_id TEXT NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
  rae_id TEXT NOT NULL REFERENCES course_raes(id) ON DELETE CASCADE,
  level TEXT NOT NULL DEFAULT '' CHECK (level IN ('', 'bajo', 'medio', 'alto')),
  ai_level TEXT NOT NULL DEFAULT '' CHECK (ai_level IN ('', 'bajo', 'medio', 'alto')),
  PRIMARY KEY (submission_id, rae_id)
);
CREATE INDEX IF NOT EXISTS review_raes_rae ON review_raes(rae_id);

CREATE TABLE IF NOT EXISTS code_annotations (
  id TEXT PRIMARY KEY,
  submission_id TEXT NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
  author_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  kind TEXT NOT NULL CHECK (kind IN ('highlight', 'strike', 'good', 'note', 'drawing')),
  line_start INTEGER,
  line_end INTEGER,
  comment TEXT NOT NULL DEFAULT '',
  data TEXT NOT NULL DEFAULT '',
  sent_at TEXT,
  created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS code_annotations_submission ON code_annotations(submission_id, created_at);

CREATE TABLE IF NOT EXISTS assignment_tests (
  id TEXT PRIMARY KEY,
  assignment_id TEXT NOT NULL REFERENCES assignments(id) ON DELETE CASCADE,
  position INTEGER NOT NULL,
  name TEXT NOT NULL,
  stdin TEXT NOT NULL DEFAULT '',
  expected TEXT NOT NULL,
  match TEXT NOT NULL DEFAULT 'contiene' CHECK (match IN ('contiene', 'exacta')),
  hidden INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS assignment_tests_assignment ON assignment_tests(assignment_id, position);

CREATE TABLE IF NOT EXISTS submission_tests (
  submission_id TEXT NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
  test_id TEXT NOT NULL REFERENCES assignment_tests(id) ON DELETE CASCADE,
  passed INTEGER NOT NULL,
  output TEXT NOT NULL DEFAULT '',
  error TEXT NOT NULL DEFAULT '',
  PRIMARY KEY (submission_id, test_id)
);
CREATE INDEX IF NOT EXISTS submission_tests_test ON submission_tests(test_id);

CREATE TABLE IF NOT EXISTS ai_jobs (
  submission_id TEXT PRIMARY KEY REFERENCES submissions(id) ON DELETE CASCADE,
  status TEXT NOT NULL CHECK (status IN ('pendiente', 'procesando', 'lista', 'error', 'omitida')),
  attempts INTEGER NOT NULL DEFAULT 0,
  error TEXT NOT NULL DEFAULT '',
  enqueued_at TEXT NOT NULL,
  started_at TEXT,
  finished_at TEXT
);
CREATE INDEX IF NOT EXISTS ai_jobs_status ON ai_jobs(status, enqueued_at);

CREATE TABLE IF NOT EXISTS audit_log (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  at TEXT NOT NULL,
  actor_id TEXT REFERENCES users(id) ON DELETE SET NULL,
  actor_name TEXT NOT NULL,
  actor_role TEXT NOT NULL,
  action TEXT NOT NULL,
  group_id TEXT REFERENCES class_groups(id) ON DELETE CASCADE,
  submission_id TEXT REFERENCES submissions(id) ON DELETE CASCADE,
  detail TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS audit_log_group ON audit_log(group_id, at);
CREATE INDEX IF NOT EXISTS audit_log_submission ON audit_log(submission_id, at);

CREATE TABLE IF NOT EXISTS ai_requests (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  at TEXT NOT NULL,
  kind TEXT NOT NULL,
  group_id TEXT REFERENCES class_groups(id) ON DELETE CASCADE,
  role TEXT NOT NULL,
  first_token_ms INTEGER,
  total_ms INTEGER NOT NULL,
  ok INTEGER NOT NULL,
  sections_ok INTEGER,
  error TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS ai_requests_group ON ai_requests(group_id, at);
