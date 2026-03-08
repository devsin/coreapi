package contact

import (
	"context"

	dbgen "github.com/devsin/coreapi/gen/db"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository wraps the sqlc-generated queries for contact messages.
type Repository struct {
	q *dbgen.Queries
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{q: dbgen.New(pool)}
}

func (r *Repository) Create(ctx context.Context, p dbgen.CreateContactMessageParams) (dbgen.ContactMessage, error) {
	return r.q.CreateContactMessage(ctx, p)
}
