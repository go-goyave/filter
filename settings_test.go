package filter

import (
	"fmt"
	"reflect"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
	"goyave.dev/goyave/v6/database"

	"goyave.dev/goyave/v6/util/typeutil"
)

type TestScopeRelation struct {
	A  string
	B  string
	ID uint
}
type TestScopeModel struct {
	Relation   *TestScopeRelation
	Name       string
	Email      string
	Computed   string `gorm:"->;-:migration" computed:"UPPER(~~~ct~~~.name)"`
	Extra      string
	ID         uint
	RelationID uint
}

type TestScopeModelNoPrimaryKey struct {
	Relation   *TestScopeRelation
	Name       string
	RelationID uint
}

var (
	testRequest = &Request{
		Filter: typeutil.NewUndefined([]*Filter{
			{Field: "name", Args: []string{"val1"}, Operator: Operators["$cont"]},
			{Field: "name", Args: []string{"val2"}, Operator: Operators["$cont"]},
		}),
		Or: typeutil.NewUndefined([]*Filter{
			{Field: "name", Args: []string{"val3"}, Or: true, Operator: Operators["$eq"]},
		}),
		Sort:    typeutil.NewUndefined([]*Sort{{Field: "name", Order: SortAscending}}),
		Join:    typeutil.NewUndefined([]*Join{{Relation: "Relation", Fields: []string{"a", "b"}}}),
		Page:    typeutil.NewUndefined(2),
		PerPage: typeutil.NewUndefined(15),
		Fields:  typeutil.NewUndefined([]string{"id", "name", "email", "computed"}),
		Search:  typeutil.NewUndefined("val"),
	}

	testSettings = &Settings[*TestScopeModel]{
		FieldsSearch: []string{"email"},
		Whitelist: Whitelist{
			Fields: AllowAllFields,
			Relations: map[string]Whitelist{
				"Relation": {Fields: AllowAllFields},
			},
		},
		SearchOperator: &Operator{
			Function: func(tx *gorm.DB, filter *Filter, column string, _ DataType) *gorm.DB {
				return tx.Or(fmt.Sprintf("%s LIKE (?)", column), filter.Args[0])
			},
			RequiredArguments: 1,
		},
	}

	expectWhereClause = clause.Clause{
		Name: "WHERE",
		Expression: clause.Where{
			Exprs: []clause.Expression{
				clause.AndConditions{
					Exprs: []clause.Expression{
						clause.OrConditions{
							Exprs: []clause.Expression{
								clause.AndConditions{
									Exprs: []clause.Expression{
										clause.Expr{SQL: `"test_scope_models"."name" LIKE ?`, Vars: []any{"%val1%"}},
										clause.Expr{SQL: `"test_scope_models"."name" LIKE ?`, Vars: []any{"%val2%"}},
									},
								},
							},
						},
						clause.OrConditions{
							Exprs: []clause.Expression{
								clause.AndConditions{
									Exprs: []clause.Expression{
										clause.Expr{SQL: `"test_scope_models"."name" = ?`, Vars: []any{"val3"}},
									},
								},
							},
						},
					},
				},
				clause.AndConditions{
					Exprs: []clause.Expression{
						clause.Expr{
							SQL:                `"test_scope_models"."email" LIKE ?`,
							Vars:               []any{"%val%"},
							WithoutParentheses: false,
						},
					},
				},
			},
		},
	}
	expectOrderByClause = clause.Clause{
		Name: "ORDER BY",
		Expression: clause.OrderBy{
			Columns: []clause.OrderByColumn{
				{
					Column: clause.Column{
						Table: "test_scope_models",
						Name:  "name",
					},
				},
			},
		},
	}
	expectSelectClause = clause.Clause{
		Name: "SELECT",
		Expression: clause.Select{
			Columns: []clause.Column{
				{Raw: true, Name: `"test_scope_models"."name"`},
				{Raw: true, Name: `"test_scope_models"."email"`},
				{Raw: true, Name: `(UPPER("test_scope_models".name)) "computed"`},
				{Raw: true, Name: `"test_scope_models"."id"`},
				{Raw: true, Name: `"test_scope_models"."relation_id"`},
			},
		},
	}
	expectFromClause = clause.Clause{
		Name:       "FROM",
		Expression: clause.From{},
	}
	expectSelects = []string{`"test_scope_models"."name"`, `"test_scope_models"."email"`, `(UPPER("test_scope_models".name)) "computed"`, `"test_scope_models"."id"`, `"test_scope_models"."relation_id"`}
)

func openMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)

	dialector := &postgres.Dialector{
		Config: &postgres.Config{
			Conn: mockDB,
		},
	}

	t.Cleanup(func() {
		mock.ExpectClose()
		assert.NoError(t, mockDB.Close())
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	db, err := gorm.Open(dialector)
	db.Logger = db.Logger.LogMode(logger.Silent)
	require.NoError(t, err)
	return db, mock
}

func runScope(t *testing.T, settings *Settings[*TestScopeModel], request *Request, sqlExpect func(mock sqlmock.Sqlmock)) (*database.Paginator[*TestScopeModel], error) {
	db, mock := openMockDB(t)

	sqlExpect(mock)

	results := []*TestScopeModel{}
	if settings == nil {
		return Scope(db, request, &results)
	}

	return settings.Scope(db, request, &results)
}

func runScopeUnpaginated(t *testing.T, settings *Settings[*TestScopeModel], request *Request, sqlExpect func(mock sqlmock.Sqlmock)) ([]*TestScopeModel, *gorm.DB) {
	db, mock := openMockDB(t)

	sqlExpect(mock)

	results := []*TestScopeModel{}
	if settings == nil {
		return results, ScopeUnpaginated(db, request, &results)
	}

	return results, settings.ScopeUnpaginated(db, request, &results)
}

func TestScope(t *testing.T) {
	settings := testSettings

	sqlExpect := func(mock sqlmock.Sqlmock) {
		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "test_scope_models" WHERE (("test_scope_models"."name" LIKE $1 AND "test_scope_models"."name" LIKE $2) OR "test_scope_models"."name" = $3) AND "test_scope_models"."email" LIKE ($4)`)).
			WithArgs("%val1%", "%val2%", "val3", "val").
			WillReturnRows(sqlmock.NewRows([]string{"count(*)"}).AddRow(19))
		mock.ExpectExec(`SAVEPOINT .+`).WithoutArgs().WillReturnResult(sqlmock.NewResult(0, 0))
		// Actual models
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT "test_scope_models"."name","test_scope_models"."email",(UPPER("test_scope_models".name)) "computed","test_scope_models"."id","test_scope_models"."relation_id" FROM "test_scope_models" WHERE (("test_scope_models"."name" LIKE $1 AND "test_scope_models"."name" LIKE $2) OR "test_scope_models"."name" = $3) AND "test_scope_models"."email" LIKE ($4) ORDER BY "test_scope_models"."name" LIMIT $5 OFFSET $6`)).
			WithArgs("%val1%", "%val2%", "val3", "val", 15, 15).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email", "computed", "relation_id"}).AddRow(123, "name", "email@example.org", "NAME", 456))
		// Preload relation
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT "test_scope_relations"."a","test_scope_relations"."b","test_scope_relations"."id" FROM "test_scope_relations" WHERE "test_scope_relations"."id" = $1`)).
			WithArgs(456).
			WillReturnRows(sqlmock.NewRows([]string{"a", "b", "id"}).AddRow("a", "b", 456))
		mock.ExpectCommit()
	}

	paginator, err := runScope(t, settings, testRequest, sqlExpect)
	assert.NotNil(t, paginator)
	assert.NoError(t, err)

	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.AndConditions{
						Exprs: []clause.Expression{
							clause.OrConditions{
								Exprs: []clause.Expression{
									clause.AndConditions{
										Exprs: []clause.Expression{
											clause.Expr{SQL: `"test_scope_models"."name" LIKE ?`, Vars: []any{"%val1%"}},
											clause.Expr{SQL: `"test_scope_models"."name" LIKE ?`, Vars: []any{"%val2%"}},
										},
									},
								},
							},
							clause.OrConditions{
								Exprs: []clause.Expression{
									clause.AndConditions{
										Exprs: []clause.Expression{
											clause.Expr{SQL: `"test_scope_models"."name" = ?`, Vars: []any{"val3"}},
										},
									},
								},
							},
						},
					},
					clause.AndConditions{
						Exprs: []clause.Expression{
							clause.Expr{
								SQL:                `"test_scope_models"."email" LIKE (?)`, // Search operator override
								Vars:               []any{"val"},
								WithoutParentheses: false,
							},
						},
					},
				},
			},
		},
		"ORDER BY": expectOrderByClause,
		"LIMIT": {
			Expression: clause.Limit{
				Limit:  new(15),
				Offset: 15,
			},
		},
		"FROM":   expectFromClause,
		"SELECT": expectSelectClause,
	}
	assert.Equal(t, expected, paginator.DB.Statement.Clauses)
	assert.Contains(t, paginator.DB.Statement.Preloads, "Relation")
	assert.Equal(t, expectSelects, paginator.DB.Statement.Selects)
}

func TestScopeUnpaginated(t *testing.T) {
	settings := testSettings

	sqlExpect := func(mock sqlmock.Sqlmock) {
		// No Count query, no transaction
		// Actual models
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT "test_scope_models"."name","test_scope_models"."email",(UPPER("test_scope_models".name)) "computed","test_scope_models"."id","test_scope_models"."relation_id" FROM "test_scope_models" WHERE (("test_scope_models"."name" LIKE $1 AND "test_scope_models"."name" LIKE $2) OR "test_scope_models"."name" = $3) AND "test_scope_models"."email" LIKE ($4) ORDER BY "test_scope_models"."name"`)).
			WithArgs("%val1%", "%val2%", "val3", "val").
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email", "computed", "relation_id"}).AddRow(123, "name", "email@example.org", "NAME", 456))
		// Preload relation
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT "test_scope_relations"."a","test_scope_relations"."b","test_scope_relations"."id" FROM "test_scope_relations" WHERE "test_scope_relations"."id" = $1`)).
			WithArgs(456).
			WillReturnRows(sqlmock.NewRows([]string{"a", "b", "id"}).AddRow("a", "b", 456))
	}

	results, db := runScopeUnpaginated(t, settings, testRequest, sqlExpect)
	assert.NotNil(t, results)
	assert.NoError(t, db.Error)

	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.AndConditions{
						Exprs: []clause.Expression{
							clause.OrConditions{
								Exprs: []clause.Expression{
									clause.AndConditions{
										Exprs: []clause.Expression{
											clause.Expr{SQL: `"test_scope_models"."name" LIKE ?`, Vars: []any{"%val1%"}},
											clause.Expr{SQL: `"test_scope_models"."name" LIKE ?`, Vars: []any{"%val2%"}},
										},
									},
								},
							},
							clause.OrConditions{
								Exprs: []clause.Expression{
									clause.AndConditions{
										Exprs: []clause.Expression{
											clause.Expr{SQL: `"test_scope_models"."name" = ?`, Vars: []any{"val3"}},
										},
									},
								},
							},
						},
					},
					clause.AndConditions{
						Exprs: []clause.Expression{
							clause.Expr{
								SQL:                `"test_scope_models"."email" LIKE (?)`, // Search operator override
								Vars:               []any{"val"},
								WithoutParentheses: false,
							},
						},
					},
				},
			},
		},
		"ORDER BY": expectOrderByClause,
		"FROM":     expectFromClause,
		"SELECT":   expectSelectClause,
	}
	assert.Equal(t, expected, db.Statement.Clauses)
	assert.Contains(t, db.Statement.Preloads, "Relation")
	assert.Equal(t, expectSelects, db.Statement.Selects)
}

func TestScopeDisableFields(t *testing.T) {
	settings := &Settings[*TestScopeModel]{
		DisableFields: true, // With "fields" disabled, all whitelisted columns are expected to be selected
		FieldsSearch:  []string{"email"},
		Whitelist: Whitelist{
			Fields: AllowAllFields,
			Relations: map[string]Whitelist{
				"Relation": {Fields: AllowAllFields},
			},
		},
	}

	sqlExpect := func(mock sqlmock.Sqlmock) {
		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "test_scope_models" WHERE (("test_scope_models"."name" LIKE $1 AND "test_scope_models"."name" LIKE $2) OR "test_scope_models"."name" = $3) AND "test_scope_models"."email" LIKE $4`)).
			WithArgs("%val1%", "%val2%", "val3", "%val%").
			WillReturnRows(sqlmock.NewRows([]string{"count(*)"}).AddRow(19))
		mock.ExpectExec(`SAVEPOINT .+`).WithoutArgs().WillReturnResult(sqlmock.NewResult(0, 0))
		// Actual models
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT "test_scope_models"."name","test_scope_models"."email",(UPPER("test_scope_models".name)) "computed","test_scope_models"."extra","test_scope_models"."id","test_scope_models"."relation_id" FROM "test_scope_models" WHERE (("test_scope_models"."name" LIKE $1 AND "test_scope_models"."name" LIKE $2) OR "test_scope_models"."name" = $3) AND "test_scope_models"."email" LIKE $4 ORDER BY "test_scope_models"."name" LIMIT $5 OFFSET $6`)).
			WithArgs("%val1%", "%val2%", "val3", "%val%", 15, 15).
			WillReturnRows(sqlmock.NewRows([]string{"name", "email", "computed", "extra", "id", "relation_id"}).AddRow("name", "email@example.org", "NAME", "extra", 123, 456))
		// Preload relation
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT "test_scope_relations"."a","test_scope_relations"."b","test_scope_relations"."id" FROM "test_scope_relations" WHERE "test_scope_relations"."id" = $1`)).
			WithArgs(456).
			WillReturnRows(sqlmock.NewRows([]string{"a", "b", "id"}).AddRow("a", "b", 456))
		mock.ExpectCommit()
	}

	paginator, err := runScope(t, settings, testRequest, sqlExpect)
	assert.NotNil(t, paginator)
	assert.NoError(t, err)

	expected := map[string]clause.Clause{
		"WHERE":    expectWhereClause,
		"ORDER BY": expectOrderByClause,
		"LIMIT": {
			Expression: clause.Limit{
				Limit:  new(15),
				Offset: 15,
			},
		},
		"FROM": expectFromClause,
		"SELECT": {
			Name: "SELECT",
			Expression: clause.Select{
				Columns: []clause.Column{
					{Raw: true, Name: `"test_scope_models"."name"`},
					{Raw: true, Name: `"test_scope_models"."email"`},
					{Raw: true, Name: `(UPPER("test_scope_models".name)) "computed"`},
					{Raw: true, Name: `"test_scope_models"."extra"`},
					{Raw: true, Name: `"test_scope_models"."id"`},
					{Raw: true, Name: `"test_scope_models"."relation_id"`},
				},
			},
		},
	}
	assert.Equal(t, expected, paginator.DB.Statement.Clauses)
	assert.Contains(t, paginator.DB.Statement.Preloads, "Relation")
	assert.Equal(t, []string{`"test_scope_models"."name"`, `"test_scope_models"."email"`, `(UPPER("test_scope_models".name)) "computed"`, `"test_scope_models"."extra"`, `"test_scope_models"."id"`, `"test_scope_models"."relation_id"`}, paginator.DB.Statement.Selects)
}

