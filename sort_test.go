package filter

import (
	"reflect"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

func TestSortScope(t *testing.T) {
	db, mock := openMockDB(t)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "table" ORDER BY "test_models"."name"`)).
		WithoutArgs().
		WillReturnRows(sqlmock.NewRows([]string{"name", "id"}).AddRow("name", 123))

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

	assert.Nil(t, sort.Scope(Whitelist{Fields: AllowAllFields}, schema, false))

	sort.Field = "name"

	results := []map[string]any{}
	db = db.Scopes(sort.Scope(Whitelist{Fields: AllowAllFields}, schema, false)).Table("table").Find(&results)
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
		"FROM": expectFromClause,
		"SELECT": {
			Name:       "SELECT",
			Expression: clause.Select{},
		},
	}
	assert.NoError(t, db.Error)
	assert.Equal(t, expected, db.Statement.Clauses)

	sort.Order = SortDescending
	results = []map[string]any{}
	db, mock = openMockDB(t)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "table" ORDER BY "test_models"."name" DESC`)).
		WithoutArgs().
		WillReturnRows(sqlmock.NewRows([]string{"name", "id"}).AddRow("name", 123))

	db = db.Scopes(sort.Scope(Whitelist{Fields: AllowAllFields}, schema, false)).Table("table").Find(&results)
	assert.NoError(t, db.Error)
	expected["ORDER BY"].Expression.(clause.OrderBy).Columns[0].Desc = true
	assert.Equal(t, expected, db.Statement.Clauses)

	// Using struct field name
	sort.Field = "Name"
	results = []map[string]any{}
	db, mock = openMockDB(t)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "table" ORDER BY "test_models"."name" DESC`)).
		WithoutArgs().
		WillReturnRows(sqlmock.NewRows([]string{"name", "id"}).AddRow("name", 123))

	db = db.Scopes(sort.Scope(Whitelist{Fields: AllowAllFields}, schema, false)).Table("table").Find(&results)
	assert.NoError(t, db.Error)
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
	assert.Nil(t, sort.Scope(Whitelist{Fields: []string{}}, schema, false))
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
	db, mock := openMockDB(t)
	sort := &Sort{Field: "Relation.name", Order: SortAscending}

	results := []*SortTestModel{}
	schema, err := parseModel(db, &results)
	require.NoError(t, err)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT "sort_test_models"."name","sort_test_models"."id" FROM "sort_test_models" LEFT JOIN "sort_test_relations" "Relation" ON "sort_test_models"."id" = "Relation"."parent_id" ORDER BY "Relation"."name"`)).
		WithoutArgs().
		WillReturnRows(sqlmock.NewRows([]string{"name", "id"}).AddRow("name", 123))

	whitelist := Whitelist{
		Relations: map[string]Whitelist{
			"Relation": {Fields: []string{"Name"}},
		},
		Fields: AllowAllFields,
	}

	db = db.Model(&results).Scopes(sort.Scope(whitelist, schema, false)).Find(&results)
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
	assert.NoError(t, db.Error)
	assert.Equal(t, expected, db.Statement.Clauses)
}

func TestSortScopeWithJoinInvalidModel(t *testing.T) {
	db, _ := openMockDB(t)
	sort := &Sort{Field: "Relation.name", Order: SortDescending}

	results := []*SortTestModel{}
	sch, err := parseModel(db, &results)
	require.NoError(t, err)

	whitelist := Whitelist{
		Relations: map[string]Whitelist{
			"Relation": {Fields: AllowAllFields},
		},
		Fields: AllowAllFields,
	}

	db = db.Scopes(sort.Scope(whitelist, sch, false)).Find(&results)
	assert.Equal(t, "unsupported data type: <nil>", db.Error.Error())
}

