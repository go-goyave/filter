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
	schema := &schema.Schema{
		FieldsByDBName: map[string]*schema.Field{
			"name": {Name: "Name"},
		},
		Table: "test_scope_models",
	}

	assert.Nil(t, filter.Scope(&Settings{}, schema))

	filter.Field = "name"

	db = db.Scopes(filter.Scope(&Settings{}, schema)).Find(nil)
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
	schema := &schema.Schema{
		FieldsByDBName: map[string]*schema.Field{
			"name": {Name: "Name"},
		},
	}

	assert.Nil(t, filter.Scope(&Settings{Blacklist: Blacklist{FieldsBlacklist: []string{"name"}}}, schema))
}

type FilterTestNestedRelation struct {
	Field    string
	ID       uint
	ParentID uint
}

type FilterTestRelation struct {
	NestedRelation *FilterTestNestedRelation `gorm:"foreignKey:ParentID"`
	Name           string
	ID             uint
	ParentID       uint
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
	schema, err := parseModel(db, &results)
	if !assert.Nil(t, err) {
		return
	}

	db = db.Model(&results).Scopes(filter.Scope(&Settings{}, schema)).Find(&results)
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
										Table: "filter_test_models",
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
	schema, err := parseModel(db, &results)
	if !assert.Nil(t, err) {
		return
	}

	settings := &Settings{
		Blacklist: Blacklist{
			RelationsBlacklist: []string{"Relation"},
		},
	}

	assert.Nil(t, filter.Scope(settings, schema))
}

func TestFilterScopeWithJoinHasMany(t *testing.T) {
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
	filter := &Filter{Field: "Relation.name", Args: []string{"val1"}, Operator: Operators["$eq"]}

	results := []*FilterTestModel{}
	sch, err := parseModel(db, &results)
	if !assert.Nil(t, err) {
		return
	}
	sch.Relationships.Relations["Relation"].Type = schema.HasMany
	assert.Nil(t, filter.Scope(&Settings{}, sch))
	sch.Relationships.Relations["Relation"].Type = schema.HasOne
}

func TestFilterScopeWithJoinInvalidModel(t *testing.T) {
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
	filter := &Filter{Field: "Relation.name", Args: []string{"val1"}, Operator: Operators["$eq"]}

	results := []*FilterTestModel{}
	sch, err := parseModel(db, &results)
	if !assert.Nil(t, err) {
		return
	}

	db = db.Scopes(filter.Scope(&Settings{}, sch)).Find(&results)
	assert.Equal(t, "unsupported data type: <nil>", db.Error.Error())
}

func TestFilterScopeWithJoinNestedRelation(t *testing.T) {
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
	filter := &Filter{Field: "Relation.NestedRelation.field", Args: []string{"val1"}, Operator: Operators["$eq"]}

	results := []*FilterTestModel{}
	sch, err := parseModel(db, &results)
	if !assert.Nil(t, err) {
		return
	}

	scp := filter.Scope(&Settings{}, sch)
	assert.NotNil(t, scp)
	db = db.Model(&results).Scopes(scp).Find(&results)
	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.Expr{SQL: "`filter_test_nested_relations`.`field` = ?", Vars: []interface{}{"val1"}},
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
										Table: "filter_test_models",
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
					{
						Type: clause.LeftJoin,
						Table: clause.Table{
							Name: "filter_test_nested_relations",
						},
						ON: clause.Where{
							Exprs: []clause.Expression{
								clause.Eq{
									Column: clause.Column{
										Table: "filter_test_relations",
										Name:  "id",
									},
									Value: clause.Column{
										Table: "filter_test_nested_relations",
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
