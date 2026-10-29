package filter

import (
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type JoinTestModel struct {
	Relation *JoinRelationModel
	Name     string
	ID       int `gorm:"primaryKey"`
	RelID    int `gorm:"column:relation_id"`
}

func (m *JoinTestModel) TableName() string {
	return "table"
}

type JoinRelationModel struct {
	C string `gorm:"->;-:migration" computed:"UPPER(~~~ct~~~.b)"`
	B string
	A int `gorm:"primaryKey"`
}

func (m *JoinRelationModel) TableName() string {
	return "relation"
}

func TestJoinScope(t *testing.T) {
	db, mock := openMockDB(t)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "table"`)).
		WithoutArgs().
		WillReturnRows(sqlmock.NewRows([]string{"name", "id", "relation_id"}).AddRow("name", 123, 456))

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT "relation"."b","relation"."a" FROM "table"`)).
		WithoutArgs().
		WillReturnRows(sqlmock.NewRows([]string{"relation.a", "relation.b"}).AddRow("a", "b"))

	join := &Join{Relation: "notarelation", Fields: []string{"a", "b", "notacolumn"}}
	join.selectCache = map[string][]string{}

	schema, err := parseModel(db, &JoinTestModel{})
	require.NoError(t, err)

	whitelist := Whitelist{
		Relations: map[string]Whitelist{
			"Relation": {Fields: AllowAllFields},
		},
		Fields: AllowAllFields,
	}

	assert.Nil(t, join.Scopes(whitelist, schema))
	join.Relation = "Relation"

	db = db.Model(&JoinTestModel{}).Scopes(join.Scopes(whitelist, schema)...).Find(&map[string]any{})
	assert.NoError(t, db.Error)
	if assert.Contains(t, db.Statement.Preloads, "Relation") {
		tx := db.Scopes(db.Statement.Preloads["Relation"][0].(func(*gorm.DB) *gorm.DB)).Find(&map[string]any{})
		assert.NoError(t, tx.Error)
		assert.Equal(t, []string{`"relation"."b"`, `"relation"."a"`}, tx.Statement.Selects)
	}
	assert.Equal(t, []string{"a", "b", "notacolumn"}, join.selectCache["Relation"])
}

func TestJoinScopeAutoSelectFieldsAllFieldsWhitelisted(t *testing.T) {
	db, mock := openMockDB(t)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "table"`)).
		WithoutArgs().
		WillReturnRows(sqlmock.NewRows([]string{"name", "id", "relation_id"}).AddRow("name", 123, 456))

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT (UPPER("relation".b)) "c","relation"."b","relation"."a" FROM "table"`)).
		WithoutArgs().
		WillReturnRows(sqlmock.NewRows([]string{"c", "relation.a", "relation.b"}).AddRow("B", "a", "b"))

	join := &Join{Relation: "Relation", Fields: nil}
	join.selectCache = map[string][]string{}
	schema, err := parseModel(db, &JoinTestModel{})
	require.NoError(t, err)

	whitelist := Whitelist{
		Relations: map[string]Whitelist{
			"Relation": {Fields: AllowAllFields},
		},
		Fields: AllowAllFields,
	}

	db = db.Model(&JoinTestModel{}).Scopes(join.Scopes(whitelist, schema)...).Find(&map[string]any{})
	assert.NoError(t, db.Error)
	if assert.Contains(t, db.Statement.Preloads, "Relation") {
		tx := db.Scopes(db.Statement.Preloads["Relation"][0].(func(*gorm.DB) *gorm.DB)).Find(&map[string]any{})
		assert.NoError(t, tx.Error)
		assert.Equal(t, []string{`(UPPER("relation".b)) "c"`, `"relation"."b"`, `"relation"."a"`}, tx.Statement.Selects)
	}
	assert.Nil(t, join.selectCache["Relation"])
}

