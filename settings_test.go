package filter

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
	"gorm.io/gorm/utils/tests"
	"goyave.dev/goyave/v4"
	"goyave.dev/goyave/v4/database"
)

type TestScopeRelation struct {
	A  string
	B  string
	ID uint
}
type TestScopeModel struct {
	Relation   *TestScopeRelation
	Name       string
	Email      string
	ID         uint
	RelationID uint
}
type TestScopeModelNoPrimaryKey struct {
	Relation   *TestScopeRelation
	Name       string
	RelationID uint
}

func prepareTestScope(settings *Settings) (*database.Paginator, *gorm.DB) {
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
			"fields":   "id,name,email",
			"search":   "val",
		},
		Lang: "en-US",
	}
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)

	results := []*TestScopeModel{}
	if settings == nil {
		return Scope(db, request, results)
	}

	return settings.Scope(db, request, results)
}

func TestScope(t *testing.T) {
	paginator, db := prepareTestScope(&Settings{
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
				Limit:  15,
				Offset: 15,
			},
		},
	}
	assert.Equal(t, expected, db.Statement.Clauses)
	assert.Contains(t, db.Statement.Preloads, "Relation")
	assert.Equal(t, []string{"`test_scope_models`.`id`", "`test_scope_models`.`name`", "`test_scope_models`.`email`", "`test_scope_models`.`relation_id`"}, db.Statement.Selects)
}

func TestScopeDisableFields(t *testing.T) {
	paginator, db := prepareTestScope(&Settings{DisableFields: true, FieldsSearch: []string{"email"}})
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
				Limit:  15,
				Offset: 15,
			},
		},
	}
	assert.Equal(t, expected, db.Statement.Clauses)
	assert.ElementsMatch(t, []string{"`test_scope_models`.`id`", "`test_scope_models`.`relation_id`", "`test_scope_models`.`name`", "`test_scope_models`.`email`"}, db.Statement.Selects)
}

func TestScopeDisableFilter(t *testing.T) {
	paginator, db := prepareTestScope(&Settings{DisableFilter: true, FieldsSearch: []string{"email"}})
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
				Limit:  15,
				Offset: 15,
			},
		},
	}
	assert.Equal(t, expected, db.Statement.Clauses)
	assert.Equal(t, []string{"`test_scope_models`.`id`", "`test_scope_models`.`name`", "`test_scope_models`.`email`", "`test_scope_models`.`relation_id`"}, db.Statement.Selects)
}

func TestScopeDisableSort(t *testing.T) {
	paginator, db := prepareTestScope(&Settings{DisableSort: true, FieldsSearch: []string{"email"}})
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
				Limit:  15,
				Offset: 15,
			},
		},
	}
	assert.Equal(t, expected, db.Statement.Clauses)
	assert.Equal(t, []string{"`test_scope_models`.`id`", "`test_scope_models`.`name`", "`test_scope_models`.`email`", "`test_scope_models`.`relation_id`"}, db.Statement.Selects)
}

func TestScopeDisableJoin(t *testing.T) {
	paginator, db := prepareTestScope(&Settings{DisableJoin: true, FieldsSearch: []string{"email"}})
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
				Limit:  15,
				Offset: 15,
			},
		},
	}
	assert.Equal(t, expected, db.Statement.Clauses)
	assert.Empty(t, db.Statement.Preloads)
	assert.Equal(t, []string{"`test_scope_models`.`id`", "`test_scope_models`.`name`", "`test_scope_models`.`email`"}, db.Statement.Selects)
}

func TestScopeDisableSearch(t *testing.T) {
	paginator, db := prepareTestScope(&Settings{DisableSearch: true, FieldsSearch: []string{"name"}})
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
				Limit:  15,
				Offset: 15,
			},
		},
	}

	assert.Equal(t, expected, db.Statement.Clauses)
	assert.Equal(t, []string{"`test_scope_models`.`id`", "`test_scope_models`.`name`", "`test_scope_models`.`email`", "`test_scope_models`.`relation_id`"}, db.Statement.Selects)
}

func TestScopeNoPrimaryKey(t *testing.T) {
	request := &goyave.Request{
		Data: map[string]interface{}{
			"fields": "name",
			"join":   []*Join{{Relation: "Relation", Fields: []string{"a", "b"}}},
		},
		Lang: "en-US",
	}
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)

	results := []*TestScopeModelNoPrimaryKey{}
	paginator, db := Scope(db, request, results)
	assert.Nil(t, paginator)
	assert.Equal(t, "Could not find primary key. Add `gorm:\"primaryKey\"` to your model", db.Error.Error())
}

func TestScopeWithFieldsBlacklist(t *testing.T) {
	request := &goyave.Request{
		Data: map[string]interface{}{},
		Lang: "en-US",
	}
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
	settings := &Settings{
		Blacklist: Blacklist{
			FieldsBlacklist: []string{"name"},
		},
	}
	results := []*TestScopeModel{}
	paginator, db := settings.Scope(db, request, results)
	assert.NotNil(t, paginator)
	assert.ElementsMatch(t, []string{"`test_scope_models`.`id`", "`test_scope_models`.`relation_id`", "`test_scope_models`.`email`"}, db.Statement.Selects)
}

func TestScopeInvalidModel(t *testing.T) {
	request := &goyave.Request{
		Data: map[string]interface{}{},
		Lang: "en-US",
	}
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
	model := []string{}
	assert.Panics(t, func() {
		Scope(db, request, model)
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

	assert.ElementsMatch(t, []string{"id", "email"}, blacklist.getSelectableFields(fields))
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
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
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
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
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
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
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
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
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
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
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
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
	db = db.Scopes(selectScope("", nil, false)).Find(nil)
	assert.Empty(t, db.Statement.Selects)

	db, _ = gorm.Open(&tests.DummyDialector{}, nil)
	db = db.Scopes(selectScope("", nil, true)).Find(nil)
	assert.Empty(t, db.Statement.Selects)

	schema := &schema.Schema{Table: "test_models"}

	db, _ = gorm.Open(&tests.DummyDialector{}, nil)
	db = db.Scopes(selectScope(schema.Table, []string{"a", "b"}, false)).Find(nil)
	assert.Equal(t, []string{"`test_models`.`a`", "`test_models`.`b`"}, db.Statement.Selects)

	db, _ = gorm.Open(&tests.DummyDialector{}, nil)
	db = db.Scopes(selectScope(schema.Table, []string{"a", "b"}, true)).Find(nil)
	assert.Equal(t, []string{"`test_models`.`a`", "`test_models`.`b`"}, db.Statement.Selects)

	db, _ = gorm.Open(&tests.DummyDialector{}, nil)
	db = db.Scopes(selectScope(schema.Table, []string{"a", "b"}, false)).Select("1 + 1 AS count").Find(nil)
	assert.Equal(t, []string{"1 + 1 AS count", "`test_models`.`a`", "`test_models`.`b`"}, db.Statement.Selects)

	db, _ = gorm.Open(&tests.DummyDialector{}, nil)
	db = db.Scopes(selectScope(schema.Table, []string{}, true)).Select("*, 1 + 1 AS count").Find(nil)
	assert.Equal(t, []string{"1"}, db.Statement.Selects)
}

func TestGetFieldFinalRelation(t *testing.T) {
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
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
