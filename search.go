package filter

import (
	"gorm.io/gorm"
)

// Search structured representation of a search query.
type Search struct {
	Query    string
	Operator *Operator
	Fields   []string
}

// Scope returns the GORM scopes with the search query.
func (s *Search) Scope(modelIdentity *modelIdentity) func(*gorm.DB) *gorm.DB {
	if len(s.Fields) == 0 {
		return nil // TODO test this
	}
	return func(tx *gorm.DB) *gorm.DB {
		searchQuery := tx.Session(&gorm.Session{NewDB: true})

		for _, field := range s.Fields {
			operator := s.Operator
			if operator == nil {
				operator = Operators["$cont"]
			}
			filter := &Filter{
				Field:    field,
				Operator: operator,
				Args:     []string{s.Query},
				Or:       true,
			}

			tableName := tx.Statement.Quote(modelIdentity.TableName) + "."
			searchQuery = operator.Function(searchQuery, filter, tableName+tx.Statement.Quote(field), modelIdentity.Columns[field].Type)
		}

		return tx.Where(searchQuery)
	}
}
