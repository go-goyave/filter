package filter

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

// Sort structured representation of a sort query.
type Sort struct {
	Field string
	Order SortOrder
}

// SortOrder the allowed strings for SQL "ORDER BY" clause.
type SortOrder string

const (
	// SortAscending "ORDER BY column ASC"
	SortAscending SortOrder = "ASC"
	// SortDescending "ORDER BY column DESC"
	SortDescending SortOrder = "DESC"
)

// Scope returns the GORM scope to use in order to apply sorting.
func (s *Sort) Scope(settings *Settings, schema *schema.Schema) func(*gorm.DB) *gorm.DB {
	field, sch, joinName := getField(s.Field, schema, &settings.Blacklist)
	if field == nil {
		return nil
	}

	computed := field.StructField.Tag.Get("computed")

	return func(tx *gorm.DB) *gorm.DB {
		if joinName != "" {
			if err := tx.Statement.Parse(tx.Statement.Model); err != nil {
				tx.AddError(err)
				return tx
			}
			tx = join(tx, joinName, schema)
		}

		table := tableFromJoinName(sch.Table, joinName)
		var column clause.Column
		if computed != "" {
			column = clause.Column{
				Raw:  true,
				Name: fmt.Sprintf("(%s)", strings.ReplaceAll(computed, clause.CurrentTable, tx.Statement.Quote(table))),
			}
		} else {
			column = clause.Column{
				Table: tableFromJoinName(sch.Table, joinName),
				Name:  field.DBName,
			}
		}
		c := clause.OrderByColumn{
			Column: column,
			Desc:   s.Order == SortDescending,
		}
		return tx.Order(c)
	}
}
