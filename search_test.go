package filter

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/utils/tests"
	"testing"
)

func TestSearchScope(t *testing.T) {
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
	search := &Search{Fields: []string{"name", "email"}, Query: "My Query"}
	modelIdentity := &modelIdentity{
		Columns: map[string]*column{
			"name":  {Name: "Name"},
			"email": {Name: "Email"},
			"role":  {Name: "role"},
		},
		TableName: "test_models",
	}

	db = db.Scopes(search.Scopes(&Settings{
		FieldsSearch: []string{"name", "email"},
		SearchOperator: &Operator{
			Function: func(tx *gorm.DB, filter *Filter, column string) *gorm.DB {
				return tx.Or(fmt.Sprintf("%s LIKE (?)", filter.Field), filter.Args[0])
			},
			RequiredArguments: 1,
		},
	}, modelIdentity)).Table("table").Find(nil)
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
										SQL:                "name LIKE (?)",
										Vars:               []interface{}{"My Query"},
										WithoutParentheses: false,
									},
								},
							},
							clause.OrConditions{
								Exprs: []clause.Expression{
									clause.Expr{
										SQL:                "email LIKE (?)",
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
