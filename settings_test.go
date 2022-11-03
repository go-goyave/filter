package filter

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
	"goyave.dev/goyave/v4"
	"goyave.dev/goyave/v4/database"
)

var fifteen = 15

type TestScopeRelation struct {
	A  string
	B  string
	ID uint
}
type TestScopeModel struct {
	Relation   *TestScopeRelation
	Name       string
	Email      string
	Computed   string `gorm:"->;-:migration" computed:"UPPER(~~~ct~~~.name)"`
	ID         uint
	RelationID uint
}

type TestScopeModelNoPrimaryKey struct {
	Relation   *TestScopeRelation
	Name       string
	RelationID uint
}

func openDryRunDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:?mode=memory"), nil)
	if err != nil {
		assert.FailNow(t, "Could not open dry run DB", err)
	}
	db.DryRun = true
	return db
}

func prepareTestScope(t *testing.T, settings *Settings) (*database.Paginator, *gorm.DB) {
	request := &goyave.Request{
		Data: map[string]interface{}{
			"filter": []*Filter{
				{Field: "name", Args: []string{"val1"}, Operator: Operators["$cont"]},
				{Field: "name", Args: []string{"val2"}, Operator: Operators["$cont"]},
			},
			"or": []*Filter{
				{Field: "name", Args: []string{"val3"}, Or: true, Operator: Operators["$eq"]},
			},
			"sort":     []*Sort{{Field: "name", Order: SortAscending}},
			"join":     []*Join{{Relation: "Relation", Fields: []string{"a", "b"}}},
			"page":     2,
			"per_page": 15,
			"fields":   "id,name,email,computed",
			"search":   "val",
		},
		Lang: "en-US",
	}
	db := openDryRunDB(t)

	results := []*TestScopeModel{}
	if settings == nil {
		return Scope(db, request, results)
	}

	return settings.Scope(db, request, results)
}

func prepareTestScopeUnpaginated(t *testing.T, settings *Settings) ([]*TestScopeModel, *gorm.DB) {
	request := &goyave.Request{
		Data: map[string]interface{}{
			"filter": []*Filter{
				{Field: "name", Args: []string{"val1"}, Operator: Operators["$cont"]},
				{Field: "name", Args: []string{"val2"}, Operator: Operators["$cont"]},
			},
			"or": []*Filter{
				{Field: "name", Args: []string{"val3"}, Or: true, Operator: Operators["$eq"]},
			},
			"sort":     []*Sort{{Field: "name", Order: SortAscending}},
			"join":     []*Join{{Relation: "Relation", Fields: []string{"a", "b"}}},
			"page":     2, // Those two should be ignored since we are not paginating
			"per_page": 15,
			"fields":   "id,name,email,computed",
			"search":   "val",
		},
		Lang: "en-US",
	}
	db := openDryRunDB(t)

	results := []*TestScopeModel{}
	if settings == nil {
		return results, ScopeUnpaginated(db, request, &results)
	}

	return results, settings.ScopeUnpaginated(db, request, &results)
}

func TestScope(t *testing.T) {
	paginator, db := prepareTestScope(t, &Settings{
		FieldsSearch: []string{"email"},
		SearchOperator: &Operator{
			Function: func(tx *gorm.DB, filter *Filter, column string, dataType schema.DataType) *gorm.DB {
				return tx.Or(fmt.Sprintf("%s LIKE (?)", column), filter.Args[0])
			},
			RequiredArguments: 1,
		},
	})
	assert.NotNil(t, paginator)

	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.AndConditions{
						Exprs: []clause.Expression{
							clause.OrConditions{
								Exprs: []clause.Expression{
									clause.AndConditions{
										Exprs: []clause.Expression{
											clause.Expr{SQL: "`test_scope_models`.`name` LIKE ?", Vars: []interface{}{"%val1%"}},
											clause.Expr{SQL: "`test_scope_models`.`name` LIKE ?", Vars: []interface{}{"%val2%"}},
										},
									},
								},
							},
							clause.OrConditions{
								Exprs: []clause.Expression{
									clause.AndConditions{
										Exprs: []clause.Expression{
											clause.Expr{SQL: "`test_scope_models`.`name` = ?", Vars: []interface{}{"val3"}},
										},
									},
								},
							},
						},
					},
					clause.AndConditions{
						Exprs: []clause.Expression{
							clause.Expr{
								SQL:                "`test_scope_models`.`email` LIKE (?)",
								Vars:               []interface{}{"val"},
								WithoutParentheses: false,
							},
						},
					},
				},
			},
		},
		"ORDER BY": {
			Name: "ORDER BY",
			Expression: clause.OrderBy{
				Columns: []clause.OrderByColumn{
					{
						Column: clause.Column{
							Table: "test_scope_models",
							Name:  "name",
						},
					},
				},
			},
		},
		"LIMIT": {
			Expression: clause.Limit{
				Limit:  &fifteen,
				Offset: 15,
			},
		},
		"FROM": {
			Name:       "FROM",
			Expression: clause.From{},
		},
		"SELECT": {
			Name: "SELECT",
			Expression: clause.Select{
				Columns: []clause.Column{
					{Raw: true, Name: "`test_scope_models`.`id`"},
					{Raw: true, Name: "`test_scope_models`.`name`"},
					{Raw: true, Name: "`test_scope_models`.`email`"},
					{Raw: true, Name: "(UPPER(`test_scope_models`.name)) `computed`"},
					{Raw: true, Name: "`test_scope_models`.`relation_id`"},
				},
			},
		},
	}
	assert.Equal(t, expected, db.Statement.Clauses)
	assert.Contains(t, db.Statement.Preloads, "Relation")
	assert.Equal(t, []string{"`test_scope_models`.`id`", "`test_scope_models`.`name`", "`test_scope_models`.`email`", "(UPPER(`test_scope_models`.name)) `computed`", "`test_scope_models`.`relation_id`"}, db.Statement.Selects)
}

func TestScopeUnpaginated(t *testing.T) {
	results, db := prepareTestScopeUnpaginated(t, &Settings{
		FieldsSearch: []string{"email"},
		SearchOperator: &Operator{
			Function: func(tx *gorm.DB, filter *Filter, column string, dataType schema.DataType) *gorm.DB {
				return tx.Or(fmt.Sprintf("%s LIKE (?)", column), filter.Args[0])
			},
			RequiredArguments: 1,
		},
	})
	assert.NotNil(t, results)

	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.AndConditions{
						Exprs: []clause.Expression{
							clause.OrConditions{
								Exprs: []clause.Expression{
									clause.AndConditions{
										Exprs: []clause.Expression{
											clause.Expr{SQL: "`test_scope_models`.`name` LIKE ?", Vars: []interface{}{"%val1%"}},
											clause.Expr{SQL: "`test_scope_models`.`name` LIKE ?", Vars: []interface{}{"%val2%"}},
										},
									},
								},
							},
							clause.OrConditions{
								Exprs: []clause.Expression{
									clause.AndConditions{
										Exprs: []clause.Expression{
											clause.Expr{SQL: "`test_scope_models`.`name` = ?", Vars: []interface{}{"val3"}},
										},
									},
								},
							},
						},
					},
					clause.AndConditions{
						Exprs: []clause.Expression{
							clause.Expr{
								SQL:                "`test_scope_models`.`email` LIKE (?)",
								Vars:               []interface{}{"val"},
								WithoutParentheses: false,
							},
						},
					},
				},
			},
		},
		"ORDER BY": {
			Name: "ORDER BY",
			Expression: clause.OrderBy{
				Columns: []clause.OrderByColumn{
					{
						Column: clause.Column{
							Table: "test_scope_models",
							Name:  "name",
						},
					},
				},
			},
		},
		"FROM": {
			Name:       "FROM",
			Expression: clause.From{},
		},
		"SELECT": {
			Name: "SELECT",
			Expression: clause.Select{
				Columns: []clause.Column{
					{Raw: true, Name: "`test_scope_models`.`id`"},
					{Raw: true, Name: "`test_scope_models`.`name`"},
					{Raw: true, Name: "`test_scope_models`.`email`"},
					{Raw: true, Name: "(UPPER(`test_scope_models`.name)) `computed`"},
					{Raw: true, Name: "`test_scope_models`.`relation_id`"},
				},
			},
		},
	}
	assert.Equal(t, expected, db.Statement.Clauses)
	assert.Contains(t, db.Statement.Preloads, "Relation")
	assert.ElementsMatch(t, []string{"`test_scope_models`.`id`", "`test_scope_models`.`name`", "`test_scope_models`.`email`", "`test_scope_models`.`relation_id`", "(UPPER(`test_scope_models`.name)) `computed`"}, db.Statement.Selects)
}

