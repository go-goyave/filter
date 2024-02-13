package filter

import (
	"fmt"

	"gorm.io/gorm"
	"goyave.dev/goyave/v5/util/sqlutil"
)

// Operator used by filters to build the SQL query.
// The operator function modifies the GORM statement (most of the time by adding
// a WHERE condition) then returns the modified statement.
//
// Operators may need arguments (e.g. "$eq", equals needs a value to compare the field to);
// RequiredArguments define the minimum number of arguments a client must send in order to
// use this operator in a filter. RequiredArguments is checked during Filter parsing.
//
// Operators may return the given tx without change if they don't support the given dataType or
// add a condition that will always be false.
type Operator struct {
	Function          func(tx *gorm.DB, filter *Filter, column string, dataType DataType) *gorm.DB
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
			Function: func(tx *gorm.DB, filter *Filter, column string, dataType DataType) *gorm.DB {
				if dataType != DataTypeText && dataType != DataTypeEnum {
					return filter.Where(tx, "FALSE")
				}
				query := castEnumAsText(column, dataType) + " LIKE ?"
				value := sqlutil.EscapeLike(filter.Args[0]) + "%"
				return filter.Where(tx, query, value)
			},
			RequiredArguments: 1,
		},
		"$ends": {
			Function: func(tx *gorm.DB, filter *Filter, column string, dataType DataType) *gorm.DB {
				if dataType != DataTypeText && dataType != DataTypeEnum {
					return filter.Where(tx, "FALSE")
				}
				query := castEnumAsText(column, dataType) + " LIKE ?"
				value := "%" + sqlutil.EscapeLike(filter.Args[0])
				return filter.Where(tx, query, value)
			},
			RequiredArguments: 1,
		},
		"$cont": {
			Function: func(tx *gorm.DB, filter *Filter, column string, dataType DataType) *gorm.DB {
				if dataType != DataTypeText && dataType != DataTypeEnum {
					return filter.Where(tx, "FALSE")
				}
				query := castEnumAsText(column, dataType) + " LIKE ?"
				value := "%" + sqlutil.EscapeLike(filter.Args[0]) + "%"
				return filter.Where(tx, query, value)
			},
			RequiredArguments: 1,
		},
		"$excl": {
			Function: func(tx *gorm.DB, filter *Filter, column string, dataType DataType) *gorm.DB {
				if dataType != DataTypeText && dataType != DataTypeEnum {
					return filter.Where(tx, "FALSE")
				}
				query := castEnumAsText(column, dataType) + " NOT LIKE ?"
				value := "%" + sqlutil.EscapeLike(filter.Args[0]) + "%"
				return filter.Where(tx, query, value)
			},
			RequiredArguments: 1,
		},
		"$in":    {Function: multiComparison("IN"), RequiredArguments: 1},
		"$notin": {Function: multiComparison("NOT IN"), RequiredArguments: 1},
		"$isnull": {
			Function: func(tx *gorm.DB, filter *Filter, column string, _ DataType) *gorm.DB {
				return filter.Where(tx, column+" IS NULL")
			},
			RequiredArguments: 0,
		},
		"$istrue": {
			Function: func(tx *gorm.DB, filter *Filter, column string, dataType DataType) *gorm.DB {
				if dataType != DataTypeBool {
					return filter.Where(tx, "FALSE")
				}
				return filter.Where(tx, column+" IS TRUE")
			},
			RequiredArguments: 0,
		},
		"$isfalse": {
			Function: func(tx *gorm.DB, filter *Filter, column string, dataType DataType) *gorm.DB {
				if dataType != DataTypeBool {
					return filter.Where(tx, "FALSE")
				}
				return filter.Where(tx, column+" IS FALSE")
			},
			RequiredArguments: 0,
		},
		"$notnull": {
			Function: func(tx *gorm.DB, filter *Filter, column string, _ DataType) *gorm.DB {
				return filter.Where(tx, column+" IS NOT NULL")
			},
			RequiredArguments: 0,
		},
		"$between": {
			Function: func(tx *gorm.DB, filter *Filter, column string, dataType DataType) *gorm.DB {
				if dataType.IsArray() {
					return filter.Where(tx, "FALSE")
				}
				args, ok := ConvertArgsToSafeType(filter.Args[:2], dataType)
				if !ok {
					return filter.Where(tx, "FALSE")
				}
				query := castEnumAsText(column, dataType) + " BETWEEN ? AND ?"
				return filter.Where(tx, query, args...)
			},
			RequiredArguments: 2,
		},
	}
)

func castEnumAsText(column string, dataType DataType) string {
	if dataType == DataTypeEnum || dataType == DataTypeEnumArray {
		return fmt.Sprintf("CAST(%s AS TEXT)", column)
	}
	return column
}

func basicComparison(op string) func(tx *gorm.DB, filter *Filter, column string, dataType DataType) *gorm.DB {
	return func(tx *gorm.DB, filter *Filter, column string, dataType DataType) *gorm.DB {
		if dataType.IsArray() {
			return filter.Where(tx, "FALSE")
		}
		arg, ok := ConvertToSafeType(filter.Args[0], dataType)
		if !ok {
			return filter.Where(tx, "FALSE")
		}

		query := fmt.Sprintf("%s %s ?", castEnumAsText(column, dataType), op)
		return filter.Where(tx, query, arg)
	}
}

func multiComparison(op string) func(tx *gorm.DB, filter *Filter, column string, dataType DataType) *gorm.DB {
	return func(tx *gorm.DB, filter *Filter, column string, dataType DataType) *gorm.DB {
		if dataType.IsArray() {
			return filter.Where(tx, "FALSE")
		}
		args, ok := ConvertArgsToSafeType(filter.Args, dataType)
		if !ok {
			return filter.Where(tx, "FALSE")
		}

		query := fmt.Sprintf("%s %s ?", castEnumAsText(column, dataType), op)
		return filter.Where(tx, query, args)
	}
}
