package filter

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
	"goyave.dev/goyave/v3"
	"goyave.dev/goyave/v3/database"
	"goyave.dev/goyave/v3/helper"
)

// Settings settings to disable certain features and/or blacklist fields
// and relations.
type Settings struct {
	// FieldsBlacklist prevent the fields in this list to be selected or to
	// be used in filters and sorts. To blacklist relation fields, use a
	// dot-separated syntax: "Relation.field"
	FieldsBlacklist []string
	// RelationsBlacklist prevent joining the relations in this list.
	RelationsBlacklist []string
	// DisableFields ignore the "fields" query if true.
	DisableFields bool
	// DisableFilter ignore the "filter" query if true.
	DisableFilter bool
	// DisableSort ignore the "sort" query if true.
	DisableSort bool
	// DisableJoin ignore the "join" query if true.
	DisableJoin bool
}

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

// Scope using the default FilterSettings. See `FilterSettings.Scope()` for more details.
func Scope(db *gorm.DB, request *goyave.Request, dest interface{}) (*database.Paginator, *gorm.DB) {
	return (&Settings{}).Scope(db, request, dest)
}

// Scope apply all filters, sorts and joins defined in the request's data to the given `*gorm.DB`
// and process pagination. Returns the resulting `*database.Paginator` and the `*gorm.DB` result,
// which can be used to check for database errors.
// The given request is expected to be validated using `ApplyValidation`.
func (s *Settings) Scope(db *gorm.DB, request *goyave.Request, dest interface{}) (*database.Paginator, *gorm.DB) {
	modelIdentity := parseModel(db, dest)

	db = s.applyFilters(db, request, modelIdentity)

	if !s.DisableSort && request.Has("sort") {
		sorts, ok := request.Data["sort"].([]*Sort)
		if ok {
			for _, sort := range sorts {
				if scope := sort.Scope(s, modelIdentity); scope != nil {
					db = db.Scopes(scope)
				}
			}
		}
	}

	hasJoins := false
	if !s.DisableJoin && request.Has("join") {
		joins, ok := request.Data["join"].([]*Join)
		if ok {
			for _, j := range joins {
				hasJoins = true
				if s := j.Scope(s, modelIdentity); s != nil {
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

	if !s.DisableFields && request.Has("fields") {
		fields := strings.Split(request.String("fields"), ",")
		if hasJoins {
			if len(modelIdentity.PrimaryKeys) == 0 {
				db.AddError(fmt.Errorf("Could not find primary key. Add `gorm:\"primaryKey\"` to your model"))
				return nil, db
			}
			fields = modelIdentity.addPrimaryKeys(fields)
		}
		paginator.DB = db.Scopes(s.selectScope(modelIdentity.cleanColumns(fields)))
	}

	return paginator, paginator.Find()
}

func (s *Settings) applyFilters(db *gorm.DB, request *goyave.Request, modelIdentity *modelIdentity) *gorm.DB {
	if s.DisableFilter {
		return db
	}
	for _, queryParam := range []string{"filter", "or"} {
		if request.Has(queryParam) {
			filters, ok := request.Data[queryParam].([]*Filter)
			if ok {
				for _, f := range filters {
					if s := f.Scope(s, modelIdentity); s != nil {
						db = db.Scopes(s)
					}
				}
			}
		}
	}
	return db
}

func (s *Settings) selectScope(fields []string) func(*gorm.DB) *gorm.DB {
	return func(tx *gorm.DB) *gorm.DB {

		if fields == nil {
			return tx
		}
		// TODO Prevent "select *" to remove the fields that are blacklisted

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
				fieldsWithTableName = append(fieldsWithTableName, tableName+SQLEscape(tx, f))
			}
		}
		return tx.Select(fieldsWithTableName)
	}
}

// Scope returns the GORM scope to use in order to apply this filter.
func (f *Filter) Scope(settings *Settings, modelIdentity *modelIdentity) func(*gorm.DB) *gorm.DB {
	if helper.ContainsStr(settings.FieldsBlacklist, f.Field) {
		return nil
	}
	_, ok := modelIdentity.Columns[f.Field]
	if !ok {
		return nil
	}
	// TODO filter on relation
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
func (s *Sort) Scope(settings *Settings, modelIdentity *modelIdentity) func(*gorm.DB) *gorm.DB {
	if helper.ContainsStr(settings.FieldsBlacklist, s.Field) {
		return nil
	}
	_, ok := modelIdentity.Columns[s.Field]
	if !ok {
		return nil
	}

	return func(tx *gorm.DB) *gorm.DB {
		field := s.Field
		table := getTableName(tx)
		c := clause.OrderByColumn{
			Column: clause.Column{
				Table: table,
				Name:  field,
			},
			Desc: s.Order == SortDescending,
		}
		return tx.Order(c)
	}
}

// Scope returns the GORM scope to use in order to apply this joint.
func (j *Join) Scope(settings *Settings, modelIdentity *modelIdentity) func(*gorm.DB) *gorm.DB {
	if helper.ContainsStr(settings.RelationsBlacklist, j.Relation) {
		return nil
	}
	relationIdentity, ok := modelIdentity.Relations[j.Relation]
	if !ok {
		return nil
	}

	columns := relationIdentity.cleanColumns(j.Fields)
	// TODO handle fields blacklist

	return func(tx *gorm.DB) *gorm.DB {
		if columns != nil {
			if len(relationIdentity.PrimaryKeys) == 0 {
				tx.AddError(fmt.Errorf("Could not find %q relation's primary key. Add `gorm:\"primaryKey\"` to your model", j.Relation))
				return tx
			}
			for _, k := range relationIdentity.PrimaryKeys {
				if !helper.ContainsStr(columns, k) {
					columns = append(columns, k)
				}
			}
			if relationIdentity.Type == schema.HasMany {
				for _, v := range relationIdentity.ForeignKeys {
					if !helper.ContainsStr(columns, v) {
						columns = append(columns, v)
					}
				}
			}
		}
		return tx.Preload(j.Relation, settings.selectScope(columns))
	}

	// TODO joins with conditions (and may not want to select relation content)
	// TODO handle nested relations
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
	tx.QuoteTo(&f, strings.TrimSpace(str))
	return f.String()
}
