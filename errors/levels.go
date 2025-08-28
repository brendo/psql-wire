package errors

// Severity represents the severity of a thrown error. The possible error
// severities are ERROR, FATAL, or PANIC (in an error message), or WARNING,
// NOTICE, DEBUG, INFO, or LOG (in a notice message)
type Severity string

const (
	LevelError   Severity = "ERROR"
	LevelFatal   Severity = "FATAL"
	LevelPanic   Severity = "PANIC"
	LevelWarning Severity = "WARNING"
	LevelNotice  Severity = "NOTICE"
	LevelDebug   Severity = "DEBUG"
	LevelInfo    Severity = "INFO"
	LevelLog     Severity = "LOG"
)

// IsError returns true if the severity represents an error condition
// (ERROR, FATAL, or PANIC) that should abort the current transaction.
func (s Severity) IsError() bool {
	return s == LevelError || s == LevelFatal || s == LevelPanic
}
