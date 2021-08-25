package filter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/utils/tests"
	"goyave.dev/goyave/v3"
	"goyave.dev/goyave/v3/database"
)

type TestScopeRelation struct {
	A  string
	B  string
	ID uint
}
type TestScopeModel struct {
	Relation   *TestScopeRelation
	Name       string
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
			"fields":   "id,name",
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
	paginator, db := prepareTestScope(nil)
	assert.NotNil(t, paginator)

	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.Expr{SQL: "`test_scope_models`.`name` LIKE ?", Vars: []interface{}{"%val1%"}},
					clause.Expr{SQL: "`test_scope_models`.`name` LIKE ?", Vars: []interface{}{"%val2%"}},
					clause.OrConditions{
						Exprs: []clause.Expression{
							clause.Expr{SQL: "`test_scope_models`.`name` = ?", Vars: []interface{}{"val3"}},
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
	assert.Equal(t, []string{"`test_scope_models`.`id`", "`test_scope_models`.`name`", "`test_scope_models`.`relation_id`"}, db.Statement.Selects)
}

func TestScopeDisableFields(t *testing.T) {
	paginator, db := prepareTestScope(&Settings{DisableFields: true})
	assert.NotNil(t, paginator)

	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.Expr{SQL: "`test_scope_models`.`name` LIKE ?", Vars: []interface{}{"%val1%"}},
					clause.Expr{SQL: "`test_scope_models`.`name` LIKE ?", Vars: []interface{}{"%val2%"}},
					clause.OrConditions{
						Exprs: []clause.Expression{
							clause.Expr{SQL: "`test_scope_models`.`name` = ?", Vars: []interface{}{"val3"}},
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
	assert.ElementsMatch(t, []string{"`test_scope_models`.`id`", "`test_scope_models`.`relation_id`", "`test_scope_models`.`name`"}, db.Statement.Selects)
}

func TestScopeDisableFilter(t *testing.T) {
	paginator, db := prepareTestScope(&Settings{DisableFilter: true})
	assert.NotNil(t, paginator)

	expected := map[string]clause.Clause{
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
	assert.Equal(t, []string{"`test_scope_models`.`id`", "`test_scope_models`.`name`", "`test_scope_models`.`relation_id`"}, db.Statement.Selects)
}

func TestScopeDisableSort(t *testing.T) {
	paginator, db := prepareTestScope(&Settings{DisableSort: true})
	assert.NotNil(t, paginator)

	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.Expr{SQL: "`test_scope_models`.`name` LIKE ?", Vars: []interface{}{"%val1%"}},
					clause.Expr{SQL: "`test_scope_models`.`name` LIKE ?", Vars: []interface{}{"%val2%"}},
					clause.OrConditions{
						Exprs: []clause.Expression{
							clause.Expr{SQL: "`test_scope_models`.`name` = ?", Vars: []interface{}{"val3"}},
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
	assert.Equal(t, []string{"`test_scope_models`.`id`", "`test_scope_models`.`name`", "`test_scope_models`.`relation_id`"}, db.Statement.Selects)
}

func TestScopeDisableJoin(t *testing.T) {
	paginator, db := prepareTestScope(&Settings{DisableJoin: true})
	assert.NotNil(t, paginator)

	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.Expr{SQL: "`test_scope_models`.`name` LIKE ?", Vars: []interface{}{"%val1%"}},
					clause.Expr{SQL: "`test_scope_models`.`name` LIKE ?", Vars: []interface{}{"%val2%"}},
					clause.OrConditions{
						Exprs: []clause.Expression{
							clause.Expr{SQL: "`test_scope_models`.`name` = ?", Vars: []interface{}{"val3"}},
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
	assert.Equal(t, []string{"`test_scope_models`.`id`", "`test_scope_models`.`name`"}, db.Statement.Selects)
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
	assert.ElementsMatch(t, []string{"`test_scope_models`.`id`", "`test_scope_models`.`relation_id`"}, db.Statement.Selects)
}

func TestBlacklistGetSelectableFields(t *testing.T) {
	blacklist := &Blacklist{
		FieldsBlacklist: []string{"name"},
	}
	fields := map[string]*column{
		"id":    {},
		"name":  {},
		"email": {},
	}

	assert.ElementsMatch(t, []string{"id", "email"}, blacklist.getSelectableFields(fields))
}

func TestApplyFilters(t *testing.T) {
	request := &goyave.Request{
		Data: map[string]interface{}{
			"filter": []*Filter{
				{Field: "name", Args: []string{"val1"}, Operator: Operators["$cont"]},
				{Field: "name", Args: []string{"val2"}, Operator: Operators["$cont"]},
			},
			"or": []*Filter{
				{Field: "name", Args: []string{"val3"}, Or: true, Operator: Operators["$eq"]},
			},
		},
		Lang: "en-US",
	}
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
	modelIdentity := &modelIdentity{
		Columns: map[string]*column{
			"id":   {Name: "ID", Tags: &gormTags{PrimaryKey: true}},
			"name": {Name: "Name"},
		},
		PrimaryKeys: []string{"id"},
		Relations:   map[string]*relation{},
		TableName:   `test_scope_models`,
	}

	db = (&Settings{}).applyFilters(db, request, modelIdentity).Find(nil)
	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.Expr{SQL: "`test_scope_models`.`name` LIKE ?", Vars: []interface{}{"%val1%"}},
					clause.Expr{SQL: "`test_scope_models`.`name` LIKE ?", Vars: []interface{}{"%val2%"}},
					clause.OrConditions{
						Exprs: []clause.Expression{
							clause.Expr{SQL: "`test_scope_models`.`name` = ?", Vars: []interface{}{"val3"}},
						},
					},
				},
			},
		},
	}
	assert.Equal(t, expected, db.Statement.Clauses)
}

func TestSelectScope(t *testing.T) {
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
	db = db.Scopes(selectScope(nil, nil)).Find(nil)
	assert.Empty(t, db.Statement.Selects)

	modelIdentity := &modelIdentity{TableName: "test_models"}
	db, _ = gorm.Open(&tests.DummyDialector{}, nil)
	db = db.Scopes(selectScope(modelIdentity, []string{"a", "b"})).Find(nil)
	assert.Equal(t, []string{"`test_models`.`a`", "`test_models`.`b`"}, db.Statement.Selects)

	db, _ = gorm.Open(&tests.DummyDialector{}, nil)
	db = db.Scopes(selectScope(modelIdentity, []string{})).Find(nil)
	assert.Equal(t, []string{"1"}, db.Statement.Selects)
}
