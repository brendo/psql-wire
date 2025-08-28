package wire

import (
	"bytes"
	"errors"
	"log/slog"
	"testing"

	"github.com/jeroenrinzema/psql-wire/codes"
	psqlerr "github.com/jeroenrinzema/psql-wire/errors"
	"github.com/jeroenrinzema/psql-wire/pkg/buffer"
	"github.com/jeroenrinzema/psql-wire/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteMessage(t *testing.T) {
	tests := []struct {
		name     string
		msg      psqlerr.Error
		wantType types.ServerMessage
	}{
		{
			name: "error message",
			msg: psqlerr.Error{
				Severity: psqlerr.LevelError,
				Code:     codes.Syntax,
				Message:  "Syntax error",
			},
			wantType: types.ServerErrorResponse,
		},
		{
			name: "notice message",
			msg: psqlerr.Error{
				Severity: psqlerr.LevelNotice,
				Code:     codes.SuccessfulCompletion,
				Message:  "Notice message",
			},
			wantType: types.ServerNoticeResponse,
		},
		{
			name: "warning with details",
			msg: psqlerr.Error{
				Severity: psqlerr.LevelWarning,
				Code:     codes.WarningDeprecatedFeature,
				Message:  "Deprecated feature",
				Detail:   "This feature will be removed in version 2.0",
				Hint:     "Use the new API instead",
			},
			wantType: types.ServerNoticeResponse,
		},
		{
			name: "fatal error",
			msg: psqlerr.Error{
				Severity: psqlerr.LevelFatal,
				Code:     codes.ConnectionFailure,
				Message:  "Connection lost",
			},
			wantType: types.ServerErrorResponse,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			writer := buffer.NewWriter(slog.Default(), &buf)

			err := WriteMessage(writer, tt.msg)
			require.NoError(t, err)

			data := buf.Bytes()
			require.NotEmpty(t, data)
			assert.Equal(t, byte(tt.wantType), data[0])

			// Verify message contents
			dataStr := string(data)
			assert.Contains(t, dataStr, tt.msg.Message)
			if tt.msg.Detail != "" {
				assert.Contains(t, dataStr, tt.msg.Detail)
			}
			if tt.msg.Hint != "" {
				assert.Contains(t, dataStr, tt.msg.Hint)
			}
		})
	}
}

func TestMessageBuilder(t *testing.T) {
	t.Run("Notice message", func(t *testing.T) {
		var buf bytes.Buffer
		writer := buffer.NewWriter(slog.Default(), &buf)

		msg := NewMessage(psqlerr.LevelNotice, "Test notice")
		err := msg.Send(writer)
		require.NoError(t, err)

		data := buf.Bytes()
		assert.Equal(t, byte(types.ServerNoticeResponse), data[0])
		assert.Contains(t, string(data), "Test notice")
	})

	t.Run("Warning with details", func(t *testing.T) {
		var buf bytes.Buffer
		writer := buffer.NewWriter(slog.Default(), &buf)

		msg := NewMessage(psqlerr.LevelWarning, "Deprecated feature").
			WithDetail("This will be removed").
			WithHint("Use the new version").
			WithCode(codes.WarningDeprecatedFeature)

		err := msg.Send(writer)
		require.NoError(t, err)

		data := buf.Bytes()
		assert.Equal(t, byte(types.ServerNoticeResponse), data[0])
		dataStr := string(data)
		assert.Contains(t, dataStr, "Deprecated feature")
		assert.Contains(t, dataStr, "This will be removed")
		assert.Contains(t, dataStr, "Use the new version")
		assert.Contains(t, dataStr, string(codes.WarningDeprecatedFeature))
	})

	t.Run("Info message", func(t *testing.T) {
		var buf bytes.Buffer
		writer := buffer.NewWriter(slog.Default(), &buf)

		msg := NewMessage(psqlerr.LevelInfo, "Processing started")
		err := msg.Send(writer)
		require.NoError(t, err)

		data := buf.Bytes()
		assert.Equal(t, byte(types.ServerNoticeResponse), data[0])
		assert.Contains(t, string(data), "Processing started")
	})

	t.Run("Debug message", func(t *testing.T) {
		var buf bytes.Buffer
		writer := buffer.NewWriter(slog.Default(), &buf)

		msg := NewMessage(psqlerr.LevelDebug, "Debug information")
		err := msg.Send(writer)
		require.NoError(t, err)

		data := buf.Bytes()
		assert.Equal(t, byte(types.ServerNoticeResponse), data[0])
		assert.Contains(t, string(data), "Debug information")
	})

	t.Run("Log message", func(t *testing.T) {
		var buf bytes.Buffer
		writer := buffer.NewWriter(slog.Default(), &buf)

		msg := NewMessage(psqlerr.LevelLog, "Server log entry")
		err := msg.Send(writer)
		require.NoError(t, err)

		data := buf.Bytes()
		assert.Equal(t, byte(types.ServerNoticeResponse), data[0])
		assert.Contains(t, string(data), "Server log entry")
	})

	t.Run("Message with constraint", func(t *testing.T) {
		var buf bytes.Buffer
		writer := buffer.NewWriter(slog.Default(), &buf)

		msg := NewMessage(psqlerr.LevelWarning, "Constraint violation warning").
			WithConstraint("check_positive_value")

		err := msg.Send(writer)
		require.NoError(t, err)

		data := buf.Bytes()
		assert.Contains(t, string(data), "check_positive_value")
	})

	t.Run("Message with source", func(t *testing.T) {
		var buf bytes.Buffer
		writer := buffer.NewWriter(slog.Default(), &buf)

		msg := NewMessage(psqlerr.LevelNotice, "Source location test").
			WithSource("test.go", 42, "TestFunction")

		err := msg.Send(writer)
		require.NoError(t, err)

		data := buf.Bytes()
		dataStr := string(data)
		assert.Contains(t, dataStr, "test.go")
		assert.Contains(t, dataStr, "TestFunction")
	})
}

