package filter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm/clause"
)

type operatorTestCase struct {
	filter   *Filter
	want     map[string]clause.Clause
	desc     string
	op       string
	column   string
	dataType DataType
}

func TestEquals(t *testing.T) {
	cases := []operatorTestCase{
		{
			desc:     "ok",
			op:       "$eq",
			filter:   &Filter{Field: "name", Args: []string{"test"}},
			column:   "`test_models`.`name`",
			dataType: DataTypeText,
			want: map[string]clause.Clause{
				"WHERE": {
					Name: "WHERE",
					Expression: clause.Where{
						Exprs: []clause.Expression{
							clause.Expr{SQL: "`test_models`.`name` = ?", Vars: []interface{}{"test"}},
						},
					},
				},
			},
		},
		{
			desc:     "cannot_compare_array",
			op:       "$eq",
			filter:   &Filter{Field: "name", Args: []string{"test"}},
			column:   "`test_models`.`name`",
			dataType: DataTypeTextArray,
			want: map[string]clause.Clause{
				"WHERE": {
					Name: "WHERE",
					Expression: clause.Where{
						Exprs: []clause.Expression{
							clause.Expr{SQL: "FALSE"},
						},
					},
				},
			},
		},
		{
			desc:     "cannot_convert_to_int",
			op:       "$eq",
			filter:   &Filter{Field: "age", Args: []string{"test"}},
			column:   "`test_models`.`age`",
			dataType: DataTypeFloat,
			want: map[string]clause.Clause{
				"WHERE": {
					Name: "WHERE",
					Expression: clause.Where{
						Exprs: []clause.Expression{
							clause.Expr{SQL: "FALSE"},
						},
					},
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			db := openDryRunDB(t)
			db = Operators[c.op].Function(db, c.filter, c.column, c.dataType)
			assert.Equal(t, c.want, db.Statement.Clauses)
		})
	}
}

func TestNotEquals(t *testing.T) {
	cases := []operatorTestCase{
		{
			desc:     "ok",
			op:       "$ne",
			filter:   &Filter{Field: "name", Args: []string{"test"}},
			column:   "`test_models`.`name`",
			dataType: DataTypeText,
			want: map[string]clause.Clause{
				"WHERE": {
					Name: "WHERE",
					Expression: clause.Where{
						Exprs: []clause.Expression{
							clause.Expr{SQL: "`test_models`.`name` <> ?", Vars: []interface{}{"test"}},
						},
					},
				},
			},
		},
		{
			desc:     "cannot_compare_array",
			op:       "$ne",
			filter:   &Filter{Field: "name", Args: []string{"test"}},
			column:   "`test_models`.`name`",
			dataType: DataTypeTextArray,
			want: map[string]clause.Clause{
				"WHERE": {
					Name: "WHERE",
					Expression: clause.Where{
						Exprs: []clause.Expression{
							clause.Expr{SQL: "FALSE"},
						},
					},
				},
			},
		},
		{
			desc:     "cannot_convert_to_int",
			op:       "$ne",
			filter:   &Filter{Field: "age", Args: []string{"test"}},
			column:   "`test_models`.`age`",
			dataType: DataTypeFloat,
			want: map[string]clause.Clause{
				"WHERE": {
					Name: "WHERE",
					Expression: clause.Where{
						Exprs: []clause.Expression{
							clause.Expr{SQL: "FALSE"},
						},
					},
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			db := openDryRunDB(t)
			db = Operators[c.op].Function(db, c.filter, c.column, c.dataType)
			assert.Equal(t, c.want, db.Statement.Clauses)
		})
	}
}

func TestGreaterThan(t *testing.T) {
	cases := []operatorTestCase{
		{
			desc:     "ok",
			op:       "$gt",
			filter:   &Filter{Field: "age", Args: []string{"18"}},
			column:   "`test_models`.`age`",
			dataType: DataTypeInt,
			want: map[string]clause.Clause{
				"WHERE": {
					Name: "WHERE",
					Expression: clause.Where{
						Exprs: []clause.Expression{
							clause.Expr{SQL: "`test_models`.`age` > ?", Vars: []interface{}{int64(18)}},
						},
					},
				},
			},
		},
		{
			desc:     "cannot_compare_array",
			op:       "$gt",
			filter:   &Filter{Field: "age", Args: []string{"18"}},
			column:   "`test_models`.`age`",
			dataType: DataTypeIntArray,
			want: map[string]clause.Clause{
				"WHERE": {
					Name: "WHERE",
					Expression: clause.Where{
						Exprs: []clause.Expression{
							clause.Expr{SQL: "FALSE"},
						},
					},
				},
			},
		},
		{
			desc:     "cannot_convert_to_int",
			op:       "$gt",
			filter:   &Filter{Field: "age", Args: []string{"test"}},
			column:   "`test_models`.`age`",
			dataType: DataTypeInt,
			want: map[string]clause.Clause{
				"WHERE": {
					Name: "WHERE",
					Expression: clause.Where{
						Exprs: []clause.Expression{
							clause.Expr{SQL: "FALSE"},
						},
					},
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			db := openDryRunDB(t)
			db = Operators[c.op].Function(db, c.filter, c.column, c.dataType)
			assert.Equal(t, c.want, db.Statement.Clauses)
		})
	}
}

func TestLowerThan(t *testing.T) {
	cases := []operatorTestCase{
		{
			desc:     "ok",
			op:       "$lt",
			filter:   &Filter{Field: "age", Args: []string{"18"}},
			column:   "`test_models`.`age`",
			dataType: DataTypeInt,
			want: map[string]clause.Clause{
				"WHERE": {
					Name: "WHERE",
					Expression: clause.Where{
						Exprs: []clause.Expression{
							clause.Expr{SQL: "`test_models`.`age` < ?", Vars: []interface{}{int64(18)}},
						},
					},
				},
			},
		},
		{
			desc:     "cannot_compare_array",
			op:       "$lt",
			filter:   &Filter{Field: "age", Args: []string{"18"}},
			column:   "`test_models`.`age`",
			dataType: DataTypeIntArray,
			want: map[string]clause.Clause{
				"WHERE": {
					Name: "WHERE",
					Expression: clause.Where{
						Exprs: []clause.Expression{
							clause.Expr{SQL: "FALSE"},
						},
					},
				},
			},
		},
		{
			desc:     "cannot_convert_to_int",
			op:       "$lt",
			filter:   &Filter{Field: "age", Args: []string{"test"}},
			column:   "`test_models`.`age`",
			dataType: DataTypeInt,
			want: map[string]clause.Clause{
				"WHERE": {
					Name: "WHERE",
					Expression: clause.Where{
						Exprs: []clause.Expression{
							clause.Expr{SQL: "FALSE"},
						},
					},
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			db := openDryRunDB(t)
			db = Operators[c.op].Function(db, c.filter, c.column, c.dataType)
			assert.Equal(t, c.want, db.Statement.Clauses)
		})
	}
}

func TestGreaterThanEqual(t *testing.T) {
	cases := []operatorTestCase{
		{
			desc:     "ok",
			op:       "$gte",
			filter:   &Filter{Field: "age", Args: []string{"18"}},
			column:   "`test_models`.`age`",
			dataType: DataTypeInt,
			want: map[string]clause.Clause{
				"WHERE": {
					Name: "WHERE",
					Expression: clause.Where{
						Exprs: []clause.Expression{
							clause.Expr{SQL: "`test_models`.`age` >= ?", Vars: []interface{}{int64(18)}},
						},
					},
				},
			},
		},
		{
			desc:     "cannot_compare_array",
			op:       "$gte",
			filter:   &Filter{Field: "age", Args: []string{"18"}},
			column:   "`test_models`.`age`",
			dataType: DataTypeIntArray,
			want: map[string]clause.Clause{
				"WHERE": {
					Name: "WHERE",
					Expression: clause.Where{
						Exprs: []clause.Expression{
							clause.Expr{SQL: "FALSE"},
						},
					},
				},
			},
		},
		{
			desc:     "cannot_convert_to_int",
			op:       "$gte",
			filter:   &Filter{Field: "age", Args: []string{"test"}},
			column:   "`test_models`.`age`",
			dataType: DataTypeInt,
			want: map[string]clause.Clause{
				"WHERE": {
					Name: "WHERE",
					Expression: clause.Where{
						Exprs: []clause.Expression{
							clause.Expr{SQL: "FALSE"},
						},
					},
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			db := openDryRunDB(t)
			db = Operators[c.op].Function(db, c.filter, c.column, c.dataType)
			assert.Equal(t, c.want, db.Statement.Clauses)
		})
	}
}

func TestLowerThanEqual(t *testing.T) {
	cases := []operatorTestCase{
		{
			desc:     "ok",
			op:       "$lte",
			filter:   &Filter{Field: "age", Args: []string{"18"}},
			column:   "`test_models`.`age`",
			dataType: DataTypeInt,
			want: map[string]clause.Clause{
				"WHERE": {
					Name: "WHERE",
					Expression: clause.Where{
						Exprs: []clause.Expression{
							clause.Expr{SQL: "`test_models`.`age` <= ?", Vars: []interface{}{int64(18)}},
						},
					},
				},
			},
		},
		{
			desc:     "cannot_compare_array",
			op:       "$lte",
			filter:   &Filter{Field: "age", Args: []string{"18"}},
			column:   "`test_models`.`age`",
			dataType: DataTypeIntArray,
			want: map[string]clause.Clause{
				"WHERE": {
					Name: "WHERE",
					Expression: clause.Where{
						Exprs: []clause.Expression{
							clause.Expr{SQL: "FALSE"},
						},
					},
				},
			},
		},
		{
			desc:     "cannot_convert_to_int",
			op:       "$lte",
			filter:   &Filter{Field: "age", Args: []string{"test"}},
			column:   "`test_models`.`age`",
			dataType: DataTypeInt,
			want: map[string]clause.Clause{
				"WHERE": {
					Name: "WHERE",
					Expression: clause.Where{
						Exprs: []clause.Expression{
							clause.Expr{SQL: "FALSE"},
						},
					},
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			db := openDryRunDB(t)
			db = Operators[c.op].Function(db, c.filter, c.column, c.dataType)
			assert.Equal(t, c.want, db.Statement.Clauses)
		})
	}
}

func TestStarts(t *testing.T) {
	cases := []operatorTestCase{
		{
			desc:     "ok",
			op:       "$starts",
			filter:   &Filter{Field: "name", Args: []string{"te%_st"}},
			column:   "`test_models`.`name`",
			dataType: DataTypeText,
			want: map[string]clause.Clause{
				"WHERE": {
					Name: "WHERE",
					Expression: clause.Where{
						Exprs: []clause.Expression{
							clause.Expr{SQL: "`test_models`.`name` LIKE ?", Vars: []interface{}{"te\\%\\_st%"}},
						},
					},
				},
			},
		},
		{
			desc:     "cannot_compare_array",
			op:       "$starts",
			filter:   &Filter{Field: "name", Args: []string{"te%_st"}},
			column:   "`test_models`.`name`",
			dataType: DataTypeTextArray,
			want: map[string]clause.Clause{
				"WHERE": {
					Name: "WHERE",
					Expression: clause.Where{
						Exprs: []clause.Expression{
							clause.Expr{SQL: "FALSE"},
						},
					},
				},
			},
		},
		{
			desc:     "cannot_use_with_int",
			op:       "$starts",
			filter:   &Filter{Field: "name", Args: []string{"te%_st"}},
			column:   "`test_models`.`name`",
			dataType: DataTypeInt,
			want: map[string]clause.Clause{
				"WHERE": {
					Name: "WHERE",
					Expression: clause.Where{
						Exprs: []clause.Expression{
							clause.Expr{SQL: "FALSE"},
						},
					},
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			db := openDryRunDB(t)
			db = Operators[c.op].Function(db, c.filter, c.column, c.dataType)
			assert.Equal(t, c.want, db.Statement.Clauses)
		})
	}
}

func TestEnds(t *testing.T) {
	cases := []operatorTestCase{
		{
			desc:     "ok",
			op:       "$ends",
			filter:   &Filter{Field: "name", Args: []string{"te%_st"}},
			column:   "`test_models`.`name`",
			dataType: DataTypeText,
			want: map[string]clause.Clause{
				"WHERE": {
					Name: "WHERE",
					Expression: clause.Where{
						Exprs: []clause.Expression{
							clause.Expr{SQL: "`test_models`.`name` LIKE ?", Vars: []interface{}{"%te\\%\\_st"}},
						},
					},
				},
			},
		},
		{
			desc:     "cannot_compare_array",
			op:       "$ends",
			filter:   &Filter{Field: "name", Args: []string{"te%_st"}},
			column:   "`test_models`.`name`",
			dataType: DataTypeTextArray,
			want: map[string]clause.Clause{
				"WHERE": {
					Name: "WHERE",
					Expression: clause.Where{
						Exprs: []clause.Expression{
							clause.Expr{SQL: "FALSE"},
						},
					},
				},
			},
		},
		{
			desc:     "cannot_use_with_int",
			op:       "$ends",
			filter:   &Filter{Field: "name", Args: []string{"te%_st"}},
			column:   "`test_models`.`name`",
			dataType: DataTypeInt,
			want: map[string]clause.Clause{
				"WHERE": {
					Name: "WHERE",
					Expression: clause.Where{
						Exprs: []clause.Expression{
							clause.Expr{SQL: "FALSE"},
						},
					},
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			db := openDryRunDB(t)
			db = Operators[c.op].Function(db, c.filter, c.column, c.dataType)
			assert.Equal(t, c.want, db.Statement.Clauses)
		})
	}
}

func TestContains(t *testing.T) {
	cases := []operatorTestCase{
		{
			desc:     "ok",
			op:       "$cont",
			filter:   &Filter{Field: "name", Args: []string{"te%_st"}},
			column:   "`test_models`.`name`",
			dataType: DataTypeText,
			want: map[string]clause.Clause{
				"WHERE": {
					Name: "WHERE",
					Expression: clause.Where{
						Exprs: []clause.Expression{
							clause.Expr{SQL: "`test_models`.`name` LIKE ?", Vars: []interface{}{"%te\\%\\_st%"}},
						},
					},
				},
			},
		},
		{
			desc:     "cannot_compare_array",
			op:       "$cont",
			filter:   &Filter{Field: "name", Args: []string{"te%_st"}},
			column:   "`test_models`.`name`",
			dataType: DataTypeTextArray,
			want: map[string]clause.Clause{
				"WHERE": {
					Name: "WHERE",
					Expression: clause.Where{
						Exprs: []clause.Expression{
							clause.Expr{SQL: "FALSE"},
						},
					},
				},
			},
		},
		{
			desc:     "cannot_use_with_int",
			op:       "$cont",
			filter:   &Filter{Field: "name", Args: []string{"te%_st"}},
			column:   "`test_models`.`name`",
			dataType: DataTypeInt,
			want: map[string]clause.Clause{
				"WHERE": {
					Name: "WHERE",
					Expression: clause.Where{
						Exprs: []clause.Expression{
							clause.Expr{SQL: "FALSE"},
						},
					},
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			db := openDryRunDB(t)
			db = Operators[c.op].Function(db, c.filter, c.column, c.dataType)
			assert.Equal(t, c.want, db.Statement.Clauses)
		})
	}
}

func TestNotContains(t *testing.T) {
	cases := []operatorTestCase{
		{
			desc:     "ok",
			op:       "$excl",
			filter:   &Filter{Field: "name", Args: []string{"te%_st"}},
			column:   "`test_models`.`name`",
			dataType: DataTypeText,
			want: map[string]clause.Clause{
				"WHERE": {
					Name: "WHERE",
					Expression: clause.Where{
						Exprs: []clause.Expression{
							clause.Expr{SQL: "`test_models`.`name` NOT LIKE ?", Vars: []interface{}{"%te\\%\\_st%"}},
						},
					},
				},
			},
		},
		{
			desc:     "cannot_compare_array",
			op:       "$excl",
			filter:   &Filter{Field: "name", Args: []string{"te%_st"}},
			column:   "`test_models`.`name`",
			dataType: DataTypeTextArray,
			want: map[string]clause.Clause{
				"WHERE": {
					Name: "WHERE",
					Expression: clause.Where{
						Exprs: []clause.Expression{
							clause.Expr{SQL: "FALSE"},
						},
					},
				},
			},
		},
		{
			desc:     "cannot_use_with_int",
			op:       "$excl",
			filter:   &Filter{Field: "name", Args: []string{"te%_st"}},
			column:   "`test_models`.`name`",
			dataType: DataTypeInt,
			want: map[string]clause.Clause{
				"WHERE": {
					Name: "WHERE",
					Expression: clause.Where{
						Exprs: []clause.Expression{
							clause.Expr{SQL: "FALSE"},
						},
					},
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			db := openDryRunDB(t)
			db = Operators[c.op].Function(db, c.filter, c.column, c.dataType)
			assert.Equal(t, c.want, db.Statement.Clauses)
		})
	}
}

func TestIn(t *testing.T) {
	cases := []operatorTestCase{
		{
			desc:     "ok",
			op:       "$in",
			filter:   &Filter{Field: "name", Args: []string{"val1", "val2"}},
			column:   "`test_models`.`name`",
			dataType: DataTypeText,
			want: map[string]clause.Clause{
				"WHERE": {
					Name: "WHERE",
					Expression: clause.Where{
						Exprs: []clause.Expression{
							clause.Expr{SQL: "`test_models`.`name` IN ?", Vars: []interface{}{[]interface{}{"val1", "val2"}}},
						},
					},
				},
			},
		},
		{
			desc:     "cannot_compare_array",
			op:       "$in",
			filter:   &Filter{Field: "name", Args: []string{"val1", "val2"}},
			column:   "`test_models`.`name`",
			dataType: DataTypeTextArray,
			want: map[string]clause.Clause{
				"WHERE": {
					Name: "WHERE",
					Expression: clause.Where{
						Exprs: []clause.Expression{
							clause.Expr{SQL: "FALSE"},
						},
					},
				},
			},
		},
		{
			desc:     "cannot_convert_to_int",
			op:       "$in",
			filter:   &Filter{Field: "name", Args: []string{"18", "val2"}},
			column:   "`test_models`.`name`",
			dataType: DataTypeInt,
			want: map[string]clause.Clause{
				"WHERE": {
					Name: "WHERE",
					Expression: clause.Where{
						Exprs: []clause.Expression{
							clause.Expr{SQL: "FALSE"},
						},
					},
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			db := openDryRunDB(t)
			db = Operators[c.op].Function(db, c.filter, c.column, c.dataType)
			assert.Equal(t, c.want, db.Statement.Clauses)
		})
	}
}

func TestNotIn(t *testing.T) {
	cases := []operatorTestCase{
		{
			desc:     "ok",
			op:       "$notin",
			filter:   &Filter{Field: "name", Args: []string{"val1", "val2"}},
			column:   "`test_models`.`name`",
			dataType: DataTypeText,
			want: map[string]clause.Clause{
				"WHERE": {
					Name: "WHERE",
					Expression: clause.Where{
						Exprs: []clause.Expression{
							clause.Expr{SQL: "`test_models`.`name` NOT IN ?", Vars: []interface{}{[]interface{}{"val1", "val2"}}},
						},
					},
				},
			},
		},
		{
			desc:     "cannot_compare_array",
			op:       "$notin",
			filter:   &Filter{Field: "name", Args: []string{"val1", "val2"}},
			column:   "`test_models`.`name`",
			dataType: DataTypeTextArray,
			want: map[string]clause.Clause{
				"WHERE": {
					Name: "WHERE",
					Expression: clause.Where{
						Exprs: []clause.Expression{
							clause.Expr{SQL: "FALSE"},
						},
					},
				},
			},
		},
		{
			desc:     "cannot_convert_to_int",
			op:       "$notin",
			filter:   &Filter{Field: "name", Args: []string{"18", "val2"}},
			column:   "`test_models`.`name`",
			dataType: DataTypeInt,
			want: map[string]clause.Clause{
				"WHERE": {
					Name: "WHERE",
					Expression: clause.Where{
						Exprs: []clause.Expression{
							clause.Expr{SQL: "FALSE"},
						},
					},
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			db := openDryRunDB(t)
			db = Operators[c.op].Function(db, c.filter, c.column, c.dataType)
			assert.Equal(t, c.want, db.Statement.Clauses)
		})
	}
}

func TestIsNull(t *testing.T) {
	cases := []operatorTestCase{
		{
			desc:     "ok",
			op:       "$isnull",
			filter:   &Filter{Field: "name"},
			column:   "`test_models`.`name`",
			dataType: DataTypeText,
			want: map[string]clause.Clause{
				"WHERE": {
					Name: "WHERE",
					Expression: clause.Where{
						Exprs: []clause.Expression{
							clause.Expr{SQL: "`test_models`.`name` IS NULL"},
						},
					},
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			db := openDryRunDB(t)
			db = Operators[c.op].Function(db, c.filter, c.column, c.dataType)
			assert.Equal(t, c.want, db.Statement.Clauses)
		})
	}
}

func TestNotNull(t *testing.T) {
	cases := []operatorTestCase{
		{
			desc:     "ok",
			op:       "$notnull",
			filter:   &Filter{Field: "name"},
			column:   "`test_models`.`name`",
			dataType: DataTypeText,
			want: map[string]clause.Clause{
				"WHERE": {
					Name: "WHERE",
					Expression: clause.Where{
						Exprs: []clause.Expression{
							clause.Expr{SQL: "`test_models`.`name` IS NOT NULL"},
						},
					},
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			db := openDryRunDB(t)
			db = Operators[c.op].Function(db, c.filter, c.column, c.dataType)
			assert.Equal(t, c.want, db.Statement.Clauses)
		})
	}
}

func TestBetween(t *testing.T) {
	cases := []operatorTestCase{
		{
			desc:     "ok_int",
			op:       "$between",
			filter:   &Filter{Field: "age", Args: []string{"18", "25"}},
			column:   "`test_models`.`age`",
			dataType: DataTypeUint,
			want: map[string]clause.Clause{
				"WHERE": {
					Name: "WHERE",
					Expression: clause.Where{
						Exprs: []clause.Expression{
							clause.Expr{SQL: "`test_models`.`age` BETWEEN ? AND ?", Vars: []interface{}{uint64(18), uint64(25)}},
						},
					},
				},
			},
		},
		{
			desc:     "ok_time",
			op:       "$between",
			filter:   &Filter{Field: "birthday", Args: []string{"2023-04-04", "2023-05-05 12:00:00"}},
			column:   "`test_models`.`birthday`",
			dataType: DataTypeTime,
			want: map[string]clause.Clause{
				"WHERE": {
					Name: "WHERE",
					Expression: clause.Where{
						Exprs: []clause.Expression{
							clause.Expr{SQL: "`test_models`.`birthday` BETWEEN ? AND ?", Vars: []interface{}{"2023-04-04", "2023-05-05 12:00:00"}},
						},
					},
				},
			},
		},
		{
			desc:     "cannot_compare_array",
			op:       "$between",
			filter:   &Filter{Field: "birthday", Args: []string{"2023-04-04", "2023-05-05 12:00:00"}},
			column:   "`test_models`.`birthday`",
			dataType: DataTypeTimeArray,
			want: map[string]clause.Clause{
				"WHERE": {
					Name: "WHERE",
					Expression: clause.Where{
						Exprs: []clause.Expression{
							clause.Expr{SQL: "FALSE"},
						},
					},
				},
			},
		},
		{
			desc:     "cannot_convert_to_int",
			op:       "$between",
			filter:   &Filter{Field: "age", Args: []string{"18", "val2"}},
			column:   "`test_models`.`age`",
			dataType: DataTypeUint,
			want: map[string]clause.Clause{
				"WHERE": {
					Name: "WHERE",
					Expression: clause.Where{
						Exprs: []clause.Expression{
							clause.Expr{SQL: "FALSE"},
						},
					},
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			db := openDryRunDB(t)
			db = Operators[c.op].Function(db, c.filter, c.column, c.dataType)
			assert.Equal(t, c.want, db.Statement.Clauses)
		})
	}
}

func TestIsTrue(t *testing.T) {
	cases := []operatorTestCase{
		{
			desc:     "ok",
			op:       "$istrue",
			filter:   &Filter{Field: "is_active"},
			column:   "`test_models`.`is_active`",
			dataType: DataTypeBool,
			want: map[string]clause.Clause{
				"WHERE": {
					Name: "WHERE",
					Expression: clause.Where{
						Exprs: []clause.Expression{
							clause.Expr{SQL: "`test_models`.`is_active` IS TRUE"},
						},
					},
				},
			},
		},
		{
			desc:     "cannot_use_with_int",
			op:       "$istrue",
			filter:   &Filter{Field: "is_active"},
			column:   "`test_models`.`is_active`",
			dataType: DataTypeInt,
			want: map[string]clause.Clause{
				"WHERE": {
					Name: "WHERE",
					Expression: clause.Where{
						Exprs: []clause.Expression{
							clause.Expr{SQL: "FALSE"},
						},
					},
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			db := openDryRunDB(t)
			db = Operators[c.op].Function(db, c.filter, c.column, c.dataType)
			assert.Equal(t, c.want, db.Statement.Clauses)
		})
	}
}

func TestIsFalse(t *testing.T) {
	cases := []operatorTestCase{
		{
			desc:     "ok",
			op:       "$isfalse",
			filter:   &Filter{Field: "is_active"},
			column:   "`test_models`.`is_active`",
			dataType: DataTypeBool,
			want: map[string]clause.Clause{
				"WHERE": {
					Name: "WHERE",
					Expression: clause.Where{
						Exprs: []clause.Expression{
							clause.Expr{SQL: "`test_models`.`is_active` IS FALSE"},
						},
					},
				},
			},
		},
		{
			desc:     "cannot_use_with_int",
			op:       "$isfalse",
			filter:   &Filter{Field: "is_active"},
			column:   "`test_models`.`is_active`",
			dataType: DataTypeInt,
			want: map[string]clause.Clause{
				"WHERE": {
					Name: "WHERE",
					Expression: clause.Where{
						Exprs: []clause.Expression{
							clause.Expr{SQL: "FALSE"},
						},
					},
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			db := openDryRunDB(t)
			db = Operators[c.op].Function(db, c.filter, c.column, c.dataType)
			assert.Equal(t, c.want, db.Statement.Clauses)
		})
	}
}
