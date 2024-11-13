package filter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

func TestFilterWhere(t *testing.T) {
	db := openDryRunDB(t)
	filter := &Filter{Field: "name", Args: []string{"val1"}}
	db = filter.Where(db, "name = ?", "val1")
	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.Expr{SQL: "name = ?", Vars: []any{"val1"}},
				},
			},
		},
	}
	assert.Equal(t, expected, db.Statement.Clauses)
}

func TestFilterWhereOr(t *testing.T) {
	db := openDryRunDB(t)
	filter := &Filter{Field: "name", Args: []string{"val1"}, Or: true}
	db = filter.Where(db, "name = ?", "val1")
	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.OrConditions{
						Exprs: []clause.Expression{
							clause.Expr{SQL: "name = ?", Vars: []any{"val1"}},
						},
					},
				},
			},
		},
	}
	assert.Equal(t, expected, db.Statement.Clauses)
}

func TestFilterScope(t *testing.T) {
	db := openDryRunDB(t)
	filter := &Filter{Field: "notacolumn", Args: []string{"val1"}, Operator: Operators["$eq"]}
	field := &schema.Field{Name: "Name", DBName: "name", GORMDataType: schema.String}
	schema := &schema.Schema{
		DBNames: []string{"name"},
		FieldsByDBName: map[string]*schema.Field{
			"name": field,
		},
		FieldsByName: map[string]*schema.Field{
			"Name": field,
		},
		Table: "test_scope_models",
	}

	joinScope, conditionScope := filter.Scope(Blacklist{}, schema)
	assert.Nil(t, joinScope)
	assert.Nil(t, conditionScope)

	filter.Field = "name"

	results := []map[string]any{}
	db = db.Scopes(filter.Scope(Blacklist{}, schema)).Find(results)
	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.Expr{SQL: "`test_scope_models`.`name` = ?", Vars: []any{"val1"}},
				},
			},
		},
	}
	assert.Equal(t, expected, db.Statement.Clauses)

	// Using struct field name
	filter.Field = "Name"

	results = []map[string]any{}
	db = openDryRunDB(t)
	db = db.Scopes(filter.Scope(Blacklist{}, schema)).Find(results)
	expected = map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.Expr{SQL: "`test_scope_models`.`name` = ?", Vars: []any{"val1"}},
				},
			},
		},
	}
	assert.Equal(t, expected, db.Statement.Clauses)
}

func TestFilterScopeBlacklisted(t *testing.T) {
	filter := &Filter{Field: "name", Args: []string{"val1"}, Operator: Operators["$eq"]}
	schema := &schema.Schema{
		DBNames: []string{"name"},
		FieldsByDBName: map[string]*schema.Field{
			"name": {Name: "Name", GORMDataType: schema.String},
		},
	}

	joinScope, conditionScope := filter.Scope(Blacklist{FieldsBlacklist: []string{"name"}}, schema)
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

type FilterTestRelationUnsupported struct {
	Name     string `filterType:"-"`
	ID       uint
	ParentID uint
}

type FilterTestModelUnsupported struct {
	Relation *FilterTestRelationUnsupported `gorm:"foreignKey:ParentID"`
	Name     string
	ID       uint
}

func TestFilterScopeWithJoin(t *testing.T) {
	db := openDryRunDB(t)
	filter := &Filter{Field: "Relation.name", Args: []string{"val1"}, Operator: Operators["$eq"]}

	results := []*FilterTestModel{}
	schema, err := parseModel(db, &results)
	if !assert.Nil(t, err) {
		return
	}

	db.DryRun = true
	db = db.Model(&results).Scopes(filter.Scope(Blacklist{}, schema)).Find(&results)
	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.Expr{SQL: "`Relation`.`name` = ?", Vars: []any{"val1"}},
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
		"SELECT": {
			Name: "SELECT",
			Expression: clause.Select{
				Columns: []clause.Column{
					{Table: "filter_test_models", Name: "name"},
					{Table: "filter_test_models", Name: "id"},
				},
			},
		},
	}
	assert.Equal(t, expected, db.Statement.Clauses)
	assert.Nil(t, db.Error)
}