func TestScopeDisableFields(t *testing.T) {
	paginator, db := prepareTestScope(t, &Settings{DisableFields: true, FieldsSearch: []string{"email"}})
	assert.NotNil(t, paginator)

	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.AndConditions{
						Exprs: []clause.Expression{
							clause.OrConditions{
								Exprs: []clause.Expression{
									clause.AndConditions{
										Exprs: []clause.Expression{
											clause.Expr{SQL: "`test_scope_models`.`name` LIKE ?", Vars: []interface{}{"%val1%"}},
											clause.Expr{SQL: "`test_scope_models`.`name` LIKE ?", Vars: []interface{}{"%val2%"}},
										},
									},
								},
							},
							clause.OrConditions{
								Exprs: []clause.Expression{
									clause.AndConditions{
										Exprs: []clause.Expression{
											clause.Expr{SQL: "`test_scope_models`.`name` = ?", Vars: []interface{}{"val3"}},
										},
									},
								},
							},
						},
					},
					clause.AndConditions{
						Exprs: []clause.Expression{
							clause.Expr{
								SQL:                "`test_scope_models`.`email` LIKE ?",
								Vars:               []interface{}{"%val%"},
								WithoutParentheses: false,
							},
						},
					},
				},
			},
		},
		"ORDER BY": {
			Name: "ORDER BY",
			Expression: clause.OrderBy{
				Columns: []clause.OrderByColumn{
					{
						Column: clause.Column{
							Table: "test_scope_models",
							Name:  "name",
						},
					},
				},
			},
		},
		"LIMIT": {
			Expression: clause.Limit{
				Limit:  &fifteen,
				Offset: 15,
			},
		},
		"FROM": {
			Name:       "FROM",
			Expression: clause.From{},
		},
		"SELECT": {
			Name: "SELECT",
			Expression: clause.Select{
				Columns: []clause.Column{
					{Raw: true, Name: "`test_scope_models`.`name`"},
					{Raw: true, Name: "`test_scope_models`.`email`"},
					{Raw: true, Name: "(UPPER(`test_scope_models`.name)) `computed`"},
					{Raw: true, Name: "`test_scope_models`.`id`"},
					{Raw: true, Name: "`test_scope_models`.`relation_id`"},
				},
			},
		},
	}
	assert.Equal(t, expected, db.Statement.Clauses)
	assert.ElementsMatch(t, []string{"`test_scope_models`.`id`", "`test_scope_models`.`relation_id`", "`test_scope_models`.`name`", "`test_scope_models`.`email`", "(UPPER(`test_scope_models`.name)) `computed`"}, db.Statement.Selects)
}

func TestScopeUnpaginatedDisableFields(t *testing.T) {
	results, db := prepareTestScopeUnpaginated(t, &Settings{DisableFields: true, FieldsSearch: []string{"email"}})
	assert.NotNil(t, results)

	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.AndConditions{
						Exprs: []clause.Expression{
							clause.OrConditions{
								Exprs: []clause.Expression{
									clause.AndConditions{
										Exprs: []clause.Expression{
											clause.Expr{SQL: "`test_scope_models`.`name` LIKE ?", Vars: []interface{}{"%val1%"}},
											clause.Expr{SQL: "`test_scope_models`.`name` LIKE ?", Vars: []interface{}{"%val2%"}},
										},
									},
								},
							},
							clause.OrConditions{
								Exprs: []clause.Expression{
									clause.AndConditions{
										Exprs: []clause.Expression{
											clause.Expr{SQL: "`test_scope_models`.`name` = ?", Vars: []interface{}{"val3"}},
										},
									},
								},
							},
						},
					},
					clause.AndConditions{
						Exprs: []clause.Expression{
							clause.Expr{
								SQL:                "`test_scope_models`.`email` LIKE ?",
								Vars:               []interface{}{"%val%"},
								WithoutParentheses: false,
							},
						},
					},
				},
			},
		},
		"ORDER BY": {
			Name: "ORDER BY",
			Expression: clause.OrderBy{
				Columns: []clause.OrderByColumn{
					{
						Column: clause.Column{
							Table: "test_scope_models",
							Name:  "name",
						},
					},
				},
			},
		},
		"FROM": {
			Name:       "FROM",
			Expression: clause.From{},
		},
		"SELECT": {
			Name: "SELECT",
			Expression: clause.Select{
				Columns: []clause.Column{
					{Raw: true, Name: "`test_scope_models`.`name`"},
					{Raw: true, Name: "`test_scope_models`.`email`"},
					{Raw: true, Name: "(UPPER(`test_scope_models`.name)) `computed`"},
					{Raw: true, Name: "`test_scope_models`.`id`"},
					{Raw: true, Name: "`test_scope_models`.`relation_id`"},
				},
			},
		},
	}
	assert.Equal(t, expected, db.Statement.Clauses)
	assert.ElementsMatch(t, []string{"`test_scope_models`.`id`", "`test_scope_models`.`relation_id`", "`test_scope_models`.`name`", "`test_scope_models`.`email`", "(UPPER(`test_scope_models`.name)) `computed`"}, db.Statement.Selects)
}

func TestScopeDisableFilter(t *testing.T) {
	paginator, db := prepareTestScope(t, &Settings{DisableFilter: true, FieldsSearch: []string{"email"}})
	assert.NotNil(t, paginator)

	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.AndConditions{
						Exprs: []clause.Expression{
							clause.Expr{
								SQL:                "`test_scope_models`.`email` LIKE ?",
								Vars:               []interface{}{"%val%"},
								WithoutParentheses: false,
							},
						},
					},
				},
			},
		},
		"ORDER BY": {
			Name: "ORDER BY",
			Expression: clause.OrderBy{
				Columns: []clause.OrderByColumn{
					{
						Column: clause.Column{
							Table: "test_scope_models",
							Name:  "name",
						},
					},
				},
			},
		},
		"LIMIT": {
			Expression: clause.Limit{
				Limit:  &fifteen,
				Offset: 15,
			},
		},
		"FROM": {
			Name:       "FROM",
			Expression: clause.From{},
		},
		"SELECT": {
			Name: "SELECT",
			Expression: clause.Select{
				Columns: []clause.Column{
					{Raw: true, Name: "`test_scope_models`.`id`"},
					{Raw: true, Name: "`test_scope_models`.`name`"},
					{Raw: true, Name: "`test_scope_models`.`email`"},
					{Raw: true, Name: "(UPPER(`test_scope_models`.name)) `computed`"},
					{Raw: true, Name: "`test_scope_models`.`relation_id`"},
				},
			},
		},
	}
	assert.Equal(t, expected, db.Statement.Clauses)
	assert.ElementsMatch(t, []string{"`test_scope_models`.`id`", "`test_scope_models`.`name`", "`test_scope_models`.`email`", "`test_scope_models`.`relation_id`", "(UPPER(`test_scope_models`.name)) `computed`"}, db.Statement.Selects)
}

