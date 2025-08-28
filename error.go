package wire

import (
	"github.com/jeroenrinzema/psql-wire/codes"
	psqlerr "github.com/jeroenrinzema/psql-wire/errors"
	"github.com/jeroenrinzema/psql-wire/pkg/buffer"
	"github.com/jeroenrinzema/psql-wire/pkg/types"
)

// ErrorCode writes an error message as response to a command with the given
// severity and error message. A ready for query message is written back to the
// client once the error has been written indicating the end of a command cycle.
// https://www.postgresql.org/docs/current/static/protocol-error-fields.html
func ErrorCode(writer *buffer.Writer, err error) error {
	desc := psqlerr.Flatten(err)

	if err := WriteMessage(writer, desc); err != nil {
		return err
	}

	// NOTE: we are writing a ready for query message to indicate the end of a
	// command cycle. However, for authentication failures, we skip this
	// because the connection will be terminated.
	if desc.Code == codes.InvalidPassword {
		return nil
	}

	return readyForQuery(writer, types.ServerIdle)
}
