package filter

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"goyave.dev/goyave/v3/helper"
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
func (s *Sort) Scope(settings *Settings, modelIdentity *modelIdentity) func(*gorm.DB) *gorm.DB {
	if helper.ContainsStr(settings.FieldsBlacklist, s.Field) {
		return nil
	}
	_, ok := modelIdentity.Columns[s.Field]
	if !ok {
		return nil
	}

	return func(tx *gorm.DB) *gorm.DB {
		c := clause.OrderByColumn{
			Column: clause.Column{
				Table: modelIdentity.TableName,
				Name:  s.Field,
			},
			Desc: s.Order == SortDescending,
		}
		return tx.Order(c)
	}
}
