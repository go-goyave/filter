package filter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
	"gorm.io/gorm/utils/tests"
)

func TestEquals(t *testing.T) {
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
	db = Operators["$eq"].Function(db, &Filter{Field: "name", Args: []string{"test"}}, "`test_models`.`name`", schema.String)

	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.Expr{SQL: "`test_models`.`name` = ?", Vars: []interface{}{"test"}},
				},
			},
		},
	}
	assert.Equal(t, expected, db.Statement.Clauses)
}

func TestNotEquals(t *testing.T) {
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
	db = Operators["$ne"].Function(db, &Filter{Field: "name", Args: []string{"test"}}, "`test_models`.`name`", schema.String)

	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.Expr{SQL: "`test_models`.`name` <> ?", Vars: []interface{}{"test"}},
				},
			},
		},
	}
	assert.Equal(t, expected, db.Statement.Clauses)
}

func TestGreaterThan(t *testing.T) {
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
	db = Operators["$gt"].Function(db, &Filter{Field: "age", Args: []string{"18"}}, "`test_models`.`age`", schema.String)

	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.Expr{SQL: "`test_models`.`age` > ?", Vars: []interface{}{"18"}},
				},
			},
		},
	}
	assert.Equal(t, expected, db.Statement.Clauses)
}

func TestLowerThan(t *testing.T) {
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
	db = Operators["$lt"].Function(db, &Filter{Field: "age", Args: []string{"18"}}, "`test_models`.`age`", schema.String)

	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.Expr{SQL: "`test_models`.`age` < ?", Vars: []interface{}{"18"}},
				},
			},
		},
	}
	assert.Equal(t, expected, db.Statement.Clauses)
}

func TestGreaterThanEqual(t *testing.T) {
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
	db = Operators["$gte"].Function(db, &Filter{Field: "age", Args: []string{"18"}}, "`test_models`.`age`", schema.String)

	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.Expr{SQL: "`test_models`.`age` >= ?", Vars: []interface{}{"18"}},
				},
			},
		},
	}
	assert.Equal(t, expected, db.Statement.Clauses)
}

func TestLowerThanEqual(t *testing.T) {
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
	db = Operators["$lte"].Function(db, &Filter{Field: "age", Args: []string{"18"}}, "`test_models`.`age`", schema.String)

	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.Expr{SQL: "`test_models`.`age` <= ?", Vars: []interface{}{"18"}},
				},
			},
		},
	}
	assert.Equal(t, expected, db.Statement.Clauses)
}

func TestStarts(t *testing.T) {
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
	db = Operators["$starts"].Function(db, &Filter{Field: "name", Args: []string{"test"}}, "`test_models`.`name`", schema.String)

	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.Expr{SQL: "`test_models`.`name` LIKE ?", Vars: []interface{}{"test%"}},
				},
			},
		},
	}
	assert.Equal(t, expected, db.Statement.Clauses)
}

func TestEnds(t *testing.T) {
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
	db = Operators["$ends"].Function(db, &Filter{Field: "name", Args: []string{"test"}}, "`test_models`.`name`", schema.String)

	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.Expr{SQL: "`test_models`.`name` LIKE ?", Vars: []interface{}{"%test"}},
				},
			},
		},
	}
	assert.Equal(t, expected, db.Statement.Clauses)
}

func TestContains(t *testing.T) {
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
	db = Operators["$cont"].Function(db, &Filter{Field: "name", Args: []string{"test"}}, "`test_models`.`name`", schema.String)

	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.Expr{SQL: "`test_models`.`name` LIKE ?", Vars: []interface{}{"%test%"}},
				},
			},
		},
	}
	assert.Equal(t, expected, db.Statement.Clauses)
}

func TestNotContains(t *testing.T) {
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
	db = Operators["$excl"].Function(db, &Filter{Field: "name", Args: []string{"test"}}, "`test_models`.`name`", schema.String)

	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.Expr{SQL: "`test_models`.`name` NOT LIKE ?", Vars: []interface{}{"%test%"}},
				},
			},
		},
	}
	assert.Equal(t, expected, db.Statement.Clauses)
}

func TestIn(t *testing.T) {
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
	db = Operators["$in"].Function(db, &Filter{Field: "name", Args: []string{"val1", "val2"}}, "`test_models`.`name`", schema.String)

	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.Expr{SQL: "`test_models`.`name` IN ?", Vars: []interface{}{[]string{"val1", "val2"}}},
				},
			},
		},
	}
	assert.Equal(t, expected, db.Statement.Clauses)
}

func TestNotIn(t *testing.T) {
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
	db = Operators["$notin"].Function(db, &Filter{Field: "name", Args: []string{"val1", "val2"}}, "`test_models`.`name`", schema.String)

	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.Expr{SQL: "`test_models`.`name` NOT IN ?", Vars: []interface{}{[]string{"val1", "val2"}}},
				},
			},
		},
	}
	assert.Equal(t, expected, db.Statement.Clauses)
}

func TestIsNull(t *testing.T) {
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
	db = Operators["$isnull"].Function(db, &Filter{Field: "name"}, "`test_models`.`name`", schema.String)

	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.Expr{SQL: "`test_models`.`name` IS NULL"},
				},
			},
		},
	}
	assert.Equal(t, expected, db.Statement.Clauses)
}

func TestNotNull(t *testing.T) {
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
	db = Operators["$notnull"].Function(db, &Filter{Field: "name"}, "`test_models`.`name`", schema.String)

	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.Expr{SQL: "`test_models`.`name` IS NOT NULL"},
				},
			},
		},
	}
	assert.Equal(t, expected, db.Statement.Clauses)
}

func TestBetween(t *testing.T) {
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
	db = Operators["$between"].Function(db, &Filter{Field: "age", Args: []string{"18", "25"}}, "`test_models`.`age`", schema.Uint)

	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.Expr{SQL: "`test_models`.`age` BETWEEN ? AND ?", Vars: []interface{}{"18", "25"}},
				},
			},
		},
	}
	assert.Equal(t, expected, db.Statement.Clauses)
}

func TestIsTrue(t *testing.T) {
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
	db = Operators["$istrue"].Function(db, &Filter{Field: "isActive"}, "`test_models`.`is_active`", schema.Bool)

	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.Expr{SQL: "`test_models`.`is_active` IS TRUE"},
				},
			},
		},
	}
	assert.Equal(t, expected, db.Statement.Clauses)

	db, _ = gorm.Open(&tests.DummyDialector{}, nil)
	db = Operators["$istrue"].Function(db, &Filter{Field: "isActive"}, "`test_models`.`is_active`", schema.String) // Unsupported type
	assert.Empty(t, db.Statement.Clauses)
}

func TestIsFalse(t *testing.T) {
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
	db = Operators["$isfalse"].Function(db, &Filter{Field: "isActive"}, "`test_models`.`is_active`", schema.Bool)

	expected := map[string]clause.Clause{
		"WHERE": {
			Name: "WHERE",
			Expression: clause.Where{
				Exprs: []clause.Expression{
					clause.Expr{SQL: "`test_models`.`is_active` IS FALSE"},
				},
			},
		},
	}
	assert.Equal(t, expected, db.Statement.Clauses)

	db, _ = gorm.Open(&tests.DummyDialector{}, nil)
	db = Operators["$isfalse"].Function(db, &Filter{Field: "isActive"}, "`test_models`.`is_active`", schema.String) // Unsupported type
	assert.Empty(t, db.Statement.Clauses)
}
