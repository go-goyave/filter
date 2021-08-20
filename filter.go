package filter

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"goyave.dev/goyave/v3"
	"goyave.dev/goyave/v3/database"
	"goyave.dev/goyave/v3/helper"
)

// Filter structured representation of a filter query.
type Filter struct {
	Field    string
	Operator *Operator
	Args     []string
	Or       bool
}

// Sort structured representation of a sort query.
type Sort struct {
	Field string
	Order SortOrder
}

// Join structured representation of a join query.
type Join struct {
	Relation string
	Fields   []string
}

// SortOrder the allowed strings for SQL "ORDER BY" clause.
type SortOrder string

var (
	// DefaultPageSize the default pagination page size if the "per_page" query param
	// isn't provided.
	DefaultPageSize = 10
)

const (
	// SortAscending "ORDER BY column ASC"
	SortAscending SortOrder = "ASC"
	// SortDescending "ORDER BY column DESC"
	SortDescending SortOrder = "DESC"
)

func selectScope(fields []string) func(*gorm.DB) *gorm.DB {
	return func(tx *gorm.DB) *gorm.DB {

		if fields == nil {
			return tx
		}
		// TODO ability to specify fields that cannot be selected / sorted by, joined, etc
		// Prevent "select *" to remove the fields that are blacklisted

		var fieldsWithTableName []string
		if len(fields) == 0 {
			fieldsWithTableName = []string{"1"}
		} else {
			fieldsWithTableName = make([]string, 0, len(fields))
			for _, f := range fields {
				tableName := getTableName(tx)
				if tableName != "" {
					tableName = SQLEscape(tx, tableName) + "."
				}
				fieldsWithTableName = append(fieldsWithTableName, tableName+f)
			}
		}
		return tx.Select(fieldsWithTableName)
	}
}

// Scope apply all filters, sorts and joins defined in the request's data to the given `*gorm.DB`
// and process pagination. Returns the resulting `*database.Paginator` and the `*gorm.DB` result,
// which can be used to check for database errors.
// The given request is expected to be validated using `ApplyValidation`.
func Scope(db *gorm.DB, request *goyave.Request, dest interface{}) (*database.Paginator, *gorm.DB) {
	// TODO ability to disable certain features (disable sort, join, etc)
	modelIdentity := parseModel(db, dest)

	db = applyFilters(db, request, modelIdentity)

	if request.Has("sort") {
		sorts, ok := request.Data["sort"].([]*Sort)
		if ok {
			for _, s := range sorts {
				if scope := s.Scope(modelIdentity); scope != nil {
					db = db.Scopes(scope)
				}
			}
		}
	}

	if request.Has("join") {
		joins, ok := request.Data["join"].([]*Join)
		if ok {
			for _, j := range joins {
				if s := j.Scope(modelIdentity); s != nil {
					db = db.Scopes(s)
				}
			}
		}
	}

	page := 1
	if request.Has("page") {
		page = request.Integer("page")
	}
	pageSize := DefaultPageSize
	if request.Has("per_page") {
		pageSize = request.Integer("per_page")
	}

	paginator := database.NewPaginator(db, page, pageSize, dest)
	paginator.UpdatePageInfo()

	if request.Has("fields") {
		fields := strings.Split(request.String("fields"), ",")
		paginator.DB = db.Scopes(selectScope(modelIdentity.cleanColumns(fields)))
	}

	return paginator, paginator.Find()
}

func applyFilters(db *gorm.DB, request *goyave.Request, modelIdentity *modelIdentity) *gorm.DB {
	for _, queryParam := range []string{"filter", "or"} {
		if request.Has(queryParam) {
			filters, ok := request.Data[queryParam].([]*Filter)
			if ok {
				for _, f := range filters {
					if s := f.Scope(modelIdentity); s != nil {
						db = db.Scopes(s)
					}
				}
			}
		}
	}
	return db
}

// Scope returns the GORM scope to use in order to apply this filter.
func (f *Filter) Scope(modelIdentity *modelIdentity) func(*gorm.DB) *gorm.DB {
	_, ok := modelIdentity.Columns[f.Field]
	if !ok {
		return nil
	}
	return func(tx *gorm.DB) *gorm.DB {
		return f.Operator.Function(tx, f)
	}
}

// Where applies a condition to given transaction, automatically taking the "Or"
// filter value into account.
func (f *Filter) Where(tx *gorm.DB, query string, args ...interface{}) *gorm.DB {
	if f.Or {
		return tx.Or(query, args...)
	}
	return tx.Where(query, args...)
}

// Scope returns the GORM scope to use in order to apply sorting.
func (s *Sort) Scope(modelIdentity *modelIdentity) func(*gorm.DB) *gorm.DB {
	_, ok := modelIdentity.Columns[s.Field]
	if !ok {
		return nil
	}

	return func(tx *gorm.DB) *gorm.DB {
		field := SQLEscape(tx, s.Field)
		if !strings.Contains(field, ".") {
			field = getTableName(tx) + field
		}
		return tx.Order(fmt.Sprintf("%s %s", field, s.Order))
	}
}

// Scope returns the GORM scope to use in order to apply this joint.
func (j *Join) Scope(modelIdentity *modelIdentity) func(*gorm.DB) *gorm.DB {
	relationIdentity, ok := modelIdentity.Relations[j.Relation]
	if !ok {
		return nil
	}

	columns := relationIdentity.cleanColumns(j.Fields)

	return func(tx *gorm.DB) *gorm.DB {
		if columns != nil {
			switch relationIdentity.Type {
			case schema.HasOne:
				if len(relationIdentity.PrimaryKeys) == 0 {
					tx.AddError(fmt.Errorf("Could not find primary key. Add `gorm:\"primaryKey\" to your model`"))
					return tx
				}
				for _, k := range relationIdentity.PrimaryKeys {
					if !helper.ContainsStr(columns, k) {
						columns = append(columns, k)
					}
				}
				for _, f := range relationIdentity.findForeignKey(tx, j.Relation, relationIdentity) {
					if !helper.ContainsStr(columns, f) {
						columns = append(columns, f)
					}
				}

			case schema.HasMany:
				for _, v := range relationIdentity.ForeignKeys {
					if !helper.ContainsStr(columns, v) {
						columns = append(columns, v)
					}
				}
			}
		}
		return tx.Preload(j.Relation, selectScope(columns))
	}

	// TODO joins with conditions (and may not want to select relation content)
}

func getTableName(tx *gorm.DB) string {
	if tx.Statement.Table != "" {
		return tx.Statement.Table
	}

	if tx.Statement.Model != nil {
		stmt := &gorm.Statement{DB: tx}
		if err := stmt.Parse(tx.Statement.Model); err != nil {
			tx.AddError(err)
			return ""
		}
		return stmt.Schema.Table
	}

	return ""
}

// SQLEscape escape the given string to prevent SQL injection.
func SQLEscape(tx *gorm.DB, str string) string {
	var f strings.Builder
	tx.QuoteTo(&f, str)
	return f.String()
}