func TestScopeUnpaginatedDisableFilter(t *testing.T) {
	results, db := prepareTestScopeUnpaginated(t, &Settings{DisableFilter: true, FieldsSearch: []string{"email"}})
	assert.NotNil(t, results)

	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.AndConditions{
						Exprs: []clause.Expression{
							clause.Expr{
								SQL:                "`test_scope_models`.`email` LIKE ?",
								Vars:               []interface{}{"%val%"},
								WithoutParentheses: false,
							},
						},
					},
				},
			},
		},
		"ORDER BY": {
			Name: "ORDER BY",
			Expression: clause.OrderBy{
				Columns: []clause.OrderByColumn{
					{
						Column: clause.Column{
							Table: "test_scope_models",
							Name:  "name",
						},
					},
				},
			},
		},
		"FROM": {
			Name:       "FROM",
			Expression: clause.From{},
		},
		"SELECT": {
			Name: "SELECT",
			Expression: clause.Select{
				Columns: []clause.Column{
					{Raw: true, Name: "`test_scope_models`.`id`"},
					{Raw: true, Name: "`test_scope_models`.`name`"},
					{Raw: true, Name: "`test_scope_models`.`email`"},
					{Raw: true, Name: "(UPPER(`test_scope_models`.name)) `computed`"},
					{Raw: true, Name: "`test_scope_models`.`relation_id`"},
				},
			},
		},
	}
	assert.Equal(t, expected, db.Statement.Clauses)
	assert.ElementsMatch(t, []string{"`test_scope_models`.`id`", "`test_scope_models`.`name`", "`test_scope_models`.`email`", "`test_scope_models`.`relation_id`", "(UPPER(`test_scope_models`.name)) `computed`"}, db.Statement.Selects)
}

func TestScopeDisableSort(t *testing.T) {
	paginator, db := prepareTestScope(t, &Settings{DisableSort: true, FieldsSearch: []string{"email"}})
	assert.NotNil(t, paginator)

	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.AndConditions{
						Exprs: []clause.Expression{
							clause.OrConditions{
								Exprs: []clause.Expression{
									clause.AndConditions{
										Exprs: []clause.Expression{
											clause.Expr{SQL: "`test_scope_models`.`name` LIKE ?", Vars: []interface{}{"%val1%"}},
											clause.Expr{SQL: "`test_scope_models`.`name` LIKE ?", Vars: []interface{}{"%val2%"}},
										},
									},
								},
							},
							clause.OrConditions{
								Exprs: []clause.Expression{
									clause.AndConditions{
										Exprs: []clause.Expression{
											clause.Expr{SQL: "`test_scope_models`.`name` = ?", Vars: []interface{}{"val3"}},
										},
									},
								},
							},
						},
					},
					clause.AndConditions{
						Exprs: []clause.Expression{
							clause.Expr{
								SQL:                "`test_scope_models`.`email` LIKE ?",
								Vars:               []interface{}{"%val%"},
								WithoutParentheses: false,
							},
						},
					},
				},
			},
		},
		"LIMIT": {
			Expression: clause.Limit{
				Limit:  &fifteen,
				Offset: 15,
			},
		},
		"FROM": {
			Name:       "FROM",
			Expression: clause.From{},
		},
		"SELECT": {
			Name: "SELECT",
			Expression: clause.Select{
				Columns: []clause.Column{
					{Raw: true, Name: "`test_scope_models`.`id`"},
					{Raw: true, Name: "`test_scope_models`.`name`"},
					{Raw: true, Name: "`test_scope_models`.`email`"},
					{Raw: true, Name: "(UPPER(`test_scope_models`.name)) `computed`"},
					{Raw: true, Name: "`test_scope_models`.`relation_id`"},
				},
			},
		},
	}
	assert.Equal(t, expected, db.Statement.Clauses)
	assert.ElementsMatch(t, []string{"`test_scope_models`.`id`", "`test_scope_models`.`name`", "`test_scope_models`.`email`", "`test_scope_models`.`relation_id`", "(UPPER(`test_scope_models`.name)) `computed`"}, db.Statement.Selects)
}

func TestScopeUnpaginatedDisableSort(t *testing.T) {
	results, db := prepareTestScopeUnpaginated(t, &Settings{DisableSort: true, FieldsSearch: []string{"email"}})
	assert.NotNil(t, results)

	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.AndConditions{
						Exprs: []clause.Expression{
							clause.OrConditions{
								Exprs: []clause.Expression{
									clause.AndConditions{
										Exprs: []clause.Expression{
											clause.Expr{SQL: "`test_scope_models`.`name` LIKE ?", Vars: []interface{}{"%val1%"}},
											clause.Expr{SQL: "`test_scope_models`.`name` LIKE ?", Vars: []interface{}{"%val2%"}},
										},
									},
								},
							},
							clause.OrConditions{
								Exprs: []clause.Expression{
									clause.AndConditions{
										Exprs: []clause.Expression{
											clause.Expr{SQL: "`test_scope_models`.`name` = ?", Vars: []interface{}{"val3"}},
										},
									},
								},
							},
						},
					},
					clause.AndConditions{
						Exprs: []clause.Expression{
							clause.Expr{
								SQL:                "`test_scope_models`.`email` LIKE ?",
								Vars:               []interface{}{"%val%"},
								WithoutParentheses: false,
							},
						},
					},
				},
			},
		},
		"FROM": {
			Name:       "FROM",
			Expression: clause.From{},
		},
		"SELECT": {
			Name: "SELECT",
			Expression: clause.Select{
				Columns: []clause.Column{
					{Raw: true, Name: "`test_scope_models`.`id`"},
					{Raw: true, Name: "`test_scope_models`.`name`"},
					{Raw: true, Name: "`test_scope_models`.`email`"},
					{Raw: true, Name: "(UPPER(`test_scope_models`.name)) `computed`"},
					{Raw: true, Name: "`test_scope_models`.`relation_id`"},
				},
			},
		},
	}
	assert.Equal(t, expected, db.Statement.Clauses)
	assert.ElementsMatch(t, []string{"`test_scope_models`.`id`", "`test_scope_models`.`name`", "`test_scope_models`.`email`", "`test_scope_models`.`relation_id`", "(UPPER(`test_scope_models`.name)) `computed`"}, db.Statement.Selects)
}

