package filter

import (
	"gorm.io/gorm"
	"goyave.dev/goyave/v3/helper"
)

// Operator used by filters to build the SQL query.
// The operator function modifies the GORM statement (most of the time by adding
// a WHERE condition) then returns the modified statement.
// Operators may need arguments (e.g. "$eq", equals needs a value to compare the field to);
// RequiredArguments define the minimum number of arguments a client must send in order to
// use this operator in a filter. RequiredArguments is checked during Filter parsing.
type Operator struct {
	Function          func(tx *gorm.DB, filter *Filter) *gorm.DB
	RequiredArguments uint8
}

var (
	// Operators definitions. The key is the query representation of the operator, (e.g. "$eq").
	Operators = map[string]*Operator{
		"$eq": {
			Function: func(tx *gorm.DB, filter *Filter) *gorm.DB {
				query := SQLEscape(tx, filter.Field) + "= ?"
				if filter.Or {
					return tx.Or(query, filter.Args[0])
				}
				return tx.Where(query, filter.Args[0])
			},
			RequiredArguments: 1,
		},
		"$cont": {
			Function: func(tx *gorm.DB, filter *Filter) *gorm.DB {
				query := SQLEscape(tx, filter.Field) + "LIKE ?"
				value := "%" + helper.EscapeLike(filter.Args[0]) + "%"
				if filter.Or {
					return tx.Or(query, value)
				}
				return tx.Where(query, value)
			},
			RequiredArguments: 1,
		},
		// TODO add more operators
	}
)
