package filter

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
	"gorm.io/gorm/utils/tests"
)

func TestSearchScope(t *testing.T) {
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
	search := &Search{
		Fields: []string{"name", "email"},
		Query:  "My Query",
		Operator: &Operator{
			Function: func(tx *gorm.DB, filter *Filter, column string, dataType schema.DataType) *gorm.DB {
				return tx.Or(fmt.Sprintf("%s LIKE (?)", column), filter.Args[0])
			},
			RequiredArguments: 1,
		},
	}
	modelIdentity := &modelIdentity{
		Columns: map[string]*column{
			"name":  {Name: "Name"},
			"email": {Name: "Email"},
			"role":  {Name: "role"},
		},
		TableName: "test_models",
	}

	db = db.Scopes(search.Scope(modelIdentity)).Table("table").Find(nil)
	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.AndConditions{
						Exprs: []clause.Expression{
							clause.OrConditions{
								Exprs: []clause.Expression{
									clause.Expr{
										SQL:                "`test_models`.`name` LIKE (?)",
										Vars:               []interface{}{"My Query"},
										WithoutParentheses: false,
									},
								},
							},
							clause.OrConditions{
								Exprs: []clause.Expression{
									clause.Expr{
										SQL:                "`test_models`.`email` LIKE (?)",
										Vars:               []interface{}{"My Query"},
										WithoutParentheses: false,
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

func TestSearchScopeEmptyField(t *testing.T) {
	search := &Search{
		Fields: []string{},
		Query:  "My Query",
		Operator: &Operator{
			Function: func(tx *gorm.DB, filter *Filter, column string, dataType schema.DataType) *gorm.DB {
				return tx.Or(fmt.Sprintf("%s LIKE (?)", column), filter.Args[0])
			},
			RequiredArguments: 1,
		},
	}
	modelIdentity := &modelIdentity{
		Columns: map[string]*column{
			"name":  {Name: "Name"},
			"email": {Name: "Email"},
			"role":  {Name: "role"},
		},
		TableName: "test_models",
	}

	assert.Nil(t, search.Scope(modelIdentity))
}
