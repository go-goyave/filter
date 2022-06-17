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
			"name": {Name: "Name", DBName: "name"},
		},
		Table: "test_scope_models",
	}

	joinScope, conditionScope := filter.Scope(&Settings{}, schema)
	assert.Nil(t, joinScope)
	assert.Nil(t, conditionScope)

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

	joinScope, conditionScope := filter.Scope(&Settings{Blacklist: Blacklist{FieldsBlacklist: []string{"name"}}}, schema)
	assert.Nil(t, joinScope)
	assert.Nil(t, conditionScope)
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
					clause.Expr{SQL: "`Relation`.`name` = ?", Vars: []interface{}{"val1"}},
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
							Name:  "filter_test_relations",
							Alias: "Relation",
						},
						ON: clause.Where{
							Exprs: []clause.Expression{
								clause.Eq{
									Column: clause.Column{
										Table: "filter_test_models",
										Name:  "id",
									},
									Value: clause.Column{
										Table: "Relation",
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

	joinScope, conditionScope := filter.Scope(settings, schema)
	assert.Nil(t, joinScope)
	assert.Nil(t, conditionScope)
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
	joinScope, conditionScope := filter.Scope(&Settings{}, sch)
	assert.Nil(t, joinScope)
	assert.Nil(t, conditionScope)
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

	joinScope, conditionScope := filter.Scope(&Settings{}, sch)
	assert.NotNil(t, joinScope)
	assert.NotNil(t, conditionScope)
	conditionTx := db.Session(&gorm.Session{NewDB: true}).Model(&results).Scopes(conditionScope).Find(&results)
	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.Expr{SQL: "`NestedRelation`.`field` = ?", Vars: []interface{}{"val1"}},
				},
			},
		},
	}
	assert.Equal(t, expected, conditionTx.Statement.Clauses)

	joinTx := db.Session(&gorm.Session{NewDB: true}).Model(&results).Scopes(joinScope).Find(&results)
	expected = map[string]clause.Clause{
		"FROM": {
			Name: "FROM",
			Expression: clause.From{
				Joins: []clause.Join{
					{
						Type: clause.LeftJoin,
						Table: clause.Table{
							Name:  "filter_test_relations",
							Alias: "Relation",
						},
						ON: clause.Where{
							Exprs: []clause.Expression{
								clause.Eq{
									Column: clause.Column{
										Table: "filter_test_models",
										Name:  "id",
									},
									Value: clause.Column{
										Table: "Relation",
										Name:  "parent_id",
									},
								},
							},
						},
					},
					{
						Type: clause.LeftJoin,
						Table: clause.Table{
							Name:  "filter_test_nested_relations",
							Alias: "NestedRelation",
						},
						ON: clause.Where{
							Exprs: []clause.Expression{
								clause.Eq{
									Column: clause.Column{
										Table: "Relation",
										Name:  "id",
									},
									Value: clause.Column{
										Table: "NestedRelation",
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
	assert.Equal(t, expected, joinTx.Statement.Clauses)
}

func TestFilterScopeWithJoinDontDuplicate(t *testing.T) {
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
	settings := &Settings{}
	filter := &Filter{Field: "Relation.name", Args: []string{"val1"}, Operator: Operators["$eq"]}
	filter2 := &Filter{Field: "Relation.id", Args: []string{"0"}, Operator: Operators["$gt"]}

	results := []*FilterTestModel{}
	schema, err := parseModel(db, &results)
	if !assert.Nil(t, err) {
		return
	}

	db = db.Model(&results).
		Scopes(filter.Scope(settings, schema)).
		Scopes(filter2.Scope(settings, schema)).
		Find(&results)
	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.Expr{SQL: "`Relation`.`name` = ?", Vars: []interface{}{"val1"}},
					clause.Expr{SQL: "`Relation`.`id` > ?", Vars: []interface{}{"0"}},
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
							Name:  "filter_test_relations",
							Alias: "Relation",
						},
						ON: clause.Where{
							Exprs: []clause.Expression{
								clause.Eq{
									Column: clause.Column{
										Table: "filter_test_models",
										Name:  "id",
									},
									Value: clause.Column{
										Table: "Relation",
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

func TestFilterScopeWithAlreadyExistingJoin(t *testing.T) {
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
	filter := &Filter{Field: "Relation.name", Args: []string{"val1"}, Operator: Operators["$eq"]}

	results := []*FilterTestModel{}
	schema, err := parseModel(db, &results)
	if !assert.Nil(t, err) {
		return
	}

	// We manually join a relation with a condition.
	// We expect this join to not be removed nor duplicated, with the condition kept.
	db = db.Joins("Relation", db.Session(&gorm.Session{NewDB: true}).Where("id > ?", 0))

	db = db.Model(&results).Scopes(filter.Scope(&Settings{}, schema)).Find(&results)
	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.Expr{SQL: "`Relation`.`name` = ?", Vars: []interface{}{"val1"}},
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
							Name:  "filter_test_relations",
							Alias: "Relation",
						},
						ON: clause.Where{
							Exprs: []clause.Expression{
								clause.Eq{
									Column: clause.Column{
										Table: "filter_test_models",
										Name:  "id",
									},
									Value: clause.Column{
										Table: "Relation",
										Name:  "parent_id",
									},
								},
								clause.Where{
									Exprs: []clause.Expression{
										clause.Expr{SQL: "id > ?", Vars: []interface{}{0}},
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
	assert.Empty(t, db.Statement.Joins)
}
