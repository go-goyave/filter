package filter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
	"gorm.io/gorm/utils/tests"
	"goyave.dev/goyave/v3"
	"goyave.dev/goyave/v3/database"
)

func TestSQLEscape(t *testing.T) {
	tx := &gorm.DB{Config: &gorm.Config{
		Dialector: tests.DummyDialector{},
	}}
	assert.Equal(t, "`name`", SQLEscape(tx, "  name "))
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
	db = db.Scopes(selectScope(nil, nil)).Find(nil)
	assert.Empty(t, db.Statement.Selects)

	db, _ = gorm.Open(&tests.DummyDialector{}, nil)
	db = db.Scopes(selectScope(nil, []string{"a", "b"})).Find(nil)
	assert.Equal(t, []string{"`a`", "`b`"}, db.Statement.Selects)

	db, _ = gorm.Open(&tests.DummyDialector{}, nil)
	db = db.Scopes(selectScope(nil, []string{})).Find(nil)
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

	assert.Nil(t, filter.Scope(&Settings{}, modelIdentity))

	filter.Field = "name"

	db = db.Scopes(filter.Scope(&Settings{}, modelIdentity)).Find(nil)
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

func TestFilterScopeBlacklisted(t *testing.T) {
	filter := &Filter{Field: "name", Args: []string{"val1"}, Operator: Operators["$eq"]}
	modelIdentity := &modelIdentity{
		Columns: map[string]*column{
			"name": {Name: "Name"},
		},
	}
	assert.Nil(t, filter.Scope(&Settings{Blacklist: Blacklist{FieldsBlacklist: []string{"name"}}}, modelIdentity))
}

func TestSortScope(t *testing.T) {
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
	sort := &Sort{Field: "notacolumn", Order: SortAscending}
	modelIdentity := &modelIdentity{
		Columns: map[string]*column{
			"name": {Name: "Name"},
		},
	}

	assert.Nil(t, sort.Scope(&Settings{}, modelIdentity))

	sort.Field = "name"

	db = db.Scopes(sort.Scope(&Settings{}, modelIdentity)).Table("table").Find(nil)
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
	db = db.Scopes(sort.Scope(&Settings{}, modelIdentity)).Table("table").Find(nil)
	expected["ORDER BY"].Expression.(clause.OrderBy).Columns[0].Desc = true
	assert.Equal(t, expected, db.Statement.Clauses)
}

func TestSortScopeBlacklisted(t *testing.T) {
	sort := &Sort{Field: "name", Order: SortAscending}
	modelIdentity := &modelIdentity{
		Columns: map[string]*column{
			"name": {Name: "Name"},
		},
	}
	assert.Nil(t, sort.Scope(&Settings{Blacklist: Blacklist{FieldsBlacklist: []string{"name"}}}, modelIdentity))
}

func TestJoinScope(t *testing.T) {
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
	join := &Join{Relation: "notarelation", Fields: []string{"a", "b", "notacolumn"}}
	join.selectCache = map[string][]string{}
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
	assert.Nil(t, join.Scopes(&Settings{}, modelIdentity))
	join.Relation = "Relation"

	results := map[string]interface{}{}
	db = db.Scopes(join.Scopes(&Settings{}, modelIdentity)...).Table("table").Find(&results)
	if assert.Contains(t, db.Statement.Preloads, "Relation") {
		tx := db.Scopes(db.Statement.Preloads["Relation"][0].(func(*gorm.DB) *gorm.DB)).Find(nil)
		assert.Equal(t, []string{"`table`.`a`", "`table`.`b`"}, tx.Statement.Selects)
	}
	assert.Equal(t, []string{"a", "b", "notacolumn"}, join.selectCache["Relation"])
}

func TestJoinScopeBlacklisted(t *testing.T) {
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
	assert.Nil(t, join.Scopes(&Settings{Blacklist: Blacklist{RelationsBlacklist: []string{"Relation"}}}, modelIdentity))
}

func TestJoinScopeBlacklistedRelationHop(t *testing.T) {
	join := &Join{Relation: "Relation.Parent.Relation", Fields: []string{"name", "id"}}
	join.selectCache = map[string][]string{}
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
	modelIdentity.Relations["Relation"].Relations["Parent"] = &relation{
		modelIdentity: modelIdentity,
		Type:          schema.HasOne,
		Tags:          &gormTags{},
		ForeignKeys:   []string{"relation_id"},
	}

	settings := &Settings{
		Blacklist: Blacklist{
			Relations: map[string]*Blacklist{
				"Relation": {
					RelationsBlacklist: []string{"Parent"},
				},
			},
		},
	}

	assert.Nil(t, join.Scopes(settings, modelIdentity))
}

func TestJoinScopeNoPrimaryKey(t *testing.T) {
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
	join := &Join{Relation: "Relation", Fields: []string{"a", "b", "notacolumn"}}
	join.selectCache = map[string][]string{}
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
	db = db.Scopes(join.Scopes(&Settings{}, modelIdentity)...).Table("table").Find(&results)
	assert.Empty(t, db.Statement.Preloads)
	assert.Empty(t, db.Statement.Selects)
	assert.Equal(t, "Could not find \"Relation\" relation's primary key. Add `gorm:\"primaryKey\"` to your model", db.Error.Error())
	assert.Equal(t, []string{"a", "b", "notacolumn"}, join.selectCache["Relation"])
}

func TestJoinScopePrimaryKeyNotSelected(t *testing.T) {
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
	join := &Join{Relation: "Relation", Fields: []string{"b"}}
	join.selectCache = map[string][]string{}
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
	db = db.Scopes(join.Scopes(&Settings{}, modelIdentity)...).Table("table").Find(&results)
	if assert.Contains(t, db.Statement.Preloads, "Relation") {
		tx := db.Scopes(db.Statement.Preloads["Relation"][0].(func(*gorm.DB) *gorm.DB)).Find(nil)
		assert.Equal(t, []string{"`table`.`b`", "`table`.`a`"}, tx.Statement.Selects)
	}
	assert.Equal(t, []string{"b"}, join.selectCache["Relation"])
}

func TestJoinScopeHasMany(t *testing.T) {
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
	join := &Join{Relation: "Relation", Fields: []string{"a", "b"}}
	join.selectCache = map[string][]string{}
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
	db = db.Scopes(join.Scopes(&Settings{}, modelIdentity)...).Table("table").Find(&results)
	if assert.Contains(t, db.Statement.Preloads, "Relation") {
		tx := db.Scopes(db.Statement.Preloads["Relation"][0].(func(*gorm.DB) *gorm.DB)).Find(nil)
		assert.Equal(t, []string{"`table`.`a`", "`table`.`b`", "`table`.`parent_id`"}, tx.Statement.Selects)
	}
	assert.Equal(t, []string{"a", "b"}, join.selectCache["Relation"])
}

func TestJoinScopeNestedRelations(t *testing.T) {
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
	join := &Join{Relation: "Relation.Parent", Fields: []string{"id", "relation_id"}}
	join.selectCache = map[string][]string{}
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
		PrimaryKeys: []string{"id"},
	}
	modelIdentity.Relations["Relation"].Relations["Parent"] = &relation{
		modelIdentity: modelIdentity,
		Type:          schema.HasOne,
		Tags:          &gormTags{},
		ForeignKeys:   []string{"relation_id"},
	}
	settings := &Settings{
		Blacklist: Blacklist{
			FieldsBlacklist: []string{"name"},
			Relations: map[string]*Blacklist{
				"Relation": {
					FieldsBlacklist: []string{"b"},
					Relations: map[string]*Blacklist{
						"Parent": {
							FieldsBlacklist: []string{"relation_id"},
							IsFinal:         true,
						},
					},
				},
			},
		},
	}

	results := map[string]interface{}{}
	db = db.Scopes(join.Scopes(settings, modelIdentity)...).Table("table").Find(&results)
	if assert.Contains(t, db.Statement.Preloads, "Relation.Parent") {
		tx := db.Session(&gorm.Session{}).Scopes(db.Statement.Preloads["Relation.Parent"][0].(func(*gorm.DB) *gorm.DB)).Find(nil)
		assert.Equal(t, []string{"`table`.`id`"}, tx.Statement.Selects)
	}
	if assert.Contains(t, db.Statement.Preloads, "Relation") {
		tx := db.Session(&gorm.Session{}).Scopes(db.Statement.Preloads["Relation"][0].(func(*gorm.DB) *gorm.DB)).Find(nil)
		assert.Equal(t, []string{"`table`.`a`", "`table`.`parent_id`"}, tx.Statement.Selects)
	}
	assert.NotContains(t, join.selectCache, "Relation")
	assert.Equal(t, []string{"id", "relation_id"}, join.selectCache["Relation.Parent"])
}