func TestJoinScopeAutoSelectFieldsWithBlacklist(t *testing.T) {
	db, mock := openMockDB(t)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "table"`)).
		WithoutArgs().
		WillReturnRows(sqlmock.NewRows([]string{"name", "id", "relation_id"}).AddRow("name", 123, 456))

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT "relation"."a",(UPPER("relation".b)) "c" FROM "table"`)).
		WithoutArgs().
		WillReturnRows(sqlmock.NewRows([]string{"c", "relation.b"}).AddRow("B", "b"))

	join := &Join{Relation: "Relation", Fields: nil}
	join.selectCache = map[string][]string{}
	schema, err := parseModel(db, &JoinTestModel{})
	require.NoError(t, err)

	whitelist := Whitelist{
		Relations: map[string]Whitelist{
			"Relation": {Fields: []string{"A", "C"}},
		},
		Fields: AllowAllFields,
	}

	db = db.Model(&JoinTestModel{}).Scopes(join.Scopes(whitelist, schema)...).Find(&map[string]any{})
	assert.NoError(t, db.Error)
	if assert.Contains(t, db.Statement.Preloads, "Relation") {
		tx := db.Scopes(db.Statement.Preloads["Relation"][0].(func(*gorm.DB) *gorm.DB)).Find(&map[string]any{})
		assert.NoError(t, tx.Error)
		assert.Equal(t, []string{`"relation"."a"`, `(UPPER("relation".b)) "c"`}, tx.Statement.Selects)
	}
	assert.Nil(t, join.selectCache["Relation"])
}

func TestJoinScopeAnonymousRelation(t *testing.T) {
	db, _ := openMockDB(t)
	join := &Join{Relation: "notarelation", Fields: []string{"a", "b", "notacolumn"}}
	join.selectCache = map[string][]string{}

	type JoinTestModel struct {
		Relation *struct {
			B string
			A int `gorm:"primaryKey"`
		}
		Name  string
		ID    int `gorm:"primaryKey"`
		RelID int `gorm:"column:relation_id"`
	}

	schema, err := parseModel(db, &JoinTestModel{})
	require.NoError(t, err)

	whitelist := Whitelist{
		Relations: map[string]Whitelist{
			"Relation": {Fields: AllowAllFields},
		},
		Fields: AllowAllFields,
	}

	assert.Nil(t, join.Scopes(whitelist, schema))
	join.Relation = "Relation"

	db = db.Model(&JoinTestModel{}).Scopes(join.Scopes(whitelist, schema)...).Find(&map[string]any{})
	assert.Empty(t, db.Statement.Preloads)
	assert.Empty(t, db.Statement.Selects)
	assert.Equal(t, "relation \"Relation\" is anonymous, could not get table name", db.Error.Error())
	assert.Equal(t, []string{"a", "b", "notacolumn"}, join.selectCache["Relation"])
}

func TestJoinScopeBlacklisted(t *testing.T) {
	db, _ := openMockDB(t)
	join := &Join{Relation: "Relation", Fields: []string{"a", "b", "notacolumn"}}

	schema, err := parseModel(db, &JoinTestModel{})
	require.NoError(t, err)

	whitelist := Whitelist{
		// No relation allowed
		Fields: AllowAllFields,
	}

	assert.Nil(t, join.Scopes(whitelist, schema))
}

type JoinHopTestModel struct {
	Relation *JoinHopTestChildModel
	Name     string
	ID       int `gorm:"primaryKey"`
	RelID    int `gorm:"column:relation_id"`
}

type JoinHopTestChildModel struct {
	Parent   *JoinHopTestModel
	B        string
	A        int `gorm:"primaryKey"`
	ParentID int
}

func (m *JoinHopTestChildModel) TableName() string {
	return "relation"
}

type JoinHopManyTestModel struct {
	Name     string
	Relation []*JoinHopManyTestChildModel `gorm:"foreignKey:A"`
	ID       int                          `gorm:"primaryKey"`
}

type JoinHopManyTestChildModel struct {
	Parent   *JoinHopManyTestModel
	B        string
	A        int `gorm:"primaryKey"`
	ParentID int
}

func (m *JoinHopManyTestChildModel) TableName() string {
	return "relation"
}

func TestJoinScopeBlacklistedRelationHop(t *testing.T) {
	db, _ := openMockDB(t)
	join := &Join{Relation: "Relation.Parent.Relation", Fields: []string{"name", "id"}}
	join.selectCache = map[string][]string{}

	schema, err := parseModel(db, &JoinHopManyTestModel{})
	require.NoError(t, err)

	whitelist := Whitelist{
		Relations: map[string]Whitelist{
			"Relation": {Fields: AllowAllFields}, // Parent relation not allowed
		},
		Fields: AllowAllFields,
	}

	assert.Nil(t, join.Scopes(whitelist, schema))
}

