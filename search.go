package filter

import (
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// Search structured representation of a search query.
type Search struct {
	Query    string
	Operator *Operator
	Fields   []string
}

// Scope returns the GORM scopes with the search query.
func (s *Search) Scope(schema *schema.Schema) func(*gorm.DB) *gorm.DB {
	if len(s.Fields) == 0 {
		return nil
	}

	return func(tx *gorm.DB) *gorm.DB {
		searchQuery := tx.Session(&gorm.Session{NewDB: true})

		for _, field := range s.Fields {

			filter := &Filter{
				Field:    field,
				Operator: s.Operator,
				Args:     []string{s.Query},
				Or:       true,
			}

			tableName := tx.Statement.Quote(schema.Table) + "."
			searchQuery = s.Operator.Function(searchQuery, filter, tableName+tx.Statement.Quote(field), schema.FieldsByDBName[field].DataType)
		}

		return tx.Where(searchQuery)
	}
}