func TestScopeDisableJoin(t *testing.T) {
	paginator, db := prepareTestScope(t, &Settings{DisableJoin: true, FieldsSearch: []string{"email"}})
	assert.NotNil(t, paginator)

	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.AndConditions{
						Exprs: []clause.Expression{
							clause.OrConditions{
								Exprs: []clause.Expression{
									clause.AndConditions{
										Exprs: []clause.Expression{
											clause.Expr{SQL: "`test_scope_models`.`name` LIKE ?", Vars: []interface{}{"%val1%"}},
											clause.Expr{SQL: "`test_scope_models`.`name` LIKE ?", Vars: []interface{}{"%val2%"}},
										},
									},
								},
							},
							clause.OrConditions{
								Exprs: []clause.Expression{
									clause.AndConditions{
										Exprs: []clause.Expression{
											clause.Expr{SQL: "`test_scope_models`.`name` = ?", Vars: []interface{}{"val3"}},
										},
									},
								},
							},
						},
					},
					clause.AndConditions{
						Exprs: []clause.Expression{
							clause.Expr{
								SQL:                "`test_scope_models`.`email` LIKE ?",
								Vars:               []interface{}{"%val%"},
								WithoutParentheses: false,
							},
						},
					},
				},
			},
		},
		"ORDER BY": {
			Name: "ORDER BY",
			Expression: clause.OrderBy{
				Columns: []clause.OrderByColumn{
					{
						Column: clause.Column{
							Table: "test_scope_models",
							Name:  "name",
						},
					},
				},
			},
		},
		"LIMIT": {
			Expression: clause.Limit{
				Limit:  &fifteen,
				Offset: 15,
			},
		},
		"FROM": {
			Name:       "FROM",
			Expression: clause.From{},
		},
		"SELECT": {
			Name: "SELECT",
			Expression: clause.Select{
				Columns: []clause.Column{
					{Raw: true, Name: "`test_scope_models`.`id`"},
					{Raw: true, Name: "`test_scope_models`.`name`"},
					{Raw: true, Name: "`test_scope_models`.`email`"},
					{Raw: true, Name: "(UPPER(`test_scope_models`.name)) `computed`"},
				},
			},
		},
	}
	assert.Equal(t, expected, db.Statement.Clauses)
	assert.Empty(t, db.Statement.Preloads)
	assert.ElementsMatch(t, []string{"`test_scope_models`.`id`", "`test_scope_models`.`name`", "`test_scope_models`.`email`", "(UPPER(`test_scope_models`.name)) `computed`"}, db.Statement.Selects)
}

func TestScopeUnpaginatedDisableJoin(t *testing.T) {
	results, db := prepareTestScopeUnpaginated(t, &Settings{DisableJoin: true, FieldsSearch: []string{"email"}})
	assert.NotNil(t, results)

	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.AndConditions{
						Exprs: []clause.Expression{
							clause.OrConditions{
								Exprs: []clause.Expression{
									clause.AndConditions{
										Exprs: []clause.Expression{
											clause.Expr{SQL: "`test_scope_models`.`name` LIKE ?", Vars: []interface{}{"%val1%"}},
											clause.Expr{SQL: "`test_scope_models`.`name` LIKE ?", Vars: []interface{}{"%val2%"}},
										},
									},
								},
							},
							clause.OrConditions{
								Exprs: []clause.Expression{
									clause.AndConditions{
										Exprs: []clause.Expression{
											clause.Expr{SQL: "`test_scope_models`.`name` = ?", Vars: []interface{}{"val3"}},
										},
									},
								},
							},
						},
					},
					clause.AndConditions{
						Exprs: []clause.Expression{
							clause.Expr{
								SQL:                "`test_scope_models`.`email` LIKE ?",
								Vars:               []interface{}{"%val%"},
								WithoutParentheses: false,
							},
						},
					},
				},
			},
		},
		"ORDER BY": {
			Name: "ORDER BY",
			Expression: clause.OrderBy{
				Columns: []clause.OrderByColumn{
					{
						Column: clause.Column{
							Table: "test_scope_models",
							Name:  "name",
						},
					},
				},
			},
		},
		"FROM": {
			Name:       "FROM",
			Expression: clause.From{},
		},
		"SELECT": {
			Name: "SELECT",
			Expression: clause.Select{
				Columns: []clause.Column{
					{Raw: true, Name: "`test_scope_models`.`id`"},
					{Raw: true, Name: "`test_scope_models`.`name`"},
					{Raw: true, Name: "`test_scope_models`.`email`"},
					{Raw: true, Name: "(UPPER(`test_scope_models`.name)) `computed`"},
				},
			},
		},
	}
	assert.Equal(t, expected, db.Statement.Clauses)
	assert.Empty(t, db.Statement.Preloads)
	assert.ElementsMatch(t, []string{"`test_scope_models`.`id`", "`test_scope_models`.`name`", "`test_scope_models`.`email`", "(UPPER(`test_scope_models`.name)) `computed`"}, db.Statement.Selects)
}