func TestJoinScopeBlacklistedRelation(t *testing.T) {
	db, _ := openMockDB(t)
	join := &Join{Relation: "Relation.Parent", Fields: []string{"name", "id"}}
	join.selectCache = map[string][]string{}

	schema, err := parseModel(db, &JoinHopManyTestModel{})
	require.NoError(t, err)

	whitelist := Whitelist{
		Relations: map[string]Whitelist{},
		Fields:    AllowAllFields,
	}

	assert.Nil(t, join.Scopes(whitelist, schema))
}

func TestJoinScopePrimaryKeyNotSelected(t *testing.T) {
	db, mock := openMockDB(t)
	join := &Join{Relation: "Relation", Fields: []string{"b"}}
	join.selectCache = map[string][]string{}
	schema, err := parseModel(db, &JoinHopTestModel{})
	require.NoError(t, err)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "join_hop_test_models"`)).
		WithoutArgs().
		WillReturnRows(sqlmock.NewRows([]string{"name", "id", "relation_id"}).AddRow("name", 123, 456))

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT "relation"."b","relation"."a","relation"."parent_id" FROM "relation" WHERE "relation"."a" = $1`)).
		WithArgs(456).
		WillReturnRows(sqlmock.NewRows([]string{"b", "a", "parent_id"}).AddRow("b", 456, 123))

	whitelist := Whitelist{
		Relations: map[string]Whitelist{
			"Relation": {Fields: AllowAllFields},
		},
		Fields: AllowAllFields,
	}

	results := []*JoinHopTestModel{}
	db = db.Model(&JoinHopTestModel{}).Scopes(join.Scopes(whitelist, schema)...).Find(&results)
	assert.NoError(t, db.Error)
	assert.Equal(t, []string{"b"}, join.selectCache["Relation"])

	require.Len(t, results, 1)
	if assert.Len(t, results, 1) && assert.NotNil(t, results[0].Relation) {
		assert.Equal(t, &JoinHopTestChildModel{B: "b", A: 456, ParentID: 123}, results[0].Relation)
	}

	// Don't select it if it's not whitelisted
	whitelist = Whitelist{
		Relations: map[string]Whitelist{
			"Relation": {Fields: []string{"b", "parent_id"}},
		},
		Fields: AllowAllFields,
	}
	db, mock = openMockDB(t)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "join_hop_test_models"`)).
		WithoutArgs().
		WillReturnRows(sqlmock.NewRows([]string{"name", "id", "relation_id"}).AddRow("name", 123, 456))

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT "relation"."b","relation"."parent_id" FROM "relation" WHERE "relation"."a" = $1`)).
		WithArgs(456).
		WillReturnRows(sqlmock.NewRows([]string{"b", "parent_id"}).AddRow("b", 123))

	db = db.Model(&JoinHopTestModel{}).Scopes(join.Scopes(whitelist, schema)...).Find(&[]*JoinHopTestModel{})
	// The relation primary key is not selected so Gorm cannot assign the association
	assert.ErrorContains(t, db.Error, "failed to assign association")
}

