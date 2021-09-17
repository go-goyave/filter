package filter

import (
	"gorm.io/gorm"
	"goyave.dev/goyave/v4/helper"
	"strings"
)

type Search struct {
	Field string
	Query string
}

// Scope returns the GORM scope to use in order to apply sorting.
func (s *Search) Scope(settings *Settings, modelIdentity *modelIdentity) func(*gorm.DB) *gorm.DB {
	if !helper.ContainsStr(settings.FieldsSearch, s.Field) {
		return nil
	}

	_, ok := modelIdentity.Columns[s.Field]
	if !ok {
		return nil
	}

	return func(tx *gorm.DB) *gorm.DB {
		tableWithColumn := []string{tx.Statement.Quote(modelIdentity.TableName), tx.Statement.Quote(s.Field)}

		return tx.Or("(lower(CAST("+ strings.Join(tableWithColumn, ".") + " AS TEXT))) LIKE lower(?)", "%" + helper.EscapeLike(s.Query) + "%")
	}
}