package main

import (
	"context"
	"log"

	wire "github.com/jeroenrinzema/psql-wire"
	psqlerr "github.com/jeroenrinzema/psql-wire/errors"
	"github.com/lib/pq/oid"
)

func main() {
	log.Println("PostgreSQL server is up and running at [127.0.0.1:5432]")
	wire.ListenAndServe("127.0.0.1:5432", handler)
}

var table = wire.Columns{
	{
		Table: 0,
		Name:  "result",
		Oid:   oid.T_text,
		Width: 256,
	},
}

func handler(ctx context.Context, query string) (wire.PreparedStatements, error) {
	log.Println("incoming SQL query:", query)

	handle := func(ctx context.Context, writer wire.DataWriter, parameters []wire.Parameter) error {
		// Send a NOTICE message to the client
		// These are informational and don't abort the transaction
		err := writer.SendMessage(
			wire.NewMessage(psqlerr.LevelNotice, "This is a notice message").
				WithDetail("Additional information about the notice"),
		)
		if err != nil {
			return err
		}

		// Send a WARNING message to the client
		// These indicate potential issues but don't abort the transaction
		err = writer.SendMessage(
			wire.NewMessage(psqlerr.LevelWarning, "Deprecated feature used").
				WithHint("Consider using the new API instead"),
		)
		if err != nil {
			return err
		}

		// Return the actual query results
		writer.Row([]any{"Query completed successfully"})
		return writer.Complete("SELECT 1")
	}

	return wire.Prepared(wire.NewStatement(handle, wire.WithColumns(table))), nil
}