func TestJoinScopeHasMany(t *testing.T) {
	db, mock := openMockDB(t)
	join := &Join{Relation: "Relation", Fields: []string{"a", "b"}}
	join.selectCache = map[string][]string{}
	schema, err := parseModel(db, &JoinHopManyTestModel{})
	require.NoError(t, err)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "join_hop_many_test_models"`)).
		WithoutArgs().
		WillReturnRows(sqlmock.NewRows([]string{"name", "id", "relation_id"}).AddRow("name", 123, 456))

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT "relation"."b","relation"."a","relation"."parent_id" FROM "join_hop_many_test_models"`)).
		WithoutArgs().
		WillReturnRows(sqlmock.NewRows([]string{"relation.a", "relation.b", "relation.parent_id"}).AddRow("a", "b", 123))

	whitelist := Whitelist{
		Relations: map[string]Whitelist{
			"Relation": {Fields: AllowAllFields},
		},
		Fields: AllowAllFields,
	}

	db = db.Model(&JoinHopManyTestModel{}).Scopes(join.Scopes(whitelist, schema)...).Find(&map[string]any{})
	assert.NoError(t, db.Error)
	if assert.Contains(t, db.Statement.Preloads, "Relation") {
		tx := db.Scopes(db.Statement.Preloads["Relation"][0].(func(*gorm.DB) *gorm.DB)).Find(&map[string]any{})
		assert.NoError(t, tx.Error)
		assert.Equal(t, []string{`"relation"."b"`, `"relation"."a"`, `"relation"."parent_id"`}, tx.Statement.Selects)
	}
	assert.Equal(t, []string{"a", "b"}, join.selectCache["Relation"])

	// Don't select parent_id if not whitelisted
	whitelist = Whitelist{
		Relations: map[string]Whitelist{
			"Relation": {Fields: []string{"a", "b"}},
		},
		Fields: AllowAllFields,
	}
	db, mock = openMockDB(t)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "join_hop_many_test_models"`)).
		WithoutArgs().
		WillReturnRows(sqlmock.NewRows([]string{"name", "id", "relation_id"}).AddRow("name", 123, 456))

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT "relation"."a","relation"."b" FROM "join_hop_many_test_models"`)).
		WithoutArgs().
		WillReturnRows(sqlmock.NewRows([]string{"relation.a", "relation.b"}).AddRow("a", "b"))

	db = db.Model(&JoinHopManyTestModel{}).Scopes(join.Scopes(whitelist, schema)...).Find(&map[string]any{})
	assert.NoError(t, db.Error)
	if assert.Contains(t, db.Statement.Preloads, "Relation") {
		tx := db.Scopes(db.Statement.Preloads["Relation"][0].(func(*gorm.DB) *gorm.DB)).Find(&map[string]any{})
		assert.NoError(t, tx.Error)
		assert.Equal(t, []string{`"relation"."a"`, `"relation"."b"`}, tx.Statement.Selects)
	}
}

