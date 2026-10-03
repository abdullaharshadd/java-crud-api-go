// Package user contains the User domain model, its persistence schema and
// related helpers for the Smart Contact Manager service.
package user

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Table and column names. Spring Boot's default physical naming strategy
// lower-cases the explicit @Table/@Column names from the JPA entity, so
// "USER"/"User_id" become "user"/"user_id" in MySQL.
const (
	TableName      = "user"
	ColumnID       = "user_id"
	ColumnName     = "user_name"
	ColumnEmail    = "user_email"
	ColumnPassword = "user_password"
	ColumnRole     = "user_role"
	ColumnAbout    = "user_about"
)

// NameBlankMessage is the @NotBlank message declared on the name field.
const NameBlankMessage = "please Add the department Name"

// ErrValidation is returned (wrapped) by Validate when a constraint fails.
var ErrValidation = errors.New("validation failed")

// ErrIDOutOfRange is returned when a decoded id does not fit into a Java int (int32).
var ErrIDOutOfRange = errors.New("user id out of int32 range")

// User mirrors the JPA User entity. String fields are pointers so that a
// Java null stays distinct from an empty string (both in JSON and in SQL).
//
// MIGRATION_NOTE: the plaintext password is stored and serialised exactly as
// the Java app does (parity). This is a security issue that must be fixed
// after parity is confirmed.
type User struct {
	ID       int     `json:"id"`
	Name     *string `json:"name"`
	Email    *string `json:"email"`
	Password *string `json:"password"`
	Role     *string `json:"role"`
	About    *string `json:"about"`
}

// NewUser returns an empty User with all fields at their zero values
// (equivalent to the Lombok @NoArgsConstructor).
func NewUser() *User {
	return &User{}
}

// NewUserWithAll returns a User with every field set, in declaration order
// (equivalent to the Lombok @AllArgsConstructor).
func NewUserWithAll(id int, name, email, password, role, about *string) *User {
	return &User{
		ID:       id,
		Name:     name,
		Email:    email,
		Password: password,
		Role:     role,
		About:    about,
	}
}

// Builder is a fluent builder for User (equivalent to Lombok @Builder).
type Builder struct {
	u User
}

// NewBuilder returns a new, empty User builder.
func NewBuilder() *Builder {
	return &Builder{}
}

// ID sets the id.
func (b *Builder) ID(id int) *Builder { b.u.ID = id; return b }

// Name sets the name.
func (b *Builder) Name(name *string) *Builder { b.u.Name = name; return b }

// Email sets the email.
func (b *Builder) Email(email *string) *Builder { b.u.Email = email; return b }

// Password sets the password.
func (b *Builder) Password(password *string) *Builder { b.u.Password = password; return b }

// Role sets the role.
func (b *Builder) Role(role *string) *Builder { b.u.Role = role; return b }

// About sets the about text.
func (b *Builder) About(about *string) *Builder { b.u.About = about; return b }

// Build returns a new User holding the builder's values.
func (b *Builder) Build() *User {
	u := b.u
	return &u
}

// GetID returns the id.
func (u *User) GetID() int { return u.ID }

// SetID sets the id.
func (u *User) SetID(id int) { u.ID = id }

// GetName returns the name (nil if unset).
func (u *User) GetName() *string { return u.Name }

// SetName sets the name.
func (u *User) SetName(name *string) { u.Name = name }

// GetEmail returns the email (nil if unset).
func (u *User) GetEmail() *string { return u.Email }

// SetEmail sets the email.
func (u *User) SetEmail(email *string) { u.Email = email }

// GetPassword returns the password (nil if unset).
func (u *User) GetPassword() *string { return u.Password }

// SetPassword sets the password.
func (u *User) SetPassword(password *string) { u.Password = password }

// GetRole returns the role (nil if unset).
func (u *User) GetRole() *string { return u.Role }

// SetRole sets the role.
func (u *User) SetRole(role *string) { u.Role = role }

// GetAbout returns the about text (nil if unset).
func (u *User) GetAbout() *string { return u.About }

// SetAbout sets the about text.
func (u *User) SetAbout(about *string) { u.About = about }