func TestScopeUnpaginatedDisableFields(t *testing.T) {
	settings := &Settings[*TestScopeModel]{
		DisableFields: true, // With "fields" disabled, all whitelisted columns are expected to be selected
		FieldsSearch:  []string{"email"},
		Whitelist: Whitelist{
			Fields: AllowAllFields,
			Relations: map[string]Whitelist{
				"Relation": {Fields: AllowAllFields},
			},
		},
	}

	sqlExpect := func(mock sqlmock.Sqlmock) {
		// No Count query, no transaction
		// Actual models
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT "test_scope_models"."name","test_scope_models"."email",(UPPER("test_scope_models".name)) "computed","test_scope_models"."extra","test_scope_models"."id","test_scope_models"."relation_id" FROM "test_scope_models" WHERE (("test_scope_models"."name" LIKE $1 AND "test_scope_models"."name" LIKE $2) OR "test_scope_models"."name" = $3) AND "test_scope_models"."email" LIKE $4 ORDER BY "test_scope_models"."name"`)).
			WithArgs("%val1%", "%val2%", "val3", "%val%").
			WillReturnRows(sqlmock.NewRows([]string{"name", "email", "computed", "extra", "id", "relation_id"}).AddRow("name", "email@example.org", "NAME", "extra", 123, 456))
		// Preload relation
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT "test_scope_relations"."a","test_scope_relations"."b","test_scope_relations"."id" FROM "test_scope_relations" WHERE "test_scope_relations"."id" = $1`)).
			WithArgs(456).
			WillReturnRows(sqlmock.NewRows([]string{"a", "b", "id"}).AddRow("a", "b", 456))
	}

	results, db := runScopeUnpaginated(t, settings, testRequest, sqlExpect)
	assert.NotNil(t, results)
	assert.NoError(t, db.Error)

	expected := map[string]clause.Clause{
		"WHERE":    expectWhereClause,
		"ORDER BY": expectOrderByClause,
		"FROM":     expectFromClause,
		"SELECT": {
			Name: "SELECT",
			Expression: clause.Select{
				Columns: []clause.Column{
					{Raw: true, Name: `"test_scope_models"."name"`},
					{Raw: true, Name: `"test_scope_models"."email"`},
					{Raw: true, Name: `(UPPER("test_scope_models".name)) "computed"`},
					{Raw: true, Name: `"test_scope_models"."extra"`},
					{Raw: true, Name: `"test_scope_models"."id"`},
					{Raw: true, Name: `"test_scope_models"."relation_id"`},
				},
			},
		},
	}
	assert.Equal(t, expected, db.Statement.Clauses)
	assert.Contains(t, db.Statement.Preloads, "Relation")
	assert.Equal(t, []string{`"test_scope_models"."name"`, `"test_scope_models"."email"`, `(UPPER("test_scope_models".name)) "computed"`, `"test_scope_models"."extra"`, `"test_scope_models"."id"`, `"test_scope_models"."relation_id"`}, db.Statement.Selects)
}

func TestScopeDisableFilter(t *testing.T) {
	settings := &Settings[*TestScopeModel]{
		DisableFilter: true,
		FieldsSearch:  []string{"email"},
		Whitelist: Whitelist{
			Fields: AllowAllFields,
			Relations: map[string]Whitelist{
				"Relation": {Fields: AllowAllFields},
			},
		},
	}

	sqlExpect := func(mock sqlmock.Sqlmock) {
		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "test_scope_models" WHERE "test_scope_models"."email" LIKE $1`)).
			WithArgs("%val%").
			WillReturnRows(sqlmock.NewRows([]string{"count(*)"}).AddRow(19))
		mock.ExpectExec(`SAVEPOINT .+`).WithoutArgs().WillReturnResult(sqlmock.NewResult(0, 0))
		// Actual models
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT "test_scope_models"."name","test_scope_models"."email",(UPPER("test_scope_models".name)) "computed","test_scope_models"."id","test_scope_models"."relation_id" FROM "test_scope_models" WHERE "test_scope_models"."email" LIKE $1 ORDER BY "test_scope_models"."name" LIMIT $2 OFFSET $3`)).
			WithArgs("%val%", 15, 15).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email", "computed", "relation_id"}).AddRow(123, "name", "email@example.org", "NAME", 456))
		// Preload relation
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT "test_scope_relations"."a","test_scope_relations"."b","test_scope_relations"."id" FROM "test_scope_relations" WHERE "test_scope_relations"."id" = $1`)).
			WithArgs(456).
			WillReturnRows(sqlmock.NewRows([]string{"a", "b", "id"}).AddRow("a", "b", 456))
		mock.ExpectCommit()
	}

	paginator, err := runScope(t, settings, testRequest, sqlExpect)
	assert.NotNil(t, paginator)
	assert.NoError(t, err)

	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.AndConditions{
						Exprs: []clause.Expression{
							clause.Expr{
								SQL:                `"test_scope_models"."email" LIKE ?`,
								Vars:               []any{"%val%"},
								WithoutParentheses: false,
							},
						},
					},
				},
			},
		},
		"ORDER BY": expectOrderByClause,
		"LIMIT": {
			Expression: clause.Limit{
				Limit:  new(15),
				Offset: 15,
			},
		},
		"FROM":   expectFromClause,
		"SELECT": expectSelectClause,
	}
	assert.Equal(t, expected, paginator.DB.Statement.Clauses)
	assert.Contains(t, paginator.DB.Statement.Preloads, "Relation")
	assert.Equal(t, expectSelects, paginator.DB.Statement.Selects)
}

func TestScopeUnpaginatedDisableFilter(t *testing.T) {
	settings := &Settings[*TestScopeModel]{
		DisableFilter: true,
		FieldsSearch:  []string{"email"},
		Whitelist: Whitelist{
			Fields: AllowAllFields,
			Relations: map[string]Whitelist{
				"Relation": {Fields: AllowAllFields},
			},
		},
	}

	sqlExpect := func(mock sqlmock.Sqlmock) {
		// No Count query, not transaction
		// Actual models
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT "test_scope_models"."name","test_scope_models"."email",(UPPER("test_scope_models".name)) "computed","test_scope_models"."id","test_scope_models"."relation_id" FROM "test_scope_models" WHERE "test_scope_models"."email" LIKE $1 ORDER BY "test_scope_models"."name"`)).
			WithArgs("%val%").
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email", "computed", "relation_id"}).AddRow(123, "name", "email@example.org", "NAME", 456))
		// Preload relation
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT "test_scope_relations"."a","test_scope_relations"."b","test_scope_relations"."id" FROM "test_scope_relations" WHERE "test_scope_relations"."id" = $1`)).
			WithArgs(456).
			WillReturnRows(sqlmock.NewRows([]string{"a", "b", "id"}).AddRow("a", "b", 456))
	}

	results, db := runScopeUnpaginated(t, settings, testRequest, sqlExpect)
	assert.NotNil(t, results)
	assert.NoError(t, db.Error)

	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.AndConditions{
						Exprs: []clause.Expression{
							clause.Expr{
								SQL:                `"test_scope_models"."email" LIKE ?`,
								Vars:               []any{"%val%"},
								WithoutParentheses: false,
							},
						},
					},
				},
			},
		},
		"ORDER BY": expectOrderByClause,
		"FROM":     expectFromClause,
		"SELECT":   expectSelectClause,
	}
	assert.Equal(t, expected, db.Statement.Clauses)
	assert.Contains(t, db.Statement.Preloads, "Relation")
	assert.Equal(t, expectSelects, db.Statement.Selects)
}

func TestScopeDisableSort(t *testing.T) {
	settings := &Settings[*TestScopeModel]{
		DisableSort:  true,
		FieldsSearch: []string{"email"},
		Whitelist: Whitelist{
			Fields: AllowAllFields,
			Relations: map[string]Whitelist{
				"Relation": {Fields: AllowAllFields},
			},
		},
	}

	sqlExpect := func(mock sqlmock.Sqlmock) {
		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "test_scope_models" WHERE (("test_scope_models"."name" LIKE $1 AND "test_scope_models"."name" LIKE $2) OR "test_scope_models"."name" = $3) AND "test_scope_models"."email" LIKE $4`)).
			WithArgs("%val1%", "%val2%", "val3", "%val%").
			WillReturnRows(sqlmock.NewRows([]string{"count(*)"}).AddRow(19))
		mock.ExpectExec(`SAVEPOINT .+`).WithoutArgs().WillReturnResult(sqlmock.NewResult(0, 0))
		// Actual models
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT "test_scope_models"."name","test_scope_models"."email",(UPPER("test_scope_models".name)) "computed","test_scope_models"."id","test_scope_models"."relation_id" FROM "test_scope_models" WHERE (("test_scope_models"."name" LIKE $1 AND "test_scope_models"."name" LIKE $2) OR "test_scope_models"."name" = $3) AND "test_scope_models"."email" LIKE $4 LIMIT $5 OFFSET $6`)).
			WithArgs("%val1%", "%val2%", "val3", "%val%", 15, 15).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email", "computed", "relation_id"}).AddRow(123, "name", "email@example.org", "NAME", 456))
		// Preload relation
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT "test_scope_relations"."a","test_scope_relations"."b","test_scope_relations"."id" FROM "test_scope_relations" WHERE "test_scope_relations"."id" = $1`)).
			WithArgs(456).
			WillReturnRows(sqlmock.NewRows([]string{"a", "b", "id"}).AddRow("a", "b", 456))
		mock.ExpectCommit()
	}

	paginator, err := runScope(t, settings, testRequest, sqlExpect)
	assert.NotNil(t, paginator)
	assert.NoError(t, err)

	expected := map[string]clause.Clause{
		"WHERE": expectWhereClause,
		"LIMIT": {
			Expression: clause.Limit{
				Limit:  new(15),
				Offset: 15,
			},
		},
		"FROM":   expectFromClause,
		"SELECT": expectSelectClause,
	}
	assert.Equal(t, expected, paginator.DB.Statement.Clauses)
	assert.Contains(t, paginator.DB.Statement.Preloads, "Relation")
	assert.Equal(t, expectSelects, paginator.DB.Statement.Selects)
}