func TestJoinScopeNestedRelations(t *testing.T) {
	db, mock := openMockDB(t)
	join := &Join{Relation: "Relation.Parent", Fields: []string{"id", "relation_id"}}
	join.selectCache = map[string][]string{}
	schema, err := parseModel(db, &JoinHopManyTestModel{})
	require.NoError(t, err)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "join_hop_many_test_models"`)).
		WithoutArgs().
		WillReturnRows(sqlmock.NewRows([]string{"name", "id", "relation_id"}).AddRow("name", 123, 456))

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT "join_hop_many_test_models"."id" FROM "join_hop_many_test_models"`)).
		WithoutArgs().
		WillReturnRows(sqlmock.NewRows([]string{"join_hop_many_test_models.id"}).AddRow(456))

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT "relation"."a","relation"."parent_id" FROM "join_hop_many_test_models"`)).
		WithoutArgs().
		WillReturnRows(sqlmock.NewRows([]string{"relation.a", "relation.parent_id"}).AddRow("a", 456))

	whitelist := Whitelist{
		Fields: []string{"id"},
		Relations: map[string]Whitelist{
			"Relation": {
				Fields: []string{"a", "parent_id"},
				Relations: map[string]Whitelist{
					"Parent": {
						Fields: []string{"ID"},
					},
				},
			},
		},
	}

	db = db.Model(&JoinHopManyTestModel{}).Scopes(join.Scopes(whitelist, schema)...).Find(&map[string]any{})
	assert.NoError(t, db.Error)
	if assert.Contains(t, db.Statement.Preloads, "Relation.Parent") {
		tx := db.Session(&gorm.Session{}).Scopes(db.Statement.Preloads["Relation.Parent"][0].(func(*gorm.DB) *gorm.DB)).Find(&map[string]any{})
		assert.NoError(t, tx.Error)
		assert.Equal(t, []string{`"join_hop_many_test_models"."id"`}, tx.Statement.Selects)
	}
	if assert.Contains(t, db.Statement.Preloads, "Relation") {
		tx := db.Session(&gorm.Session{}).Scopes(db.Statement.Preloads["Relation"][0].(func(*gorm.DB) *gorm.DB)).Find(&map[string]any{})
		assert.NoError(t, tx.Error)
		assert.Equal(t, []string{`"relation"."a"`, `"relation"."parent_id"`}, tx.Statement.Selects)
	}
	assert.NotContains(t, join.selectCache, "Relation")
	assert.Equal(t, []string{"id", "relation_id"}, join.selectCache["Relation.Parent"])
}

func TestJoinScopeFinal(t *testing.T) {
	db, _ := openMockDB(t)
	join := &Join{Relation: "Relation", Fields: []string{"a", "b"}}
	join.selectCache = map[string][]string{}
	schema, err := parseModel(db, &JoinHopManyTestModel{})
	require.NoError(t, err)
	assert.Nil(t, join.Scopes(Whitelist{Fields: AllowAllFields}, schema))
}

func TestJoinNestedRelationsWithSelect(t *testing.T) {
	db, mock := openMockDB(t)
	join := &Join{Relation: "Relation", Fields: []string{"b"}}
	join.selectCache = map[string][]string{}
	join2 := &Join{Relation: "Relation.Parent", Fields: []string{"id", "relation_id"}}
	join2.selectCache = join.selectCache
	schema, err := parseModel(db, &JoinHopManyTestModel{})
	require.NoError(t, err)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "join_hop_many_test_models"`)).
		WithoutArgs().
		WillReturnRows(sqlmock.NewRows([]string{"name", "id", "relation_id"}).AddRow("name", 123, 456))

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT "join_hop_many_test_models"."id" FROM "join_hop_many_test_models"`)).
		WithoutArgs().
		WillReturnRows(sqlmock.NewRows([]string{"join_hop_many_test_models.id"}).AddRow(456))

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT "relation"."b","relation"."a","relation"."parent_id" FROM "join_hop_many_test_models"`)).
		WithoutArgs().
		WillReturnRows(sqlmock.NewRows([]string{"relation.b", "relation.a", "relation.parent_id"}).AddRow("b", "a", 456))

	whitelist := Whitelist{
		Fields: []string{"id"},
		Relations: map[string]Whitelist{
			"Relation": {
				Fields: []string{"a", "b", "parent_id"},
				Relations: map[string]Whitelist{
					"Parent": {
						Fields: AllowAllFields,
					},
				},
			},
		},
	}

	db = db.Model(&JoinHopManyTestModel{}).Scopes(join.Scopes(whitelist, schema)...).Scopes(join2.Scopes(whitelist, schema)...).Find(&map[string]any{})
	assert.NoError(t, db.Error)
	if assert.Contains(t, db.Statement.Preloads, "Relation.Parent") {
		tx := db.Session(&gorm.Session{}).Scopes(db.Statement.Preloads["Relation.Parent"][0].(func(*gorm.DB) *gorm.DB)).Find(&map[string]any{})
		assert.NoError(t, tx.Error)
		assert.Equal(t, []string{`"join_hop_many_test_models"."id"`}, tx.Statement.Selects)
	}
	if assert.Contains(t, db.Statement.Preloads, "Relation") {
		tx := db.Session(&gorm.Session{}).Scopes(db.Statement.Preloads["Relation"][0].(func(*gorm.DB) *gorm.DB)).Find(&map[string]any{})
		assert.NoError(t, tx.Error)
		assert.Equal(t, []string{`"relation"."b"`, `"relation"."a"`, `"relation"."parent_id"`}, tx.Statement.Selects)
	}
	assert.Equal(t, []string{"b"}, join.selectCache["Relation"])
	assert.Equal(t, []string{"id", "relation_id"}, join.selectCache["Relation.Parent"])
}

func TestJoinScopeInvalidSyntax(t *testing.T) {
	db, _ := openMockDB(t)
	join := &Join{Relation: "Relation.", Fields: []string{"a", "b"}} // A dot at the end of the relation name is invalid
	join.selectCache = map[string][]string{}
	schema, err := parseModel(db, &JoinHopManyTestModel{})
	require.NoError(t, err)
	whitelist := Whitelist{
		Relations: map[string]Whitelist{
			"Relation": {Fields: AllowAllFields},
		},
		Fields: AllowAllFields,
	}
	assert.Nil(t, join.Scopes(whitelist, schema))
}

