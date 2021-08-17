package filter

import (
	"gorm.io/gorm"
	"goyave.dev/goyave/v3/helper"
)

type Operator struct {
	Function       func(tx *gorm.DB, filter *Filter) *gorm.DB
	RequiredParams uint8
}

var (
	Operators = map[string]*Operator{
		"$eq": {
			Function: func(tx *gorm.DB, filter *Filter) *gorm.DB {
				query := escape(tx, filter.Field) + "= ?"
				if filter.Or {
					return tx.Or(query, filter.Args[0])
				}
				return tx.Where(query, filter.Args[0])
			},
			RequiredParams: 1,
		},
		"$cont": {
			Function: func(tx *gorm.DB, filter *Filter) *gorm.DB {
				query := escape(tx, filter.Field) + "LIKE ?"
				value := "%" + helper.EscapeLike(filter.Args[0]) + "%"
				if filter.Or {
					return tx.Or(query, value)
				}
				return tx.Where(query, value)
			},
			RequiredParams: 1,
		},
	}
)
