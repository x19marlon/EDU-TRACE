package backup

import (
	"archive/tar"
	"compress/gzip"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"filippo.io/age"
	_ "modernc.org/sqlite"
)

const (
	dbName = "edutrace.db"
	filesDir = "assignments"
	prefix = "edutrace-"
	suffix = ".tar.gz"
	encrypted = ".age"
)

var ErrNeedIdentity = errors.New("el respaldo está cifrado: indica la clave privada con -identidad o BACKUP_IDENTITY")

func ParseRecipients(text string) ([]age.Recipient, error) {
	text = strings.TrimSpace(strings.ReplaceAll(text, ",", "\n"))
	if text == "" {
		return nil, nil
	}
	return age.ParseRecipients(strings.NewReader(text))
}

func LoadIdentities(path string) ([]age.Identity, error) {
	if path == "" {
		return nil, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return age.ParseIdentities(f)
}

func Encrypted(path string) bool { return strings.HasSuffix(path, suffix+encrypted) }

type archiveReader struct {
	io.Reader
	file *os.File
}

func (a archiveReader) Close() error { return a.file.Close() }

func openArchive(path string, ids []age.Identity) (io.ReadCloser, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	if !Encrypted(path) {
		return f, nil
	}
	if len(ids) == 0 {
		f.Close()
		return nil, ErrNeedIdentity
	}
	r, err := age.Decrypt(f, ids...)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("descifrar: %w", err)
	}
	return archiveReader{Reader: r, file: f}, nil
}

func encryptFile(src, dst string, recipients []age.Recipient) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	w, err := age.Encrypt(out, recipients...)
	if err == nil {
		_, err = io.Copy(w, in)
		if cerr := w.Close(); err == nil {
			err = cerr
		}
	}
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	return err
}

type Summary struct {
	Users int
	Courses int
	Submissions int
	Files int
	MissingFiles int
}

func (s Summary) String() string {
	return fmt.Sprintf("%d usuarios, %d materias, %d entregas, %d archivos de talleres", s.Users, s.Courses, s.Submissions, s.Files)
}

func Create(dataDir, destDir string, now time.Time, recipients []age.Recipient) (string, Summary, error) {
	if _, err := os.Stat(filepath.Join(dataDir, dbName)); err != nil {
		return "", Summary{}, fmt.Errorf("no encuentro la base: %w", err)
	}
	if err := os.MkdirAll(destDir, 0o700); err != nil {
		return "", Summary{}, err
	}
	work, err := os.MkdirTemp(destDir, ".respaldo-")
	if err != nil {
		return "", Summary{}, err
	}
	defer os.RemoveAll(work)
	snapshot := filepath.Join(work, dbName)
	db, err := sql.Open("sqlite", "file:"+filepath.Join(dataDir, dbName)+"?mode=ro&_pragma=busy_timeout(10000)")
	if err != nil {
		return "", Summary{}, err
	}
	_, err = db.Exec(`VACUUM INTO ?`, snapshot)
	db.Close()
	if err != nil {
		return "", Summary{}, fmt.Errorf("copiar la base: %w", err)
	}
	name := filepath.Join(destDir, prefix+now.Format("20060102-150405")+suffix)
	tmp := filepath.Join(work, "respaldo"+suffix)
	if err := writeArchive(tmp, snapshot, filepath.Join(dataDir, filesDir)); err != nil {
		return "", Summary{}, err
	}
	summary, err := Verify(tmp, nil)
	if err != nil {
		return "", Summary{}, fmt.Errorf("el respaldo no pasó la verificación: %w", err)
	}
	if len(recipients) > 0 {
		name += encrypted
		sealed := filepath.Join(work, "respaldo"+suffix+encrypted)
		if err := encryptFile(tmp, sealed, recipients); err != nil {
			return "", Summary{}, fmt.Errorf("cifrar: %w", err)
		}
		tmp = sealed
	}
	if err := os.Rename(tmp, name); err != nil {
		return "", Summary{}, err
	}
	return name, summary, nil
}

func writeArchive(path, snapshot, files string) error {
	out, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	gz := gzip.NewWriter(out)
	tw := tar.NewWriter(gz)
	err = addFile(tw, snapshot, dbName)
	if err == nil {
		err = filepath.WalkDir(files, func(p string, d fs.DirEntry, err error) error {
			if errors.Is(err, fs.ErrNotExist) && p == files {
				return filepath.SkipDir
			}
			if err != nil || !d.Type().IsRegular() {
				return err
			}
			rel, err := filepath.Rel(filepath.Dir(files), p)
			if err != nil {
				return err
			}
			return addFile(tw, p, filepath.ToSlash(rel))
		})
	}
	for _, c := range []io.Closer{tw, gz, out} {
		if cerr := c.Close(); err == nil {
			err = cerr
		}
	}
	return err
}