func TestSortScopeWithJoinNestedRelation(t *testing.T) {
	db, mock := openMockDB(t)
	sort := &Sort{Field: "Relation.NestedRelation.field", Order: SortAscending}

	results := []*SortTestModel{}
	schema, err := parseModel(db, &results)
	require.NoError(t, err)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT "sort_test_models"."name","sort_test_models"."id" FROM "sort_test_models" LEFT JOIN "sort_test_relations" "Relation" ON "sort_test_models"."id" = "Relation"."parent_id" LEFT JOIN "sort_test_nested_relations" "NestedRelation" ON "Relation"."id" = "NestedRelation"."parent_id" ORDER BY "NestedRelation"."field"`)).
		WithoutArgs().
		WillReturnRows(sqlmock.NewRows([]string{"name", "id"}).AddRow("name", 123))

	whitelist := Whitelist{
		Relations: map[string]Whitelist{
			"Relation": {
				Relations: map[string]Whitelist{
					"NestedRelation": {
						Fields: AllowAllFields,
					},
				},
				Fields: AllowAllFields,
			},
		},
		Fields: AllowAllFields,
	}

	db = db.Model(&results).Scopes(sort.Scope(whitelist, schema, false)).Find(&results)
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
	assert.NoError(t, db.Error)
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
	db, mock := openMockDB(t)
	sort := &Sort{Field: "computed", Order: SortAscending}

	results := []*SortTestModelComputed{}
	schema, err := parseModel(db, &results)
	require.NoError(t, err)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "sort_test_model_computeds" ORDER BY ("sort_test_model_computeds".computedcolumn)`)).
		WithoutArgs().
		WillReturnRows(sqlmock.NewRows([]string{"name", "id"}).AddRow("name", 123))

	db = db.Model(&results).Scopes(sort.Scope(Whitelist{Fields: AllowAllFields}, schema, false)).Find(&results)
	expected := map[string]clause.Clause{
		"ORDER BY": {
			Name: "ORDER BY",
			Expression: clause.OrderBy{
				Columns: []clause.OrderByColumn{
					{
						Column: clause.Column{
							Raw:  true,
							Name: `("sort_test_model_computeds".computedcolumn)`,
						},
					},
				},
			},
		},
		"FROM": expectFromClause,
		"SELECT": {
			Name:       "SELECT",
			Expression: clause.Select{},
		},
	}
	assert.NoError(t, db.Error)
	assert.Equal(t, expected, db.Statement.Clauses)
}

func TestSortScopeComputedWithJoin(t *testing.T) {
	db, mock := openMockDB(t)
	sort := &Sort{Field: "Relation.computed", Order: SortAscending}

	results := []*SortTestModelComputed{}
	schema, err := parseModel(db, &results)
	require.NoError(t, err)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT "sort_test_model_computeds"."name","sort_test_model_computeds"."computed","sort_test_model_computeds"."id" FROM "sort_test_model_computeds" LEFT JOIN "sort_test_model_computed_relations" "Relation" ON "sort_test_model_computeds"."id" = "Relation"."parent_id" ORDER BY ("Relation".computedcolumnrelation)`)).
		WithoutArgs().
		WillReturnRows(sqlmock.NewRows([]string{"name", "id"}).AddRow("name", 123))

	whitelist := Whitelist{
		Relations: map[string]Whitelist{
			"Relation": {Fields: AllowAllFields},
		},
		Fields: AllowAllFields,
	}

	db = db.Model(&results).Scopes(sort.Scope(whitelist, schema, false)).Find(&results)
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
							Name: `("Relation".computedcolumnrelation)`,
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
	assert.NoError(t, db.Error)
	assert.Equal(t, expected, db.Statement.Clauses)
}

func TestSortScopeCaseInsensitive(t *testing.T) {
	db, mock := openMockDB(t)
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

	assert.Nil(t, sort.Scope(Whitelist{Fields: AllowAllFields}, schema, false))

	sort.Field = "name"

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "table" ORDER BY LOWER("test_models"."name")`)).
		WithoutArgs().
		WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("name"))

	results := []map[string]any{}
	db = db.Scopes(sort.Scope(Whitelist{Fields: AllowAllFields}, schema, true)).Table("table").Find(&results)
	expected := map[string]clause.Clause{
		"ORDER BY": {
			Name: "ORDER BY",
			Expression: clause.OrderBy{
				Columns: []clause.OrderByColumn{
					{
						Column: clause.Column{
							Raw:  true,
							Name: `LOWER("test_models"."name")`,
						},
					},
				},
			},
		},
		"FROM": expectFromClause,
		"SELECT": {
			Name:       "SELECT",
			Expression: clause.Select{},
		},
	}
	assert.NoError(t, db.Error)
	assert.Equal(t, expected, db.Statement.Clauses)

	sort.Order = SortDescending
	results = []map[string]any{}
	db, mock = openMockDB(t)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "table" ORDER BY LOWER("test_models"."name") DESC`)).
		WithoutArgs().
		WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("name"))

	db = db.Scopes(sort.Scope(Whitelist{Fields: AllowAllFields}, schema, true)).Table("table").Find(&results)
	assert.NoError(t, db.Error)
	expected["ORDER BY"].Expression.(clause.OrderBy).Columns[0].Desc = true
	assert.Equal(t, expected, db.Statement.Clauses)

	// Using struct field name
	sort.Field = "Name"
	results = []map[string]any{}
	db, mock = openMockDB(t)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "table" ORDER BY LOWER("test_models"."name") DESC`)).
		WithoutArgs().
		WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("name"))

	db = db.Scopes(sort.Scope(Whitelist{Fields: AllowAllFields}, schema, true)).Table("table").Find(&results)
	assert.NoError(t, db.Error)
	assert.Equal(t, expected, db.Statement.Clauses)
}

