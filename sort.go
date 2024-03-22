package filter

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

// Sort structured representation of a sort query.
// The generic parameter is the type pointer type of the model.
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
// If caseInsensitive is true, the column is wrapped in a `LOWER()` function.
func (s *Sort) Scope(blacklist Blacklist, schema *schema.Schema, caseInsensitive bool) func(*gorm.DB) *gorm.DB {
	field, sch, joinName := getField(s.Field, schema, &blacklist)
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
		} else if caseInsensitive && getDataType(field) == DataTypeText {
			column = clause.Column{
				Raw:  true,
				Name: fmt.Sprintf("LOWER(%s.%s)", tx.Statement.Quote(table), tx.Statement.Quote(field.DBName)),
			}
		} else {
			column = clause.Column{
				Table: table,
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
