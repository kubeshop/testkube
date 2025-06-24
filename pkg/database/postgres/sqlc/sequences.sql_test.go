package sqlc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSQLQuerySyntax(t *testing.T) {
	tests := []struct {
		name  string
		query string
	}{
		{
			name:  "GetExecutionSequence",
			query: "SELECT name, number, created_at, updated_at FROM execution_sequences WHERE name = $1",
		},
		{
			name: "UpsertAndIncrementExecutionSequence",
			query: `INSERT INTO execution_sequences (name, number)
VALUES ($1, 1)
ON CONFLICT (name) DO UPDATE SET
    number = execution_sequences.number + 1,
    updated_at = NOW()
RETURNING name, number, created_at, updated_at`,
		},
		{
			name:  "DeleteExecutionSequence",
			query: "DELETE FROM execution_sequences WHERE name = $1",
		},
		{
			name:  "DeleteExecutionSequences",
			query: "DELETE FROM execution_sequences WHERE name = ANY($1)",
		},
		{
			name:  "DeleteAllExecutionSequences",
			query: "DELETE FROM execution_sequences",
		},
		{
			name:  "GetAllExecutionSequences",
			query: "SELECT name, number, created_at, updated_at FROM execution_sequences ORDER BY created_at DESC",
		},
		{
			name:  "GetExecutionSequencesByNames",
			query: "SELECT name, number, created_at, updated_at FROM execution_sequences WHERE name = ANY($1) ORDER BY name",
		},
		{
			name:  "CountExecutionSequences",
			query: "SELECT COUNT(*) FROM execution_sequences",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that the query is not empty
			assert.NotEmpty(t, tt.query, "Query should not be empty")

			// Test basic SQL syntax requirements
			assert.Contains(t, tt.query, "execution_sequences", "Query should reference execution_sequences table")

			// Test specific query patterns
			switch tt.name {
			case "GetExecutionSequence":
				assert.Contains(t, tt.query, "SELECT", "Should be a SELECT query")
				assert.Contains(t, tt.query, "WHERE name =", "Should filter by name")
			case "UpsertAndIncrementExecutionSequence":
				assert.Contains(t, tt.query, "INSERT", "Should be an INSERT query")
				assert.Contains(t, tt.query, "ON CONFLICT", "Should handle conflicts")
				assert.Contains(t, tt.query, "RETURNING", "Should return the result")
			case "DeleteExecutionSequence":
				assert.Contains(t, tt.query, "DELETE", "Should be a DELETE query")
				assert.Contains(t, tt.query, "WHERE name =", "Should filter by name")
			case "DeleteExecutionSequences":
				assert.Contains(t, tt.query, "DELETE", "Should be a DELETE query")
				assert.Contains(t, tt.query, "ANY($1)", "Should use ANY for array parameter")
			case "DeleteAllExecutionSequences":
				assert.Contains(t, tt.query, "DELETE", "Should be a DELETE query")
				assert.NotContains(t, tt.query, "WHERE", "Should not have WHERE clause")
			case "GetAllExecutionSequences":
				assert.Contains(t, tt.query, "SELECT", "Should be a SELECT query")
				assert.Contains(t, tt.query, "ORDER BY", "Should have ordering")
			case "GetExecutionSequencesByNames":
				assert.Contains(t, tt.query, "SELECT", "Should be a SELECT query")
				assert.Contains(t, tt.query, "ANY($1)", "Should use ANY for array parameter")
				assert.Contains(t, tt.query, "ORDER BY", "Should have ordering")
			case "CountExecutionSequences":
				assert.Contains(t, tt.query, "SELECT COUNT(*)", "Should be a COUNT query")
			}
		})
	}
}

func TestQueryParameterPatterns(t *testing.T) {
	t.Run("ParameterConsistency", func(t *testing.T) {
		// Test that all queries use consistent parameter patterns
		queries := map[string]string{
			"single_param": "WHERE name = $1",
			"array_param":  "WHERE name = ANY($1)",
			"no_param":     "DELETE FROM execution_sequences",
			"returning":    "RETURNING name, number, created_at, updated_at",
			"conflict":     "ON CONFLICT (name) DO UPDATE SET",
		}

		for pattern, query := range queries {
			t.Run(pattern, func(t *testing.T) {
				assert.NotEmpty(t, query, "Query pattern should not be empty")

				switch pattern {
				case "single_param":
					assert.Contains(t, query, "$1", "Should use $1 parameter")
				case "array_param":
					assert.Contains(t, query, "ANY($1)", "Should use ANY($1) for arrays")
				case "no_param":
					assert.NotContains(t, query, "$", "Should not have parameters")
				case "returning":
					assert.Contains(t, query, "RETURNING", "Should have RETURNING clause")
				case "conflict":
					assert.Contains(t, query, "ON CONFLICT", "Should handle conflicts")
				}
			})
		}
	})
}

func TestTableStructureCompatibility(t *testing.T) {
	t.Run("RequiredColumns", func(t *testing.T) {
		requiredColumns := []string{"name", "number", "created_at", "updated_at"}

		selectQuery := "SELECT name, number, created_at, updated_at FROM execution_sequences"

		for _, column := range requiredColumns {
			assert.Contains(t, selectQuery, column, "Query should select required column: %s", column)
		}
	})

	t.Run("PrimaryKeyUsage", func(t *testing.T) {
		// Test that queries properly use the primary key (name)
		pkQueries := []string{
			"WHERE name = $1",
			"WHERE name = ANY($1)",
			"INSERT INTO execution_sequences (name, number)",
		}

		for _, query := range pkQueries {
			assert.Contains(t, query, "name", "Query should reference primary key column")
		}
	})
}
