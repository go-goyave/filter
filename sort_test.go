package filter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
	"gorm.io/gorm/utils/tests"
)

func TestSortScope(t *testing.T) {
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
	sort := &Sort{Field: "notacolumn", Order: SortAscending}
	schema := &schema.Schema{
		FieldsByDBName: map[string]*schema.Field{
			"name": {Name: "Name", DBName: "name"},
		},
		Table: "test_models",
	}

	assert.Nil(t, sort.Scope(&Settings{}, schema))

	sort.Field = "name"

	db = db.Scopes(sort.Scope(&Settings{}, schema)).Table("table").Find(nil)
	expected := map[string]clause.Clause{
		"ORDER BY": {
			Name: "ORDER BY",
			Expression: clause.OrderBy{
				Columns: []clause.OrderByColumn{
					{
						Column: clause.Column{
							Table: "test_models",
							Name:  "name",
						},
					},
				},
			},
		},
	}
	assert.Equal(t, expected, db.Statement.Clauses)

	sort.Order = SortDescending
	db, _ = gorm.Open(&tests.DummyDialector{}, nil)
	db = db.Scopes(sort.Scope(&Settings{}, schema)).Table("table").Find(nil)
	expected["ORDER BY"].Expression.(clause.OrderBy).Columns[0].Desc = true
	assert.Equal(t, expected, db.Statement.Clauses)
}

func TestSortScopeBlacklisted(t *testing.T) {
	sort := &Sort{Field: "name", Order: SortAscending}
	schema := &schema.Schema{
		FieldsByDBName: map[string]*schema.Field{
			"name": {Name: "Name"},
		},
		Table: "test_models",
	}
	assert.Nil(t, sort.Scope(&Settings{Blacklist: Blacklist{FieldsBlacklist: []string{"name"}}}, schema))
}

type SortTestNestedRelation struct {
	Field    string
	ID       uint
	ParentID uint
}

type SortTestRelation struct {
	NestedRelation *SortTestNestedRelation `gorm:"foreignKey:ParentID"`
	Name           string
	ID             uint
	ParentID       uint
}

type SortTestModel struct {
	Relation *SortTestRelation `gorm:"foreignKey:ParentID"`
	Name     string
	ID       uint
}

func TestSortScopeWithJoin(t *testing.T) {
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
	sort := &Sort{Field: "Relation.name", Order: SortAscending}

	results := []*SortTestModel{}
	schema, err := parseModel(db, &results)
	if !assert.Nil(t, err) {
		return
	}

	db = db.Model(&results).Scopes(sort.Scope(&Settings{}, schema)).Find(&results)
	expected := map[string]clause.Clause{
		"FROM": {
			Name: "FROM",
			Expression: clause.From{
				Joins: []clause.Join{
					{
						Type: clause.LeftJoin,
						Table: clause.Table{
							Name:  "sort_test_relations",
							Alias: "Relation",
						},
						ON: clause.Where{
							Exprs: []clause.Expression{
								clause.Eq{
									Column: clause.Column{
										Table: "sort_test_models",
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
		"ORDER BY": {
			Name: "ORDER BY",
			Expression: clause.OrderBy{
				Columns: []clause.OrderByColumn{
					{
						Column: clause.Column{
							Table: "Relation",
							Name:  "name",
						},
					},
				},
			},
		},
	}
	assert.Equal(t, expected, db.Statement.Clauses)
}

func TestSortScopeWithJoinInvalidModel(t *testing.T) {
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
	sort := &Sort{Field: "Relation.name", Order: SortDescending}

	results := []*SortTestModel{}
	sch, err := parseModel(db, &results)
	if !assert.Nil(t, err) {
		return
	}

	db = db.Scopes(sort.Scope(&Settings{}, sch)).Find(&results)
	assert.Equal(t, "unsupported data type: <nil>", db.Error.Error())
}

func TestSortScopeWithJoinNestedRelation(t *testing.T) {
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
	sort := &Sort{Field: "Relation.NestedRelation.field", Order: SortAscending}

	results := []*SortTestModel{}
	schema, err := parseModel(db, &results)
	if !assert.Nil(t, err) {
		return
	}

	db = db.Model(&results).Scopes(sort.Scope(&Settings{}, schema)).Find(&results)
	expected := map[string]clause.Clause{
		"FROM": {
			Name: "FROM",
			Expression: clause.From{
				Joins: []clause.Join{
					{
						Type: clause.LeftJoin,
						Table: clause.Table{
							Name:  "sort_test_relations",
							Alias: "Relation",
						},
						ON: clause.Where{
							Exprs: []clause.Expression{
								clause.Eq{
									Column: clause.Column{
										Table: "sort_test_models",
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
							Name:  "sort_test_nested_relations",
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
		"ORDER BY": {
			Name: "ORDER BY",
			Expression: clause.OrderBy{
				Columns: []clause.OrderByColumn{
					{
						Column: clause.Column{
							Table: "NestedRelation",
							Name:  "field",
						},
					},
				},
			},
		},
	}
	assert.Equal(t, expected, db.Statement.Clauses)
}
