package filter

import (
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/utils/tests"
	"testing"
)

func TestSearchScope(t *testing.T) {
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
	search := &Search{Field: "name", Query: "Name"}
	modelIdentity := &modelIdentity{
		Columns: map[string]*column{
			"name": {Name: "Name"},
		},
		TableName: "test_models",
	}

	assert.Nil(t, search.Scope(&Settings{}, modelIdentity))

	db = db.Scopes(search.Scope(&Settings{FieldsSearch: []string{"name"}}, modelIdentity)).Table("table").Find(nil)
	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.OrConditions{
						Exprs: []clause.Expression{
							clause.Expr{
								SQL: "(lower(CAST(`test_models`.`name` AS TEXT))) LIKE lower(?)",
								Vars: []interface {}{"%Name%"},
								WithoutParentheses: false,
							},
						},
					},
				},
			},
		},
	}
	assert.Equal(t, expected, db.Statement.Clauses)
}

func TestSearchScopeFieldsNotDefined(t *testing.T) {
	search := &Search{Field: "name", Query: "Name"}
	modelIdentity := &modelIdentity{
		Columns: map[string]*column{
			"name": {Name: "Name"},
		},
		TableName: "test_models",
	}

	assert.Nil(t, search.Scope(&Settings{}, modelIdentity))

	scope := search.Scope(&Settings{FieldsSearch: []string{"notdefined"}}, modelIdentity)

	assert.Nil(t, scope)
}