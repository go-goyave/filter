package filter

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
	"gorm.io/gorm/utils/tests"
)

type TestRelation struct {
	Name string
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
	Promoted
	*PromotedPtr
	PromotedRelation
	Str       string `gorm:"column:"`
	Relation  *TestRelation
	Relations []*TestRelation
	ID        uint
}

func TestParseModel(t *testing.T) {
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
	identity := parseModel(db, &TestModel{})

	relation := &modelIdentity{
		Columns: map[string]column{
			"name": {Name: "Name", Tag: ""},
		},
		Relations: map[string]*modelIdentity{},
	}
	expected := &modelIdentity{
		Columns: map[string]column{
			"id":            {Name: "ID", Tag: ""},
			"str":           {Name: "Str", Tag: `gorm:"column:"`},
			"email_address": {Name: "Email", Tag: `gorm:"column:email_address"`},
		},
		Relations: map[string]*modelIdentity{
			"Relation":         relation,
			"Relations":        relation,
			"PromotedRelation": relation,
		},
	}
	assert.Equal(t, expected, identity)
	assert.Same(t, identity.Relations["Relation"], identity.Relations["Relations"])
	assert.Same(t, identity.Relations["Relation"], identity.Relations["PromotedRelation"])

	assert.Contains(t, identityCache, "goyave.dev/filter|filter.TestRelation")
	assert.Contains(t, identityCache, "goyave.dev/filter|filter.Promoted")
	assert.Contains(t, identityCache, "goyave.dev/filter|filter.PromotedPtr")
	assert.Contains(t, identityCache, "goyave.dev/filter|filter.PromotedRelation")
	assert.Contains(t, identityCache, "goyave.dev/filter|filter.TestModel")

	identity = parseModel(db, []*TestModel{})
	assert.Equal(t, expected, identity)
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
	expected := &modelIdentity{
		Columns: map[string]column{},
		Relations: map[string]*modelIdentity{
			"Relation": {
				Columns:   map[string]column{},
				Relations: map[string]*modelIdentity{},
			},
		},
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
		Columns: map[string]column{
			"embed_name": {Name: "Name", Tag: ""},
		},
		Relations: map[string]*modelIdentity{},
	}
	assert.Equal(t, expected, identity)
}

func TestGetEmbeddedInfoInvalidSyntax(t *testing.T) {
	field := reflect.StructField{
		Tag: `gorm:"embedded;embeddedPrefix:"`,
	}

	prefix, ok := getEmbeddedInfo(field)
	assert.Empty(t, prefix)
	assert.True(t, ok)
}

func TestParseNilModel(t *testing.T) {
	assert.Nil(t, parseModel(nil, 1))
}
