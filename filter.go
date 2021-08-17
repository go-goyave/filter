package filter

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
	"goyave.dev/goyave/v3"
	"goyave.dev/goyave/v3/database"
)

type Filter struct {
	Field    string
	Operator *Operator
	Args     []string
	Or       bool
}

type Sort struct {
	Field string
	Order SortOrder
}

type Join struct {
	Relation string
	Fields   []string
}

type SortOrder string

const (
	DefaultPageSize = 10

	SortAscending  SortOrder = "ASC"
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

func Scope(db *gorm.DB, request *goyave.Request, dest interface{}) (*database.Paginator, *gorm.DB) {

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
		db.Scopes(selectScope(strings.Split(request.String("fields"), ",")))
	}

	return paginator, paginator.Find()

}

func (f *Filter) Scope(tx *gorm.DB) *gorm.DB {
	return f.Operator.Function(tx, f)
}

func (s *Sort) Scope(tx *gorm.DB) *gorm.DB {
	field := escape(tx, s.Field)
	if !strings.Contains(field, ".") {
		field = getTableName(tx) + field
	}
	return tx.Order(fmt.Sprintf("%s %s", field, s.Order))
}

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

func escape(tx *gorm.DB, str string) string {
	var f strings.Builder
	tx.QuoteTo(&f, str)
	return f.String()
}