func TestJoinScopeFinal(t *testing.T) {
	join := &Join{Relation: "Relation", Fields: []string{"a", "b"}}
	join.selectCache = map[string][]string{}
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
	settings := &Settings{Blacklist: Blacklist{IsFinal: true}}

	assert.Nil(t, join.Scopes(settings, modelIdentity))
}

func TestJoinNestedRelationsWithSelect(t *testing.T) {
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
	join := &Join{Relation: "Relation", Fields: []string{"b"}}
	join.selectCache = map[string][]string{}
	join2 := &Join{Relation: "Relation.Parent", Fields: []string{"id", "relation_id"}}
	join2.selectCache = join.selectCache
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
		PrimaryKeys: []string{"id"},
	}
	modelIdentity.Relations["Relation"].Relations["Parent"] = &relation{
		modelIdentity: modelIdentity,
		Type:          schema.HasOne,
		Tags:          &gormTags{},
		ForeignKeys:   []string{"relation_id"},
	}
	settings := &Settings{
		Blacklist: Blacklist{
			FieldsBlacklist: []string{"name"},
			Relations: map[string]*Blacklist{
				"Relation": {
					Relations: map[string]*Blacklist{
						"Parent": {
							FieldsBlacklist: []string{"relation_id"},
							IsFinal:         true,
						},
					},
				},
			},
		},
	}

	results := map[string]interface{}{}
	db = db.Scopes(join.Scopes(settings, modelIdentity)...).Scopes(join2.Scopes(settings, modelIdentity)...).Table("table").Find(&results)
	if assert.Contains(t, db.Statement.Preloads, "Relation.Parent") {
		tx := db.Session(&gorm.Session{}).Scopes(db.Statement.Preloads["Relation.Parent"][0].(func(*gorm.DB) *gorm.DB)).Find(nil)
		assert.Equal(t, []string{"`table`.`id`"}, tx.Statement.Selects)
	}
	if assert.Contains(t, db.Statement.Preloads, "Relation") {
		tx := db.Session(&gorm.Session{}).Scopes(db.Statement.Preloads["Relation"][0].(func(*gorm.DB) *gorm.DB)).Find(nil)
		assert.Equal(t, []string{"`table`.`b`", "`table`.`a`", "`table`.`parent_id`"}, tx.Statement.Selects)
	}
	assert.Equal(t, []string{"b"}, join.selectCache["Relation"])
	assert.Equal(t, []string{"id", "relation_id"}, join.selectCache["Relation.Parent"])
}

