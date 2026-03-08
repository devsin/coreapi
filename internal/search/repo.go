package search

import (
	"context"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	dbgen "github.com/devsin/coreapi/gen/db"
)

// Repository encapsulates database operations for search.
type Repository struct {
	pool *pgxpool.Pool
	q    *dbgen.Queries
}

// NewRepository creates a search repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool, q: dbgen.New(pool)}
}

// nonWord matches any character that is not alphanumeric or underscore.
var nonWord = regexp.MustCompile(`\W+`)

// buildPrefixQuery converts user input into a to_tsquery-compatible string with prefix matching.
func buildPrefixQuery(input string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return ""
	}

	cleaned := nonWord.ReplaceAllString(input, " ")
	words := strings.Fields(cleaned)
	if len(words) == 0 {
		return ""
	}

	terms := make([]string, 0, len(words))
	for _, w := range words {
		terms = append(terms, w+":*")
	}
	return strings.Join(terms, " & ")
}

// buildIlikePattern converts user input into an ILIKE pattern: "%input%".
func buildIlikePattern(input string) string {
	return "%" + strings.TrimSpace(input) + "%"
}

// SearchUsers performs full-text search + ILIKE fallback against users.
func (r *Repository) SearchUsers(ctx context.Context, query string, limit, offset int32) ([]dbgen.SearchUsersRow, int64, error) {
	tsQuery := buildPrefixQuery(query)
	ilikePattern := buildIlikePattern(query)

	rows, err := r.q.SearchUsers(ctx, dbgen.SearchUsersParams{
		Limit:      limit,
		Offset:     offset,
		Query:      tsQuery,
		IlikeQuery: ilikePattern,
	})
	if err != nil {
		return nil, 0, err
	}

	total, err := r.q.CountSearchUsers(ctx, dbgen.CountSearchUsersParams{
		Query:      tsQuery,
		IlikeQuery: ilikePattern,
	})
	if err != nil {
		return nil, 0, err
	}

	return rows, total, nil
}