func TestFilterScopeWithJoinBlacklistedRelation(t *testing.T) {
	db := openDryRunDB(t)
	filter := &Filter{Field: "Relation.name", Args: []string{"val1"}, Operator: Operators["$eq"]}

	results := []*FilterTestModel{}
	schema, err := parseModel(db, &results)
	if !assert.Nil(t, err) {
		return
	}

	blacklist := Blacklist{
		RelationsBlacklist: []string{"Relation"},
	}

	joinScope, conditionScope := filter.Scope(blacklist, schema)
	assert.Nil(t, joinScope)
	assert.Nil(t, conditionScope)
}

func TestFilterScopeWithJoinHasMany(t *testing.T) {
	db := openDryRunDB(t)
	filter := &Filter{Field: "Relation.name", Args: []string{"val1"}, Operator: Operators["$eq"]}

	results := []*FilterTestModel{}
	sch, err := parseModel(db, &results)
	if !assert.Nil(t, err) {
		return
	}
	sch.Relationships.Relations["Relation"].Type = schema.HasMany
	joinScope, conditionScope := filter.Scope(Blacklist{}, sch)
	assert.Nil(t, joinScope)
	assert.Nil(t, conditionScope)
	sch.Relationships.Relations["Relation"].Type = schema.HasOne
}

func TestFilterScopeWithJoinInvalidModel(t *testing.T) {
	db := openDryRunDB(t)
	filter := &Filter{Field: "Relation.name", Args: []string{"val1"}, Operator: Operators["$eq"]}

	results := []*FilterTestModel{}
	sch, err := parseModel(db, &results)
	if !assert.Nil(t, err) {
		return
	}

	db = db.Scopes(filter.Scope(Blacklist{}, sch)).Find(&results)
	assert.Equal(t, "unsupported data type: <nil>", db.Error.Error())
}

func TestFilterScopeWithJoinNestedRelation(t *testing.T) {
	db := openDryRunDB(t)
	filter := &Filter{Field: "Relation.NestedRelation.field", Args: []string{"val1"}, Operator: Operators["$eq"]}

	results := []*FilterTestModel{}
	sch, err := parseModel(db, &results)
	if !assert.Nil(t, err) {
		return
	}

	joinScope, conditionScope := filter.Scope(Blacklist{}, sch)
	assert.NotNil(t, joinScope)
	assert.NotNil(t, conditionScope)
	conditionTx := db.Session(&gorm.Session{NewDB: true}).Model(&results).Scopes(conditionScope).Find(&results)
	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.Expr{SQL: "`NestedRelation`.`field` = ?", Vars: []any{"val1"}},
				},
			},
		},
		"FROM": {
			Name:       "FROM",
			Expression: clause.From{},
		},
		"SELECT": {
			Name:       "SELECT",
			Expression: clause.Select{},
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
		"SELECT": {
			Name: "SELECT",
			Expression: clause.Select{
				Columns: []clause.Column{
					{Table: "filter_test_models", Name: "name"},
					{Table: "filter_test_models", Name: "id"},
				},
			},
		},
	}
	assert.Equal(t, expected, joinTx.Statement.Clauses)
}

func TestFilterScopeWithJoinDontDuplicate(t *testing.T) {
	db := openDryRunDB(t)
	filter := &Filter{Field: "Relation.name", Args: []string{"val1"}, Operator: Operators["$eq"]}
	filter2 := &Filter{Field: "Relation.id", Args: []string{"0"}, Operator: Operators["$gt"]}

	results := []*FilterTestModel{}
	schema, err := parseModel(db, &results)
	if !assert.Nil(t, err) {
		return
	}

	db = db.Model(&results).
		Scopes(filter.Scope(Blacklist{}, schema)).
		Scopes(filter2.Scope(Blacklist{}, schema)).
		Find(&results)
	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.Expr{SQL: "`Relation`.`name` = ?", Vars: []any{"val1"}},
					clause.Expr{SQL: "`Relation`.`id` > ?", Vars: []any{uint64(0)}},
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
		"SELECT": {
			Name: "SELECT",
			Expression: clause.Select{
				Columns: []clause.Column{
					{Table: "filter_test_models", Name: "name"},
					{Table: "filter_test_models", Name: "id"},
				},
			},
		},
	}
	assert.Equal(t, expected, db.Statement.Clauses)
}

