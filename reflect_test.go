package filter

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"gorm.io/gorm/utils/tests"
)

type TestRelation struct {
	TestModel          *TestModel `gorm:"foreignKey:TestModelID"`
	TestModelGuessed   *TestModel
	Name               string
	TestModelID        int
	TestModelGuessedID int
	ID                 int
}
type Promoted struct {
	Email string `gorm:"column:email_address"`
}
type PromotedPtr struct {
	Promoted
}
type PromotedRelation struct {
	PromotedRelation TestRelation
}
type TestModel struct {
	Relations []*TestRelation
	Relation  *TestRelation
	DeletedAt *gorm.DeletedAt
	*PromotedPtr
	Promoted
	Str     string `gorm:"column:"`
	Ignored string `gorm:"-"`
	PromotedRelation
	ID uint `gorm:"primaryKey"`
}

func TestParseModel(t *testing.T) {
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
	identity := parseModel(db, &TestModel{})

	relModelIdentity := &modelIdentity{
		Columns: map[string]*column{
			"name":                  {Name: "Name", Tags: &gormTags{}, Type: schema.String},
			"id":                    {Name: "ID", Tags: &gormTags{}, Type: schema.Uint},
			"test_model_id":         {Name: "TestModelID", Tags: &gormTags{}, Type: schema.Uint},
			"test_model_guessed_id": {Name: "TestModelGuessedID", Tags: &gormTags{}, Type: schema.Uint},
		},
		Relations:   map[string]*relation{},
		PrimaryKeys: []string{"id"},
	}
	expected := &modelIdentity{
		Columns: map[string]*column{
			"id":            {Name: "ID", Tags: &gormTags{PrimaryKey: true}, Type: schema.Uint},
			"str":           {Name: "Str", Tags: &gormTags{}, Type: schema.String},
			"email_address": {Name: "Email", Tags: &gormTags{Column: "email_address"}, Type: schema.String},
			"deleted_at":    {Name: "DeletedAt", Tags: &gormTags{}, Type: schema.Time},
		},
		PrimaryKeys: []string{"id"},
		Relations: map[string]*relation{
			"Relation": {
				modelIdentity: relModelIdentity,
				Type:          schema.HasOne,
				Tags:          &gormTags{},
				ForeignKeys:   []string{"test_model_id", "test_model_guessed_id"},
				keysProcessed: true,
			},
			"Relations": {
				modelIdentity: relModelIdentity,
				Type:          schema.HasMany,
				Tags:          &gormTags{},
				ForeignKeys:   []string{"test_model_id", "test_model_guessed_id"},
				keysProcessed: true,
			},
			"PromotedRelation": {
				modelIdentity: relModelIdentity,
				Type:          schema.HasOne,
				Tags:          &gormTags{},
				ForeignKeys:   []string{"test_model_id", "test_model_guessed_id"},
				keysProcessed: true,
			},
		},
	}
	relModelIdentity.Relations["TestModel"] = &relation{
		modelIdentity: expected,
		Type:          schema.HasOne,
		Tags:          &gormTags{ForeignKey: "TestModelID"},
		ForeignKeys:   []string{},
		keysProcessed: true,
	}
	relModelIdentity.Relations["TestModelGuessed"] = &relation{
		modelIdentity: expected,
		Type:          schema.HasOne,
		Tags:          &gormTags{},
		ForeignKeys:   []string{},
		keysProcessed: true,
	}
	assertModelIdentityEqual(t, expected, identity, []*modelIdentity{})

	assert.Same(t, identity.Relations["Relation"].modelIdentity, identity.Relations["Relations"].modelIdentity)
	assert.Same(t, identity.Relations["Relation"].modelIdentity, identity.Relations["PromotedRelation"].modelIdentity)

	assert.Contains(t, identityCache, "goyave.dev/filter|filter.TestRelation")
	assert.Contains(t, identityCache, "goyave.dev/filter|filter.Promoted")
	assert.Contains(t, identityCache, "goyave.dev/filter|filter.PromotedPtr")
	assert.Contains(t, identityCache, "goyave.dev/filter|filter.PromotedRelation")
	assert.Contains(t, identityCache, "goyave.dev/filter|filter.TestModel")

	identity = parseModel(db, []*TestModel{})
	assertModelIdentityEqual(t, expected, identity, []*modelIdentity{})
}

