package filter

import (
	"gorm.io/gorm"
)

type Search struct {
	Fields []string
	Query  string
}

func (s *Search) Scopes(settings *Settings, modelIdentity *modelIdentity) func(*gorm.DB) *gorm.DB {
	var fields []string

	// Remove columns that not exist in the table
	for _, field := range s.Fields {
		_, ok := modelIdentity.Columns[field]
		if ok {
			fields = append(fields, field)
		}
	}

	if len(fields) == 0 {
		return nil
	}

	return func(tx *gorm.DB) *gorm.DB {
		searchQuery := tx.Session(&gorm.Session{NewDB: true})

		for _, field := range fields {
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