func TestFilterScopeWithAlreadyExistingJoin(t *testing.T) {
	db := openDryRunDB(t)
	filter := &Filter{Field: "Relation.name", Args: []string{"val1"}, Operator: Operators["$eq"]}

	results := []*FilterTestModel{}
	schema, err := parseModel(db, &results)
	if !assert.Nil(t, err) {
		return
	}

	// We manually join a relation with a condition.
	// We expect this join to not be removed nor duplicated, with the condition kept.
	db = db.Joins("Relation", db.Session(&gorm.Session{NewDB: true}).Where("id > ?", 0))

	db = db.Model(&results).Scopes(filter.Scope(Blacklist{}, schema)).Find(&results)
	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.Expr{SQL: "`Relation`.`name` = ?", Vars: []any{"val1"}},
				},
			},
		},
		"FROM": {
			Name: "FROM",
			Expression: clause.From{
				Joins: []clause.Join{},
			},
		},
	}
	assert.Equal(t, expected["FROM"], db.Statement.Clauses["FROM"])
	assert.Equal(t, expected["WHERE"], db.Statement.Clauses["WHERE"])
	require.Len(t, db.Statement.Joins, 1)
	j := db.Statement.Joins[0]
	assert.Equal(t, "Relation", j.Name)
	assert.Equal(t, clause.LeftJoin, j.JoinType)
	assert.Empty(t, j.Omits)
	if assert.Len(t, j.Conds, 1) {
		assert.Equal(t, clause.Where{Exprs: []clause.Expression{clause.Expr{SQL: "id > ?", Vars: []any{0}}}}, j.Conds[0].(*gorm.DB).Statement.Clauses["WHERE"].Expression)
	}
	assert.Empty(t, j.Selects)
	expectedOn := &clause.Where{
		Exprs: []clause.Expression{
			clause.Expr{SQL: "id > ?", Vars: []any{0}},
		},
	}
	assert.Equal(t, expectedOn, j.On)
}

func TestFilterScopeWithAlreadyExistingRawJoin(t *testing.T) {
	db := openDryRunDB(t)
	filter := &Filter{Field: "Relation.name", Args: []string{"val1"}, Operator: Operators["$eq"]}

	results := []*FilterTestModel{}
	schema, err := parseModel(db, &results)
	if !assert.Nil(t, err) {
		return
	}

	// We manually join a relation with a condition.
	// We expect this join to not be removed nor duplicated, with the condition kept.
	db = db.Joins(`LEFT JOIN filter_test_relations AS "Relation" ON id > ?`, 0)

	db = db.Model(&results).Scopes(filter.Scope(Blacklist{}, schema)).Find(&results)
	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.Expr{SQL: "`Relation`.`name` = ?", Vars: []any{"val1"}},
				},
			},
		},
		"FROM": {
			Name: "FROM",
			Expression: clause.From{
				Joins: []clause.Join{
					// {
					// 	Expression: clause.NamedExpr{
					// 		SQL:  `LEFT JOIN filter_test_relations AS "Relation" ON id > ?`,
					// 		Vars: []any{0},
					// 	},
					// },
				},
			},
		},
		"SELECT": {
			Name: "SELECT",
			Expression: clause.Select{
				Columns: []clause.Column{
					{Table: "filter_test_models", Name: "name"},
					{Table: "filter_test_models", Name: "id"},
				},
			},
		},
	}
	assert.Equal(t, expected, db.Statement.Clauses)

	require.Len(t, db.Statement.Joins, 1)
	j := db.Statement.Joins[0]
	assert.Equal(t, `LEFT JOIN filter_test_relations AS "Relation" ON id > ?`, j.Name)
	assert.Equal(t, clause.LeftJoin, j.JoinType)
	assert.Empty(t, j.Omits)
	assert.Empty(t, j.Selects)
	if assert.Len(t, j.Conds, 1) {
		assert.Equal(t, 0, j.Conds[0])
	}
	assert.Nil(t, j.On)
}