func assertModelIdentityEqual(t *testing.T, expected *modelIdentity, actual *modelIdentity, explored []*modelIdentity) {
	assert.Equal(t, expected.Columns, actual.Columns)
	assert.Equal(t, expected.PrimaryKeys, actual.PrimaryKeys)
	for k, v := range expected.Relations {
		if assert.Contains(t, actual.Relations, k) {
			v2 := actual.Relations[k]
			explored = append(explored, v.modelIdentity)
			if !isExplored(explored, v.modelIdentity) {
				assertModelIdentityEqual(t, v.modelIdentity, v2.modelIdentity, explored)
			}
			assert.Equal(t, v.Type, v2.Type)
			assert.Equal(t, v.Tags, v2.Tags)
			assert.Equal(t, v.keysProcessed, v2.keysProcessed)
			assert.ElementsMatch(t, v.ForeignKeys, v2.ForeignKeys)
		}
	}
	for k := range actual.Relations {
		assert.Contains(t, expected.Relations, k)
	}
	assert.Equal(t, expected.Columns, actual.Columns)
}

func isExplored(explored []*modelIdentity, identity *modelIdentity) bool {
	for _, v := range explored {
		if v == identity {
			return true
		}
	}
	return false
}

type TestRelationCycle struct {
	Parent *TestModelRelationCycle
}
type TestModelRelationCycle struct {
	*TestModelRelationCycle
	Relation *TestRelationCycle
}

func TestParseModelRelationCycle(t *testing.T) {
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
	identity := parseModel(db, &TestModelRelationCycle{})

	rel := &relation{
		modelIdentity: &modelIdentity{
			Columns:     map[string]*column{},
			Relations:   map[string]*relation{},
			PrimaryKeys: []string{},
			TableName:   "test_relation_cycles",
		},
		Type:          schema.HasOne,
		Tags:          &gormTags{},
		ForeignKeys:   []string{},
		LocalKeys:     []string{},
		keysProcessed: true,
	}
	expected := &modelIdentity{
		Columns: map[string]*column{},
		Relations: map[string]*relation{
			"Relation": rel,
		},
		PrimaryKeys: []string{},
		TableName:   "test_model_relation_cycles",
	}
	rel.Relations["Parent"] = &relation{
		modelIdentity: expected,
		Type:          schema.HasOne,
		Tags:          &gormTags{},
		ForeignKeys:   []string{},
		LocalKeys:     []string{},
		keysProcessed: true,
	}
	assert.Equal(t, expected, identity)
}

type TestModelBelongsTo struct {
	Parent      *TestModelBelongsToParent `gorm:"foreignKey:BelongsToID"`
	BelongsToID uint
}

type TestModelBelongsToParent struct {
	ID uint
}

func TestParseModelLocalKeys(t *testing.T) {
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
	identity := parseModel(db, &TestModelBelongsTo{})

	expected := &modelIdentity{
		Columns: map[string]*column{
			"belongs_to_id": {
				Name: "BelongsToID",
				Tags: &gormTags{},
				Type: schema.Uint,
			},
		},
		Relations: map[string]*relation{
			"Parent": {
				modelIdentity: &modelIdentity{
					Columns: map[string]*column{
						"id": {
							Name: "ID",
							Tags: &gormTags{},
							Type: schema.Uint,
						},
					},
					Relations:   map[string]*relation{},
					TableName:   "test_model_belongs_to_parents",
					PrimaryKeys: []string{"id"},
				},
				Type:          schema.HasOne,
				Tags:          &gormTags{ForeignKey: "BelongsToID"},
				ForeignKeys:   []string{},
				LocalKeys:     []string{"belongs_to_id"},
				keysProcessed: true,
			},
		},
		PrimaryKeys: []string{},
		TableName:   "test_model_belongs_tos",
	}
	assert.Equal(t, expected, identity)
}

