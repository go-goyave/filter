package filter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
	"gorm.io/gorm/utils/tests"
	"goyave.dev/goyave/v3"
)

func TestSQLEscape(t *testing.T) {
	tx := &gorm.DB{Config: &gorm.Config{
		Dialector: tests.DummyDialector{},
	}}
	assert.Equal(t, "`name`", SQLEscape(tx, "name"))
}

func TestGetTableName(t *testing.T) {
	type testModel struct {
		Name string
		ID   uint
	}

	tx := &gorm.DB{
		Config:    &gorm.Config{Dialector: tests.DummyDialector{}},
		Statement: &gorm.Statement{},
	}
	tx.Statement.DB = tx

	assert.Empty(t, getTableName(tx))

	tx = tx.Table("users")

	assert.Equal(t, "users", getTableName(tx))

	tx, _ = gorm.Open(tests.DummyDialector{}, nil)
	tx = tx.Model(&testModel{})

	assert.Equal(t, "test_models", getTableName(tx))

	tx, _ = gorm.Open(tests.DummyDialector{}, nil)
	tx = tx.Model(1)
	getTableName(tx)
	assert.Equal(t, "unsupported data type: 1", tx.Error.Error())
}

func TestFilterWhere(t *testing.T) {
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
	filter := &Filter{Field: "name", Args: []string{"val1"}}
	db = filter.Where(db, "name = ?", "val1")
	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.Expr{SQL: "name = ?", Vars: []interface{}{"val1"}},
				},
			},
		},
	}
	assert.Equal(t, expected, db.Statement.Clauses)
}

func TestFilterWhereOr(t *testing.T) {
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
	filter := &Filter{Field: "name", Args: []string{"val1"}, Or: true}
	db = filter.Where(db, "name = ?", "val1")
	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.OrConditions{
						Exprs: []clause.Expression{
							clause.Expr{SQL: "name = ?", Vars: []interface{}{"val1"}},
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
	db = db.Scopes(selectScope(nil)).Find(nil)
	assert.Empty(t, db.Statement.Selects)

	db, _ = gorm.Open(&tests.DummyDialector{}, nil)
	db = db.Scopes(selectScope([]string{"a", "b"})).Find(nil)
	assert.Equal(t, []string{"`a`", "`b`"}, db.Statement.Selects)

	db, _ = gorm.Open(&tests.DummyDialector{}, nil)
	db = db.Scopes(selectScope([]string{})).Find(nil)
	assert.Equal(t, []string{"1"}, db.Statement.Selects)
}

func TestFilterScope(t *testing.T) {
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
	filter := &Filter{Field: "notacolumn", Args: []string{"val1"}, Operator: Operators["$eq"]}
	modelIdentity := &modelIdentity{
		Columns: map[string]*column{
			"name": {Name: "Name"},
		},
	}

	assert.Nil(t, filter.Scope(modelIdentity))

	filter.Field = "name"

	db = db.Scopes(filter.Scope(modelIdentity)).Find(nil)
	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.Expr{SQL: "`name`= ?", Vars: []interface{}{"val1"}},
				},
			},
		},
	}
	assert.Equal(t, expected, db.Statement.Clauses)
}

func TestSortScope(t *testing.T) {
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
	sort := &Sort{Field: "notacolumn", Order: SortAscending}
	modelIdentity := &modelIdentity{
		Columns: map[string]*column{
			"name": {Name: "Name"},
		},
	}

	assert.Nil(t, sort.Scope(modelIdentity))

	sort.Field = "name"

	db = db.Scopes(sort.Scope(modelIdentity)).Table("table").Find(nil)
	expected := map[string]clause.Clause{
		"ORDER BY": {
			Name: "ORDER BY",
			Expression: clause.OrderBy{
				Columns: []clause.OrderByColumn{
					{
						Column: clause.Column{
							Table: "table",
							Name:  "name",
						},
					},
				},
			},
		},
	}
	assert.Equal(t, expected, db.Statement.Clauses)

	sort.Order = SortDescending
	db, _ = gorm.Open(&tests.DummyDialector{}, nil)
	db = db.Scopes(sort.Scope(modelIdentity)).Table("table").Find(nil)
	expected["ORDER BY"].Expression.(clause.OrderBy).Columns[0].Desc = true
	assert.Equal(t, expected, db.Statement.Clauses)
}

func TestJoinScope(t *testing.T) {
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
	join := &Join{Relation: "notarelation", Fields: []string{"a", "b", "notacolumn"}}
	modelIdentity := &modelIdentity{
		Columns: map[string]*column{
			"id":          {Name: "ID", Tags: &gormTags{PrimaryKey: true}},
			"name":        {Name: "Name"},
			"relation_id": {Name: "RelID"},
		},
		Relations: map[string]*relation{
			"Relation": {
				modelIdentity: &modelIdentity{
					Columns: map[string]*column{
						"a": {Name: "A", Tags: &gormTags{PrimaryKey: true}},
						"b": {Name: "B"},
					},
					Relations:   map[string]*relation{},
					PrimaryKeys: []string{"a"},
				},
				Type: schema.HasOne,
				Tags: &gormTags{},
			},
		},
	}
	assert.Nil(t, join.Scope(modelIdentity))
	join.Relation = "Relation"

	results := map[string]interface{}{}
	db = db.Scopes(join.Scope(modelIdentity)).Table("table").Find(&results)
	if assert.Contains(t, db.Statement.Preloads, "Relation") {
		tx := db.Scopes(db.Statement.Preloads["Relation"][0].(func(*gorm.DB) *gorm.DB)).Find(nil)
		assert.Equal(t, []string{"`table`.`a`", "`table`.`b`"}, tx.Statement.Selects)
	}
}

