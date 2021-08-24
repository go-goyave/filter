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
		"$eq":  {Function: basicComparison("="), RequiredArguments: 1},
		"$ne":  {Function: basicComparison("<>"), RequiredArguments: 1},
		"$gt":  {Function: basicComparison(">"), RequiredArguments: 1},
		"$lt":  {Function: basicComparison("<"), RequiredArguments: 1},
		"$gte": {Function: basicComparison(">="), RequiredArguments: 1},
		"$lte": {Function: basicComparison("<="), RequiredArguments: 1},
		"$starts": {
			Function: func(tx *gorm.DB, filter *Filter) *gorm.DB {
				query := SQLEscape(tx, filter.Field) + "LIKE ?"
				value := helper.EscapeLike(filter.Args[0]) + "%"
				return filter.Where(tx, query, value)
			},
			RequiredArguments: 1,
		},
		"$ends": {
			Function: func(tx *gorm.DB, filter *Filter) *gorm.DB {
				query := SQLEscape(tx, filter.Field) + "LIKE ?"
				value := "%" + helper.EscapeLike(filter.Args[0])
				return filter.Where(tx, query, value)
			},
			RequiredArguments: 1,
		},
		"$cont": {
			Function: func(tx *gorm.DB, filter *Filter) *gorm.DB {
				query := SQLEscape(tx, filter.Field) + "LIKE ?"
				value := "%" + helper.EscapeLike(filter.Args[0]) + "%"
				return filter.Where(tx, query, value)
			},
			RequiredArguments: 1,
		},
		"$excl": {
			Function: func(tx *gorm.DB, filter *Filter) *gorm.DB {
				query := SQLEscape(tx, filter.Field) + "NOT LIKE ?"
				value := "%" + helper.EscapeLike(filter.Args[0]) + "%"
				return filter.Where(tx, query, value)
			},
			RequiredArguments: 1,
		},
		"$in":    {Function: multiComparison("IN"), RequiredArguments: 1},
		"$notin": {Function: multiComparison("NOT IN"), RequiredArguments: 1},
		"$isnull": {
			Function: func(tx *gorm.DB, filter *Filter) *gorm.DB {
				return filter.Where(tx, SQLEscape(tx, filter.Field)+" IS NULL")
			},
			RequiredArguments: 0,
		},
		"$notnull": {
			Function: func(tx *gorm.DB, filter *Filter) *gorm.DB {
				return filter.Where(tx, SQLEscape(tx, filter.Field)+" IS NOT NULL")
			},
			RequiredArguments: 0,
		},
		"$between": {
			Function: func(tx *gorm.DB, filter *Filter) *gorm.DB {
				query := SQLEscape(tx, filter.Field) + " BETWEEN ? AND ?"
				return filter.Where(tx, query, filter.Args[0], filter.Args[1])
			},
			RequiredArguments: 2,
		},
	}
)

func basicComparison(op string) func(tx *gorm.DB, filter *Filter) *gorm.DB {
	return func(tx *gorm.DB, filter *Filter) *gorm.DB {
		query := SQLEscape(tx, filter.Field) + op + " ?"
		return filter.Where(tx, query, filter.Args[0])
	}
}

func multiComparison(op string) func(tx *gorm.DB, filter *Filter) *gorm.DB {
	return func(tx *gorm.DB, filter *Filter) *gorm.DB {
		query := SQLEscape(tx, filter.Field) + op + " ?"
		return filter.Where(tx, query, filter.Args)
	}
}
