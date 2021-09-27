package filter

import (
	"fmt"

	"gorm.io/gorm"
	"goyave.dev/goyave/v4/helper"
)

// Operator used by filters to build the SQL query.
// The operator function modifies the GORM statement (most of the time by adding
// a WHERE condition) then returns the modified statement.
// Operators may need arguments (e.g. "$eq", equals needs a value to compare the field to);
// RequiredArguments define the minimum number of arguments a client must send in order to
// use this operator in a filter. RequiredArguments is checked during Filter parsing.
type Operator struct {
	Name              string
	Function          func(tx *gorm.DB, filter *Filter, column string) *gorm.DB
	RequiredArguments uint8
}

var (
	// Operators definitions. The key is the query representation of the operator, (e.g. "$eq").
	Operators = map[string]*Operator{
		"$eq": {
			Name:              "$eq",
			Function:          basicComparison("="),
			RequiredArguments: 1,
		},
		"$ne": {
			Name:              "$ne",
			Function:          basicComparison("<>"),
			RequiredArguments: 1,
		},
		"$gt": {
			Name:              "$gt",
			Function:          basicComparison(">"),
			RequiredArguments: 1,
		},
		"$lt": {
			Name:              "$lt",
			Function:          basicComparison("<"),
			RequiredArguments: 1,
		},
		"$gte": {
			Name:              "$gte",
			Function:          basicComparison(">="),
			RequiredArguments: 1,
		},
		"$lte": {
			Name:              "$lte",
			Function:          basicComparison("<="),
			RequiredArguments: 1,
		},
		"$starts": {
			Name: "$starts",
			Function: func(tx *gorm.DB, filter *Filter, column string) *gorm.DB {
				query := column + " LIKE ?"
				value := helper.EscapeLike(filter.Args[0]) + "%"
				return filter.Where(tx, query, value)
			},
			RequiredArguments: 1,
		},
		"$ends": {
			Name: "$ends",
			Function: func(tx *gorm.DB, filter *Filter, column string) *gorm.DB {
				query := column + " LIKE ?"
				value := "%" + helper.EscapeLike(filter.Args[0])
				return filter.Where(tx, query, value)
			},
			RequiredArguments: 1,
		},
		"$cont": {
			Name: "$cont",
			Function: func(tx *gorm.DB, filter *Filter, column string) *gorm.DB {
				query := column + " LIKE ?"
				value := "%" + helper.EscapeLike(filter.Args[0]) + "%"
				return filter.Where(tx, query, value)
			},
			RequiredArguments: 1,
		},
		"$excl": {
			Name: "$excl",
			Function: func(tx *gorm.DB, filter *Filter, column string) *gorm.DB {
				query := column + " NOT LIKE ?"
				value := "%" + helper.EscapeLike(filter.Args[0]) + "%"
				return filter.Where(tx, query, value)
			},
			RequiredArguments: 1,
		},
		"$in": {
			Name:              "$in",
			Function:          multiComparison("IN"),
			RequiredArguments: 1,
		},
		"$notin": {
			Name:              "$notin",
			Function:          multiComparison("NOT IN"),
			RequiredArguments: 1,
		},
		"$isnull": {
			Name: "$isnull",
			Function: func(tx *gorm.DB, filter *Filter, column string) *gorm.DB {
				return filter.Where(tx, column+" IS NULL")
			},
			RequiredArguments: 0,
		},
		"$istrue": {
			Name: "$istrue",
			Function: func(tx *gorm.DB, filter *Filter, column string) *gorm.DB {
				return filter.Where(tx, column+" IS TRUE")
			},
			RequiredArguments: 0,
		},
		"$isfalse": {
			Name: "$isfalse",
			Function: func(tx *gorm.DB, filter *Filter, column string) *gorm.DB {
				return filter.Where(tx, column+" IS FALSE")
			},
			RequiredArguments: 0,
		},
		"$notnull": {
			Name: "$notnull",
			Function: func(tx *gorm.DB, filter *Filter, column string) *gorm.DB {
				return filter.Where(tx, column+" IS NOT NULL")
			},
			RequiredArguments: 0,
		},
		"$between": {
			Name: "$between",
			Function: func(tx *gorm.DB, filter *Filter, column string) *gorm.DB {
				query := column + " BETWEEN ? AND ?"
				return filter.Where(tx, query, filter.Args[0], filter.Args[1])
			},
			RequiredArguments: 2,
		},
	}
)

func basicComparison(op string) func(tx *gorm.DB, filter *Filter, column string) *gorm.DB {
	return func(tx *gorm.DB, filter *Filter, column string) *gorm.DB {
		query := fmt.Sprintf("%s %s ?", column, op)
		return filter.Where(tx, query, filter.Args[0])
	}
}

func multiComparison(op string) func(tx *gorm.DB, filter *Filter, column string) *gorm.DB {
	return func(tx *gorm.DB, filter *Filter, column string) *gorm.DB {
		query := fmt.Sprintf("%s %s ?", column, op)
		return filter.Where(tx, query, filter.Args)
	}
}
