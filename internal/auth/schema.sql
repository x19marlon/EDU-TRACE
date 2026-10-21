CREATE TABLE IF NOT EXISTS users (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  email TEXT NOT NULL UNIQUE,
  role TEXT NOT NULL CHECK (role IN ('student', 'teacher')),
  password_hash  TEXT NOT NULL,
  created_at TEXT NOT NULL,
  compiles INTEGER NOT NULL DEFAULT 0,
  feedbacks INTEGER NOT NULL DEFAULT 0,
  last_active_at TEXT
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
  created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS courses_teacher ON courses(teacher_id);


CREATE TABLE IF NOT EXISTS class_groups (
  id TEXT PRIMARY KEY,
  course_id  TEXT NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  join_code  TEXT NOT NULL UNIQUE,
  created_at TEXT NOT NULL
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
  created_at TEXT NOT NULL
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
  updated_at TEXT NOT NULL
);
