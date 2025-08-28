package wire

import (
	"github.com/jeroenrinzema/psql-wire/codes"
	psqlerr "github.com/jeroenrinzema/psql-wire/errors"
	"github.com/jeroenrinzema/psql-wire/pkg/buffer"
	"github.com/jeroenrinzema/psql-wire/pkg/types"
)

// errFieldType represents the error and notice message fields.
type errFieldType byte

// http://www.postgresql.org/docs/current/static/protocol-error-fields.html
//
//nolint:varcheck,deadcode
const (
	errFieldSeverity       errFieldType = 'S'
	errFieldMsgPrimary     errFieldType = 'M'
	errFieldSQLState       errFieldType = 'C'
	errFieldDetail         errFieldType = 'D'
	errFieldHint           errFieldType = 'H'
	errFieldSrcFile        errFieldType = 'F'
	errFieldSrcLine        errFieldType = 'L'
	errFieldSrcFunction    errFieldType = 'R'
	errFieldConstraintName errFieldType = 'n'
)

// WriteMessage writes a PostgreSQL wire protocol message (either ErrorResponse or NoticeResponse)
// to the client. This function handles both error and notice messages, which share the same
// wire format but differ in their message type and transaction behavior.
//
// For ERROR, FATAL, and PANIC severities, it sends an ErrorResponse (type 'E').
// For WARNING, NOTICE, DEBUG, INFO, and LOG severities, it sends a NoticeResponse (type 'N').
func WriteMessage(writer *buffer.Writer, msg psqlerr.Error) error {
	// Determine message type based on severity
	var messageType types.ServerMessage
	if msg.Severity.IsError() {
		messageType = types.ServerErrorResponse
	} else {
		messageType = types.ServerNoticeResponse
	}

	writer.Start(messageType)

	// Write all message fields (same for both errors and notices)
	writer.AddByte(byte(errFieldSeverity))
	writer.AddString(string(msg.Severity))
	writer.AddNullTerminate()

	writer.AddByte(byte(errFieldSQLState))
	writer.AddString(string(msg.Code))
	writer.AddNullTerminate()

	writer.AddByte(byte(errFieldMsgPrimary))
	writer.AddString(msg.Message)
	writer.AddNullTerminate()

	if msg.Detail != "" {
		writer.AddByte(byte(errFieldDetail))
		writer.AddString(msg.Detail)
		writer.AddNullTerminate()
	}

	if msg.Hint != "" {
		writer.AddByte(byte(errFieldHint))
		writer.AddString(msg.Hint)
		writer.AddNullTerminate()
	}

	if msg.ConstraintName != "" {
		writer.AddByte(byte(errFieldConstraintName))
		writer.AddString(msg.ConstraintName)
		writer.AddNullTerminate()
	}

	if msg.Source != nil {
		writer.AddByte(byte(errFieldSrcFile))
		writer.AddString(msg.Source.File)
		writer.AddNullTerminate()

		writer.AddByte(byte(errFieldSrcLine))
		writer.AddInt32(msg.Source.Line)
		writer.AddNullTerminate()

		writer.AddByte(byte(errFieldSrcFunction))
		writer.AddString(msg.Source.Function)
		writer.AddNullTerminate()
	}

	writer.AddNullTerminate()
	return writer.End()
}

// Message represents a notice or error message that can be sent to the client.
// It wraps a Go error and uses the existing error decorators from the errors package.
type Message struct {
	err error
}

// NewMessage creates a new message builder with the given severity and message text.
// The severity determines whether this will be sent as an ErrorResponse or NoticeResponse.
// For ERROR, FATAL, and PANIC severities, it sends an ErrorResponse.
// For WARNING, NOTICE, DEBUG, INFO, and LOG severities, it sends a NoticeResponse.
func NewMessage(severity psqlerr.Severity, message string) *Message {
	// Create a simple error with the message
	err := error(&simpleMessage{message: message})

	// Add severity
	err = psqlerr.WithSeverity(err, severity)

	// Add default code if not an error severity
	// (errors default to Uncategorized via GetCode)
	if !severity.IsError() {
		err = psqlerr.WithCode(err, codes.SuccessfulCompletion)
	}

	return &Message{err: err}
}

// WithCode sets the SQLSTATE code for the message.
func (m *Message) WithCode(code codes.Code) *Message {
	m.err = psqlerr.WithCode(m.err, code)
	return m
}

// WithDetail adds a detail message.
func (m *Message) WithDetail(detail string) *Message {
	m.err = psqlerr.WithDetail(m.err, detail)
	return m
}

// WithHint adds a hint message.
func (m *Message) WithHint(hint string) *Message {
	m.err = psqlerr.WithHint(m.err, hint)
	return m
}

// WithConstraint adds a constraint name.
func (m *Message) WithConstraint(constraint string) *Message {
	m.err = psqlerr.WithConstraintName(m.err, constraint)
	return m
}

// WithSource adds source location information.
func (m *Message) WithSource(file string, line int32, function string) *Message {
	m.err = psqlerr.WithSource(m.err, file, line, function)
	return m
}

// simpleMessage is a basic error type for notice/warning messages
type simpleMessage struct {
	message string
}

func (s *simpleMessage) Error() string {
	return s.message
}

// Send writes the message to the given buffer writer.
func (m *Message) Send(writer *buffer.Writer) error {
	return WriteMessage(writer, psqlerr.Flatten(m.err))
}