func TestScopeDisableSearch(t *testing.T) {
	paginator, db := prepareTestScope(t, &Settings{DisableSearch: true, FieldsSearch: []string{"name"}})
	assert.NotNil(t, paginator)

	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.AndConditions{
						Exprs: []clause.Expression{
							clause.OrConditions{
								Exprs: []clause.Expression{
									clause.AndConditions{
										Exprs: []clause.Expression{
											clause.Expr{SQL: "`test_scope_models`.`name` LIKE ?", Vars: []interface{}{"%val1%"}},
											clause.Expr{SQL: "`test_scope_models`.`name` LIKE ?", Vars: []interface{}{"%val2%"}},
										},
									},
								},
							},
							clause.OrConditions{
								Exprs: []clause.Expression{
									clause.AndConditions{
										Exprs: []clause.Expression{
											clause.Expr{SQL: "`test_scope_models`.`name` = ?", Vars: []interface{}{"val3"}},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		"ORDER BY": {
			Name: "ORDER BY",
			Expression: clause.OrderBy{
				Columns: []clause.OrderByColumn{
					{
						Column: clause.Column{
							Table: "test_scope_models",
							Name:  "name",
						},
					},
				},
			},
		},
		"LIMIT": {
			Expression: clause.Limit{
				Limit:  &fifteen,
				Offset: 15,
			},
		},
		"FROM": {
			Name:       "FROM",
			Expression: clause.From{},
		},
		"SELECT": {
			Name: "SELECT",
			Expression: clause.Select{
				Columns: []clause.Column{
					{Raw: true, Name: "`test_scope_models`.`id`"},
					{Raw: true, Name: "`test_scope_models`.`name`"},
					{Raw: true, Name: "`test_scope_models`.`email`"},
					{Raw: true, Name: "(UPPER(`test_scope_models`.name)) `computed`"},
					{Raw: true, Name: "`test_scope_models`.`relation_id`"},
				},
			},
		},
	}

	assert.Equal(t, expected, db.Statement.Clauses)
	assert.ElementsMatch(t, []string{"`test_scope_models`.`id`", "`test_scope_models`.`name`", "`test_scope_models`.`email`", "`test_scope_models`.`relation_id`", "(UPPER(`test_scope_models`.name)) `computed`"}, db.Statement.Selects)
}

func TestScopeUnpaginatedDisableSearch(t *testing.T) {
	results, db := prepareTestScopeUnpaginated(t, &Settings{DisableSearch: true, FieldsSearch: []string{"name"}})
	assert.NotNil(t, results)

	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.AndConditions{
						Exprs: []clause.Expression{
							clause.OrConditions{
								Exprs: []clause.Expression{
									clause.AndConditions{
										Exprs: []clause.Expression{
											clause.Expr{SQL: "`test_scope_models`.`name` LIKE ?", Vars: []interface{}{"%val1%"}},
											clause.Expr{SQL: "`test_scope_models`.`name` LIKE ?", Vars: []interface{}{"%val2%"}},
										},
									},
								},
							},
							clause.OrConditions{
								Exprs: []clause.Expression{
									clause.AndConditions{
										Exprs: []clause.Expression{
											clause.Expr{SQL: "`test_scope_models`.`name` = ?", Vars: []interface{}{"val3"}},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		"ORDER BY": {
			Name: "ORDER BY",
			Expression: clause.OrderBy{
				Columns: []clause.OrderByColumn{
					{
						Column: clause.Column{
							Table: "test_scope_models",
							Name:  "name",
						},
					},
				},
			},
		},
		"FROM": {
			Name:       "FROM",
			Expression: clause.From{},
		},
		"SELECT": {
			Name: "SELECT",
			Expression: clause.Select{
				Columns: []clause.Column{
					{Raw: true, Name: "`test_scope_models`.`id`"},
					{Raw: true, Name: "`test_scope_models`.`name`"},
					{Raw: true, Name: "`test_scope_models`.`email`"},
					{Raw: true, Name: "(UPPER(`test_scope_models`.name)) `computed`"},
					{Raw: true, Name: "`test_scope_models`.`relation_id`"},
				},
			},
		},
	}

	assert.Equal(t, expected, db.Statement.Clauses)
	assert.ElementsMatch(t, []string{"`test_scope_models`.`id`", "`test_scope_models`.`name`", "`test_scope_models`.`email`", "`test_scope_models`.`relation_id`", "(UPPER(`test_scope_models`.name)) `computed`"}, db.Statement.Selects)
}

func TestScopeNoPrimaryKey(t *testing.T) {
	request := &goyave.Request{
		Data: map[string]interface{}{
			"fields": "name",
			"join":   []*Join{{Relation: "Relation", Fields: []string{"a", "b"}}},
		},
		Lang: "en-US",
	}
	db := openDryRunDB(t)

	results := []*TestScopeModelNoPrimaryKey{}
	paginator, db := Scope(db, request, results)
	assert.Nil(t, paginator)
	assert.Equal(t, "Could not find primary key. Add `gorm:\"primaryKey\"` to your model", db.Error.Error())
}

func TestScopeUnpaginatedNoPrimaryKey(t *testing.T) {
	request := &goyave.Request{
		Data: map[string]interface{}{
			"fields": "name",
			"join":   []*Join{{Relation: "Relation", Fields: []string{"a", "b"}}},
		},
		Lang: "en-US",
	}
	db := openDryRunDB(t)

	var results []*TestScopeModelNoPrimaryKey
	db = ScopeUnpaginated(db, request, &results)
	assert.Nil(t, results)
	assert.Equal(t, "Could not find primary key. Add `gorm:\"primaryKey\"` to your model", db.Error.Error())
}

func TestScopeWithFieldsBlacklist(t *testing.T) {
	request := &goyave.Request{
		Data: map[string]interface{}{},
		Lang: "en-US",
	}
	db := openDryRunDB(t)
	settings := &Settings{
		Blacklist: Blacklist{
			FieldsBlacklist: []string{"name"},
		},
	}
	results := []*TestScopeModel{}
	paginator, db := settings.Scope(db, request, results)
	assert.NotNil(t, paginator)
	assert.ElementsMatch(t, []string{"`test_scope_models`.`id`", "`test_scope_models`.`relation_id`", "`test_scope_models`.`email`", "(UPPER(`test_scope_models`.name)) `computed`"}, db.Statement.Selects)
}

func TestScopeUnpaginatedWithFieldsBlacklist(t *testing.T) {
	request := &goyave.Request{
		Data: map[string]interface{}{},
		Lang: "en-US",
	}
	db := openDryRunDB(t)
	settings := &Settings{
		Blacklist: Blacklist{
			FieldsBlacklist: []string{"name"},
		},
	}
	var results []*TestScopeModel
	db = settings.ScopeUnpaginated(db, request, &results)
	assert.Nil(t, db.Error)
	assert.ElementsMatch(t, []string{"`test_scope_models`.`id`", "`test_scope_models`.`relation_id`", "`test_scope_models`.`email`", "(UPPER(`test_scope_models`.name)) `computed`"}, db.Statement.Selects)
}

func TestScopeInvalidModel(t *testing.T) {
	request := &goyave.Request{
		Data: map[string]interface{}{},
		Lang: "en-US",
	}
	db := openDryRunDB(t)
	model := []string{}
	assert.Panics(t, func() {
		Scope(db, request, model)
	})
}

func TestScopeUnpaginatedInvalidModel(t *testing.T) {
	request := &goyave.Request{
		Data: map[string]interface{}{},
		Lang: "en-US",
	}
	db := openDryRunDB(t)
	model := []string{}
	assert.Panics(t, func() {
		ScopeUnpaginated(db, request, model)
	})
}

func TestBlacklistGetSelectableFields(t *testing.T) {
	blacklist := &Blacklist{
		FieldsBlacklist: []string{"name"},
	}
	fields := map[string]*schema.Field{
		"id":    {},
		"name":  {},
		"email": {},
	}

	sch := &schema.Schema{
		DBNames:        []string{"id", "name", "email"},
		FieldsByDBName: fields,
	}

	assert.ElementsMatch(t, []*schema.Field{fields["id"], fields["email"]}, getSelectableFields(blacklist, sch))
	assert.ElementsMatch(t, []*schema.Field{fields["id"], fields["email"], fields["name"]}, getSelectableFields(nil, sch))
}

type TestFilterScopeModel struct {
	Name string
	ID   int `gorm:"primaryKey"`
}

func (m *TestFilterScopeModel) TableName() string {
	return "test_scope_models"
}

func TestApplyFiltersAnd(t *testing.T) {
	request := &goyave.Request{
		Data: map[string]interface{}{
			"filter": []*Filter{
				{Field: "name", Args: []string{"val1"}, Operator: Operators["$cont"]},
				{Field: "name", Args: []string{"val2"}, Operator: Operators["$cont"]},
			},
		},
		Lang: "en-US",
	}
	db := openDryRunDB(t)
	schema, err := parseModel(db, &TestFilterScopeModel{})
	if !assert.Nil(t, err) {
		return
	}

	db = (&Settings{}).applyFilters(db, request, schema).Find(nil)
	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.AndConditions{
						Exprs: []clause.Expression{
							clause.AndConditions{
								Exprs: []clause.Expression{
									clause.Expr{SQL: "`test_scope_models`.`name` LIKE ?", Vars: []interface{}{"%val1%"}},
									clause.Expr{SQL: "`test_scope_models`.`name` LIKE ?", Vars: []interface{}{"%val2%"}},
								},
							},
						},
					},
				},
			},
		},
		"FROM": {
			Name:       "FROM",
			Expression: clause.From{},
		},
		"SELECT": {
			Name:       "SELECT",
			Expression: clause.Select{},
		},
	}

	assert.Equal(t, expected, db.Statement.Clauses)
}

func TestApplyFiltersOr(t *testing.T) {
	request := &goyave.Request{
		Data: map[string]interface{}{
			"or": []*Filter{
				{Field: "name", Args: []string{"val1"}, Operator: Operators["$cont"], Or: true},
				{Field: "name", Args: []string{"val2"}, Operator: Operators["$cont"], Or: true},
			},
		},
		Lang: "en-US",
	}
	db := openDryRunDB(t)
	schema, err := parseModel(db, &TestFilterScopeModel{})
	if !assert.Nil(t, err) {
		return
	}

	db = (&Settings{}).applyFilters(db, request, schema).Find(nil)
	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.AndConditions{
						Exprs: []clause.Expression{
							clause.AndConditions{
								Exprs: []clause.Expression{
									clause.OrConditions{
										Exprs: []clause.Expression{
											clause.Expr{SQL: "`test_scope_models`.`name` LIKE ?", Vars: []interface{}{"%val1%"}},
										},
									},
									clause.OrConditions{
										Exprs: []clause.Expression{
											clause.Expr{SQL: "`test_scope_models`.`name` LIKE ?", Vars: []interface{}{"%val2%"}},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		"FROM": {
			Name:       "FROM",
			Expression: clause.From{},
		},
		"SELECT": {
			Name:       "SELECT",
			Expression: clause.Select{},
		},
	}

	assert.Equal(t, expected, db.Statement.Clauses)
}

func TestApplyFiltersMixed(t *testing.T) {
	request := &goyave.Request{
		Data: map[string]interface{}{
			"filter": []*Filter{
				{Field: "name", Args: []string{"val1"}, Operator: Operators["$cont"]},
				{Field: "name", Args: []string{"val2"}, Operator: Operators["$cont"]},
			},
			"or": []*Filter{
				{Field: "name", Args: []string{"val3"}, Or: true, Operator: Operators["$eq"]},
				{Field: "name", Args: []string{"val4"}, Or: true, Operator: Operators["$eq"]},
			},
		},
		Lang: "en-US",
	}
	db := openDryRunDB(t)
	schema, err := parseModel(db, &TestFilterScopeModel{})
	if !assert.Nil(t, err) {
		return
	}

	db = (&Settings{}).applyFilters(db, request, schema).Find(nil)
	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.AndConditions{
						Exprs: []clause.Expression{
							clause.OrConditions{
								Exprs: []clause.Expression{
									clause.AndConditions{
										Exprs: []clause.Expression{
											clause.Expr{SQL: "`test_scope_models`.`name` LIKE ?", Vars: []interface{}{"%val1%"}},
											clause.Expr{SQL: "`test_scope_models`.`name` LIKE ?", Vars: []interface{}{"%val2%"}},
										},
									},
								},
							},
							clause.OrConditions{
								Exprs: []clause.Expression{
									clause.AndConditions{
										Exprs: []clause.Expression{
											clause.Expr{SQL: "`test_scope_models`.`name` = ?", Vars: []interface{}{"val3"}},
											clause.Expr{SQL: "`test_scope_models`.`name` = ?", Vars: []interface{}{"val4"}},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		"FROM": {
			Name:       "FROM",
			Expression: clause.From{},
		},
		"SELECT": {
			Name:       "SELECT",
			Expression: clause.Select{},
		},
	}

	assert.Equal(t, expected, db.Statement.Clauses)
}

func TestApplyFiltersWithJoin(t *testing.T) {
	request := &goyave.Request{
		Data: map[string]interface{}{
			"filter": []*Filter{
				{Field: "Relation.name", Args: []string{"val1"}, Operator: Operators["$cont"]},
			},
		},
		Lang: "en-US",
	}
	db := openDryRunDB(t)
	schema, err := parseModel(db, &FilterTestModel{})
	if !assert.Nil(t, err) {
		return
	}

	results := []*FilterTestModel{}
	db = db.Model(&results)
	db = (&Settings{}).applyFilters(db, request, schema).Find(&results)
	assert.Nil(t, db.Statement.Error)
	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.AndConditions{
						Exprs: []clause.Expression{
							clause.Expr{SQL: "`Relation`.`name` LIKE ?", Vars: []interface{}{"%val1%"}},
						},
					},
				},
			},
		},
		"FROM": {
			Name: "FROM",
			Expression: clause.From{
				Joins: []clause.Join{
					{
						Type: clause.LeftJoin,
						Table: clause.Table{
							Name:  "filter_test_relations",
							Alias: "Relation",
						},
						ON: clause.Where{
							Exprs: []clause.Expression{
								clause.Eq{
									Column: clause.Column{
										Table: "filter_test_models",
										Name:  "id",
									},
									Value: clause.Column{
										Table: "Relation",
										Name:  "parent_id",
									},
								},
							},
						},
					},
				},
			},
		},
		"SELECT": {
			Name: "SELECT",
			Expression: clause.Select{
				Columns: []clause.Column{
					{Table: "filter_test_models", Name: "name"},
					{Table: "filter_test_models", Name: "id"},
				},
			},
		},
	}

	assert.Equal(t, expected, db.Statement.Clauses)
}

func TestApplySearch(t *testing.T) {
	request := &goyave.Request{
		Data: map[string]interface{}{
			"search": "val",
		},
		Lang: "en-US",
	}
	db := openDryRunDB(t)
	schema, err := parseModel(db, &TestFilterScopeModel{})
	if !assert.Nil(t, err) {
		return
	}

	search := (&Settings{}).applySearch(request, schema)
	assert.NotNil(t, search)
	assert.ElementsMatch(t, []string{"id", "name"}, search.Fields)
	assert.Equal(t, "val", search.Query)
	assert.Equal(t, Operators["$cont"], search.Operator)
}

func TestApplySearchNoQuery(t *testing.T) {
	request := &goyave.Request{
		Data: map[string]interface{}{},
		Lang: "en-US",
	}
	assert.Nil(t, (&Settings{}).applySearch(request, &schema.Schema{}))
}

func TestSelectScope(t *testing.T) {
	db := openDryRunDB(t)
	db = db.Scopes(selectScope("", nil, false)).Find(nil)
	assert.Empty(t, db.Statement.Selects)
	assert.Empty(t, db.Statement.Clauses["SELECT"].Expression.(clause.Select).Columns)

	db = openDryRunDB(t)
	db = db.Scopes(selectScope("", nil, true)).Find(nil)
	assert.Empty(t, db.Statement.Selects)
	assert.Empty(t, db.Statement.Clauses["SELECT"].Expression.(clause.Select).Columns)

	sch := &schema.Schema{
		Table: "test_models",
	}

	db = openDryRunDB(t)
	db = db.Scopes(selectScope(sch.Table, []*schema.Field{{DBName: "a"}, {DBName: "b"}}, false)).Find(nil)
	assert.Equal(t, []string{"`test_models`.`a`", "`test_models`.`b`"}, db.Statement.Selects)
	assert.Equal(t, []clause.Column{{Raw: true, Name: "`test_models`.`a`"}, {Raw: true, Name: "`test_models`.`b`"}}, db.Statement.Clauses["SELECT"].Expression.(clause.Select).Columns)

	db = openDryRunDB(t)
	db = db.Scopes(selectScope(sch.Table, []*schema.Field{{DBName: "a"}, {DBName: "b"}}, true)).Find(nil)
	assert.Equal(t, []string{"`test_models`.`a`", "`test_models`.`b`"}, db.Statement.Selects)
	assert.Equal(t, []clause.Column{{Raw: true, Name: "`test_models`.`a`"}, {Raw: true, Name: "`test_models`.`b`"}}, db.Statement.Clauses["SELECT"].Expression.(clause.Select).Columns)

	db = openDryRunDB(t)
	db = db.Scopes(selectScope(sch.Table, []*schema.Field{{DBName: "a"}, {DBName: "b"}}, false)).Select("1 + 1 AS count").Find(nil)
	assert.Equal(t, []string{"1 + 1 AS count", "`test_models`.`a`", "`test_models`.`b`"}, db.Statement.Selects)
	assert.Equal(t, []clause.Column{{Raw: true, Name: "1 + 1 AS count"}, {Raw: true, Name: "`test_models`.`a`"}, {Raw: true, Name: "`test_models`.`b`"}}, db.Statement.Clauses["SELECT"].Expression.(clause.Select).Columns)

	db = openDryRunDB(t)
	db = db.Scopes(selectScope(sch.Table, []*schema.Field{}, true)).Select("*, 1 + 1 AS count").Find(nil)
	assert.Equal(t, []string{"1"}, db.Statement.Selects)
	assert.Equal(t, []clause.Column{{Raw: true, Name: "1"}}, db.Statement.Clauses["SELECT"].Expression.(clause.Select).Columns)

	db = openDryRunDB(t)
	db = db.Scopes(selectScope(sch.Table, []*schema.Field{{DBName: "c", StructField: reflect.StructField{Tag: `computed:"a+b"`}}}, true)).Select("*, 1 + 1 AS count").Find(nil)
	assert.Equal(t, []string{"(a+b) `c`"}, db.Statement.Selects)
	assert.Equal(t, []clause.Column{{Raw: true, Name: "(a+b) `c`"}}, db.Statement.Clauses["SELECT"].Expression.(clause.Select).Columns)

	db = openDryRunDB(t)
	db = db.Joins("relation").Scopes(selectScope(sch.Table, []*schema.Field{{DBName: "c", StructField: reflect.StructField{Tag: `computed:"a+b"`}}}, true)).Select("*, 1 + 1 AS count").Find(nil)
	assert.Equal(t, []string{"(a+b) `c`"}, db.Statement.Selects)
	assert.Equal(t, []clause.Column{{Raw: true, Name: "(a+b) `c`"}}, db.Statement.Clauses["SELECT"].Expression.(clause.Select).Columns)
}

func TestGetFieldFinalRelation(t *testing.T) {
	db := openDryRunDB(t)
	schema, err := parseModel(db, &FilterTestModel{})
	if !assert.Nil(t, err) {
		return
	}

	settings := &Settings{Blacklist: Blacklist{IsFinal: true}}
	field, sch, joinName := getField("Relation.name", schema, &settings.Blacklist)
	assert.Nil(t, field)
	assert.Nil(t, sch)
	assert.Empty(t, joinName)

	settings = &Settings{Blacklist: Blacklist{
		Relations: map[string]*Blacklist{
			"Relation": {IsFinal: true},
		},
	}}
	field, sch, joinName = getField("Relation.NestedRelation.field", schema, &settings.Blacklist)
	assert.Nil(t, field)
	assert.Nil(t, sch)
	assert.Empty(t, joinName)
}

func TestSettingsComputedFieldWithAutoFields(t *testing.T) {
	request := &goyave.Request{
		Data: map[string]interface{}{
			"filter": []*Filter{
				{Field: "Relation.a", Args: []string{"val1"}, Operator: Operators["$cont"]},
			},
			"per_page": 15,
		},
		Lang: "en-US",
	}
	db := openDryRunDB(t)

	results := []*TestScopeModel{}
	paginator, db := Scope(db, request, results)

	assert.NotNil(t, paginator)

	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.AndConditions{
						Exprs: []clause.Expression{
							clause.Expr{SQL: "`Relation`.`a` LIKE ?", Vars: []interface{}{"%val1%"}},
						},
					},
				},
			},
		},
		"LIMIT": {
			Expression: clause.Limit{
				Limit:  &fifteen,
				Offset: 0,
			},
		},
		"FROM": {
			Name: "FROM",
			Expression: clause.From{
				Joins: []clause.Join{
					{
						Type: clause.LeftJoin,
						Table: clause.Table{
							Name:  "test_scope_relations",
							Alias: "Relation",
						},
						ON: clause.Where{
							Exprs: []clause.Expression{
								clause.Eq{
									Column: clause.Column{
										Table: "test_scope_models",
										Name:  "relation_id",
									},
									Value: clause.Column{
										Table: "Relation",
										Name:  "id",
									},
								},
							},
						},
					},
				},
			},
		},
		"SELECT": {
			Name: "SELECT",
			Expression: clause.Select{
				Columns: []clause.Column{
					{Raw: true, Name: "`test_scope_models`.`name`"},
					{Raw: true, Name: "`test_scope_models`.`email`"},
					{Raw: true, Name: "(UPPER(`test_scope_models`.name)) `computed`"},
					{Raw: true, Name: "`test_scope_models`.`id`"},
					{Raw: true, Name: "`test_scope_models`.`relation_id`"},
				},
			},
		},
	}
	assert.Equal(t, expected, db.Statement.Clauses)
	assert.ElementsMatch(t, []string{"`test_scope_models`.`name`", "`test_scope_models`.`email`", "(UPPER(`test_scope_models`.name)) `computed`", "`test_scope_models`.`id`", "`test_scope_models`.`relation_id`"}, db.Statement.Selects)
}

func TestSettingsSelectWithExistingJoin(t *testing.T) {
	request := &goyave.Request{
		Data: map[string]interface{}{
			"filter": []*Filter{
				{Field: "Relation.a", Args: []string{"val1"}, Operator: Operators["$cont"]},
			},
			"per_page": 15,
		},
		Lang: "en-US",
	}
	db := openDryRunDB(t)

	// We manually join a relation with a condition.
	// We expect this join to not be removed nor duplicated, with the condition kept and the fields selected.
	db = db.Joins("Relation", db.Session(&gorm.Session{NewDB: true}).Where("Relation.id > ?", 0))

	results := []*TestScopeModel{}
	paginator, db := Scope(db, request, results)

	assert.NotNil(t, paginator)

	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.AndConditions{
						Exprs: []clause.Expression{
							clause.Expr{SQL: "`Relation`.`a` LIKE ?", Vars: []interface{}{"%val1%"}},
						},
					},
				},
			},
		},
		"FROM": {
			Name: "FROM",
			Expression: clause.From{
				Joins: []clause.Join{
					{
						Type: clause.LeftJoin,
						Table: clause.Table{
							Name:  "test_scope_relations",
							Alias: "Relation",
						},
						ON: clause.Where{
							Exprs: []clause.Expression{
								clause.Eq{
									Column: clause.Column{
										Table: clause.CurrentTable,
										Name:  "relation_id",
									},
									Value: clause.Column{
										Table: "Relation",
										Name:  "id",
									},
								},
								clause.Expr{SQL: "Relation.id > ?", Vars: []interface{}{0}},
							},
						},
					},
				},
			},
		},
		"LIMIT": {
			Expression: clause.Limit{
				Limit:  &fifteen,
				Offset: 0,
			},
		},
		"SELECT": {
			Name: "SELECT",
			Expression: clause.Select{
				Columns: []clause.Column{
					{Raw: true, Name: "`Relation`.`a` `Relation__a`"},
					{Raw: true, Name: "`Relation`.`b` `Relation__b`"},
					{Raw: true, Name: "`Relation`.`id` `Relation__id`"},
					{Raw: true, Name: "`test_scope_models`.`name`"},
					{Raw: true, Name: "`test_scope_models`.`email`"},
					{Raw: true, Name: "(UPPER(`test_scope_models`.name)) `computed`"},
					{Raw: true, Name: "`test_scope_models`.`id`"},
					{Raw: true, Name: "`test_scope_models`.`relation_id`"},
				},
			},
		},
	}
	assert.Equal(t, expected, db.Statement.Clauses)
	assert.Empty(t, db.Statement.Joins)
}

type TestScopeRelationWithComputed struct {
	A  string
	B  string
	C  string `gorm:"->;-:migration" computed:"UPPER(~~~ct~~~.b)"`
	ID uint
}
type TestScopeModelWithComputed struct {
	Relation   *TestScopeRelationWithComputed
	Name       string
	Email      string
	Computed   string `gorm:"->;-:migration" computed:"UPPER(~~~ct~~~.name)"`
	ID         uint
	RelationID uint
}

func TestSettingsSelectWithExistingJoinAndComputed(t *testing.T) {
	request := &goyave.Request{
		Data: map[string]interface{}{
			"filter": []*Filter{
				{Field: "Relation.a", Args: []string{"val1"}, Operator: Operators["$cont"]},
			},
			"per_page": 15,
		},
		Lang: "en-US",
	}
	db := openDryRunDB(t)

	db = db.Joins("Relation")

	results := []*TestScopeModelWithComputed{}
	paginator, db := Scope(db, request, results)

	assert.NotNil(t, paginator)

	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.AndConditions{
						Exprs: []clause.Expression{
							clause.Expr{SQL: "`Relation`.`a` LIKE ?", Vars: []interface{}{"%val1%"}},
						},
					},
				},
			},
		},
		"FROM": {
			Name: "FROM",
			Expression: clause.From{
				Joins: []clause.Join{
					{
						Type: clause.LeftJoin,
						Table: clause.Table{
							Name:  "test_scope_relation_with_computeds",
							Alias: "Relation",
						},
						ON: clause.Where{
							Exprs: []clause.Expression{
								clause.Eq{
									Column: clause.Column{
										Table: clause.CurrentTable,
										Name:  "relation_id",
									},
									Value: clause.Column{
										Table: "Relation",
										Name:  "id",
									},
								},
							},
						},
					},
				},
			},
		},
		"LIMIT": {
			Expression: clause.Limit{
				Limit:  &fifteen,
				Offset: 0,
			},
		},
		"SELECT": {
			Name: "SELECT",
			Expression: clause.Select{
				Columns: []clause.Column{
					{Raw: true, Name: "`Relation`.`a` `Relation__a`"},
					{Raw: true, Name: "`Relation`.`b` `Relation__b`"},
					{Raw: true, Name: "(UPPER(`Relation`.b)) `Relation__c`"},
					{Raw: true, Name: "`Relation`.`id` `Relation__id`"},
					{Raw: true, Name: "`test_scope_model_with_computeds`.`name`"},
					{Raw: true, Name: "`test_scope_model_with_computeds`.`email`"},
					{Raw: true, Name: "(UPPER(`test_scope_model_with_computeds`.name)) `computed`"},
					{Raw: true, Name: "`test_scope_model_with_computeds`.`id`"},
					{Raw: true, Name: "`test_scope_model_with_computeds`.`relation_id`"},
				},
			},
		},
	}
	assert.Equal(t, expected, db.Statement.Clauses)
	assert.Empty(t, db.Statement.Joins)
}

