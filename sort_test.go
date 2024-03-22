package filter

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

func TestSortScope(t *testing.T) {
	db := openDryRunDB(t)
	sort := &Sort{Field: "notacolumn", Order: SortAscending}
	field := &schema.Field{Name: "Name", DBName: "name", GORMDataType: schema.String}
	schema := &schema.Schema{
		FieldsByDBName: map[string]*schema.Field{
			"name": field,
		},
		FieldsByName: map[string]*schema.Field{
			"Name": field,
		},
		Table: "test_models",
	}

	assert.Nil(t, sort.Scope(Blacklist{}, schema, false))

	sort.Field = "name"

	results := []map[string]any{}
	db = db.Scopes(sort.Scope(Blacklist{}, schema, false)).Table("table").Find(&results)
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

	sort.Order = SortDescending
	results = []map[string]any{}
	db = openDryRunDB(t)
	db = db.Scopes(sort.Scope(Blacklist{}, schema, false)).Table("table").Find(&results)
	expected["ORDER BY"].Expression.(clause.OrderBy).Columns[0].Desc = true
	assert.Equal(t, expected, db.Statement.Clauses)

	// Using struct field name
	sort.Field = "Name"
	results = []map[string]any{}
	db = openDryRunDB(t)
	db = db.Scopes(sort.Scope(Blacklist{}, schema, false)).Table("table").Find(&results)
	assert.Equal(t, expected, db.Statement.Clauses)
}

func TestSortScopeBlacklisted(t *testing.T) {
	sort := &Sort{Field: "name", Order: SortAscending}
	schema := &schema.Schema{
		FieldsByDBName: map[string]*schema.Field{
			"name": {Name: "Name", GORMDataType: schema.String},
		},
		Table: "test_models",
	}
	assert.Nil(t, sort.Scope(Blacklist{FieldsBlacklist: []string{"name"}}, schema, false))
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
	db := openDryRunDB(t)
	sort := &Sort{Field: "Relation.name", Order: SortAscending}

	results := []*SortTestModel{}
	schema, err := parseModel(db, &results)
	if !assert.Nil(t, err) {
		return
	}

	db = db.Model(&results).Scopes(sort.Scope(Blacklist{}, schema, false)).Find(&results)
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
		"SELECT": {
			Name: "SELECT",
			Expression: clause.Select{
				Columns: []clause.Column{
					{Table: "sort_test_models", Name: "name"},
					{Table: "sort_test_models", Name: "id"},
				},
			},
		},
	}
	assert.Equal(t, expected, db.Statement.Clauses)
}

func TestSortScopeWithJoinInvalidModel(t *testing.T) {
	db := openDryRunDB(t)
	sort := &Sort{Field: "Relation.name", Order: SortDescending}

	results := []*SortTestModel{}
	sch, err := parseModel(db, &results)
	if !assert.Nil(t, err) {
		return
	}

	db = db.Scopes(sort.Scope(Blacklist{}, sch, false)).Find(&results)
	assert.Equal(t, "unsupported data type: <nil>", db.Error.Error())
}

func TestSortScopeWithJoinNestedRelation(t *testing.T) {
	db := openDryRunDB(t)
	sort := &Sort{Field: "Relation.NestedRelation.field", Order: SortAscending}

	results := []*SortTestModel{}
	schema, err := parseModel(db, &results)
	if !assert.Nil(t, err) {
		return
	}

	db = db.Model(&results).Scopes(sort.Scope(Blacklist{}, schema, false)).Find(&results)
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
		"SELECT": {
			Name: "SELECT",
			Expression: clause.Select{
				Columns: []clause.Column{
					{Table: "sort_test_models", Name: "name"},
					{Table: "sort_test_models", Name: "id"},
				},
			},
		},
	}
	assert.Equal(t, expected, db.Statement.Clauses)
}

