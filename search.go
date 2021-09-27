package filter

import (
	"gorm.io/gorm"
)

// Search structured representation of a search query.
type Search struct {
	Fields []string
	Query  string
}

// Scopes returns the GORM scopes with the search query.
func (s *Search) Scopes(settings *Settings, modelIdentity *modelIdentity) func(*gorm.DB) *gorm.DB {
	return func(tx *gorm.DB) *gorm.DB {
		searchQuery := tx.Session(&gorm.Session{NewDB: true})

		for _, field := range settings.FieldsSearch {
			filter := &Filter{
				Field:    field,
				Operator: settings.SearchOperator,
				Args:     []string{s.Query},
				Or:       true,
			}

			if settings.SearchOperator != nil {
				searchQuery = settings.SearchOperator.Function(searchQuery, filter, field)
			} else {
				searchQuery = Operators["$cont"].Function(searchQuery, filter, field)
			}
		}

		return tx.Where(searchQuery)
	}
}