func TestSettingsSelectWithExistingJoinAndComputedOmit(t *testing.T) {
	request := &goyave.Request{
		Data: map[string]interface{}{
			"filter": []*Filter{
				{Field: "Relation.a", Args: []string{"val1"}, Operator: Operators["$cont"]},
			},
			"per_page": 15,
		},
		Lang: "en-US",
	}
	db := openDryRunDB(t)

	// We use Omit on Relation to avoid Gorm auto-selecting it
	db = db.Joins("Relation", db.Session(&gorm.Session{NewDB: true}).Omit("c"))

	results := []*TestScopeModelWithComputed{}
	paginator, db := Scope(db, request, results)

	assert.NotNil(t, paginator)

	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.AndConditions{
						Exprs: []clause.Expression{
							clause.Expr{SQL: "`Relation`.`a` LIKE ?", Vars: []interface{}{"%val1%"}},
						},
					},
				},
			},
		},
		"FROM": {
			Name: "FROM",
			Expression: clause.From{
				Joins: []clause.Join{
					{
						Type: clause.LeftJoin,
						Table: clause.Table{
							Name:  "test_scope_relation_with_computeds",
							Alias: "Relation",
						},
						ON: clause.Where{
							Exprs: []clause.Expression{
								clause.Eq{
									Column: clause.Column{
										Table: clause.CurrentTable,
										Name:  "relation_id",
									},
									Value: clause.Column{
										Table: "Relation",
										Name:  "id",
									},
								},
							},
						},
					},
				},
			},
		},
		"LIMIT": {
			Expression: clause.Limit{
				Limit:  &fifteen,
				Offset: 0,
			},
		},
		"SELECT": {
			Name: "SELECT",
			Expression: clause.Select{
				Columns: []clause.Column{
					{Raw: true, Name: "`Relation`.`a` `Relation__a`"},
					{Raw: true, Name: "`Relation`.`b` `Relation__b`"},
					{Raw: true, Name: "`Relation`.`id` `Relation__id`"},
					{Raw: true, Name: "`test_scope_model_with_computeds`.`name`"},
					{Raw: true, Name: "`test_scope_model_with_computeds`.`email`"},
					{Raw: true, Name: "(UPPER(`test_scope_model_with_computeds`.name)) `computed`"},
					{Raw: true, Name: "`test_scope_model_with_computeds`.`id`"},
					{Raw: true, Name: "`test_scope_model_with_computeds`.`relation_id`"},
				},
			},
		},
	}
	assert.Equal(t, expected, db.Statement.Clauses)
	assert.Empty(t, db.Statement.Joins)
}