func TestParseModelEmbeddedStruct(t *testing.T) {
	type TestEmbed struct {
		Name string
	}
	type TestModelEmbedded struct {
		Embed TestEmbed `gorm:"embedded;embeddedPrefix:embed_"`
	}

	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
	identity := parseModel(db, &TestModelEmbedded{})
	expected := &modelIdentity{
		Columns: map[string]*column{
			"embed_name": {Name: "Name", Tags: &gormTags{}, Type: schema.String},
		},
		Relations:   map[string]*relation{},
		PrimaryKeys: []string{},
		TableName:   "test_model_embeddeds",
	}
	assert.Equal(t, expected, identity)
}

func TestParseGormTags(t *testing.T) {
	type gormTagsModel struct {
		CustomColumn string `gorm:"column:custom_column"`
		Relation     string `gorm:"foreignKey:id_relation;references:relation"`
		Embedded     string `gorm:"embedded;embeddedPrefix:prefix_"`
		ID           int    `gorm:"primaryKey"`
		IDAlt        int    `gorm:"primary_key"`
		Ignored      int    `gorm:"-"`
	}

	ty := reflect.TypeOf(gormTagsModel{})
	expected := &gormTags{Column: "custom_column"}
	assert.Equal(t, expected, parseGormTags(ty.Field(0)))

	expected = &gormTags{ForeignKey: "id_relation", References: "relation"}
	assert.Equal(t, expected, parseGormTags(ty.Field(1)))

	expected = &gormTags{Embedded: true, EmbeddedPrefix: "prefix_"}
	assert.Equal(t, expected, parseGormTags(ty.Field(2)))

	expected = &gormTags{PrimaryKey: true}
	assert.Equal(t, expected, parseGormTags(ty.Field(3)))

	expected = &gormTags{PrimaryKey: true}
	assert.Equal(t, expected, parseGormTags(ty.Field(4)))

	expected = &gormTags{Ignored: true}
	assert.Equal(t, expected, parseGormTags(ty.Field(5)))
}

func TestCleanColumns(t *testing.T) {
	id := &modelIdentity{
		Columns: map[string]*column{
			"id":   {},
			"name": {},
		},
	}
	assert.Equal(t, []string{"id"}, id.cleanColumns([]string{"id", "test", "name", "notacolumn"}, []string{"name"}))
}

func TestFindColumn(t *testing.T) {
	id := &modelIdentity{
		Columns: map[string]*column{
			"id":   {Name: "ID"},
			"name": {Name: "Name"},
		},
	}
	col, name := id.findColumn("Name")
	assert.Same(t, id.Columns["name"], col)
	assert.Equal(t, "name", name)

	col, name = id.findColumn("NotaCol")
	assert.Nil(t, col)
	assert.Empty(t, name)
}

func TestAddPrimaryKeys(t *testing.T) {
	id := &modelIdentity{
		PrimaryKeys: []string{"id_1", "id_2"},
	}

	fields := []string{"id_2"}
	fields = id.addPrimaryKeys(fields)
	assert.Equal(t, []string{"id_2", "id_1"}, fields)
}

func TestParseNilModel(t *testing.T) {
	assert.Nil(t, parseModel(nil, 1))
}

type Numbers []int

func (n *Numbers) Scan(value interface{}) error {
	return nil
}

func TestParseModelSliceField(t *testing.T) {
	type DataTypeModel struct {
		Numbers    Numbers
		NumbersPtr *Numbers
		SliceOfInt []int
		ID         uint
	}
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
	identity := parseModel(db, &DataTypeModel{})

	assert.Contains(t, identity.Columns, "numbers")
	assert.Contains(t, identity.Columns, "numbers_ptr")
	assert.NotContains(t, identity.Columns, "slice_of_int")
}
