package filter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
	"gorm.io/gorm/utils/tests"
)

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

func TestFilterScope(t *testing.T) {
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
	filter := &Filter{Field: "notacolumn", Args: []string{"val1"}, Operator: Operators["$eq"]}
	modelIdentity := &modelIdentity{
		Columns: map[string]*column{
			"name": {Name: "Name"},
		},
		TableName: "test_scope_models",
	}

	assert.Nil(t, filter.Scope(&Settings{}, modelIdentity))

	filter.Field = "name"

	db = db.Scopes(filter.Scope(&Settings{}, modelIdentity)).Find(nil)
	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.Expr{SQL: "`test_scope_models`.`name` = ?", Vars: []interface{}{"val1"}},
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

type FilterTestRelation struct {
	Name     string
	ID       uint
	ParentID uint
}

type FilterTestModel struct {
	Relation *FilterTestRelation `gorm:"foreignKey:ParentID"`
	Name     string
	ID       uint
}

func TestFilterScopeWithJoin(t *testing.T) {
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
	filter := &Filter{Field: "Relation.name", Args: []string{"val1"}, Operator: Operators["$eq"]}

	results := []*FilterTestModel{}
	modelIdentity := parseModel(db, &results)

	db = db.Model(&results).Scopes(filter.Scope(&Settings{}, modelIdentity)).Find(&results)
	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.Expr{SQL: "`filter_test_relations`.`name` = ?", Vars: []interface{}{"val1"}},
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
							Name: "filter_test_relations",
						},
						ON: clause.Where{
							Exprs: []clause.Expression{
								clause.Eq{
									Column: clause.Column{
										Table: clause.CurrentTable,
										Name:  "id",
									},
									Value: clause.Column{
										Table: "filter_test_relations",
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

func TestFilterScopeWithJoinBlacklistedRelation(t *testing.T) {
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
	filter := &Filter{Field: "Relation.name", Args: []string{"val1"}, Operator: Operators["$eq"]}

	results := []*FilterTestModel{}
	modelIdentity := parseModel(db, &results)

	settings := &Settings{
		Blacklist: Blacklist{
			RelationsBlacklist: []string{"Relation"},
		},
	}

	assert.Nil(t, filter.Scope(settings, modelIdentity))
}

func TestFilterScopeWithJoinHasMany(t *testing.T) {
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
	filter := &Filter{Field: "Relation.name", Args: []string{"val1"}, Operator: Operators["$eq"]}

	results := []*FilterTestModel{}
	modelIdentity := parseModel(db, &results)
	modelIdentity.Relations["Relation"].Type = schema.HasMany
	assert.Nil(t, filter.Scope(&Settings{}, modelIdentity))
	modelIdentity.Relations["Relation"].Type = schema.HasOne
}

func TestFilterScopeWithJoinInvalidModel(t *testing.T) {
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
	filter := &Filter{Field: "Relation.name", Args: []string{"val1"}, Operator: Operators["$eq"]}

	results := []*FilterTestModel{}
	modelIdentity := parseModel(db, &results)

	db = db.Scopes(filter.Scope(&Settings{}, modelIdentity)).Find(&results)
	assert.Equal(t, "unsupported data type: <nil>", db.Error.Error())
}