func TestScopeUnpaginatedDisableSort(t *testing.T) {
	settings := &Settings[*TestScopeModel]{
		DisableSort:  true,
		FieldsSearch: []string{"email"},
		Whitelist: Whitelist{
			Fields: AllowAllFields,
			Relations: map[string]Whitelist{
				"Relation": {Fields: AllowAllFields},
			},
		},
	}

	sqlExpect := func(mock sqlmock.Sqlmock) {
		// No Count query, no transaction
		// Actual models
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT "test_scope_models"."name","test_scope_models"."email",(UPPER("test_scope_models".name)) "computed","test_scope_models"."id","test_scope_models"."relation_id" FROM "test_scope_models" WHERE (("test_scope_models"."name" LIKE $1 AND "test_scope_models"."name" LIKE $2) OR "test_scope_models"."name" = $3) AND "test_scope_models"."email" LIKE $4`)).
			WithArgs("%val1%", "%val2%", "val3", "%val%").
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email", "computed", "relation_id"}).AddRow(123, "name", "email@example.org", "NAME", 456))
		// Preload relation
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT "test_scope_relations"."a","test_scope_relations"."b","test_scope_relations"."id" FROM "test_scope_relations" WHERE "test_scope_relations"."id" = $1`)).
			WithArgs(456).
			WillReturnRows(sqlmock.NewRows([]string{"a", "b", "id"}).AddRow("a", "b", 456))
	}

	resources, db := runScopeUnpaginated(t, settings, testRequest, sqlExpect)
	assert.NotNil(t, resources)
	assert.NoError(t, db.Error)

	expected := map[string]clause.Clause{
		"WHERE":  expectWhereClause,
		"FROM":   expectFromClause,
		"SELECT": expectSelectClause,
	}
	assert.Equal(t, expected, db.Statement.Clauses)
	assert.Contains(t, db.Statement.Preloads, "Relation")
	assert.Equal(t, expectSelects, db.Statement.Selects)
}

func TestScopeDisableJoin(t *testing.T) {
	settings := &Settings[*TestScopeModel]{
		DisableJoin:  true,
		FieldsSearch: []string{"email"},
		Whitelist: Whitelist{
			Fields: AllowAllFields,
			Relations: map[string]Whitelist{
				"Relation": {Fields: AllowAllFields},
			},
		},
	}

	sqlExpect := func(mock sqlmock.Sqlmock) {
		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "test_scope_models" WHERE (("test_scope_models"."name" LIKE $1 AND "test_scope_models"."name" LIKE $2) OR "test_scope_models"."name" = $3) AND "test_scope_models"."email" LIKE $4`)).
			WithArgs("%val1%", "%val2%", "val3", "%val%").
			WillReturnRows(sqlmock.NewRows([]string{"count(*)"}).AddRow(19))
		mock.ExpectExec(`SAVEPOINT .+`).WithoutArgs().WillReturnResult(sqlmock.NewResult(0, 0))
		// Actual models
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT "test_scope_models"."name","test_scope_models"."email",(UPPER("test_scope_models".name)) "computed","test_scope_models"."id" FROM "test_scope_models" WHERE (("test_scope_models"."name" LIKE $1 AND "test_scope_models"."name" LIKE $2) OR "test_scope_models"."name" = $3) AND "test_scope_models"."email" LIKE $4 ORDER BY "test_scope_models"."name" LIMIT $5 OFFSET $6`)).
			WithArgs("%val1%", "%val2%", "val3", "%val%", 15, 15).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email", "computed"}).AddRow(123, "name", "email@example.org", "NAME"))
		// No preload
		mock.ExpectCommit()
	}

	paginator, err := runScope(t, settings, testRequest, sqlExpect)
	assert.NotNil(t, paginator)
	assert.NoError(t, err)

	expected := map[string]clause.Clause{
		"WHERE": expectWhereClause,
		"LIMIT": {
			Expression: clause.Limit{
				Limit:  new(15),
				Offset: 15,
			},
		},
		"ORDER BY": expectOrderByClause,
		"FROM":     expectFromClause,
		"SELECT": {
			Name: "SELECT",
			Expression: clause.Select{
				Columns: []clause.Column{
					{Raw: true, Name: `"test_scope_models"."name"`},
					{Raw: true, Name: `"test_scope_models"."email"`},
					{Raw: true, Name: `(UPPER("test_scope_models".name)) "computed"`},
					{Raw: true, Name: `"test_scope_models"."id"`},
				},
			},
		},
	}
	assert.Equal(t, expected, paginator.DB.Statement.Clauses)
	assert.NotContains(t, paginator.DB.Statement.Preloads, "Relation")
	assert.Equal(t, []string{`"test_scope_models"."name"`, `"test_scope_models"."email"`, `(UPPER("test_scope_models".name)) "computed"`, `"test_scope_models"."id"`}, paginator.DB.Statement.Selects)
}