func TestJoinScopeNonExistingRelation(t *testing.T) {
	db, _ := openMockDB(t)
	join := &Join{Relation: "Relation.NotARelation.Parent", Fields: []string{"a", "b"}}
	join.selectCache = map[string][]string{}
	schema, err := parseModel(db, &JoinHopManyTestModel{})
	require.NoError(t, err)

	whitelist := Whitelist{
		Relations: map[string]Whitelist{
			"Relation": {
				Relations: map[string]Whitelist{
					// The relation is whitelisted even if it doesn't exist in the model.
					// It should still not return a scope.
					"NotARelation": {
						Relations: map[string]Whitelist{
							"Parent": {Fields: AllowAllFields},
						},
						Fields: AllowAllFields,
					},
				},
				Fields: AllowAllFields,
			},
		},
		Fields: AllowAllFields,
	}

	assert.Nil(t, join.Scopes(whitelist, schema))
}

func TestJoinScopeNonExistingFinalRelation(t *testing.T) {
	db, _ := openMockDB(t)
	join := &Join{Relation: "Relation.NotARelation", Fields: []string{"a", "b"}}
	join.selectCache = map[string][]string{}
	schema, err := parseModel(db, &JoinHopManyTestModel{})
	require.NoError(t, err)

	whitelist := Whitelist{
		Relations: map[string]Whitelist{
			"Relation": {
				Relations: map[string]Whitelist{
					// The relation is whitelisted even if it doesn't exist in the model.
					// It should still not return a scope.
					"NotARelation": {
						Fields: AllowAllFields,
					},
				},
				Fields: AllowAllFields,
			},
		},
		Fields: AllowAllFields,
	}

	assert.Nil(t, join.Scopes(whitelist, schema))
}

type JoinTestModelComputed struct {
	Relation *JoinRelationModelComputed
	Name     string
	ID       int `gorm:"primaryKey"`
	RelID    int `gorm:"column:relation_id"`
}

type JoinRelationModelComputed struct {
	C string `gorm:"->;-:migration" computed:"UPPER(~~~ct~~~.b)"`
	B string
	A int `gorm:"primaryKey"`
}

func TestJoinScopeComputedField(t *testing.T) {
	db, mock := openMockDB(t)
	join := &Join{Relation: "Relation", Fields: []string{"a", "b", "c"}}
	join.selectCache = map[string][]string{}

	schema, err := parseModel(db, &JoinTestModelComputed{})
	require.NoError(t, err)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "join_test_model_computeds"`)).
		WithoutArgs().
		WillReturnRows(sqlmock.NewRows([]string{"name", "id", "relation_id"}).AddRow("name", 123, 456))

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT (UPPER("join_relation_model_computeds".b)) "c","join_relation_model_computeds"."b","join_relation_model_computeds"."a" FROM "join_test_model_computeds"`)).
		WithoutArgs().
		WillReturnRows(sqlmock.NewRows([]string{"join_relation_model_computeds.a", "join_relation_model_computeds.b", "c"}).AddRow("a", "b", "B"))

	whitelist := Whitelist{
		Relations: map[string]Whitelist{
			"Relation": {Fields: AllowAllFields},
		},
		Fields: AllowAllFields,
	}

	db = db.Model(&JoinTestModelComputed{}).Scopes(join.Scopes(whitelist, schema)...).Find(&map[string]any{})
	assert.NoError(t, db.Error)
	if assert.Contains(t, db.Statement.Preloads, "Relation") {
		tx := db.Scopes(db.Statement.Preloads["Relation"][0].(func(*gorm.DB) *gorm.DB)).Find(&map[string]any{})
		assert.NoError(t, tx.Error)
		assert.Equal(t, []string{`(UPPER("join_relation_model_computeds".b)) "c"`, `"join_relation_model_computeds"."b"`, `"join_relation_model_computeds"."a"`}, tx.Statement.Selects)
	}
	assert.Equal(t, []string{"a", "b", "c"}, join.selectCache["Relation"])
}