type FilterTestModelComputedRelation struct {
	Name     string
	Computed string `computed:"~~~ct~~~.computedcolumnrelation"`
	ID       uint
	ParentID uint
}

type FilterTestModelComputed struct {
	Relation *FilterTestModelComputedRelation `gorm:"foreignKey:ParentID"`
	Name     string
	Computed string `computed:"~~~ct~~~.computedcolumn"`
	ID       uint
}

func TestFilterScopeComputed(t *testing.T) {
	db := openDryRunDB(t)
	filter := &Filter{Field: "computed", Args: []string{"val1"}, Operator: Operators["$eq"]}

	results := []*FilterTestModelComputed{}
	schema, err := parseModel(db, &results)
	if !assert.Nil(t, err) {
		return
	}

	db = db.Model(&results).Scopes(filter.Scope(Blacklist{}, schema)).Find(&results)
	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.Expr{SQL: "(`filter_test_model_computeds`.computedcolumn) = ?", Vars: []any{"val1"}},
				},
			},
		},
		"FROM": {
			Name:       "FROM",
			Expression: clause.From{},
		},
		"SELECT": {
			Name:       "SELECT",
			Expression: clause.Select{},
		},
	}
	assert.Equal(t, expected, db.Statement.Clauses)
}

func TestFilterScopeComputedRelation(t *testing.T) {
	db := openDryRunDB(t)
	filter := &Filter{Field: "Relation.computed", Args: []string{"val1"}, Operator: Operators["$eq"]}

	results := []*FilterTestModelComputed{}
	schema, err := parseModel(db, &results)
	if !assert.Nil(t, err) {
		return
	}

	db = db.Model(&results).Scopes(filter.Scope(Blacklist{}, schema)).Find(&results)
	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.Expr{SQL: "(`Relation`.computedcolumnrelation) = ?", Vars: []any{"val1"}},
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
							Name:  "filter_test_model_computed_relations",
							Alias: "Relation",
						},
						ON: clause.Where{
							Exprs: []clause.Expression{
								clause.Eq{
									Column: clause.Column{
										Table: "filter_test_model_computeds",
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
		"SELECT": {
			Name: "SELECT",
			Expression: clause.Select{
				Columns: []clause.Column{
					{Table: "filter_test_model_computeds", Name: "name"},
					{Table: "filter_test_model_computeds", Name: "computed"}, // Should not be problematic that it is added automatically by Gorm since we force only selectable fields all he time.
					{Table: "filter_test_model_computeds", Name: "id"},
				},
			},
		},
	}
	assert.Equal(t, expected, db.Statement.Clauses)
}

func TestFilterScopeWithUnsupportedDataType(t *testing.T) {
	db := openDryRunDB(t)
	filter := &Filter{Field: "name", Args: []string{"val1"}, Operator: Operators["$eq"]}
	schema := &schema.Schema{
		DBNames: []string{"name"},
		FieldsByDBName: map[string]*schema.Field{
			"name": {Name: "Name", DBName: "name", GORMDataType: "custom", DataType: "CHARACTER VARYING(255)"},
		},
		Table: "test_scope_models",
	}

	results := []map[string]any{}
	db = db.Scopes(filter.Scope(Blacklist{}, schema)).Find(results)
	expected := map[string]clause.Clause{}
	assert.Equal(t, expected, db.Statement.Clauses)
}

func TestFilterScopeWithJoinedUnsupportedDataType(t *testing.T) {
	db := openDryRunDB(t)
	filter := &Filter{Field: "Relation.name", Args: []string{"val1"}, Operator: Operators["$eq"]}

	results := []*FilterTestModelUnsupported{}
	schema, err := parseModel(db, &results)
	if !assert.Nil(t, err) {
		return
	}

	db.DryRun = true
	db = db.Model(&results).Scopes(filter.Scope(Blacklist{}, schema)).Find(&results)
	expected := map[string]clause.Clause{
		"FROM": {
			Name:       "FROM",
			Expression: clause.From{},
		},
		"SELECT": {
			Name:       "SELECT",
			Expression: clause.Select{},
		},
	}
	assert.Equal(t, expected, db.Statement.Clauses)
	assert.Nil(t, db.Error)
}