func TestScopeUnpaginatedDisableJoin(t *testing.T) {
	settings := &Settings[*TestScopeModel]{
		DisableJoin:  true,
		FieldsSearch: []string{"email"},
		Whitelist: Whitelist{
			Fields: AllowAllFields,
			Relations: map[string]Whitelist{
				"Relation": {Fields: AllowAllFields},
			},
		},
	}

	sqlExpect := func(mock sqlmock.Sqlmock) {
		// No Count query, no transaction
		// Actual models
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT "test_scope_models"."name","test_scope_models"."email",(UPPER("test_scope_models".name)) "computed","test_scope_models"."id" FROM "test_scope_models" WHERE (("test_scope_models"."name" LIKE $1 AND "test_scope_models"."name" LIKE $2) OR "test_scope_models"."name" = $3) AND "test_scope_models"."email" LIKE $4 ORDER BY "test_scope_models"."name"`)).
			WithArgs("%val1%", "%val2%", "val3", "%val%").
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email", "computed", "relation_id"}).AddRow(123, "name", "email@example.org", "NAME", 456))
		// No preload
	}

	resources, db := runScopeUnpaginated(t, settings, testRequest, sqlExpect)
	assert.NotNil(t, resources)
	assert.NoError(t, db.Error)

	expected := map[string]clause.Clause{
		"WHERE":    expectWhereClause,
		"ORDER BY": expectOrderByClause,
		"FROM":     expectFromClause,
		"SELECT": {
			Name: "SELECT",
			Expression: clause.Select{
				Columns: []clause.Column{
					{Raw: true, Name: `"test_scope_models"."name"`},
					{Raw: true, Name: `"test_scope_models"."email"`},
					{Raw: true, Name: `(UPPER("test_scope_models".name)) "computed"`},
					{Raw: true, Name: `"test_scope_models"."id"`},
				},
			},
		},
	}
	assert.Equal(t, expected, db.Statement.Clauses)
	assert.NotContains(t, db.Statement.Preloads, "Relation")
	assert.Equal(t, []string{`"test_scope_models"."name"`, `"test_scope_models"."email"`, `(UPPER("test_scope_models".name)) "computed"`, `"test_scope_models"."id"`}, db.Statement.Selects)
}

func TestScopeDisableSearch(t *testing.T) {
	settings := &Settings[*TestScopeModel]{
		DisableSearch: true,
		FieldsSearch:  []string{"name"},
		Whitelist: Whitelist{
			Fields: AllowAllFields,
			Relations: map[string]Whitelist{
				"Relation": {Fields: AllowAllFields},
			},
		},
	}

	sqlExpect := func(mock sqlmock.Sqlmock) {
		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "test_scope_models" WHERE ("test_scope_models"."name" LIKE $1 AND "test_scope_models"."name" LIKE $2) OR "test_scope_models"."name" = $3`)).
			WithArgs("%val1%", "%val2%", "val3").
			WillReturnRows(sqlmock.NewRows([]string{"count(*)"}).AddRow(19))
		mock.ExpectExec(`SAVEPOINT .+`).WithoutArgs().WillReturnResult(sqlmock.NewResult(0, 0))
		// Actual models
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT "test_scope_models"."name","test_scope_models"."email",(UPPER("test_scope_models".name)) "computed","test_scope_models"."id","test_scope_models"."relation_id" FROM "test_scope_models" WHERE ("test_scope_models"."name" LIKE $1 AND "test_scope_models"."name" LIKE $2) OR "test_scope_models"."name" = $3 ORDER BY "test_scope_models"."name" LIMIT $4 OFFSET $5`)).
			WithArgs("%val1%", "%val2%", "val3", 15, 15).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email", "computed", "relation_id"}).AddRow(123, "name", "email@example.org", "NAME", 456))
		// Preload relation
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT "test_scope_relations"."a","test_scope_relations"."b","test_scope_relations"."id" FROM "test_scope_relations" WHERE "test_scope_relations"."id" = $1`)).
			WithArgs(456).
			WillReturnRows(sqlmock.NewRows([]string{"a", "b", "id"}).AddRow("a", "b", 456))
		mock.ExpectCommit()
	}

	paginator, err := runScope(t, settings, testRequest, sqlExpect)
	assert.NotNil(t, paginator)
	assert.NoError(t, err)

	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.AndConditions{
						Exprs: []clause.Expression{
							clause.OrConditions{
								Exprs: []clause.Expression{
									clause.AndConditions{
										Exprs: []clause.Expression{
											clause.Expr{SQL: `"test_scope_models"."name" LIKE ?`, Vars: []any{"%val1%"}},
											clause.Expr{SQL: `"test_scope_models"."name" LIKE ?`, Vars: []any{"%val2%"}},
										},
									},
								},
							},
							clause.OrConditions{
								Exprs: []clause.Expression{
									clause.AndConditions{
										Exprs: []clause.Expression{
											clause.Expr{SQL: `"test_scope_models"."name" = ?`, Vars: []any{"val3"}},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		"LIMIT": {
			Expression: clause.Limit{
				Limit:  new(15),
				Offset: 15,
			},
		},
		"ORDER BY": expectOrderByClause,
		"FROM":     expectFromClause,
		"SELECT":   expectSelectClause,
	}
	assert.Equal(t, expected, paginator.DB.Statement.Clauses)
	assert.Contains(t, paginator.DB.Statement.Preloads, "Relation")
	assert.Equal(t, expectSelects, paginator.DB.Statement.Selects)
}

func TestScopeUnpaginatedDisableSearch(t *testing.T) {
	settings := &Settings[*TestScopeModel]{
		DisableSearch: true,
		FieldsSearch:  []string{"name"},
		Whitelist: Whitelist{
			Fields: AllowAllFields,
			Relations: map[string]Whitelist{
				"Relation": {Fields: AllowAllFields},
			},
		},
	}

	sqlExpect := func(mock sqlmock.Sqlmock) {
		// No Count query, no transaction
		// Actual models
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT "test_scope_models"."name","test_scope_models"."email",(UPPER("test_scope_models".name)) "computed","test_scope_models"."id","test_scope_models"."relation_id" FROM "test_scope_models" WHERE ("test_scope_models"."name" LIKE $1 AND "test_scope_models"."name" LIKE $2) OR "test_scope_models"."name" = $3 ORDER BY "test_scope_models"."name"`)).
			WithArgs("%val1%", "%val2%", "val3").
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email", "computed", "relation_id"}).AddRow(123, "name", "email@example.org", "NAME", 456))
		// Preload relation
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT "test_scope_relations"."a","test_scope_relations"."b","test_scope_relations"."id" FROM "test_scope_relations" WHERE "test_scope_relations"."id" = $1`)).
			WithArgs(456).
			WillReturnRows(sqlmock.NewRows([]string{"a", "b", "id"}).AddRow("a", "b", 456))
	}

	resources, db := runScopeUnpaginated(t, settings, testRequest, sqlExpect)
	assert.NotNil(t, resources)
	assert.NoError(t, db.Error)

	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.AndConditions{
						Exprs: []clause.Expression{
							clause.OrConditions{
								Exprs: []clause.Expression{
									clause.AndConditions{
										Exprs: []clause.Expression{
											clause.Expr{SQL: `"test_scope_models"."name" LIKE ?`, Vars: []any{"%val1%"}},
											clause.Expr{SQL: `"test_scope_models"."name" LIKE ?`, Vars: []any{"%val2%"}},
										},
									},
								},
							},
							clause.OrConditions{
								Exprs: []clause.Expression{
									clause.AndConditions{
										Exprs: []clause.Expression{
											clause.Expr{SQL: `"test_scope_models"."name" = ?`, Vars: []any{"val3"}},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		"ORDER BY": expectOrderByClause,
		"FROM":     expectFromClause,
		"SELECT":   expectSelectClause,
	}
	assert.Equal(t, expected, db.Statement.Clauses)
	assert.Contains(t, db.Statement.Preloads, "Relation")
	assert.Equal(t, expectSelects, db.Statement.Selects)
}

func TestScopeNoPrimaryKey(t *testing.T) {
	request := &Request{
		Fields: typeutil.NewUndefined([]string{"name"}),
		Join:   typeutil.NewUndefined([]*Join{{Relation: "Relation", Fields: []string{"a", "b"}}}),
	}
	db, mock := openMockDB(t)
	mock.ExpectBegin() // Count query transaction
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "test_scope_model_no_primary_keys"`)).
		WithoutArgs().
		WillReturnRows(sqlmock.NewRows([]string{"count(*)"}).AddRow(13))
	mock.ExpectRollback()

	results := []*TestScopeModelNoPrimaryKey{}
	paginator, err := Scope(db, request, &results)
	assert.Equal(t, "could not find primary key. Add `gorm:\"primaryKey\"` to your model", err.Error())
	assert.Equal(t, err, paginator.DB.Error)
}

func TestScopeUnpaginatedNoPrimaryKey(t *testing.T) {
	request := &Request{
		Fields: typeutil.NewUndefined([]string{"name"}),
		Join:   typeutil.NewUndefined([]*Join{{Relation: "Relation", Fields: []string{"a", "b"}}}),
	}
	db, _ := openMockDB(t)

	var results []*TestScopeModelNoPrimaryKey
	db = ScopeUnpaginated(db, request, &results)
	assert.Nil(t, results)
	assert.Equal(t, "could not find primary key. Add `gorm:\"primaryKey\"` to your model", db.Error.Error())
}

func TestScopeInvalidModel(t *testing.T) {
	request := &Request{}
	db, mock := openMockDB(t)

	mock.ExpectBegin()

	model := []string{}
	assert.Panics(t, func() {
		// Not called because the count query returns 0
		_, _ = Scope(db, request, &model)
	})
}

func TestScopeUnpaginatedInvalidModel(t *testing.T) {
	request := &Request{}
	db, _ := openMockDB(t)
	model := []string{}
	assert.Panics(t, func() {
		ScopeUnpaginated(db, request, &model)
	})
}

func TestScopeWithFieldsWhitelist(t *testing.T) {
	settings := &Settings[*TestScopeModel]{
		Whitelist: Whitelist{
			Fields: []string{"email", "computed", "id", "relation_id"},
		},
	}

	db, mock := openMockDB(t)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "test_scope_models"`)).
		WithoutArgs().
		WillReturnRows(sqlmock.NewRows([]string{"count(*)"}).AddRow(19))
		// Actual models
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT "test_scope_models"."email",(UPPER("test_scope_models".name)) "computed","test_scope_models"."id","test_scope_models"."relation_id" FROM "test_scope_models" LIMIT $1`)).
		WithArgs(DefaultPageSize).
		WillReturnRows(sqlmock.NewRows([]string{"email", "computed", "id", "relation_id"}).AddRow("email@example.org", "NAME", 123, 456))
	mock.ExpectCommit()

	results := []*TestScopeModel{}
	paginator, err := settings.Scope(db, &Request{}, &results)
	assert.NotNil(t, paginator)
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{`"test_scope_models"."id"`, `"test_scope_models"."relation_id"`, `"test_scope_models"."email"`, `(UPPER("test_scope_models".name)) "computed"`}, paginator.DB.Statement.Selects)
}

func TestScopeUnpaginatedWithFieldsWhitelist(t *testing.T) {
	settings := &Settings[*TestScopeModel]{
		Whitelist: Whitelist{
			Fields: []string{"email", "computed", "id", "relation_id"},
		},
	}

	db, mock := openMockDB(t)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT "test_scope_models"."email",(UPPER("test_scope_models".name)) "computed","test_scope_models"."id","test_scope_models"."relation_id" FROM "test_scope_models"`)).
		WithoutArgs().
		WillReturnRows(sqlmock.NewRows([]string{"email", "computed", "id", "relation_id"}).AddRow("email@example.org", "NAME", 123, 456))

	var results []*TestScopeModel
	tx := settings.ScopeUnpaginated(db, &Request{}, &results)
	assert.NotNil(t, results)
	assert.NoError(t, tx.Error)
	assert.ElementsMatch(t, []string{`"test_scope_models"."id"`, `"test_scope_models"."relation_id"`, `"test_scope_models"."email"`, `(UPPER("test_scope_models".name)) "computed"`}, tx.Statement.Selects)
}

func TestWhitelistGetSelectableFields(t *testing.T) {
	type selectableModel struct {
		Relation   *TestScopeRelation
		Ignored    string `gorm:"-"`
		Email      string
		ID         uint
		RelationID uint
	}

	db, _ := openMockDB(t)
	sch, err := parseModel(db, &selectableModel{})
	require.NoError(t, err)

	whitelist := Whitelist{
		Fields: []string{
			"id",
			"email",
			"Email",    // deduplication
			"ignored",  // Struct tag ignore
			"Relation", // Relation are fields that don't have a DBName
		},
	}

	assert.ElementsMatch(t, []*schema.Field{sch.FieldsByDBName["id"], sch.FieldsByDBName["email"]}, getSelectableFields(whitelist, sch))
	assert.ElementsMatch(t, []*schema.Field{sch.FieldsByDBName["id"], sch.FieldsByDBName["email"], sch.FieldsByDBName["relation_id"]}, getSelectableFields(Whitelist{Fields: AllowAllFields}, sch))
}

type TestFilterScopeModel struct {
	Name string
	ID   int `gorm:"primaryKey"`
}

func (m *TestFilterScopeModel) TableName() string {
	return "test_scope_models"
}

func TestApplyFiltersAnd(t *testing.T) {
	request := &Request{
		Filter: typeutil.NewUndefined([]*Filter{
			{Field: "name", Args: []string{"val1"}, Operator: Operators["$cont"]},
			{Field: "name", Args: []string{"val2"}, Operator: Operators["$cont"]},
		}),
	}
	db, mock := openMockDB(t)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "test_scope_models" WHERE ("test_scope_models"."name" LIKE $1 AND "test_scope_models"."name" LIKE $2)`)).
		WithArgs("%val1%", "%val2%").
		WillReturnRows(sqlmock.NewRows([]string{"name", "id"}).AddRow("name", 123))

	schema, err := parseModel(db, &TestFilterScopeModel{})
	require.NoError(t, err)

	settings := &Settings[*TestFilterScopeModel]{
		Whitelist: Whitelist{Fields: AllowAllFields},
	}

	db = settings.applyFilters(db, request, schema).Find(&[]*TestFilterScopeModel{})
	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.AndConditions{
						Exprs: []clause.Expression{
							clause.AndConditions{
								Exprs: []clause.Expression{
									clause.Expr{SQL: `"test_scope_models"."name" LIKE ?`, Vars: []any{"%val1%"}},
									clause.Expr{SQL: `"test_scope_models"."name" LIKE ?`, Vars: []any{"%val2%"}},
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
}

func TestApplyFiltersMixed(t *testing.T) {
	request := &Request{
		Filter: typeutil.NewUndefined([]*Filter{
			{Field: "name", Args: []string{"val1"}, Operator: Operators["$cont"]},
			{Field: "name", Args: []string{"val2"}, Operator: Operators["$cont"]},
		}),
		Or: typeutil.NewUndefined([]*Filter{
			{Field: "name", Args: []string{"val3"}, Operator: Operators["$eq"], Or: true},
			{Field: "name", Args: []string{"val4"}, Operator: Operators["$eq"], Or: true},
		}),
	}
	db, mock := openMockDB(t)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "test_scope_models" WHERE ("test_scope_models"."name" LIKE $1 AND "test_scope_models"."name" LIKE $2) OR ("test_scope_models"."name" = $3 AND "test_scope_models"."name" = $4)`)).
		WithArgs("%val1%", "%val2%", "val3", "val4").
		WillReturnRows(sqlmock.NewRows([]string{"name", "id"}).AddRow("name", 123))

	schema, err := parseModel(db, &TestFilterScopeModel{})
	require.NoError(t, err)

	settings := &Settings[*TestFilterScopeModel]{
		Whitelist: Whitelist{Fields: AllowAllFields},
	}

	db = settings.applyFilters(db, request, schema).Find(&[]*TestFilterScopeModel{})
	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.AndConditions{
						Exprs: []clause.Expression{
							clause.OrConditions{
								Exprs: []clause.Expression{
									clause.AndConditions{
										Exprs: []clause.Expression{
											clause.Expr{SQL: `"test_scope_models"."name" LIKE ?`, Vars: []any{"%val1%"}},
											clause.Expr{SQL: `"test_scope_models"."name" LIKE ?`, Vars: []any{"%val2%"}},
										},
									},
								},
							},
							clause.OrConditions{
								Exprs: []clause.Expression{
									clause.AndConditions{
										Exprs: []clause.Expression{
											clause.Expr{SQL: `"test_scope_models"."name" = ?`, Vars: []any{"val3"}},
											clause.Expr{SQL: `"test_scope_models"."name" = ?`, Vars: []any{"val4"}},
										},
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
}

func TestApplyFiltersOr(t *testing.T) {
	cases := []struct {
		desc string
		or   bool
	}{
		{desc: "with_or_flag", or: true},
		// If the request was built programmatically, the Filter.Flag flag may be false
		// even if it's in the Request.Or slice. Make sure it is still builts as a OR filter. (see PR #13)
		{desc: "without_or_flag", or: false},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			request := &Request{
				Or: typeutil.NewUndefined([]*Filter{
					{Field: "name", Args: []string{"val1"}, Operator: Operators["$cont"], Or: c.or},
					{Field: "name", Args: []string{"val2"}, Operator: Operators["$cont"], Or: c.or},
				}),
			}
			db, mock := openMockDB(t)

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "test_scope_models" WHERE ("test_scope_models"."name" LIKE $1 OR "test_scope_models"."name" LIKE $2)`)).
				WithArgs("%val1%", "%val2%").
				WillReturnRows(sqlmock.NewRows([]string{"name", "id"}).AddRow("name", 123))

			schema, err := parseModel(db, &TestFilterScopeModel{})
			require.NoError(t, err)

			settings := &Settings[*TestFilterScopeModel]{
				Whitelist: Whitelist{Fields: AllowAllFields},
			}

			db = settings.applyFilters(db, request, schema).Find(&[]*TestFilterScopeModel{})
			expected := map[string]clause.Clause{
				"WHERE": {
					Name: "WHERE",
					Expression: clause.Where{
						Exprs: []clause.Expression{
							clause.AndConditions{
								Exprs: []clause.Expression{
									clause.AndConditions{
										Exprs: []clause.Expression{
											clause.OrConditions{
												Exprs: []clause.Expression{
													clause.Expr{SQL: `"test_scope_models"."name" LIKE ?`, Vars: []any{"%val1%"}},
												},
											},
											clause.OrConditions{
												Exprs: []clause.Expression{
													clause.Expr{SQL: `"test_scope_models"."name" LIKE ?`, Vars: []any{"%val2%"}},
												},
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
		})
	}
}

func TestApplyFiltersWithJoin(t *testing.T) {
	request := &Request{
		Filter: typeutil.NewUndefined([]*Filter{
			{Field: "Relation.name", Args: []string{"val1"}, Operator: Operators["$cont"]},
		}),
	}
	db, mock := openMockDB(t)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT "filter_test_models"."name","filter_test_models"."id" FROM "filter_test_models" LEFT JOIN "filter_test_relations" "Relation" ON "filter_test_models"."id" = "Relation"."parent_id" WHERE "Relation"."name" LIKE $1`)).
		WithArgs("%val1%").
		WillReturnRows(sqlmock.NewRows([]string{"name", "id"}).AddRow("name", 123))

	schema, err := parseModel(db, &FilterTestModel{})
	require.NoError(t, err)

	settings := &Settings[*TestFilterScopeModel]{
		Whitelist: Whitelist{
			Relations: map[string]Whitelist{
				"Relation": {Fields: AllowAllFields},
			},
			Fields: AllowAllFields,
		},
	}

	results := []*FilterTestModel{}
	db = db.Model(&results)
	db = settings.applyFilters(db, request, schema).Find(&results)
	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.AndConditions{
						Exprs: []clause.Expression{
							clause.Expr{SQL: `"Relation"."name" LIKE ?`, Vars: []any{"%val1%"}},
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

	assert.NoError(t, db.Error)
	assert.Equal(t, expected, db.Statement.Clauses)
}

func TestApplySearch(t *testing.T) {
	db, _ := openMockDB(t)
	schema, err := parseModel(db, &TestFilterScopeModel{})
	require.NoError(t, err)

	settings := &Settings[*TestFilterScopeModel]{
		Whitelist: Whitelist{Fields: AllowAllFields},
	}

	search := settings.applySearch("val", schema)
	assert.NotNil(t, search)
	assert.ElementsMatch(t, []string{"id", "name"}, search.Fields)
	assert.Equal(t, "val", search.Query)
	assert.Equal(t, Operators["$cont"], search.Operator)
}

func TestSelectScope(t *testing.T) {
	type TestModel struct {
		A string
		B string
	}

	cases := []struct {
		desc              string
		selectScope       func(*gorm.DB) *gorm.DB
		extraScope        func(*gorm.DB) *gorm.DB
		wantSelects       []string
		wantClauseColumns []clause.Column
	}{
		// Note: when we say "override" it's in the scenario of a join, "no override" a regular select
		{
			desc:              "nil_selectable_fields_no_override",
			selectScope:       selectScope("", nil, false),
			wantSelects:       nil,
			wantClauseColumns: nil,
		},
		{
			desc:              "nil_selectable_fields_override",
			selectScope:       selectScope("", nil, true),
			wantSelects:       nil,
			wantClauseColumns: nil,
		},
		{
			desc:              "no_override",
			selectScope:       selectScope("test_models", []*schema.Field{{DBName: "a"}, {DBName: "b"}}, false),
			wantSelects:       []string{`"test_models"."a"`, `"test_models"."b"`},
			wantClauseColumns: []clause.Column{{Raw: true, Name: `"test_models"."a"`}, {Raw: true, Name: `"test_models"."b"`}},
		},
		{
			desc:              "override",
			selectScope:       selectScope("test_models", []*schema.Field{{DBName: "a"}, {DBName: "b"}}, true),
			wantSelects:       []string{`"test_models"."a"`, `"test_models"."b"`},
			wantClauseColumns: []clause.Column{{Raw: true, Name: `"test_models"."a"`}, {Raw: true, Name: `"test_models"."b"`}},
		},
		{
			desc:              "extra_select_no_override",
			selectScope:       selectScope("test_models", []*schema.Field{{DBName: "a"}, {DBName: "b"}}, false),
			extraScope:        func(db *gorm.DB) *gorm.DB { return db.Select("1 + 1 AS count") },
			wantSelects:       []string{"1 + 1 AS count", `"test_models"."a"`, `"test_models"."b"`},
			wantClauseColumns: []clause.Column{{Raw: true, Name: "1 + 1 AS count"}, {Raw: true, Name: `"test_models"."a"`}, {Raw: true, Name: `"test_models"."b"`}},
		},
		{
			desc:              "no_selectable_fields_select_override",
			selectScope:       selectScope("test_models", []*schema.Field{}, true),
			extraScope:        func(db *gorm.DB) *gorm.DB { return db.Select("*, 1 + 1 AS count") },
			wantSelects:       []string{"1", "2"},
			wantClauseColumns: []clause.Column{{Raw: true, Name: "1"}, {Raw: true, Name: "2"}},
		},
		{
			desc:              "computed_column_with_override",
			selectScope:       selectScope("test_models", []*schema.Field{{DBName: "c", StructField: reflect.StructField{Tag: `computed:"a+b"`}}}, true),
			extraScope:        func(db *gorm.DB) *gorm.DB { return db.Select("*, 1 + 1 AS count") },
			wantSelects:       []string{`(a+b) "c"`},
			wantClauseColumns: []clause.Column{{Raw: true, Name: `(a+b) "c"`}},
		},
		{
			desc:              "computed_column_with_joins",
			selectScope:       selectScope("test_models", []*schema.Field{{DBName: "c", StructField: reflect.StructField{Tag: `computed:"a+b"`}}}, true),
			extraScope:        func(db *gorm.DB) *gorm.DB { return db.Joins("relation").Select("*, 1 + 1 AS count") },
			wantSelects:       []string{`(a+b) "c"`},
			wantClauseColumns: []clause.Column{{Raw: true, Name: `(a+b) "c"`}},
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			db, _ := openMockDB(t)
			db.DryRun = true
			if c.extraScope != nil {
				db = db.Scopes(c.extraScope)
			}
			db = db.Model(&TestModel{}).Scopes(c.selectScope).Find(&[]*TestModel{})
			assert.NoError(t, db.Error)
			assert.Equal(t, c.wantSelects, db.Statement.Selects)
			assert.Equal(t, c.wantClauseColumns, db.Statement.Clauses["SELECT"].Expression.(clause.Select).Columns)
		})
	}
}

func TestGetFieldFinalRelation(t *testing.T) {
	db, _ := openMockDB(t)
	schema, err := parseModel(db, &FilterTestModel{})
	require.NoError(t, err)

	settings := &Settings[*FilterTestModel]{
		Whitelist: Whitelist{Fields: AllowAllFields},
	}
	field, sch, joinName := getField("Relation.name", schema, &settings.Whitelist)
	assert.Nil(t, field)
	assert.Nil(t, sch)
	assert.Empty(t, joinName)

	settings = &Settings[*FilterTestModel]{
		Whitelist: Whitelist{
			Relations: map[string]Whitelist{
				"Relation": {
					Fields:    []string{"id", "name", "relation_id"},
					Relations: map[string]Whitelist{},
				},
			},
		},
	}
	field, sch, joinName = getField("Relation.NestedRelation.field", schema, &settings.Whitelist)
	assert.Nil(t, field)
	assert.Nil(t, sch)
	assert.Empty(t, joinName)

	field, sch, joinName = getField("Relation.field", schema, &settings.Whitelist)
	assert.Nil(t, field)
	assert.Nil(t, sch)
	assert.Empty(t, joinName)
}

func TestSettingsComputedFieldWithAutoFields(t *testing.T) {
	request := &Request{
		Filter: typeutil.NewUndefined([]*Filter{
			{Field: "Relation.a", Args: []string{"val1"}, Operator: Operators["$cont"]},
		}),
		PerPage: typeutil.NewUndefined(15),
	}
	db, mock := openMockDB(t)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "test_scope_models" LEFT JOIN "test_scope_relations" "Relation" ON "test_scope_models"."relation_id" = "Relation"."id" WHERE "Relation"."a" LIKE $1`)).
		WithArgs("%val1%").
		WillReturnRows(sqlmock.NewRows([]string{"count(*)"}).AddRow(19))
	mock.ExpectExec(`SAVEPOINT .+`).WithoutArgs().WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT "test_scope_models"."name","test_scope_models"."email",(UPPER("test_scope_models".name)) "computed","test_scope_models"."extra","test_scope_models"."id","test_scope_models"."relation_id" FROM "test_scope_models" LEFT JOIN "test_scope_relations" "Relation" ON "test_scope_models"."relation_id" = "Relation"."id" WHERE "Relation"."a" LIKE $1 LIMIT $2`)).
		WithArgs("%val1%", 15).
		WillReturnRows(sqlmock.NewRows([]string{"name", "email", "computed", "extra", "id", "relation_id"}).AddRow("name", "email@example.org", "NAME", "extra", 123, 456))
	mock.ExpectCommit()

	settings := &Settings[*TestScopeModel]{
		Whitelist: Whitelist{
			Relations: map[string]Whitelist{
				"Relation": {Fields: AllowAllFields},
			},
			Fields: AllowAllFields,
		},
	}

	results := []*TestScopeModel{}
	paginator, err := settings.Scope(db, request, &results)

	assert.NotNil(t, paginator)
	assert.NoError(t, err)

	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.AndConditions{
						Exprs: []clause.Expression{
							clause.Expr{SQL: `"Relation"."a" LIKE ?`, Vars: []any{"%val1%"}},
						},
					},
				},
			},
		},
		"LIMIT": {
			Expression: clause.Limit{
				Limit:  new(15),
				Offset: 0,
			},
		},
		"FROM": {
			Name: "FROM",
			Expression: clause.From{
				Joins: []clause.Join{
					{
						Type: clause.LeftJoin,
						Table: clause.Table{
							Name:  "test_scope_relations",
							Alias: "Relation",
						},
						ON: clause.Where{
							Exprs: []clause.Expression{
								clause.Eq{
									Column: clause.Column{
										Table: "test_scope_models",
										Name:  "relation_id",
									},
									Value: clause.Column{
										Table: "Relation",
										Name:  "id",
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
					{Raw: true, Name: `"test_scope_models"."name"`},
					{Raw: true, Name: `"test_scope_models"."email"`},
					{Raw: true, Name: `(UPPER("test_scope_models".name)) "computed"`},
					{Raw: true, Name: `"test_scope_models"."extra"`},
					{Raw: true, Name: `"test_scope_models"."id"`},
					{Raw: true, Name: `"test_scope_models"."relation_id"`},
				},
			},
		},
	}
	assert.Equal(t, expected, paginator.DB.Statement.Clauses)
	assert.ElementsMatch(t, []string{`"test_scope_models"."id"`, `"test_scope_models"."name"`, `"test_scope_models"."email"`, `(UPPER("test_scope_models".name)) "computed"`, `"test_scope_models"."extra"`, `"test_scope_models"."relation_id"`}, paginator.DB.Statement.Selects)
}

func TestSettingsSelectWithExistingJoin(t *testing.T) {
	request := &Request{
		Filter: typeutil.NewUndefined([]*Filter{
			{Field: "Relation.a", Args: []string{"val1"}, Operator: Operators["$cont"]},
		}),
		PerPage: typeutil.NewUndefined(15),
	}
	db, mock := openMockDB(t)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "test_scope_models" LEFT JOIN "test_scope_relations" "Relation" ON "test_scope_models"."relation_id" = "Relation"."id" AND Relation.id > $1 WHERE "Relation"."a" LIKE $2`)).
		WithArgs(0, "%val1%").
		WillReturnRows(sqlmock.NewRows([]string{"count(*)"}).AddRow(19))
	mock.ExpectExec(`SAVEPOINT .+`).WithoutArgs().WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT "Relation"."a" "Relation__a","Relation"."b" "Relation__b","Relation"."id" "Relation__id","test_scope_models"."name","test_scope_models"."email",(UPPER("test_scope_models".name)) "computed","test_scope_models"."extra","test_scope_models"."id","test_scope_models"."relation_id" FROM "test_scope_models" LEFT JOIN "test_scope_relations" "Relation" ON "test_scope_models"."relation_id" = "Relation"."id" AND Relation.id > $1 WHERE "Relation"."a" LIKE $2 LIMIT $3`)).
		WithArgs(0, "%val1%", 15).
		WillReturnRows(
			sqlmock.NewRows([]string{"Relation__a", "Relation__b", "Relation__id", "name", "email", "computed", "extra", "id", "relation_id"}).
				AddRow("a", "b", 456, "name", "email@example.org", "NAME", "extra", 123, 456),
		)
	mock.ExpectCommit()

	// We manually join a relation with a condition.
	// We expect this join to not be removed nor duplicated, with the condition kept and the fields selected.
	db = db.Joins("Relation", db.Session(&gorm.Session{NewDB: true}).Where("Relation.id > ?", 0))

	settings := &Settings[*TestScopeModel]{
		Whitelist: Whitelist{
			Relations: map[string]Whitelist{
				"Relation": {Fields: AllowAllFields},
			},
			Fields: AllowAllFields,
		},
	}

	results := []*TestScopeModel{}
	paginator, err := settings.Scope(db, request, &results)

	assert.NotNil(t, paginator)
	assert.NoError(t, err)

	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.AndConditions{
						Exprs: []clause.Expression{
							clause.Expr{SQL: `"Relation"."a" LIKE ?`, Vars: []any{"%val1%"}},
						},
					},
				},
			},
		},
		"FROM": {
			Name: "FROM",
			Expression: clause.From{
				Joins: []clause.Join{},
			},
		},
		"LIMIT": {
			Expression: clause.Limit{
				Limit:  new(15),
				Offset: 0,
			},
		},
		"SELECT": {
			Name: "SELECT",
			Expression: clause.Select{
				Columns: []clause.Column{
					{Raw: true, Name: `"Relation"."a" "Relation__a"`},
					{Raw: true, Name: `"Relation"."b" "Relation__b"`},
					{Raw: true, Name: `"Relation"."id" "Relation__id"`},
					{Raw: true, Name: `"test_scope_models"."name"`},
					{Raw: true, Name: `"test_scope_models"."email"`},
					{Raw: true, Name: `(UPPER("test_scope_models".name)) "computed"`},
					{Raw: true, Name: `"test_scope_models"."extra"`},
					{Raw: true, Name: `"test_scope_models"."id"`},
					{Raw: true, Name: `"test_scope_models"."relation_id"`},
				},
			},
		},
	}
	assert.Equal(t, expected, paginator.DB.Statement.Clauses)
	require.Len(t, paginator.DB.Statement.Joins, 1)
	j := paginator.DB.Statement.Joins[0]
	assert.Equal(t, "Relation", j.Name)
	assert.Equal(t, clause.LeftJoin, j.JoinType)
	assert.Equal(t, []string{"*"}, j.Omits)
	if assert.Len(t, j.Conds, 1) {
		assert.Equal(t, clause.Where{Exprs: []clause.Expression{clause.Expr{SQL: "Relation.id > ?", Vars: []any{0}}}}, j.Conds[0].(*gorm.DB).Statement.Clauses["WHERE"].Expression)
	}
	assert.Empty(t, j.Selects)
	expectedOn := &clause.Where{
		Exprs: []clause.Expression{
			clause.Expr{SQL: "Relation.id > ?", Vars: []any{0}},
		},
	}
	assert.Equal(t, expectedOn, j.On)
}

type TestScopeRelationWithComputed struct {
	A  string
	B  string
	C  string `gorm:"->;-:migration" computed:"UPPER(~~~ct~~~.b)"`
	ID uint
}
type TestScopeModelWithComputed struct {
	Relation   *TestScopeRelationWithComputed
	Name       string
	Email      string
	Computed   string `gorm:"->;-:migration" computed:"UPPER(~~~ct~~~.name)"`
	ID         uint
	RelationID uint
}

func TestSettingsSelectWithExistingJoinAndComputed(t *testing.T) {
	request := &Request{
		Filter: typeutil.NewUndefined([]*Filter{
			{Field: "Relation.a", Args: []string{"val1"}, Operator: Operators["$cont"]},
		}),
		PerPage: typeutil.NewUndefined(15),
	}
	db, mock := openMockDB(t)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "test_scope_model_with_computeds" LEFT JOIN "test_scope_relation_with_computeds" "Relation" ON "test_scope_model_with_computeds"."relation_id" = "Relation"."id" WHERE "Relation"."a" LIKE $1`)).
		WithArgs("%val1%").
		WillReturnRows(sqlmock.NewRows([]string{"count(*)"}).AddRow(19))
	mock.ExpectExec(`SAVEPOINT .+`).WithoutArgs().WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT "Relation"."a" "Relation__a","Relation"."b" "Relation__b",(UPPER("Relation".b)) "Relation__c","Relation"."id" "Relation__id","test_scope_model_with_computeds"."name","test_scope_model_with_computeds"."email",(UPPER("test_scope_model_with_computeds".name)) "computed","test_scope_model_with_computeds"."id","test_scope_model_with_computeds"."relation_id" FROM "test_scope_model_with_computeds" LEFT JOIN "test_scope_relation_with_computeds" "Relation" ON "test_scope_model_with_computeds"."relation_id" = "Relation"."id" WHERE "Relation"."a" LIKE $1 LIMIT $2`)).
		WithArgs("%val1%", 15).
		WillReturnRows(
			sqlmock.NewRows([]string{"Relation__a", "Relation__b", "Relation__c", "Relation__id", "name", "email", "computed", "id", "relation_id"}).
				AddRow("a", "b", "B", 456, "name", "email@example.org", "NAME", 123, 456),
		)
	mock.ExpectCommit()

	db = db.Joins("Relation")

	settings := &Settings[*TestScopeModelWithComputed]{
		Whitelist: Whitelist{
			Relations: map[string]Whitelist{
				"Relation": {Fields: AllowAllFields},
			},
			Fields: AllowAllFields,
		},
	}

	results := []*TestScopeModelWithComputed{}
	paginator, err := settings.Scope(db, request, &results)

	assert.NotNil(t, paginator)
	assert.NoError(t, err)

	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.AndConditions{
						Exprs: []clause.Expression{
							clause.Expr{SQL: `"Relation"."a" LIKE ?`, Vars: []any{"%val1%"}},
						},
					},
				},
			},
		},
		"FROM": {
			Name: "FROM",
			Expression: clause.From{
				Joins: []clause.Join{},
			},
		},
		"LIMIT": {
			Expression: clause.Limit{
				Limit:  new(15),
				Offset: 0,
			},
		},
		"SELECT": {
			Name: "SELECT",
			Expression: clause.Select{
				Columns: []clause.Column{
					{Raw: true, Name: `"Relation"."a" "Relation__a"`},
					{Raw: true, Name: `"Relation"."b" "Relation__b"`},
					{Raw: true, Name: `(UPPER("Relation".b)) "Relation__c"`},
					{Raw: true, Name: `"Relation"."id" "Relation__id"`},
					{Raw: true, Name: `"test_scope_model_with_computeds"."name"`},
					{Raw: true, Name: `"test_scope_model_with_computeds"."email"`},
					{Raw: true, Name: `(UPPER("test_scope_model_with_computeds".name)) "computed"`},
					{Raw: true, Name: `"test_scope_model_with_computeds"."id"`},
					{Raw: true, Name: `"test_scope_model_with_computeds"."relation_id"`},
				},
			},
		},
	}
	assert.Equal(t, expected, paginator.DB.Statement.Clauses)
	expectedSelect := []string{
		`"Relation"."a" "Relation__a"`,
		`"Relation"."b" "Relation__b"`,
		`(UPPER("Relation".b)) "Relation__c"`,
		`"Relation"."id" "Relation__id"`,
		`"test_scope_model_with_computeds"."name"`,
		`"test_scope_model_with_computeds"."email"`,
		`(UPPER("test_scope_model_with_computeds".name)) "computed"`,
		`"test_scope_model_with_computeds"."id"`,
		`"test_scope_model_with_computeds"."relation_id"`,
	}
	assert.Equal(t, expectedSelect, paginator.DB.Statement.Selects)
	require.Len(t, paginator.DB.Statement.Joins, 1)
	j := paginator.DB.Statement.Joins[0]
	assert.Equal(t, "Relation", j.Name)
	assert.Equal(t, clause.LeftJoin, j.JoinType)
	assert.Equal(t, []string{"*"}, j.Omits)
	assert.Empty(t, j.Conds)
	assert.Empty(t, j.Selects)
	assert.Empty(t, j.On)
}

func TestSettingsSelectWithExistingJoinAndComputedOmit(t *testing.T) {
	request := &Request{
		Filter: typeutil.NewUndefined([]*Filter{
			{Field: "Relation.a", Args: []string{"val1"}, Operator: Operators["$cont"]},
		}),
		PerPage: typeutil.NewUndefined(15),
	}
	db, mock := openMockDB(t)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "test_scope_model_with_computeds" LEFT JOIN "test_scope_relation_with_computeds" "Relation" ON "test_scope_model_with_computeds"."relation_id" = "Relation"."id" WHERE "Relation"."a" LIKE $1`)).
		WithArgs("%val1%").
		WillReturnRows(sqlmock.NewRows([]string{"count(*)"}).AddRow(19))
	mock.ExpectExec(`SAVEPOINT .+`).WithoutArgs().WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT "Relation"."a" "Relation__a","Relation"."b" "Relation__b","Relation"."id" "Relation__id","test_scope_model_with_computeds"."name","test_scope_model_with_computeds"."email",(UPPER("test_scope_model_with_computeds".name)) "computed","test_scope_model_with_computeds"."id","test_scope_model_with_computeds"."relation_id" FROM "test_scope_model_with_computeds" LEFT JOIN "test_scope_relation_with_computeds" "Relation" ON "test_scope_model_with_computeds"."relation_id" = "Relation"."id" WHERE "Relation"."a" LIKE $1 LIMIT $2`)).
		WithArgs("%val1%", 15).
		WillReturnRows(
			sqlmock.NewRows([]string{"Relation__a", "Relation__b", "Relation__id", "name", "email", "computed", "id", "relation_id"}).
				AddRow("a", "b", 456, "name", "email@example.org", "NAME", 123, 456),
		)
	mock.ExpectCommit()

	// We use Omit on Relation to avoid Gorm auto-selecting it
	db = db.Joins("Relation", db.Session(&gorm.Session{NewDB: true}).Omit("c"))

	settings := &Settings[*TestScopeModelWithComputed]{
		Whitelist: Whitelist{
			Relations: map[string]Whitelist{
				"Relation": {Fields: AllowAllFields},
			},
			Fields: AllowAllFields,
		},
	}

	results := []*TestScopeModelWithComputed{}
	paginator, err := settings.Scope(db, request, &results)

	assert.NotNil(t, paginator)
	assert.NoError(t, err)

	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.AndConditions{
						Exprs: []clause.Expression{
							clause.Expr{SQL: `"Relation"."a" LIKE ?`, Vars: []any{"%val1%"}},
						},
					},
				},
			},
		},
		"FROM": {
			Name: "FROM",
			Expression: clause.From{
				Joins: []clause.Join{},
			},
		},
		"LIMIT": {
			Expression: clause.Limit{
				Limit:  new(15),
				Offset: 0,
			},
		},
		"SELECT": {
			Name: "SELECT",
			Expression: clause.Select{
				Columns: []clause.Column{
					{Raw: true, Name: `"Relation"."a" "Relation__a"`},
					{Raw: true, Name: `"Relation"."b" "Relation__b"`},
					{Raw: true, Name: `"Relation"."id" "Relation__id"`},
					{Raw: true, Name: `"test_scope_model_with_computeds"."name"`},
					{Raw: true, Name: `"test_scope_model_with_computeds"."email"`},
					{Raw: true, Name: `(UPPER("test_scope_model_with_computeds".name)) "computed"`},
					{Raw: true, Name: `"test_scope_model_with_computeds"."id"`},
					{Raw: true, Name: `"test_scope_model_with_computeds"."relation_id"`},
				},
			},
		},
	}
	assert.Equal(t, expected, paginator.DB.Statement.Clauses)
	expectedSelect := []string{
		`"Relation"."a" "Relation__a"`,
		`"Relation"."b" "Relation__b"`,
		`"Relation"."id" "Relation__id"`,
		`"test_scope_model_with_computeds"."name"`,
		`"test_scope_model_with_computeds"."email"`,
		`(UPPER("test_scope_model_with_computeds".name)) "computed"`,
		`"test_scope_model_with_computeds"."id"`,
		`"test_scope_model_with_computeds"."relation_id"`,
	}
	assert.Equal(t, expectedSelect, paginator.DB.Statement.Selects)
	require.Len(t, paginator.DB.Statement.Joins, 1)
	j := paginator.DB.Statement.Joins[0]
	assert.Equal(t, "Relation", j.Name)
	assert.Equal(t, clause.LeftJoin, j.JoinType)
	assert.Equal(t, []string{"*"}, j.Omits)
	if assert.Len(t, j.Conds, 1) {
		assert.Equal(t, []string{"c"}, j.Conds[0].(*gorm.DB).Statement.Omits)
	}
	assert.Empty(t, j.Selects)
	assert.Empty(t, j.On)
}

func TestSettingsSelectWithExistingJoinAndComputedWithoutFiltering(t *testing.T) {
	request := &Request{
		PerPage: typeutil.NewUndefined(15),
	}
	db, mock := openMockDB(t)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "test_scope_model_with_computeds" LEFT JOIN "test_scope_relation_with_computeds" "Relation" ON "test_scope_model_with_computeds"."relation_id" = "Relation"."id" AND Relation.id > $1`)).
		WithArgs(0).
		WillReturnRows(sqlmock.NewRows([]string{"count(*)"}).AddRow(19))
	mock.ExpectExec(`SAVEPOINT .+`).WithoutArgs().WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT "Relation"."a" "Relation__a","Relation"."b" "Relation__b",(UPPER("Relation".b)) "Relation__c","Relation"."id" "Relation__id","test_scope_model_with_computeds"."name","test_scope_model_with_computeds"."email",(UPPER("test_scope_model_with_computeds".name)) "computed","test_scope_model_with_computeds"."id","test_scope_model_with_computeds"."relation_id" FROM "test_scope_model_with_computeds" LEFT JOIN "test_scope_relation_with_computeds" "Relation" ON "test_scope_model_with_computeds"."relation_id" = "Relation"."id" AND Relation.id > $1 LIMIT $2`)).
		WithArgs(0, 15).
		WillReturnRows(
			sqlmock.NewRows([]string{"Relation__a", "Relation__b", "Relation__c", "Relation__id", "name", "email", "computed", "id", "relation_id"}).
				AddRow("a", "b", "B", 456, "name", "email@example.org", "NAME", 123, 456),
		)
	mock.ExpectCommit()

	// Gorm will automatically select all the fields of the relation.
	// We expect the computed fields to work properly.
	db = db.Joins("Relation", db.Session(&gorm.Session{NewDB: true}).Where("Relation.id > ?", 0))

	results := []*TestScopeModelWithComputed{}
	paginator, err := Scope(db, request, &results)

	assert.NotNil(t, paginator)
	assert.NoError(t, err)

	expected := map[string]clause.Clause{
		"FROM": {
			Name: "FROM",
			Expression: clause.From{
				Joins: []clause.Join{},
			},
		},
		"LIMIT": {
			Expression: clause.Limit{
				Limit:  new(15),
				Offset: 0,
			},
		},
		"SELECT": {
			Name: "SELECT",
			Expression: clause.Select{
				Columns: []clause.Column{
					{Raw: true, Name: `"Relation"."a" "Relation__a"`},
					{Raw: true, Name: `"Relation"."b" "Relation__b"`},
					{Raw: true, Name: `(UPPER("Relation".b)) "Relation__c"`},
					{Raw: true, Name: `"Relation"."id" "Relation__id"`},
					{Raw: true, Name: `"test_scope_model_with_computeds"."name"`},
					{Raw: true, Name: `"test_scope_model_with_computeds"."email"`},
					{Raw: true, Name: `(UPPER("test_scope_model_with_computeds".name)) "computed"`},
					{Raw: true, Name: `"test_scope_model_with_computeds"."id"`},
					{Raw: true, Name: `"test_scope_model_with_computeds"."relation_id"`},
				},
			},
		},
	}
	assert.Equal(t, expected, paginator.DB.Statement.Clauses)
	expectedSelect := []string{
		`"Relation"."a" "Relation__a"`,
		`"Relation"."b" "Relation__b"`,
		`(UPPER("Relation".b)) "Relation__c"`,
		`"Relation"."id" "Relation__id"`,
		`"test_scope_model_with_computeds"."name"`,
		`"test_scope_model_with_computeds"."email"`,
		`(UPPER("test_scope_model_with_computeds".name)) "computed"`,
		`"test_scope_model_with_computeds"."id"`,
		`"test_scope_model_with_computeds"."relation_id"`,
	}
	assert.Equal(t, expectedSelect, paginator.DB.Statement.Selects)
	require.Len(t, paginator.DB.Statement.Joins, 1)
	j := paginator.DB.Statement.Joins[0]
	assert.Equal(t, "Relation", j.Name)
	assert.Equal(t, clause.LeftJoin, j.JoinType)
	assert.Equal(t, []string{"*"}, j.Omits)
	if assert.Len(t, j.Conds, 1) {
		assert.Equal(t, clause.Where{Exprs: []clause.Expression{clause.Expr{SQL: "Relation.id > ?", Vars: []any{0}}}}, j.Conds[0].(*gorm.DB).Statement.Clauses["WHERE"].Expression)
	}
	assert.Empty(t, j.Selects)
	expectedOn := &clause.Where{
		Exprs: []clause.Expression{
			clause.Expr{SQL: "Relation.id > ?", Vars: []any{0}},
		},
	}
	assert.Equal(t, expectedOn, j.On)
}

func TestSettingsDefaultSort(t *testing.T) {
	settings := &Settings[*TestScopeModel]{
		DefaultSort: []*Sort{
			{Field: "name", Order: SortAscending},
			{Field: "email", Order: SortDescending},
		},
		Whitelist: Whitelist{Fields: AllowAllFields},
	}

	t.Run("sort_not_provided", func(t *testing.T) {
		request := &Request{
			Filter: typeutil.NewUndefined([]*Filter{
				{Field: "name", Args: []string{"val1"}, Operator: Operators["$cont"]},
			}),
			Fields:  typeutil.NewUndefined([]string{"id", "name", "email"}),
			Page:    typeutil.NewUndefined(3),
			PerPage: typeutil.NewUndefined(15),
		}
		db, mock := openMockDB(t)

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "test_scope_models" WHERE "test_scope_models"."name" LIKE $1`)).
			WithArgs("%val1%").
			WillReturnRows(sqlmock.NewRows([]string{"count(*)"}).AddRow(19))
		mock.ExpectExec(`SAVEPOINT .+`).WithoutArgs().WillReturnResult(sqlmock.NewResult(0, 0))
		// Actual models
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT "test_scope_models"."name","test_scope_models"."email","test_scope_models"."id" FROM "test_scope_models" WHERE "test_scope_models"."name" LIKE $1 ORDER BY "test_scope_models"."name","test_scope_models"."email" DESC LIMIT $2 OFFSET $3`)).
			WithArgs("%val1%", 15, 30).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email"}).AddRow(123, "name", "email@example.org"))
		mock.ExpectCommit()

		results := []*TestScopeModel{}

		paginator, err := settings.Scope(db, request, &results)
		assert.NotNil(t, paginator)
		assert.NoError(t, err)

		expected := map[string]clause.Clause{
			"WHERE": {
				Name: "WHERE",
				Expression: clause.Where{
					Exprs: []clause.Expression{
						clause.AndConditions{
							Exprs: []clause.Expression{
								clause.Expr{SQL: `"test_scope_models"."name" LIKE ?`, Vars: []any{"%val1%"}},
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
								Table: "test_scope_models",
								Name:  "name",
							},
						},
						{
							Column: clause.Column{
								Table: "test_scope_models",
								Name:  "email",
							},
							Desc: true,
						},
					},
				},
			},
			"LIMIT": {
				Expression: clause.Limit{
					Limit:  new(15),
					Offset: 30,
				},
			},
			"FROM": expectFromClause,
			"SELECT": {
				Name: "SELECT",
				Expression: clause.Select{
					Columns: []clause.Column{
						{Raw: true, Name: `"test_scope_models"."name"`},
						{Raw: true, Name: `"test_scope_models"."email"`},
						{Raw: true, Name: `"test_scope_models"."id"`},
					},
				},
			},
		}
		assert.Equal(t, expected, paginator.DB.Statement.Clauses)
	})

	t.Run("sort_provided", func(t *testing.T) {
		request := &Request{
			Filter: typeutil.NewUndefined([]*Filter{
				{Field: "name", Args: []string{"val1"}, Operator: Operators["$cont"]},
			}),
			Sort:    typeutil.NewUndefined([]*Sort{{Field: "name", Order: SortDescending}}),
			Fields:  typeutil.NewUndefined([]string{"id", "name", "email"}),
			Page:    typeutil.NewUndefined(3),
			PerPage: typeutil.NewUndefined(15),
		}

		db, mock := openMockDB(t)

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "test_scope_models" WHERE "test_scope_models"."name" LIKE $1`)).
			WithArgs("%val1%").
			WillReturnRows(sqlmock.NewRows([]string{"count(*)"}).AddRow(19))
		mock.ExpectExec(`SAVEPOINT .+`).WithoutArgs().WillReturnResult(sqlmock.NewResult(0, 0))
		// Actual models
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT "test_scope_models"."name","test_scope_models"."email","test_scope_models"."id" FROM "test_scope_models" WHERE "test_scope_models"."name" LIKE $1 ORDER BY "test_scope_models"."name" DESC LIMIT $2 OFFSET $3`)).
			WithArgs("%val1%", 15, 30).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email"}).AddRow(123, "name", "email@example.org"))
		mock.ExpectCommit()

		results := []*TestScopeModel{}
		paginator, err := settings.Scope(db, request, &results)

		assert.NotNil(t, paginator)
		assert.NoError(t, err)

		expected := map[string]clause.Clause{
			"WHERE": {
				Name: "WHERE",
				Expression: clause.Where{
					Exprs: []clause.Expression{
						clause.AndConditions{
							Exprs: []clause.Expression{
								clause.Expr{SQL: `"test_scope_models"."name" LIKE ?`, Vars: []any{"%val1%"}},
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
								Table: "test_scope_models",
								Name:  "name",
							},
							Desc: true,
						},
					},
				},
			},
			"LIMIT": {
				Expression: clause.Limit{
					Limit:  new(15),
					Offset: 30,
				},
			},
			"FROM": expectFromClause,
			"SELECT": {
				Name: "SELECT",
				Expression: clause.Select{
					Columns: []clause.Column{
						{Raw: true, Name: `"test_scope_models"."name"`},
						{Raw: true, Name: `"test_scope_models"."email"`},
						{Raw: true, Name: `"test_scope_models"."id"`},
					},
				},
			},
		}
		assert.Equal(t, expected, paginator.DB.Statement.Clauses)
	})
}

func TestScopeWithCaseInsensitiveSort(t *testing.T) {
	request := &Request{
		Sort: typeutil.NewUndefined([]*Sort{{Field: "name", Order: SortAscending}}),
	}
	db, mock := openMockDB(t)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "test_scope_models"`)).
		WithoutArgs().
		WillReturnRows(sqlmock.NewRows([]string{"count(*)"}).AddRow(19))
	mock.ExpectExec(`SAVEPOINT .+`).WithoutArgs().WillReturnResult(sqlmock.NewResult(0, 0))
	// Actual models
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT "test_scope_models"."name","test_scope_models"."email",(UPPER("test_scope_models".name)) "computed","test_scope_models"."extra","test_scope_models"."id","test_scope_models"."relation_id" FROM "test_scope_models" ORDER BY LOWER("test_scope_models"."name") LIMIT $1`)).
		WithArgs(DefaultPageSize).
		WillReturnRows(sqlmock.NewRows([]string{"name", "email", "computed", "extra", "id", "relation_id"}).AddRow("name", "email@example.org", "NAME", "extra", 123, 456))
	mock.ExpectCommit()

	results := []*TestScopeModel{}
	settings := &Settings[*TestScopeModel]{
		CaseInsensitiveSort: true,
		Whitelist:           Whitelist{Fields: AllowAllFields},
	}
	paginator, err := settings.Scope(db, request, &results)
	assert.NotNil(t, paginator)
	assert.NoError(t, err)

	expected := map[string]clause.Clause{
		"ORDER BY": {
			Name: "ORDER BY",
			Expression: clause.OrderBy{
				Columns: []clause.OrderByColumn{
					{
						Column: clause.Column{
							Raw:  true,
							Name: `LOWER("test_scope_models"."name")`,
						},
					},
				},
			},
		},
		"FROM": expectFromClause,
		"SELECT": {
			Name: "SELECT",
			Expression: clause.Select{
				Columns: []clause.Column{
					{Raw: true, Name: `"test_scope_models"."name"`},
					{Raw: true, Name: `"test_scope_models"."email"`},
					{Raw: true, Name: `(UPPER("test_scope_models".name)) "computed"`},
					{Raw: true, Name: `"test_scope_models"."extra"`},
					{Raw: true, Name: `"test_scope_models"."id"`},
					{Raw: true, Name: `"test_scope_models"."relation_id"`},
				},
			},
		},
		"LIMIT": {
			Expression: clause.Limit{
				Limit:  new(10),
				Offset: 0,
			},
		},
	}
	assert.Equal(t, expected, paginator.DB.Statement.Clauses)
}

func TestSettingsGlobalScope(t *testing.T) {
	sqlExpect := func(mock sqlmock.Sqlmock) {
		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "test_scope_models" WHERE ("test_scope_models"."name" LIKE $1 AND "test_scope_models"."name" LIKE $2) OR "test_scope_models"."name" = $3`)).
			WithArgs("%val1%", "%val2%", "val3").
			WillReturnRows(sqlmock.NewRows([]string{"count(*)"}).AddRow(19))
		// Actual models
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT "test_scope_models"."name","test_scope_models"."email",(UPPER("test_scope_models".name)) "computed","test_scope_models"."id","test_scope_models"."relation_id" FROM "test_scope_models" WHERE ("test_scope_models"."name" LIKE $1 AND "test_scope_models"."name" LIKE $2) OR "test_scope_models"."name" = $3 ORDER BY "test_scope_models"."name" LIMIT $4 OFFSET $5`)).
			WithArgs("%val1%", "%val2%", "val3", 15, 15).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email", "computed", "relation_id"}).AddRow(123, "name", "email@example.org", "NAME", 456))
		// Don't preload relation, the default whitelist doesn't allow it
		mock.ExpectCommit()
	}

	paginator, err := runScope(t, nil, testRequest, sqlExpect)
	assert.NotNil(t, paginator)
	assert.NoError(t, err)

	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.AndConditions{
						Exprs: []clause.Expression{
							clause.OrConditions{
								Exprs: []clause.Expression{
									clause.AndConditions{
										Exprs: []clause.Expression{
											clause.Expr{SQL: `"test_scope_models"."name" LIKE ?`, Vars: []any{"%val1%"}},
											clause.Expr{SQL: `"test_scope_models"."name" LIKE ?`, Vars: []any{"%val2%"}},
										},
									},
								},
							},
							clause.OrConditions{
								Exprs: []clause.Expression{
									clause.AndConditions{
										Exprs: []clause.Expression{
											clause.Expr{SQL: `"test_scope_models"."name" = ?`, Vars: []any{"val3"}},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		"ORDER BY": expectOrderByClause,
		"LIMIT": {
			Expression: clause.Limit{
				Limit:  new(15),
				Offset: 15,
			},
		},
		"FROM":   expectFromClause,
		"SELECT": expectSelectClause,
	}
	assert.Equal(t, expected, paginator.DB.Statement.Clauses)
	assert.Empty(t, paginator.DB.Statement.Preloads)
	assert.Equal(t, expectSelects, paginator.DB.Statement.Selects)
}

func TestSettingsGlobalScopeUnpaginated(t *testing.T) {
	sqlExpect := func(mock sqlmock.Sqlmock) {
		// No Count query, no transaction
		// Actual models
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT "test_scope_models"."name","test_scope_models"."email",(UPPER("test_scope_models".name)) "computed","test_scope_models"."id","test_scope_models"."relation_id" FROM "test_scope_models" WHERE ("test_scope_models"."name" LIKE $1 AND "test_scope_models"."name" LIKE $2) OR "test_scope_models"."name" = $3 ORDER BY "test_scope_models"."name"`)).
			WithArgs("%val1%", "%val2%", "val3").
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email", "computed", "relation_id"}).AddRow(123, "name", "email@example.org", "NAME", 456))
		// Do not preload relation, the default whitelist doesn't allow it
	}

	results, db := runScopeUnpaginated(t, nil, testRequest, sqlExpect)
	assert.NotNil(t, results)
	assert.NoError(t, db.Error)

	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.AndConditions{
						Exprs: []clause.Expression{
							clause.OrConditions{
								Exprs: []clause.Expression{
									clause.AndConditions{
										Exprs: []clause.Expression{
											clause.Expr{SQL: `"test_scope_models"."name" LIKE ?`, Vars: []any{"%val1%"}},
											clause.Expr{SQL: `"test_scope_models"."name" LIKE ?`, Vars: []any{"%val2%"}},
										},
									},
								},
							},
							clause.OrConditions{
								Exprs: []clause.Expression{
									clause.AndConditions{
										Exprs: []clause.Expression{
											clause.Expr{SQL: `"test_scope_models"."name" = ?`, Vars: []any{"val3"}},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		"ORDER BY": expectOrderByClause,
		"FROM":     expectFromClause,
		"SELECT":   expectSelectClause,
	}
	assert.Equal(t, expected, db.Statement.Clauses)
	assert.Empty(t, db.Statement.Preloads)
	assert.Equal(t, expectSelects, db.Statement.Selects)
}

func TestNewRequest(t *testing.T) {
	cases := []struct {
		query map[string]any
		want  *Request
		desc  string
	}{
		{
			desc: "all_fields",
			query: map[string]any{
				QueryParamFilter: []*Filter{
					{Field: "name", Args: []string{"val1"}, Operator: Operators["$cont"]},
					{Field: "name", Args: []string{"val2"}, Operator: Operators["$cont"]},
				},
				QueryParamOr: []*Filter{
					{Field: "name", Args: []string{"val3"}, Or: true, Operator: Operators["$eq"]},
				},
				QueryParamSort:    []*Sort{{Field: "name", Order: SortDescending}},
				QueryParamJoin:    []*Join{{Relation: "Relation", Fields: []string{"a", "b"}}},
				QueryParamPage:    2,
				QueryParamPerPage: 15,
				QueryParamFields:  []string{"id", "name", "email", "computed"},
				QueryParamSearch:  "val",
			},
			want: &Request{
				Filter: typeutil.NewUndefined([]*Filter{
					{Field: "name", Args: []string{"val1"}, Operator: Operators["$cont"]},
					{Field: "name", Args: []string{"val2"}, Operator: Operators["$cont"]},
				}),
				Or: typeutil.NewUndefined([]*Filter{
					{Field: "name", Args: []string{"val3"}, Or: true, Operator: Operators["$eq"]},
				}),
				Sort:    typeutil.NewUndefined([]*Sort{{Field: "name", Order: SortDescending}}),
				Join:    typeutil.NewUndefined([]*Join{{Relation: "Relation", Fields: []string{"a", "b"}}}),
				Page:    typeutil.NewUndefined(2),
				PerPage: typeutil.NewUndefined(15),
				Fields:  typeutil.NewUndefined([]string{"id", "name", "email", "computed"}),
				Search:  typeutil.NewUndefined("val"),
			},
		},
		{
			desc: "partial",
			query: map[string]any{
				QueryParamFilter: []*Filter{
					{Field: "name", Args: []string{"val1"}, Operator: Operators["$cont"]},
					{Field: "name", Args: []string{"val2"}, Operator: Operators["$cont"]},
				},
				QueryParamSort:    []*Sort{{Field: "name", Order: SortDescending}},
				QueryParamPerPage: 15,
			},
			want: &Request{
				Filter: typeutil.NewUndefined([]*Filter{
					{Field: "name", Args: []string{"val1"}, Operator: Operators["$cont"]},
					{Field: "name", Args: []string{"val2"}, Operator: Operators["$cont"]},
				}),
				Sort:    typeutil.NewUndefined([]*Sort{{Field: "name", Order: SortDescending}}),
				PerPage: typeutil.NewUndefined(15),
			},
		},
		{
			desc: "incorrect_type",
			query: map[string]any{
				QueryParamFilter:  "a",
				QueryParamOr:      "b",
				QueryParamSort:    "c",
				QueryParamJoin:    "d",
				QueryParamPage:    "e",
				QueryParamPerPage: "f",
				QueryParamFields:  "g",
				QueryParamSearch:  1,
			},
			want: &Request{},
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			request := NewRequest(c.query)
			assert.Equal(t, c.want, request)
		})
	}
}

func TestWhitelistAllowed(t *testing.T) {
	cases := []struct {
		desc      string
		field     *schema.Field
		whitelist Whitelist
		want      bool
	}{
		{
			desc:      "all",
			whitelist: Whitelist{Fields: AllowAllFields},
			field:     &schema.Field{DBName: "column", Name: "Column"},
			want:      true,
		},
		{
			desc:      "using_dbname",
			whitelist: Whitelist{Fields: []string{"column"}},
			field:     &schema.Field{DBName: "column", Name: "Column"},
			want:      true,
		},
		{
			desc:      "using_name",
			whitelist: Whitelist{Fields: []string{"Column"}},
			field:     &schema.Field{DBName: "column", Name: "Column"},
			want:      true,
		},
		{
			desc:      "not_allowed",
			whitelist: Whitelist{Fields: []string{"FieldName"}},
			field:     &schema.Field{DBName: "column", Name: "Column"},
			want:      false,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			assert.Equal(t, c.want, c.whitelist.allowed(c.field))
		})
	}
}

func TestWhitelistNothing(t *testing.T) {
	settings := &Settings[*TestScopeModel]{
		FieldsSearch: []string{"email"},
		Whitelist:    Whitelist{},
		SearchOperator: &Operator{
			Function: func(tx *gorm.DB, filter *Filter, column string, _ DataType) *gorm.DB {
				return tx.Or(fmt.Sprintf("%s LIKE (?)", column), filter.Args[0])
			},
			RequiredArguments: 1,
		},
	}

	sqlExpect := func(mock sqlmock.Sqlmock) {
		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "test_scope_models" WHERE "test_scope_models"."email" LIKE ($1)`)).
			WithArgs("val").
			WillReturnRows(sqlmock.NewRows([]string{"count(*)"}).AddRow(19))
		mock.ExpectExec(`SAVEPOINT .+`).WithoutArgs().WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT 1,2 FROM "test_scope_models" WHERE "test_scope_models"."email" LIKE ($1) LIMIT $2 OFFSET $3`)).
			WithArgs("val", 15, 15).
			WillReturnRows(sqlmock.NewRows([]string{"1", "2"}).AddRow(1, 2))
		mock.ExpectCommit()
	}

	paginator, err := runScope(t, settings, testRequest, sqlExpect)
	assert.NotNil(t, paginator)
	assert.NoError(t, err)

	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.AndConditions{
						Exprs: []clause.Expression{
							clause.Expr{
								SQL:                `"test_scope_models"."email" LIKE (?)`, // Search operator override
								Vars:               []any{"val"},
								WithoutParentheses: false,
							},
						},
					},
				},
			},
		},
		"LIMIT": {
			Expression: clause.Limit{
				Limit:  new(15),
				Offset: 15,
			},
		},
		"FROM": expectFromClause,
		"SELECT": {
			Name: "SELECT",
			Expression: clause.Select{
				Columns: []clause.Column{
					{Raw: true, Name: `1`},
					{Raw: true, Name: `2`},
				},
			},
		},
	}
	assert.Equal(t, expected, paginator.DB.Statement.Clauses)
	assert.Empty(t, paginator.DB.Statement.Preloads)
	assert.Equal(t, []string{"1", "2"}, paginator.DB.Statement.Selects)
}
