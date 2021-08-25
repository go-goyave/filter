package filter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"gorm.io/gorm/utils/tests"
)

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
					TableName:   "relation",
				},
				Type: schema.HasOne,
				Tags: &gormTags{},
			},
		},
		TableName: "table",
	}
	assert.Nil(t, join.Scopes(&Settings{}, modelIdentity))
	join.Relation = "Relation"

	results := map[string]interface{}{}
	db = db.Scopes(join.Scopes(&Settings{}, modelIdentity)...).Table("table").Find(&results)
	if assert.Contains(t, db.Statement.Preloads, "Relation") {
		tx := db.Scopes(db.Statement.Preloads["Relation"][0].(func(*gorm.DB) *gorm.DB)).Find(nil)
		assert.Equal(t, []string{"`relation`.`a`", "`relation`.`b`"}, tx.Statement.Selects)
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
					TableName:   "relation",
				},
				Type: schema.HasOne,
				Tags: &gormTags{},
			},
		},
		TableName: "table",
	}
	results := map[string]interface{}{}
	db = db.Scopes(join.Scopes(&Settings{}, modelIdentity)...).Table("table").Find(&results)
	if assert.Contains(t, db.Statement.Preloads, "Relation") {
		tx := db.Scopes(db.Statement.Preloads["Relation"][0].(func(*gorm.DB) *gorm.DB)).Find(nil)
		assert.Equal(t, []string{"`relation`.`b`", "`relation`.`a`"}, tx.Statement.Selects)
	}
	assert.Equal(t, []string{"b"}, join.selectCache["Relation"])

	// Don't select it if it's blacklisted
	settings := &Settings{
		Blacklist: Blacklist{
			Relations: map[string]*Blacklist{
				"Relation": {
					FieldsBlacklist: []string{"a"},
				},
			},
		},
	}
	db = db.Scopes(join.Scopes(settings, modelIdentity)...).Table("table").Find(&results)
	if assert.Contains(t, db.Statement.Preloads, "Relation") {
		tx := db.Scopes(db.Statement.Preloads["Relation"][0].(func(*gorm.DB) *gorm.DB)).Find(nil)
		assert.Equal(t, []string{"`relation`.`b`"}, tx.Statement.Selects)
	}
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
					TableName:   "relation",
				},
				Type:        schema.HasMany,
				Tags:        &gormTags{},
				ForeignKeys: []string{"parent_id"},
			},
		},
		TableName: "table",
	}

	results := map[string]interface{}{}
	db = db.Scopes(join.Scopes(&Settings{}, modelIdentity)...).Table("table").Find(&results)
	if assert.Contains(t, db.Statement.Preloads, "Relation") {
		tx := db.Scopes(db.Statement.Preloads["Relation"][0].(func(*gorm.DB) *gorm.DB)).Find(nil)
		assert.Equal(t, []string{"`relation`.`a`", "`relation`.`b`", "`relation`.`parent_id`"}, tx.Statement.Selects)
	}
	assert.Equal(t, []string{"a", "b"}, join.selectCache["Relation"])

	// Don't select parent_id if blacklisted
	settings := &Settings{
		Blacklist: Blacklist{
			Relations: map[string]*Blacklist{
				"Relation": {
					FieldsBlacklist: []string{"parent_id"},
				},
			},
		},
	}
	db = db.Scopes(join.Scopes(settings, modelIdentity)...).Table("table").Find(&results)
	if assert.Contains(t, db.Statement.Preloads, "Relation") {
		tx := db.Scopes(db.Statement.Preloads["Relation"][0].(func(*gorm.DB) *gorm.DB)).Find(nil)
		assert.Equal(t, []string{"`relation`.`a`", "`relation`.`b`"}, tx.Statement.Selects)
	}
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
					TableName:   "relation",
				},
				Type:        schema.HasMany,
				Tags:        &gormTags{},
				ForeignKeys: []string{"parent_id"},
			},
		},
		PrimaryKeys: []string{"id"},
		TableName:   "table",
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
		assert.Equal(t, []string{"`relation`.`a`", "`relation`.`parent_id`"}, tx.Statement.Selects)
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
					TableName:   "relation",
				},
				Type:        schema.HasMany,
				Tags:        &gormTags{},
				ForeignKeys: []string{"parent_id"},
			},
		},
		PrimaryKeys: []string{"id"},
		TableName:   "table",
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
		assert.Equal(t, []string{"`relation`.`b`", "`relation`.`a`", "`relation`.`parent_id`"}, tx.Statement.Selects)
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
