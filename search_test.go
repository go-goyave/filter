package filter

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

func TestSearchScope(t *testing.T) {
	db, mock := openMockDB(t)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "table" WHERE "test_models"."name" LIKE ($1) OR "test_models"."email" LIKE ($2)`)).
		WithArgs("My Query", "My Query").
		WillReturnRows(sqlmock.NewRows([]string{"name", "id", "relation_id"}).AddRow("name", 123, 456))

	search := &Search{
		Fields: []string{"name", "email"},
		Query:  "My Query",
		Operator: &Operator{
			Function: func(tx *gorm.DB, filter *Filter, column string, _ DataType) *gorm.DB {
				return tx.Or(fmt.Sprintf("%s LIKE (?)", column), filter.Args[0])
			},
			RequiredArguments: 1,
		},
	}

	schema := &schema.Schema{
		FieldsByDBName: map[string]*schema.Field{
			"name":  {Name: "Name", DBName: "name", GORMDataType: schema.String},
			"email": {Name: "Email", DBName: "email", GORMDataType: schema.String},
			"role":  {Name: "Role", DBName: "role", GORMDataType: schema.String},
		},
		FieldsByName: map[string]*schema.Field{
			"Name":  {Name: "Name", DBName: "name", GORMDataType: schema.String},
			"Email": {Name: "Email", DBName: "email", GORMDataType: schema.String},
			"Role":  {Name: "Role", DBName: "role", GORMDataType: schema.String},
		},
		Table: "test_models",
	}

	db = db.Scopes(search.Scope(schema)).Table("table").Find(&map[string]any{})
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
										SQL:                `"test_models"."name" LIKE (?)`,
										Vars:               []any{"My Query"},
										WithoutParentheses: false,
									},
								},
							},
							clause.OrConditions{
								Exprs: []clause.Expression{
									clause.Expr{
										SQL:                `"test_models"."email" LIKE (?)`,
										Vars:               []any{"My Query"},
										WithoutParentheses: false,
									},
								},
							},
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

	// Using struct field names
	search.Fields = []string{"Name", "Email"}

	db, mock = openMockDB(t)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "table" WHERE "test_models"."name" LIKE ($1) OR "test_models"."email" LIKE ($2)`)).
		WithArgs("My Query", "My Query").
		WillReturnRows(sqlmock.NewRows([]string{"name", "id", "relation_id"}).AddRow("name", 123, 456))

	db = db.Scopes(search.Scope(schema)).Table("table").Find(&map[string]any{})
	assert.NoError(t, db.Error)
	assert.Equal(t, expected, db.Statement.Clauses)
}

func TestSearchScopeEmptyField(t *testing.T) {
	search := &Search{
		Fields: []string{},
		Query:  "My Query",
		Operator: &Operator{
			Function: func(tx *gorm.DB, filter *Filter, column string, _ DataType) *gorm.DB {
				return tx.Or(fmt.Sprintf("%s LIKE (?)", column), filter.Args[0])
			},
			RequiredArguments: 1,
		},
	}
	schema := &schema.Schema{
		FieldsByDBName: map[string]*schema.Field{
			"name":  {Name: "Name", GORMDataType: schema.String},
			"email": {Name: "Email", GORMDataType: schema.String},
			"role":  {Name: "Role", GORMDataType: schema.String},
		},
		Table: "test_models",
	}

	assert.Nil(t, search.Scope(schema))
}

type SearchTestNestedRelation struct {
	Field    string
	ID       uint
	ParentID uint
}

type SearchTestRelation struct {
	NestedRelation *SearchTestNestedRelation `gorm:"foreignKey:ParentID"`
	Name           string
	ID             uint
	ParentID       uint
}

type SearchTestModel struct {
	Relation *SearchTestRelation `gorm:"foreignKey:ParentID"`
	Name     string
	ID       uint
}

