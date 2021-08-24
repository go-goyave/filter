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
	Blacklist

	// DisableFields ignore the "fields" query if true.
	DisableFields bool
	// DisableFilter ignore the "filter" query if true.
	DisableFilter bool
	// DisableSort ignore the "sort" query if true.
	DisableSort bool
	// DisableJoin ignore the "join" query if true.
	DisableJoin bool
}

// Blacklist definition of blacklisted relations and fields.
type Blacklist struct {
	Relations map[string]*Blacklist

	// FieldsBlacklist prevent the fields in this list to be selected or to
	// be used in filters and sorts.
	FieldsBlacklist []string
	// RelationsBlacklist prevent joining the relations in this list.
	RelationsBlacklist []string

	// IsFinal if true, prevent joining any relation
	IsFinal bool
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
	selectCache map[string][]string
	Relation    string
	Fields      []string
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
			selectCache := map[string][]string{}
			for _, j := range joins {
				hasJoins = true
				j.selectCache = selectCache
				if s := j.Scopes(s, modelIdentity); s != nil {
					db = db.Scopes(s...)
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
		paginator.DB = db.Scopes(selectScope(modelIdentity, modelIdentity.cleanColumns(fields, s.FieldsBlacklist)))
	} else {
		paginator.DB = db.Scopes(selectScope(modelIdentity, s.getSelectableFields(modelIdentity.Columns)))
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

func (b *Blacklist) getSelectableFields(fields map[string]*column) []string {
	if b.FieldsBlacklist == nil {
		return nil
	}
	columns := make([]string, 0, len(fields))
	for k := range fields {
		if !helper.ContainsStr(b.FieldsBlacklist, k) {
			columns = append(columns, k)
		}
	}

	return columns
}

func selectScope(modelIdentity *modelIdentity, fields []string) func(*gorm.DB) *gorm.DB {
	return func(tx *gorm.DB) *gorm.DB {

		if fields == nil {
			return tx
		}

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

// Scopes returns the GORM scopes to use in order to apply this joint.
func (j *Join) Scopes(settings *Settings, modelIdentity *modelIdentity) []func(*gorm.DB) *gorm.DB {
	// TODO joins with conditions (and may not want to select relation content)
	scopes := j.applyRelation(modelIdentity, &settings.Blacklist, j.Relation, 0, make([]func(*gorm.DB) *gorm.DB, 0, strings.Count(j.Relation, ".")+1))
	if scopes != nil {
		return scopes
	}
	return nil
}

func (j *Join) applyRelation(modelIdentity *modelIdentity, blacklist *Blacklist, relationName string, startIndex int, scopes []func(*gorm.DB) *gorm.DB) []func(*gorm.DB) *gorm.DB {
	if blacklist != nil && blacklist.IsFinal {
		return nil
	}
	trimmedRelationName := relationName[startIndex:]
	i := strings.Index(trimmedRelationName, ".")
	if i == -1 {
		if blacklist != nil && helper.ContainsStr(blacklist.RelationsBlacklist, trimmedRelationName) {
			return nil
		}

		b, _ := blacklist.Relations[trimmedRelationName]
		r, ok := modelIdentity.Relations[trimmedRelationName]
		if !ok {
			return nil
		}

		j.selectCache[relationName] = j.Fields
		return append(scopes, joinScope(relationName, r, j.Fields, b))
	}

	if startIndex+i+1 >= len(relationName) {
		return nil
	}

	name := trimmedRelationName[:i]
	var b *Blacklist
	if blacklist != nil {
		if helper.ContainsStr(blacklist.RelationsBlacklist, name) {
			return nil
		}
		b, _ = blacklist.Relations[name]
	}
	r, ok := modelIdentity.Relations[name]
	if !ok {
		return nil
	}
	n := relationName[:startIndex+i]
	fields := []string{}
	if f, ok := j.selectCache[n]; ok {
		fields = f
	}
	scopes = append(scopes, joinScope(n, r, fields, b))

	return j.applyRelation(r.modelIdentity, b, relationName, startIndex+i+1, scopes)
}

func joinScope(relationName string, relationIdentity *relation, fields []string, blacklist *Blacklist) func(*gorm.DB) *gorm.DB {
	var b []string
	if blacklist != nil {
		b = blacklist.FieldsBlacklist
	}
	var columns []string
	if fields != nil {
		columns = relationIdentity.cleanColumns(fields, b)
	} else if blacklist != nil {
		columns = blacklist.getSelectableFields(relationIdentity.Columns)
	}

	return func(tx *gorm.DB) *gorm.DB {
		if columns != nil {
			if len(relationIdentity.PrimaryKeys) == 0 {
				tx.AddError(fmt.Errorf("Could not find %q relation's primary key. Add `gorm:\"primaryKey\"` to your model", relationName))
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
		return tx.Preload(relationName, selectScope(relationIdentity.modelIdentity, columns))
	}
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