func TestDefaultCodeConsistency(t *testing.T) {
	// Verify that NewMessage defaults match what Flatten would produce
	t.Run("Error without code defaults to Uncategorized", func(t *testing.T) {
		// Create an error without a code
		err := errors.New("test error")
		err = psqlerr.WithSeverity(err, psqlerr.LevelError)
		flattened := psqlerr.Flatten(err)

		// Create a message with the same severity
		msg := NewMessage(psqlerr.LevelError, "test error")
		msgFlattened := psqlerr.Flatten(msg.err)

		// They should have the same default code
		assert.Equal(t, flattened.Code, msgFlattened.Code)
		assert.Equal(t, codes.Uncategorized, msgFlattened.Code)
	})

	t.Run("Notice defaults to SuccessfulCompletion", func(t *testing.T) {
		msg := NewMessage(psqlerr.LevelNotice, "test notice")
		msgFlattened := psqlerr.Flatten(msg.err)
		assert.Equal(t, codes.SuccessfulCompletion, msgFlattened.Code)
	})
}

func TestMessageChaining(t *testing.T) {
	var buf bytes.Buffer
	writer := buffer.NewWriter(slog.Default(), &buf)

	// Test that all builder methods can be chained
	msg := NewMessage(psqlerr.LevelWarning, "Warning message").
		WithCode(codes.Warning).
		WithDetail("Detailed information").
		WithHint("Helpful suggestion").
		WithConstraint("some_constraint").
		WithSource("file.go", 100, "function")

	err := msg.Send(writer)
	require.NoError(t, err)

	data := buf.String()
	assert.Contains(t, data, "Warning message")
	assert.Contains(t, data, "Detailed information")
	assert.Contains(t, data, "Helpful suggestion")
	assert.Contains(t, data, "some_constraint")
	assert.Contains(t, data, "file.go")
}

func TestMessageSeverityTypes(t *testing.T) {
	tests := []struct {
		name        string
		severity    psqlerr.Severity
		wantType    types.ServerMessage
		wantDefCode codes.Code
	}{
		{"debug", psqlerr.LevelDebug, types.ServerNoticeResponse, codes.SuccessfulCompletion},
		{"log", psqlerr.LevelLog, types.ServerNoticeResponse, codes.SuccessfulCompletion},
		{"info", psqlerr.LevelInfo, types.ServerNoticeResponse, codes.SuccessfulCompletion},
		{"notice", psqlerr.LevelNotice, types.ServerNoticeResponse, codes.SuccessfulCompletion},
		{"warning", psqlerr.LevelWarning, types.ServerNoticeResponse, codes.SuccessfulCompletion},
		{"error", psqlerr.LevelError, types.ServerErrorResponse, codes.Uncategorized},
		{"fatal", psqlerr.LevelFatal, types.ServerErrorResponse, codes.Uncategorized},
		{"panic", psqlerr.LevelPanic, types.ServerErrorResponse, codes.Uncategorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			writer := buffer.NewWriter(slog.Default(), &buf)

			msg := NewMessage(tt.severity, "Test message")
			err := msg.Send(writer)
			require.NoError(t, err)

			data := buf.Bytes()
			assert.Equal(t, byte(tt.wantType), data[0],
				"Severity %s should produce message type %v", tt.severity, tt.wantType)

			// Verify default code is correct
			assert.Contains(t, string(data), string(tt.wantDefCode),
				"Severity %s should have default code %s", tt.severity, tt.wantDefCode)
		})
	}
}