func TestSeachScopeWithJoin(t *testing.T) {
	db, mock := openMockDB(t)
	search := &Search{
		Fields: []string{"name", "Relation.name"},
		Query:  "My Query",
		Operator: &Operator{
			Function: func(tx *gorm.DB, filter *Filter, column string, _ DataType) *gorm.DB {
				return tx.Or(fmt.Sprintf("%s LIKE (?)", column), filter.Args[0])
			},
			RequiredArguments: 1,
		},
	}

	results := []*SearchTestModel{}
	schema, err := parseModel(db, &results)
	if !assert.Nil(t, err) {
		return
	}

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT "search_test_models"."name","search_test_models"."id" FROM "search_test_models" LEFT JOIN "search_test_relations" "Relation" ON "search_test_models"."id" = "Relation"."parent_id" WHERE "search_test_models"."name" LIKE ($1) OR "Relation"."name" LIKE ($2)`)).
		WithArgs("My Query", "My Query").
		WillReturnRows(sqlmock.NewRows([]string{"name", "id"}).AddRow("name", 123))

	db = db.Model(&results).Scopes(search.Scope(schema)).Find(&results)
	expected := map[string]clause.Clause{
		"FROM": {
			Name: "FROM",
			Expression: clause.From{
				Joins: []clause.Join{
					{
						Type: clause.LeftJoin,
						Table: clause.Table{
							Name:  "search_test_relations",
							Alias: "Relation",
						},
						ON: clause.Where{
							Exprs: []clause.Expression{
								clause.Eq{
									Column: clause.Column{
										Table: "search_test_models",
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
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.AndConditions{
						Exprs: []clause.Expression{
							clause.OrConditions{
								Exprs: []clause.Expression{
									clause.Expr{
										SQL:                `"search_test_models"."name" LIKE (?)`,
										Vars:               []any{"My Query"},
										WithoutParentheses: false,
									},
								},
							},
							clause.OrConditions{
								Exprs: []clause.Expression{
									clause.Expr{
										SQL:                `"Relation"."name" LIKE (?)`,
										Vars:               []any{"My Query"},
										WithoutParentheses: false,
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
					{Table: "search_test_models", Name: "name"},
					{Table: "search_test_models", Name: "id"},
				},
			},
		},
	}
	assert.NoError(t, db.Error)
	assert.Equal(t, expected, db.Statement.Clauses)
}

func TestSeachScopeWithJoinInvalidModel(t *testing.T) {
	db, _ := openMockDB(t)
	search := &Search{
		Fields:   []string{"name", "Relation.name"},
		Query:    "My Query",
		Operator: Operators["$eq"],
	}

	results := []*SearchTestModel{}
	sch, err := parseModel(db, &results)
	if !assert.Nil(t, err) {
		return
	}

	db = db.Scopes(search.Scope(sch)).Find(&results)
	assert.Equal(t, "unsupported data type: <nil>", db.Error.Error())
}

func TestSeachScopeWithJoinNestedRelation(t *testing.T) {
	db, mock := openMockDB(t)
	search := &Search{
		Fields: []string{"name", "Relation.NestedRelation.field"},
		Query:  "My Query",
		Operator: &Operator{
			Function: func(tx *gorm.DB, filter *Filter, column string, _ DataType) *gorm.DB {
				return tx.Or(fmt.Sprintf("%s LIKE (?)", column), filter.Args[0])
			},
			RequiredArguments: 1,
		},
	}

	results := []*SearchTestModel{}
	schema, err := parseModel(db, &results)
	if !assert.Nil(t, err) {
		return
	}

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT "search_test_models"."name","search_test_models"."id" FROM "search_test_models" LEFT JOIN "search_test_relations" "Relation" ON "search_test_models"."id" = "Relation"."parent_id" LEFT JOIN "search_test_nested_relations" "NestedRelation" ON "Relation"."id" = "NestedRelation"."parent_id" WHERE "search_test_models"."name" LIKE ($1) OR "NestedRelation"."field" LIKE ($2)`)).
		WithArgs("My Query", "My Query").
		WillReturnRows(sqlmock.NewRows([]string{"name", "id", "relation_id"}).AddRow("name", 123, 456))

	db = db.Model(&results).Scopes(search.Scope(schema)).Find(&results)
	expected := map[string]clause.Clause{
		"FROM": {
			Name: "FROM",
			Expression: clause.From{
				Joins: []clause.Join{
					{
						Type: clause.LeftJoin,
						Table: clause.Table{
							Name:  "search_test_relations",
							Alias: "Relation",
						},
						ON: clause.Where{
							Exprs: []clause.Expression{
								clause.Eq{
									Column: clause.Column{
										Table: "search_test_models",
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
							Name:  "search_test_nested_relations",
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
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.AndConditions{
						Exprs: []clause.Expression{
							clause.OrConditions{
								Exprs: []clause.Expression{
									clause.Expr{
										SQL:                `"search_test_models"."name" LIKE (?)`,
										Vars:               []any{"My Query"},
										WithoutParentheses: false,
									},
								},
							},
							clause.OrConditions{
								Exprs: []clause.Expression{
									clause.Expr{
										SQL:                `"NestedRelation"."field" LIKE (?)`,
										Vars:               []any{"My Query"},
										WithoutParentheses: false,
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
					{Table: "search_test_models", Name: "name"},
					{Table: "search_test_models", Name: "id"},
				},
			},
		},
	}
	assert.NoError(t, db.Error)
	assert.Equal(t, expected, db.Statement.Clauses)
}

type SearchTestModelComputedRelation struct {
	Name     string
	Computed string `computed:"~~~ct~~~.computedcolumnrelation"`
	ID       uint
	ParentID uint
}

type SearchTestModelComputed struct {
	Relation *SearchTestModelComputedRelation `gorm:"foreignKey:ParentID"`
	Name     string
	Computed string `computed:"~~~ct~~~.computedcolumn"`
	ID       uint
}

func TestSearchScopeComputed(t *testing.T) {
	db, mock := openMockDB(t)
	search := &Search{
		Fields:   []string{"computed", "Relation.computed"},
		Query:    "My Query",
		Operator: Operators["$eq"],
	}

	results := []*SearchTestModelComputed{}
	schema, err := parseModel(db, &results)
	if !assert.Nil(t, err) {
		return
	}

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT "search_test_model_computeds"."name","search_test_model_computeds"."computed","search_test_model_computeds"."id" FROM "search_test_model_computeds" LEFT JOIN "search_test_model_computed_relations" "Relation" ON "search_test_model_computeds"."id" = "Relation"."parent_id" WHERE ("search_test_model_computeds".computedcolumn) = $1 OR ("Relation".computedcolumnrelation) = $2`)).
		WithArgs("My Query", "My Query").
		WillReturnRows(sqlmock.NewRows([]string{"name", "id"}).AddRow("name", 123))

	db = db.Model(&results).Scopes(search.Scope(schema)).Find(&results)
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
										SQL:                `("search_test_model_computeds".computedcolumn) = ?`,
										Vars:               []any{"My Query"},
										WithoutParentheses: false,
									},
								},
							},
							clause.OrConditions{
								Exprs: []clause.Expression{
									clause.Expr{
										SQL:                `("Relation".computedcolumnrelation) = ?`,
										Vars:               []any{"My Query"},
										WithoutParentheses: false,
									},
								},
							},
						},
					},
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
							Name:  "search_test_model_computed_relations",
							Alias: "Relation",
						},
						ON: clause.Where{
							Exprs: []clause.Expression{
								clause.Eq{
									Column: clause.Column{
										Table: "search_test_model_computeds",
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
					{Table: "search_test_model_computeds", Name: "name"},
					{Table: "search_test_model_computeds", Name: "computed"}, // Should not be problematic that it is added automatically by Gorm since we force only selectable fields all he time.
					{Table: "search_test_model_computeds", Name: "id"},
				},
			},
		},
	}
	assert.NoError(t, db.Error)
	assert.Equal(t, expected, db.Statement.Clauses)
}

func TestSearchScopeWithUnsupportedDataType(t *testing.T) {
	db, mock := openMockDB(t)
	search := &Search{
		Fields: []string{"name", "email"},
		Query:  "My Query",
		Operator: &Operator{
			Function: func(tx *gorm.DB, filter *Filter, column string, _ DataType) *gorm.DB {
				return tx.Or(fmt.Sprintf("%s LIKE (?)", column), filter.Args[0])
			},
			RequiredArguments: 1,
		},
	}

	schema := &schema.Schema{
		FieldsByDBName: map[string]*schema.Field{
			"name":  {Name: "Name", DBName: "name", GORMDataType: schema.String},
			"email": {Name: "Email", DBName: "email", GORMDataType: "custom", DataType: "CHARACTER VARYING(255)"},
			"role":  {Name: "Role", DBName: "role", GORMDataType: schema.String},
		},
		Table: "test_models",
	}

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "table" WHERE "test_models"."name" LIKE ($1)`)).
		WithArgs("My Query").
		WillReturnRows(sqlmock.NewRows([]string{"name", "email"}).AddRow("name", "email@example.org"))

	db = db.Scopes(search.Scope(schema)).Table("table").Find(&map[string]any{})
	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.AndConditions{
						Exprs: []clause.Expression{
							clause.Expr{
								SQL:                `"test_models"."name" LIKE (?)`,
								Vars:               []any{"My Query"},
								WithoutParentheses: false,
							},
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