type SortTestModelComputedRelation struct {
	Name     string
	Computed string `gorm:"->;-:migration" computed:"~~~ct~~~.computedcolumnrelation"`
	ID       uint
	ParentID uint
}

type SortTestModelComputed struct {
	Relation *SortTestModelComputedRelation `gorm:"foreignKey:ParentID"`
	Name     string
	Computed string `gorm:"->;-:migration" computed:"~~~ct~~~.computedcolumn"`
	ID       uint
}

func TestSortScopeComputed(t *testing.T) {
	db := openDryRunDB(t)
	sort := &Sort{Field: "computed", Order: SortAscending}

	results := []*SortTestModelComputed{}
	schema, err := parseModel(db, &results)
	if !assert.Nil(t, err) {
		return
	}

	db = db.Model(&results).Scopes(sort.Scope(Blacklist{}, schema, false)).Find(&results)
	expected := map[string]clause.Clause{
		"ORDER BY": {
			Name: "ORDER BY",
			Expression: clause.OrderBy{
				Columns: []clause.OrderByColumn{
					{
						Column: clause.Column{
							Raw:  true,
							Name: "(`sort_test_model_computeds`.computedcolumn)",
						},
					},
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

func TestSortScopeComputedWithJoin(t *testing.T) {
	db := openDryRunDB(t)
	sort := &Sort{Field: "Relation.computed", Order: SortAscending}

	results := []*SortTestModelComputed{}
	schema, err := parseModel(db, &results)
	if !assert.Nil(t, err) {
		return
	}

	db = db.Model(&results).Scopes(sort.Scope(Blacklist{}, schema, false)).Find(&results)
	expected := map[string]clause.Clause{
		"FROM": {
			Name: "FROM",
			Expression: clause.From{
				Joins: []clause.Join{
					{
						Type: clause.LeftJoin,
						Table: clause.Table{
							Name:  "sort_test_model_computed_relations",
							Alias: "Relation",
						},
						ON: clause.Where{
							Exprs: []clause.Expression{
								clause.Eq{
									Column: clause.Column{
										Table: "sort_test_model_computeds",
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
							Raw:  true,
							Name: "(`Relation`.computedcolumnrelation)",
						},
					},
				},
			},
		},
		"SELECT": {
			Name: "SELECT",
			Expression: clause.Select{
				Columns: []clause.Column{
					{Table: "sort_test_model_computeds", Name: "name"},
					{Table: "sort_test_model_computeds", Name: "computed"}, // Should not be problematic that it is added automatically by Gorm since we force only selectable fields all he time.
					{Table: "sort_test_model_computeds", Name: "id"},
				},
			},
		},
	}
	assert.Equal(t, expected, db.Statement.Clauses)
}

func TestSortScopeCaseInsensitive(t *testing.T) {
	db := openDryRunDB(t)
	sort := &Sort{Field: "notacolumn", Order: SortAscending}
	field := &schema.Field{Name: "Name", DBName: "name", GORMDataType: schema.String}
	schema := &schema.Schema{
		FieldsByDBName: map[string]*schema.Field{
			"name": field,
		},
		FieldsByName: map[string]*schema.Field{
			"Name": field,
		},
		Table: "test_models",
	}

	assert.Nil(t, sort.Scope(Blacklist{}, schema, false))

	sort.Field = "name"

	results := []map[string]any{}
	db = db.Scopes(sort.Scope(Blacklist{}, schema, true)).Table("table").Find(&results)
	expected := map[string]clause.Clause{
		"ORDER BY": {
			Name: "ORDER BY",
			Expression: clause.OrderBy{
				Columns: []clause.OrderByColumn{
					{
						Column: clause.Column{
							Raw:  true,
							Name: "LOWER(`test_models`.`name`)",
						},
					},
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

	sort.Order = SortDescending
	results = []map[string]any{}
	db = openDryRunDB(t)
	db = db.Scopes(sort.Scope(Blacklist{}, schema, true)).Table("table").Find(&results)
	expected["ORDER BY"].Expression.(clause.OrderBy).Columns[0].Desc = true
	assert.Equal(t, expected, db.Statement.Clauses)

	// Using struct field name
	sort.Field = "Name"
	results = []map[string]any{}
	db = openDryRunDB(t)
	db = db.Scopes(sort.Scope(Blacklist{}, schema, true)).Table("table").Find(&results)
	assert.Equal(t, expected, db.Statement.Clauses)
}

func TestSortScopeCaseInsensitiveNotString(t *testing.T) {
	db := openDryRunDB(t)
	sort := &Sort{Field: "notacolumn", Order: SortAscending}
	field := &schema.Field{Name: "ID", DBName: "id", GORMDataType: schema.Int}
	schema := &schema.Schema{
		FieldsByDBName: map[string]*schema.Field{
			"id": field,
		},
		FieldsByName: map[string]*schema.Field{
			"ID": field,
		},
		Table: "test_models",
	}

	assert.Nil(t, sort.Scope(Blacklist{}, schema, false))

	sort.Field = "id"

	results := []map[string]any{}
	db = db.Scopes(sort.Scope(Blacklist{}, schema, true)).Table("table").Find(&results)
	expected := map[string]clause.Clause{
		"ORDER BY": {
			Name: "ORDER BY",
			Expression: clause.OrderBy{
				Columns: []clause.OrderByColumn{
					{
						Column: clause.Column{
							Table: "test_models",
							Name:  "id",
						},
					},
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

	sort.Order = SortDescending
	results = []map[string]any{}
	db = openDryRunDB(t)
	db = db.Scopes(sort.Scope(Blacklist{}, schema, true)).Table("table").Find(&results)
	expected["ORDER BY"].Expression.(clause.OrderBy).Columns[0].Desc = true
	assert.Equal(t, expected, db.Statement.Clauses)

	// Using struct field name
	sort.Field = "ID"
	results = []map[string]any{}
	db = openDryRunDB(t)
	db = db.Scopes(sort.Scope(Blacklist{}, schema, true)).Table("table").Find(&results)
	assert.Equal(t, expected, db.Statement.Clauses)
}

func TestSortScopeCaseInsensitiveComputed(t *testing.T) {
	db := openDryRunDB(t)
	sort := &Sort{Field: "notacolumn", Order: SortAscending}
	field := &schema.Field{Name: "Name", DBName: "name", GORMDataType: schema.String, StructField: reflect.StructField{Tag: `computed:"UPPER(name)"`}}
	schema := &schema.Schema{
		FieldsByDBName: map[string]*schema.Field{
			"name": field,
		},
		FieldsByName: map[string]*schema.Field{
			"Name": field,
		},
		Table: "test_models",
	}

	assert.Nil(t, sort.Scope(Blacklist{}, schema, false))

	sort.Field = "name"

	results := []map[string]any{}
	db = db.Scopes(sort.Scope(Blacklist{}, schema, true)).Table("table").Find(&results)
	expected := map[string]clause.Clause{
		"ORDER BY": {
			Name: "ORDER BY",
			Expression: clause.OrderBy{
				Columns: []clause.OrderByColumn{
					{
						Column: clause.Column{
							Raw:  true,
							Name: "(UPPER(name))",
						},
					},
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

	sort.Order = SortDescending
	results = []map[string]any{}
	db = openDryRunDB(t)
	db = db.Scopes(sort.Scope(Blacklist{}, schema, true)).Table("table").Find(&results)
	expected["ORDER BY"].Expression.(clause.OrderBy).Columns[0].Desc = true
	assert.Equal(t, expected, db.Statement.Clauses)

	// Using struct field name
	sort.Field = "Name"
	results = []map[string]any{}
	db = openDryRunDB(t)
	db = db.Scopes(sort.Scope(Blacklist{}, schema, true)).Table("table").Find(&results)
	assert.Equal(t, expected, db.Statement.Clauses)
}
