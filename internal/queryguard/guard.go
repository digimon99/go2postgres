// Package queryguard splits SQL scripts into individual statements and validates
// them against a blocklist of dangerous commands.
package queryguard

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
)

// ErrBlockedStatement is returned when a statement is on the blocklist.
var ErrBlockedStatement = errors.New("statement not permitted")

// ErrReadOnlyViolation is returned when a write statement is sent to a readonly key.
var ErrReadOnlyViolation = errors.New("write statement not permitted for read-only key")

// blockedPrefixes are SQL command prefixes that are never allowed.
var blockedPrefixes = []string{
	"COPY",
	"ALTER SYSTEM",
	"CREATE EXTENSION",
	"DROP DATABASE",
	"DROP ROLE",
	"DROP USER",
	"DROP TABLESPACE",
	"REASSIGN OWNED",
	"DROP OWNED",
	"LOAD",
	"CLUSTER",           // can take exclusive locks
	"VACUUM",
	"REINDEX",
	"CHECKPOINT",
	"PG_READ_FILE",
	"PG_WRITE_FILE",
	"PG_LS_DIR",
	"LO_IMPORT",
	"LO_EXPORT",
	"PG_SLEEP",
}

// readOnlyAllowed are command prefixes allowed for readonly keys.
var readOnlyAllowed = []string{
	"SELECT",
	"WITH",      // CTEs that begin a SELECT
	"EXPLAIN",
	"SHOW",
	"TABLE",     // shorthand for SELECT * FROM
}

// SplitStatements splits a SQL script into individual non-empty statements using
// a minimal state machine (handles single-quoted strings, double-quoted identifiers,
// line comments, block comments, and dollar-quoted strings).
func SplitStatements(sql string) []string {
	var statements []string
	var buf strings.Builder

	type state int
	const (
		normal state = iota
		inSingleQuote
		inDoubleQuote
		inLineComment
		inBlockComment
		inDollarQuote
	)

	st := normal
	dollarTag := ""
	i := 0
	runes := []rune(sql)
	n := len(runes)

	flush := func() {
		s := strings.TrimSpace(buf.String())
		if s != "" {
			statements = append(statements, s)
		}
		buf.Reset()
	}

	for i < n {
		ch := runes[i]

		switch st {
		case inLineComment:
			if ch == '\n' {
				st = normal
			}
			buf.WriteRune(ch)
		case inBlockComment:
			buf.WriteRune(ch)
			if ch == '*' && i+1 < n && runes[i+1] == '/' {
				buf.WriteRune(runes[i+1])
				i += 2
				st = normal
				continue
			}
		case inDollarQuote:
			buf.WriteRune(ch)
			// check for closing dollar tag
			if ch == '$' {
				end := i + 1 + len(dollarTag)
				if end < n && string(runes[i+1:end+1]) == dollarTag+"$" {
					for j := 0; j <= len(dollarTag); j++ {
						buf.WriteRune(runes[i+1+j])
					}
					i = end + 1
					st = normal
					continue
				}
			}
		case inSingleQuote:
			buf.WriteRune(ch)
			if ch == '\'' {
				if i+1 < n && runes[i+1] == '\'' {
					// escaped quote
					buf.WriteRune(runes[i+1])
					i += 2
					continue
				}
				st = normal
			}
		case inDoubleQuote:
			buf.WriteRune(ch)
			if ch == '"' {
				st = normal
			}
		case normal:
			switch {
			case ch == ';':
				buf.WriteRune(ch)
				flush()
			case ch == '-' && i+1 < n && runes[i+1] == '-':
				buf.WriteRune(ch)
				st = inLineComment
			case ch == '/' && i+1 < n && runes[i+1] == '*':
				buf.WriteRune(ch)
				buf.WriteRune(runes[i+1])
				i += 2
				st = inBlockComment
				continue
			case ch == '\'':
				buf.WriteRune(ch)
				st = inSingleQuote
			case ch == '"':
				buf.WriteRune(ch)
				st = inDoubleQuote
			case ch == '$':
				// possible dollar-quoting: look ahead for tag
				j := i + 1
				for j < n && (unicode.IsLetter(runes[j]) || unicode.IsDigit(runes[j]) || runes[j] == '_') {
					j++
				}
				if j < n && runes[j] == '$' {
					tag := string(runes[i+1 : j])
					dollarTag = tag
					for k := i; k <= j; k++ {
						buf.WriteRune(runes[k])
					}
					i = j + 1
					st = inDollarQuote
					continue
				}
				buf.WriteRune(ch)
			default:
				buf.WriteRune(ch)
			}
		}
		i++
	}
	flush() // last statement without trailing semicolon
	return statements
}

// commandPrefix extracts the first keyword token from a statement (uppercased).
// e.g. "  select * from t" → "SELECT"
func commandPrefix(stmt string) string {
	s := strings.TrimLeftFunc(stmt, unicode.IsSpace)
	// skip comment lines at the start
	for strings.HasPrefix(s, "--") {
		idx := strings.IndexByte(s, '\n')
		if idx < 0 {
			return ""
		}
		s = strings.TrimLeftFunc(s[idx+1:], unicode.IsSpace)
	}
	// For block comments at start
	for strings.HasPrefix(s, "/*") {
		end := strings.Index(s, "*/")
		if end < 0 {
			return ""
		}
		s = strings.TrimLeftFunc(s[end+2:], unicode.IsSpace)
	}

	end := strings.IndexFunc(s, func(r rune) bool {
		return unicode.IsSpace(r) || r == '(' || r == ';'
	})
	if end < 0 {
		return strings.ToUpper(s)
	}
	return strings.ToUpper(s[:end])
}

// twoWordPrefix returns the first two keywords, e.g. "ALTER SYSTEM".
func twoWordPrefix(stmt string) string {
	s := strings.TrimLeftFunc(stmt, unicode.IsSpace)
	fields := strings.Fields(s)
	if len(fields) >= 2 {
		return strings.ToUpper(fields[0]) + " " + strings.ToUpper(fields[1])
	}
	return commandPrefix(stmt)
}

// CheckBlocked returns an error if any statement in stmts is on the blocklist.
// It returns the index of the offending statement and the reason.
func CheckBlocked(stmts []string) (int, error) {
	for i, stmt := range stmts {
		upper := strings.ToUpper(strings.TrimSpace(stmt))
		if upper == "" || upper == ";" {
			continue
		}
		for _, blocked := range blockedPrefixes {
			// Match full keyword (word boundary check via space/( after prefix)
			if strings.HasPrefix(upper, blocked) {
				rest := upper[len(blocked):]
				if rest == "" || rest == ";" || strings.IndexFunc(rest, func(r rune) bool {
					return unicode.IsSpace(r) || r == '(' || r == ';'
				}) == 0 {
					return i, fmt.Errorf("%w: %s", ErrBlockedStatement, blocked)
				}
			}
		}
	}
	return -1, nil
}

// CheckReadOnly returns an error if any statement performs a write operation.
// For use with APIKeyTypeReadOnly keys.
func CheckReadOnly(stmts []string) (int, error) {
	for i, stmt := range stmts {
		trimmed := strings.TrimSpace(stmt)
		if trimmed == "" || trimmed == ";" {
			continue
		}
		cmd := commandPrefix(trimmed)
		allowed := false
		for _, pfx := range readOnlyAllowed {
			if cmd == pfx {
				allowed = true
				break
			}
		}
		if !allowed {
			return i, fmt.Errorf("%w: %s", ErrReadOnlyViolation, cmd)
		}
	}
	return -1, nil
}
