package filter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