func addFile(tw *tar.Writer, path, name string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return err
	}
	if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o600, Size: st.Size(), ModTime: st.ModTime(), Typeflag: tar.TypeReg}); err != nil {
		return err
	}
	_, err = io.Copy(tw, f)
	return err
}

func safeName(name string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(name))
	if name == "" || filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("ruta no permitida en el respaldo: %q", name)
	}
	if clean != dbName && !strings.HasPrefix(clean, filesDir+string(filepath.Separator)) {
		return "", fmt.Errorf("archivo inesperado en el respaldo: %q", name)
	}
	return clean, nil
}

func Extract(archive, dir string, ids []age.Identity) error {
	if _, err := os.Stat(dir); err == nil {
		return fmt.Errorf("%s ya existe: elige una carpeta nueva", dir)
	}
	in, err := openArchive(archive, ids)
	if err != nil {
		return err
	}
	defer in.Close()
	gz, err := gzip.NewReader(in)
	if err != nil {
		return err
	}
	defer gz.Close()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if h.Typeflag != tar.TypeReg {
			return fmt.Errorf("el respaldo solo debe tener archivos normales: %q", h.Name)
		}
		name, err := safeName(h.Name)
		if err != nil {
			return err
		}
		target := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err != nil {
			return err
		}
		_, err = io.Copy(out, tr)
		if cerr := out.Close(); err == nil {
			err = cerr
		}
		if err != nil {
			return err
		}
	}
	if _, err := os.Stat(filepath.Join(dir, dbName)); err != nil {
		return fmt.Errorf("el respaldo no tiene la base de datos: %w", err)
	}
	return nil
}

func Inspect(dataDir string) (Summary, error) {
	var s Summary
	db, err := sql.Open("sqlite", "file:"+filepath.Join(dataDir, dbName)+"?mode=ro")
	if err != nil {
		return s, err
	}
	defer db.Close()
	var check string
	if err := db.QueryRow(`PRAGMA integrity_check`).Scan(&check); err != nil {
		return s, fmt.Errorf("revisar la base: %w", err)
	}
	if check != "ok" {
		return s, fmt.Errorf("la base está dañada: %s", check)
	}
	if err := db.QueryRow(`SELECT (SELECT COUNT(*) FROM users), (SELECT COUNT(*) FROM courses), (SELECT COUNT(*) FROM submissions)`).
		Scan(&s.Users, &s.Courses, &s.Submissions); err != nil {
		return s, err
	}
	rows, err := db.Query(`SELECT assignment_id, id FROM assignment_files`)
	if err != nil {
		return s, err
	}
	defer rows.Close()
	for rows.Next() {
		var assignment, id string
		if err := rows.Scan(&assignment, &id); err != nil {
			return s, err
		}
		s.Files++
		if _, err := os.Stat(filepath.Join(dataDir, filesDir, assignment, id)); err != nil {
			s.MissingFiles++
		}
	}
	return s, rows.Err()
}

func Verify(archive string, ids []age.Identity) (Summary, error) {
	work, err := os.MkdirTemp("", "edutrace-verificar-")
	if err != nil {
		return Summary{}, err
	}
	defer os.RemoveAll(work)
	dir := filepath.Join(work, "datos")
	if err := Extract(archive, dir, ids); err != nil {
		return Summary{}, err
	}
	s, err := Inspect(dir)
	if err == nil && s.MissingFiles > 0 {
		err = fmt.Errorf("faltan %d de %d archivos de talleres", s.MissingFiles, s.Files)
	}
	return s, err
}

func List(destDir string) ([]string, error) {
	entries, err := os.ReadDir(destDir)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if e.Type().IsRegular() && strings.HasPrefix(e.Name(), prefix) && (strings.HasSuffix(e.Name(), suffix) || strings.HasSuffix(e.Name(), suffix+encrypted)) {
			out = append(out, filepath.Join(destDir, e.Name()))
		}
	}
	sort.Strings(out)
	return out, nil
}

func Prune(destDir string, keep int) ([]string, error) {
	all, err := List(destDir)
	if err != nil || keep <= 0 || len(all) <= keep {
		return nil, err
	}
	old := all[:len(all)-keep]
	for _, p := range old {
		if err := os.Remove(p); err != nil {
			return nil, err
		}
	}
	return old, nil
}