func strPtrEqual(a, b *string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

// Equal reports value equality over all six fields (Lombok @Data equals).
func (u *User) Equal(other *User) bool {
	if u == nil || other == nil {
		return u == nil && other == nil
	}
	return u.ID == other.ID &&
		strPtrEqual(u.Name, other.Name) &&
		strPtrEqual(u.Email, other.Email) &&
		strPtrEqual(u.Password, other.Password) &&
		strPtrEqual(u.Role, other.Role) &&
		strPtrEqual(u.About, other.About)
}

func strOrNull(s *string) string {
	if s == nil {
		return "null"
	}
	return *s
}

// String renders the user in Lombok @Data toString format.
func (u *User) String() string {
	if u == nil {
		return "null"
	}
	return fmt.Sprintf("User(id=%d, name=%s, email=%s, password=%s, role=%s, about=%s)",
		u.ID, strOrNull(u.Name), strOrNull(u.Email), strOrNull(u.Password),
		strOrNull(u.Role), strOrNull(u.About))
}

// Validate reproduces the Bean Validation constraints of the entity: only
// @NotBlank on name. The returned error wraps ErrValidation.
func (u *User) Validate() error {
	if u.Name == nil || strings.TrimSpace(*u.Name) == "" {
		return fmt.Errorf("%w: name: %s", ErrValidation, NameBlankMessage)
	}
	return nil
}

// UnmarshalJSON decodes a User the way Jackson would: a null id becomes 0,
// numeric strings and floats are coerced, and ids outside the int32 range
// yield an error wrapping ErrIDOutOfRange (callers should answer 400).
func (u *User) UnmarshalJSON(data []byte) error {
	type alias User
	aux := struct {
		ID json.RawMessage `json:"id"`
		*alias
	}{alias: (*alias)(u)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	id, err := parseID(aux.ID)
	if err != nil {
		return err
	}
	u.ID = id
	return nil
}

func parseID(raw json.RawMessage) (int, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return 0, nil
	}
	text := string(raw)
	if raw[0] == '"' {
		if err := json.Unmarshal(raw, &text); err != nil {
			return 0, fmt.Errorf("decode user id: %w", err)
		}
		text = strings.TrimSpace(text)
		if text == "" {
			return 0, nil
		}
	}
	if n, err := strconv.ParseInt(text, 10, 64); err == nil {
		if n < math.MinInt32 || n > math.MaxInt32 {
			return 0, fmt.Errorf("%w: %d", ErrIDOutOfRange, n)
		}
		return int(n), nil
	}
	f, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return 0, fmt.Errorf("decode user id %q: %w", text, err)
	}
	if math.IsNaN(f) || f < math.MinInt32 || f > math.MaxInt32 {
		return 0, fmt.Errorf("%w: %s", ErrIDOutOfRange, text)
	}
	return int(f), nil
}

// schemaStatements is the DDL Hibernate (ddl-auto=update, MySQL dialect,
// GenerationType.AUTO) would produce for this entity: a hibernate_sequence
// table for id generation and the user table itself.
var schemaStatements = []string{
	`CREATE TABLE IF NOT EXISTS hibernate_sequence (
		next_val BIGINT
	) ENGINE=InnoDB`,
	"CREATE TABLE IF NOT EXISTS `user` (" +
		"user_id INT NOT NULL, " +
		"user_about VARCHAR(500), " +
		"user_email VARCHAR(255), " +
		"user_name VARCHAR(255), " +
		"user_password VARCHAR(255), " +
		"user_role VARCHAR(255), " +
		"PRIMARY KEY (user_id), " +
		"CONSTRAINT uk_user_email UNIQUE (user_email)" +
		") ENGINE=InnoDB",
}

// EnsureSchema creates the user table and the hibernate_sequence table if
// they do not exist, and seeds the sequence like Hibernate does. It must be
// called once at application start-up.
func EnsureSchema(ctx context.Context, db *sql.DB) error {
	for _, stmt := range schemaStatements {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("ensure user schema: %w", err)
		}
	}
	var n int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM hibernate_sequence").Scan(&n); err != nil {
		return fmt.Errorf("check hibernate_sequence: %w", err)
	}
	if n == 0 {
		if _, err := db.ExecContext(ctx, "INSERT INTO hibernate_sequence (next_val) VALUES (1)"); err != nil {
			return fmt.Errorf("seed hibernate_sequence: %w", err)
		}
	}
	return nil
}
