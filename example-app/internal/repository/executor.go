// Package repository wires the example-app's custom repository wrappers around
// the skimatik-generated code in the generated subpackage and provides the
// transactional executor plumbing they share.
package repository

import (
	"context"

	"github.com/nhalm/pgxkit/v2"
)

// txKey is the context key under which an active transaction is stored.
type txKey struct{}

// executorFromContext returns the transaction stored on ctx, or db if none
// is present. Wrapper repositories pass the result as the executor argument
// to generated methods so callers can opt into transactional execution by
// attaching a transaction to the context (blueprint-vet R-5).
func executorFromContext(ctx context.Context, db pgxkit.Executor) pgxkit.Executor {
	if tx, ok := ctx.Value(txKey{}).(pgxkit.Executor); ok && tx != nil {
		return tx
	}
	return db
}