func TestJoinScopeNoPrimaryKey(t *testing.T) {
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
	join := &Join{Relation: "Relation", Fields: []string{"a", "b", "notacolumn"}}
	modelIdentity := &modelIdentity{
		Columns: map[string]*column{
			"id":          {Name: "ID", Tags: &gormTags{PrimaryKey: true}},
			"name":        {Name: "Name"},
			"relation_id": {Name: "RelID"},
		},
		Relations: map[string]*relation{
			"Relation": {
				modelIdentity: &modelIdentity{
					Columns: map[string]*column{
						"a": {Name: "A", Tags: &gormTags{}},
						"b": {Name: "B"},
					},
					Relations:   map[string]*relation{},
					PrimaryKeys: []string{},
				},
				Type: schema.HasOne,
				Tags: &gormTags{},
			},
		},
	}
	results := map[string]interface{}{}
	db = db.Scopes(join.Scope(modelIdentity)).Table("table").Find(&results)
	assert.Empty(t, db.Statement.Preloads)
	assert.Empty(t, db.Statement.Selects)
}

func TestJoinScopePrimaryKeyNotSelected(t *testing.T) {
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
	join := &Join{Relation: "Relation", Fields: []string{"b"}}
	modelIdentity := &modelIdentity{
		Columns: map[string]*column{
			"id":          {Name: "ID", Tags: &gormTags{PrimaryKey: true}},
			"name":        {Name: "Name"},
			"relation_id": {Name: "RelID"},
		},
		Relations: map[string]*relation{
			"Relation": {
				modelIdentity: &modelIdentity{
					Columns: map[string]*column{
						"a": {Name: "A", Tags: &gormTags{PrimaryKey: true}},
						"b": {Name: "B"},
					},
					Relations:   map[string]*relation{},
					PrimaryKeys: []string{"a"},
				},
				Type: schema.HasOne,
				Tags: &gormTags{},
			},
		},
	}
	results := map[string]interface{}{}
	db = db.Scopes(join.Scope(modelIdentity)).Table("table").Find(&results)
	if assert.Contains(t, db.Statement.Preloads, "Relation") {
		tx := db.Scopes(db.Statement.Preloads["Relation"][0].(func(*gorm.DB) *gorm.DB)).Find(nil)
		assert.Equal(t, []string{"`table`.`b`", "`table`.`a`"}, tx.Statement.Selects)
	}
}

func TestJoinScopeHasMany(t *testing.T) {
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
	join := &Join{Relation: "Relation", Fields: []string{"a", "b"}}
	modelIdentity := &modelIdentity{
		Columns: map[string]*column{
			"id":          {Name: "ID", Tags: &gormTags{PrimaryKey: true}},
			"name":        {Name: "Name"},
			"relation_id": {Name: "RelID"},
		},
		Relations: map[string]*relation{
			"Relation": {
				modelIdentity: &modelIdentity{
					Columns: map[string]*column{
						"a":         {Name: "A", Tags: &gormTags{PrimaryKey: true}},
						"b":         {Name: "B"},
						"parent_id": {Name: "ParentID"},
					},
					Relations:   map[string]*relation{},
					PrimaryKeys: []string{"a"},
				},
				Type:        schema.HasMany,
				Tags:        &gormTags{},
				ForeignKeys: []string{"parent_id"},
			},
		},
	}

	results := map[string]interface{}{}
	db = db.Scopes(join.Scope(modelIdentity)).Table("table").Find(&results)
	if assert.Contains(t, db.Statement.Preloads, "Relation") {
		tx := db.Scopes(db.Statement.Preloads["Relation"][0].(func(*gorm.DB) *gorm.DB)).Find(nil)
		assert.Equal(t, []string{"`table`.`a`", "`table`.`b`", "`table`.`parent_id`"}, tx.Statement.Selects)
	}
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
	}

	db = applyFilters(db, request, modelIdentity).Find(nil)
	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.Expr{SQL: "`name`LIKE ?", Vars: []interface{}{"%val1%"}},
					clause.Expr{SQL: "`name`LIKE ?", Vars: []interface{}{"%val2%"}},
					clause.OrConditions{
						Exprs: []clause.Expression{
							clause.Expr{SQL: "`name`= ?", Vars: []interface{}{"val3"}},
						},
					},
				},
			},
		},
	}
	assert.Equal(t, expected, db.Statement.Clauses)
}