func TestSettingsSelectWithExistingJoinAndComputedWithoutFiltering(t *testing.T) {
	request := &goyave.Request{
		Data: map[string]interface{}{
			"per_page": 15,
		},
		Lang: "en-US",
	}
	db := openDryRunDB(t)

	// Gorm will automatically select all the fields of the relation.
	// We expect the computed fields to work properly.
	db = db.Joins("Relation", db.Session(&gorm.Session{NewDB: true}).Where("Relation.id > ?", 0))

	results := []*TestScopeModelWithComputed{}
	paginator, db := Scope(db, request, results)

	assert.NotNil(t, paginator)

	expected := map[string]clause.Clause{
		"FROM": {
			Name: "FROM",
			Expression: clause.From{
				Joins: []clause.Join{
					{
						Type: clause.LeftJoin,
						Table: clause.Table{
							Name:  "test_scope_relation_with_computeds",
							Alias: "Relation",
						},
						ON: clause.Where{
							Exprs: []clause.Expression{
								clause.Eq{
									Column: clause.Column{
										Table: clause.CurrentTable,
										Name:  "relation_id",
									},
									Value: clause.Column{
										Table: "Relation",
										Name:  "id",
									},
								},
								clause.Expr{SQL: "Relation.id > ?", Vars: []interface{}{0}},
							},
						},
					},
				},
			},
		},
		"LIMIT": {
			Expression: clause.Limit{
				Limit:  &fifteen,
				Offset: 0,
			},
		},
		"SELECT": {
			Name: "SELECT",
			Expression: clause.Select{
				Columns: []clause.Column{
					{Raw: true, Name: "`Relation`.`a` `Relation__a`"},
					{Raw: true, Name: "`Relation`.`b` `Relation__b`"},
					{Raw: true, Name: "(UPPER(`Relation`.b)) `Relation__c`"},
					{Raw: true, Name: "`Relation`.`id` `Relation__id`"},
					{Raw: true, Name: "`test_scope_model_with_computeds`.`name`"},
					{Raw: true, Name: "`test_scope_model_with_computeds`.`email`"},
					{Raw: true, Name: "(UPPER(`test_scope_model_with_computeds`.name)) `computed`"},
					{Raw: true, Name: "`test_scope_model_with_computeds`.`id`"},
					{Raw: true, Name: "`test_scope_model_with_computeds`.`relation_id`"},
				},
			},
		},
	}
	assert.Equal(t, expected, db.Statement.Clauses)
	assert.Empty(t, db.Statement.Joins)
}