func TestJoinScopeInvalidSyntax(t *testing.T) {
	join := &Join{Relation: "Relation.", Fields: []string{"a", "b"}} // A dot at the end of the relation name is invalid
	join.selectCache = map[string][]string{}
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
	assert.Nil(t, join.Scopes(&Settings{}, modelIdentity))
}

func TestJoinScopeNonExistingRelation(t *testing.T) {
	join := &Join{Relation: "Relation.NotARelation.Parent", Fields: []string{"a", "b"}}
	join.selectCache = map[string][]string{}
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
	assert.Nil(t, join.Scopes(&Settings{}, modelIdentity))
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

	db = (&Settings{}).applyFilters(db, request, modelIdentity).Find(nil)
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
	assert.Equal(t, []string{"`test_scope_models`.`id`", "`test_scope_models`.`name`"}, db.Statement.Selects)
}

func TestScopeDisableFields(t *testing.T) {
	paginator, db := prepareTestScope(&Settings{DisableFields: true})
	assert.NotNil(t, paginator)

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
	assert.Empty(t, db.Statement.Selects)
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
	assert.Equal(t, []string{"`test_scope_models`.`id`", "`test_scope_models`.`name`"}, db.Statement.Selects)
}

func TestScopeDisableSort(t *testing.T) {
	paginator, db := prepareTestScope(&Settings{DisableSort: true})
	assert.NotNil(t, paginator)

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
		"LIMIT": {
			Expression: clause.Limit{
				Limit:  15,
				Offset: 15,
			},
		},
	}
	assert.Equal(t, expected, db.Statement.Clauses)
	assert.Equal(t, []string{"`test_scope_models`.`id`", "`test_scope_models`.`name`"}, db.Statement.Selects)
}

func TestScopeDisableJoin(t *testing.T) {
	paginator, db := prepareTestScope(&Settings{DisableJoin: true})
	assert.NotNil(t, paginator)

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
	assert.ElementsMatch(t, []string{"`id`", "`relation_id`"}, db.Statement.Selects)
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
