package filter

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
	"goyave.dev/goyave/v3"
	"goyave.dev/goyave/v3/database"
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

const (
	// DefaultPageSize the default pagination page size if the "per_page" query param
	// isn't provided.
	DefaultPageSize = 10

	// SortAscending "ORDER BY column ASC"
	SortAscending SortOrder = "ASC"
	// SortDescending "ORDER BY column DESC"
	SortDescending SortOrder = "DESC"
)

func selectScope(fields []string) func(*gorm.DB) *gorm.DB {
	return func(tx *gorm.DB) *gorm.DB {
		// TODO ability to specify fields that cannot be selected / sorted by, joined, etc
		// Prevent "select *" to remove the fields that are blacklisted
		fieldsWithTableName := make([]string, 0, len(fields))
		for _, f := range fields {
			fieldsWithTableName = append(fieldsWithTableName, getTableName(tx)+f)
		}
		return tx.Select(fieldsWithTableName)
	}
}

// Scope apply all filters, sorts and joins defined in the request's data to the given `*gorm.DB`
// and process pagination. Returns the resulting `*database.Paginator` and the `*gorm.DB` result,
// which can be used to check for database errors.
// The given request is expected to be validated using `ApplyValidation`.
func Scope(db *gorm.DB, request *goyave.Request, dest interface{}) (*database.Paginator, *gorm.DB) {
	modelIdentity := parseModel(db, dest)

	for _, queryParam := range []string{"filter", "or"} {
		if request.Has(queryParam) {
			filters, ok := request.Data[queryParam].([]*Filter)
			if ok {
				for _, f := range filters {
					db = db.Scopes(f.Scope)
				}
			}
		}
	}

	if request.Has("sort") {
		sorts, ok := request.Data["sort"].([]*Sort)
		if ok {
			for _, s := range sorts {
				db = db.Scopes(s.Scope)
			}
		}
	}

	if request.Has("join") {
		joins, ok := request.Data["join"].([]*Join)
		if ok {
			for _, j := range joins {
				db = db.Scopes(j.Scope)
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
		// TODO validate fields exist
		// FIXME if field is a relation, should not be considered as existing
		// TODO if joins have fields, add them to the select scope
		fields := strings.Split(request.String("fields"), ",")
		fields = modelIdentity.cleanColumns(fields)
		db.Scopes(selectScope(fields))
	}

	return paginator, paginator.Find()

}

// Scope returns the GORM scope to use in order to apply this filter.
func (f *Filter) Scope(tx *gorm.DB) *gorm.DB {
	return f.Operator.Function(tx, f)
}

// Scope returns the GORM scope to use in order to apply sorting.
func (s *Sort) Scope(tx *gorm.DB) *gorm.DB {
	field := SQLEscape(tx, s.Field)
	if !strings.Contains(field, ".") {
		field = getTableName(tx) + field
	}
	return tx.Order(fmt.Sprintf("%s %s", field, s.Order))
}

// Scope returns the GORM scope to use in order to apply this joint.
func (j *Join) Scope(tx *gorm.DB) *gorm.DB {
	// FIXME If UserID not selected, cannot assign relation.
	// Manually find all relation IDs and add them if they're not here
	// if field has tag foreignKey, use that, otherwise use fieldName + "ID" OR fieldName + "Id"
	return tx.Preload(j.Relation, selectScope(j.Fields))

	// TODO joins with conditions (and may not want to select relation content)
}

func getTableName(tx *gorm.DB) string {
	if tx.Statement.Table != "" {
		return tx.Statement.Table + "."
	}

	if tx.Statement.Model != nil {
		stmt := &gorm.Statement{DB: tx}
		if err := stmt.Parse(tx.Statement.Model); err != nil {
			panic(err)
		}
		return stmt.Schema.Table + "."
	}

	return ""
}

// SQLEscape escape the given string to prevent SQL injection.
func SQLEscape(tx *gorm.DB, str string) string {
	var f strings.Builder
	tx.QuoteTo(&f, str)
	return f.String()
}
