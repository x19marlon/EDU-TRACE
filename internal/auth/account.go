package auth

import (
	"crypto/rand"
	"database/sql"
	"errors"
	"math/big"
	"strings"
)

var ErrInvalidRecovery = errors.New("el email o el código de recuperación no son correctos")

func newRecoveryCode() (string, error) {
	var b strings.Builder
	for i := 0; i < 12; i++ {
		if i > 0 && i%4 == 0 {
			b.WriteByte('-')
		}
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(joinAlphabet))))
		if err != nil {
			return "", err
		}
		b.WriteByte(joinAlphabet[n.Int64()])
	}
	return b.String(), nil
}

func NormalizeRecoveryCode(code string) string {
	return strings.NewReplacer(" ", "", "-", "").Replace(strings.ToUpper(strings.TrimSpace(code)))
}

func (s *Store) NewRecoveryCode(userID string) (string, error) {
	code, err := newRecoveryCode()
	if err != nil {
		return "", err
	}
	hash, err := HashPassword(NormalizeRecoveryCode(code))
	if err != nil {
		return "", err
	}
	if err := affected(s.db.Exec(`UPDATE users SET recovery_hash = ? WHERE id = ?`, hash, userID)); err != nil {
		return "", err
	}
	return code, nil
}

func (s *Store) ChangePassword(userID, current, next string) error {
	var hash string
	err := s.db.QueryRow(`SELECT password_hash FROM users WHERE id = ?`, userID).Scan(&hash)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if !CheckPassword(current, hash) {
		return ErrInvalidCredentials
	}
	newHash, err := HashPassword(next)
	if err != nil {
		return err
	}
	return affected(s.db.Exec(`UPDATE users SET password_hash = ? WHERE id = ?`, newHash, userID))
}

func (s *Store) DeleteOtherSessions(userID, keepToken string) error {
	_, err := s.db.Exec(`DELETE FROM sessions WHERE user_id = ? AND token_hash != ?`, userID, hashToken(keepToken))
	return err
}

func (s *Store) RecoverAccount(email, code, newPassword string) (*User, string, error) {
	u, err := scanUser(s.db.QueryRow(`SELECT `+userCols+` FROM users u WHERE u.email = ?`, normalizeEmail(email)))
	if errors.Is(err, sql.ErrNoRows) || (err == nil && u.RecoveryHash == "") {
		CheckPassword(code, dummyHash)
		return nil, "", ErrInvalidRecovery
	}
	if err != nil {
		return nil, "", err
	}
	if !CheckPassword(NormalizeRecoveryCode(code), u.RecoveryHash) {
		return nil, "", ErrInvalidRecovery
	}
	passwordHash, err := HashPassword(newPassword)
	if err != nil {
		return nil, "", err
	}
	next, err := newRecoveryCode()
	if err != nil {
		return nil, "", err
	}
	recoveryHash, err := HashPassword(NormalizeRecoveryCode(next))
	if err != nil {
		return nil, "", err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return nil, "", err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`UPDATE users SET password_hash = ?, recovery_hash = ? WHERE id = ?`, passwordHash, recoveryHash, u.ID); err != nil {
		return nil, "", err
	}
	if _, err := tx.Exec(`DELETE FROM sessions WHERE user_id = ?`, u.ID); err != nil {
		return nil, "", err
	}
	if err := tx.Commit(); err != nil {
		return nil, "", err
	}
	u.PasswordHash, u.RecoveryHash = passwordHash, recoveryHash
	return u, next, nil
}