func TestSortScopeCaseInsensitiveNotString(t *testing.T) {
	db, mock := openMockDB(t)
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

	assert.Nil(t, sort.Scope(Whitelist{Fields: AllowAllFields}, schema, false))

	sort.Field = "id"

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "table" ORDER BY "test_models"."id"`)).
		WithoutArgs().
		WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("name"))

	results := []map[string]any{}
	db = db.Scopes(sort.Scope(Whitelist{Fields: AllowAllFields}, schema, true)).Table("table").Find(&results)
	assert.NoError(t, db.Error)
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
		"FROM": expectFromClause,
		"SELECT": {
			Name:       "SELECT",
			Expression: clause.Select{},
		},
	}
	assert.Equal(t, expected, db.Statement.Clauses)

	sort.Order = SortDescending
	results = []map[string]any{}
	db, mock = openMockDB(t)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "table" ORDER BY "test_models"."id" DESC`)).
		WithoutArgs().
		WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("name"))

	db = db.Scopes(sort.Scope(Whitelist{Fields: AllowAllFields}, schema, true)).Table("table").Find(&results)
	assert.NoError(t, db.Error)
	expected["ORDER BY"].Expression.(clause.OrderBy).Columns[0].Desc = true
	assert.Equal(t, expected, db.Statement.Clauses)

	// Using struct field name
	sort.Field = "ID"
	results = []map[string]any{}
	db, mock = openMockDB(t)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "table" ORDER BY "test_models"."id" DESC`)).
		WithoutArgs().
		WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("name"))

	db = db.Scopes(sort.Scope(Whitelist{Fields: AllowAllFields}, schema, true)).Table("table").Find(&results)
	assert.NoError(t, db.Error)
	assert.Equal(t, expected, db.Statement.Clauses)
}

func TestSortScopeCaseInsensitiveComputed(t *testing.T) {
	db, mock := openMockDB(t)
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

	assert.Nil(t, sort.Scope(Whitelist{Fields: AllowAllFields}, schema, false))

	sort.Field = "name"

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "table" ORDER BY (UPPER(name))`)).
		WithoutArgs().
		WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("name"))

	results := []map[string]any{}
	db = db.Scopes(sort.Scope(Whitelist{Fields: AllowAllFields}, schema, true)).Table("table").Find(&results)
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
		"FROM": expectFromClause,
		"SELECT": {
			Name:       "SELECT",
			Expression: clause.Select{},
		},
	}
	assert.NoError(t, db.Error)
	assert.Equal(t, expected, db.Statement.Clauses)

	sort.Order = SortDescending
	results = []map[string]any{}
	db, mock = openMockDB(t)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "table" ORDER BY (UPPER(name)) DESC`)).
		WithoutArgs().
		WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("name"))

	db = db.Scopes(sort.Scope(Whitelist{Fields: AllowAllFields}, schema, true)).Table("table").Find(&results)
	assert.NoError(t, db.Error)
	expected["ORDER BY"].Expression.(clause.OrderBy).Columns[0].Desc = true
	assert.Equal(t, expected, db.Statement.Clauses)

	// Using struct field name
	sort.Field = "Name"
	results = []map[string]any{}
	db, mock = openMockDB(t)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "table" ORDER BY (UPPER(name)) DESC`)).
		WithoutArgs().
		WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("name"))

	db = db.Scopes(sort.Scope(Whitelist{Fields: AllowAllFields}, schema, true)).Table("table").Find(&results)
	assert.NoError(t, db.Error)
	assert.Equal(t, expected, db.Statement.Clauses)
}

func TestSortScopeUsingRelation(t *testing.T) {
	// When giving a relation name, no sort should be generated
	sort := &Sort{Field: "Relation", Order: SortAscending}
	schema := &schema.Schema{
		FieldsByName: map[string]*schema.Field{
			"Relation": {Name: "Relation"}, // leave DBName empty to simulate a relation
		},
		Table: "test_models",
	}
	assert.Nil(t, sort.Scope(Whitelist{Fields: AllowAllFields}, schema, false))
}
