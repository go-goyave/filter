package filter

import (
	"fmt"
	"strings"
	"sync"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
	"goyave.dev/goyave/v4"
	"goyave.dev/goyave/v4/database"
	"goyave.dev/goyave/v4/util/sliceutil"
)

// Settings settings to disable certain features and/or blacklist fields
// and relations.
type Settings struct {
	// FieldsSearch allows search for these fields
	FieldsSearch []string
	// SearchOperator is used by the search scope, by default it use the $cont operator
	SearchOperator *Operator

	Blacklist

	// DisableFields ignore the "fields" query if true.
	DisableFields bool
	// DisableFilter ignore the "filter" query if true.
	DisableFilter bool
	// DisableSort ignore the "sort" query if true.
	DisableSort bool
	// DisableJoin ignore the "join" query if true.
	DisableJoin bool
	// DisableSearch ignore the "search" query if true.
	DisableSearch bool
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

var (
	// DefaultPageSize the default pagination page size if the "per_page" query param
	// isn't provided.
	DefaultPageSize = 10

	modelCache = &sync.Map{}
)

func parseModel(db *gorm.DB, model interface{}) (*schema.Schema, error) {
	return schema.Parse(model, modelCache, db.NamingStrategy)
}

// Scope using the default FilterSettings. See `FilterSettings.Scope()` for more details.
func Scope(db *gorm.DB, request *goyave.Request, dest interface{}) (*database.Paginator, *gorm.DB) {
	return (&Settings{}).Scope(db, request, dest)
}

// ScopeUnpaginated using the default FilterSettings. See `FilterSettings.ScopeUnpaginated()` for more details.
func ScopeUnpaginated(db *gorm.DB, request *goyave.Request, dest interface{}) *gorm.DB {
	return (&Settings{}).ScopeUnpaginated(db, request, dest)
}

// Scope apply all filters, sorts and joins defined in the request's data to the given `*gorm.DB`
// and process pagination. Returns the resulting `*database.Paginator` and the `*gorm.DB` result,
// which can be used to check for database errors.
// The given request is expected to be validated using `ApplyValidation`.
func (s *Settings) Scope(db *gorm.DB, request *goyave.Request, dest interface{}) (*database.Paginator, *gorm.DB) {
	db, schema, hasJoins := s.scopeCommon(db, request, dest)

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

	paginator.DB = s.scopeSort(paginator.DB, request, schema)
	if fieldsDB := s.scopeFields(paginator.DB, request, schema, hasJoins); fieldsDB != nil {
		paginator.DB = fieldsDB
	} else {
		return nil, paginator.DB
	}

	return paginator, paginator.Find()
}

// ScopeUnpaginated apply all filters, sorts and joins defined in the request's data to the given `*gorm.DB`
// without any pagination.
// Returns the `*gorm.DB` result, which can be used to check for database errors.
// The records will be added in the given `dest` slice.
// The given request is expected to be validated using `ApplyValidation`.
func (s *Settings) ScopeUnpaginated(db *gorm.DB, request *goyave.Request, dest interface{}) *gorm.DB {
	db, schema, hasJoins := s.scopeCommon(db, request, dest)
	db = s.scopeSort(db, request, schema)
	if fieldsDB := s.scopeFields(db, request, schema, hasJoins); fieldsDB != nil {
		db = fieldsDB
	} else {
		return db
	}
	return db.Find(dest)
}

// scopeCommon applies all scopes common to both the paginated and non-paginated requests.
// The third returned valued indicates if the query contains joins.
func (s *Settings) scopeCommon(db *gorm.DB, request *goyave.Request, dest interface{}) (*gorm.DB, *schema.Schema, bool) {
	schema, err := parseModel(db, dest)
	if err != nil {
		panic(err)
	}

	db = db.Model(dest)
	db = s.applyFilters(db, request, schema)

	hasJoins := false
	if !s.DisableJoin && request.Has("join") {
		joins, ok := request.Data["join"].([]*Join)
		if ok {
			selectCache := map[string][]string{}
			for _, j := range joins {
				hasJoins = true
				j.selectCache = selectCache
				if s := j.Scopes(s, schema); s != nil {
					db = db.Scopes(s...)
				}
			}
		}
	}

	if !s.DisableSearch && request.Has("search") {
		if search := s.applySearch(request, schema); search != nil {
			if scope := search.Scope(schema); scope != nil {
				db = db.Scopes(scope)
			}
		}
	}

	db.Scopes(func(tx *gorm.DB) *gorm.DB {
		// Convert all joins' selects to support computed columns
		// for manual Joins.
		processJoinsComputedColumns(tx.Statement, schema)
		return tx
	})

	return db, schema, hasJoins
}

func (s *Settings) scopeFields(db *gorm.DB, request *goyave.Request, schema *schema.Schema, hasJoins bool) *gorm.DB {
	if !s.DisableFields && request.Has("fields") {
		fields := strings.Split(request.String("fields"), ",")
		if hasJoins {
			if len(schema.PrimaryFieldDBNames) == 0 {
				db.AddError(fmt.Errorf("Could not find primary key. Add `gorm:\"primaryKey\"` to your model"))
				return nil
			}
			fields = addPrimaryKeys(schema, fields)
			fields = addForeignKeys(schema, fields)
		}
		return db.Scopes(selectScope(schema.Table, cleanColumns(schema, fields, s.FieldsBlacklist), false))
	}
	return db.Scopes(selectScope(schema.Table, getSelectableFields(&s.Blacklist, schema), false))
}

func (s *Settings) scopeSort(db *gorm.DB, request *goyave.Request, schema *schema.Schema) *gorm.DB {
	if !s.DisableSort && request.Has("sort") {
		sorts, ok := request.Data["sort"].([]*Sort)
		if ok {
			for _, sort := range sorts {
				if scope := sort.Scope(s, schema); scope != nil {
					db = db.Scopes(scope)
				}
			}
		}
	}
	return db
}

func (s *Settings) applyFilters(db *gorm.DB, request *goyave.Request, schema *schema.Schema) *gorm.DB {
	if s.DisableFilter {
		return db
	}
	filterScopes := make([]func(*gorm.DB) *gorm.DB, 0, 2)
	joinScopes := make([]func(*gorm.DB) *gorm.DB, 0, 2)

	andLen := filterLen(request, "filter")
	orLen := filterLen(request, "or")
	mixed := orLen > 1 && andLen > 0

	for _, queryParam := range []string{"filter", "or"} {
		if request.Has(queryParam) {
			filters, ok := request.Data[queryParam].([]*Filter)
			if ok {
				group := make([]func(*gorm.DB) *gorm.DB, 0, 4)
				for _, f := range filters {
					if mixed {
						f = &Filter{
							Field:    f.Field,
							Operator: f.Operator,
							Args:     f.Args,
							Or:       false,
						}
					}
					joinScope, conditionScope := f.Scope(s, schema)
					if conditionScope != nil {
						group = append(group, conditionScope)
					}
					if joinScope != nil {
						joinScopes = append(joinScopes, joinScope)
					}
				}
				filterScopes = append(filterScopes, groupFilters(group, false))
			}
		}
	}
	if len(joinScopes) > 0 {
		db = db.Scopes(joinScopes...)
	}
	if len(filterScopes) > 0 {
		db = db.Scopes(groupFilters(filterScopes, true))
	}
	return db
}

func filterLen(request *goyave.Request, name string) int {
	count := 0
	if data, ok := request.Data[name]; ok {
		if filters, ok := data.([]*Filter); ok {
			count = len(filters)
		}
	}
	return count
}

func groupFilters(scopes []func(*gorm.DB) *gorm.DB, and bool) func(*gorm.DB) *gorm.DB {
	return func(tx *gorm.DB) *gorm.DB {
		processedFilters := tx.Session(&gorm.Session{NewDB: true})
		for _, f := range scopes {
			processedFilters = f(processedFilters)
		}
		if and {
			return tx.Where(processedFilters)
		}
		return tx.Or(processedFilters)
	}
}

func (s *Settings) applySearch(request *goyave.Request, schema *schema.Schema) *Search {
	// Note: the search condition is not in a group condition (parenthesis)
	query, ok := request.Data["search"].(string)
	if ok {
		fields := s.FieldsSearch
		if fields == nil {
			for _, f := range getSelectableFields(&s.Blacklist, schema) {
				fields = append(fields, f.DBName)
			}
		}

		operator := s.SearchOperator
		if operator == nil {
			operator = Operators["$cont"]
		}

		search := &Search{
			Query:    query,
			Operator: operator,
			Fields:   fields,
		}

		return search
	}

	return nil
}

func getSelectableFields(blacklist *Blacklist, sch *schema.Schema) []*schema.Field {
	b := []string{}
	if blacklist != nil && blacklist.FieldsBlacklist != nil {
		b = blacklist.FieldsBlacklist
	}
	columns := make([]*schema.Field, 0, len(sch.DBNames))
	for _, f := range sch.DBNames {
		if !sliceutil.ContainsStr(b, f) {
			columns = append(columns, sch.FieldsByDBName[f])
		}
	}

	return columns
}

func selectScope(table string, fields []*schema.Field, override bool) func(*gorm.DB) *gorm.DB {
	return func(tx *gorm.DB) *gorm.DB {

		if fields == nil {
			return tx
		}

		var fieldsWithTableName []string
		if len(fields) == 0 {
			fieldsWithTableName = []string{"1"}
		} else {
			fieldsWithTableName = make([]string, 0, len(fields))
			tableName := tx.Statement.Quote(table)
			for _, f := range fields {
				computed := f.StructField.Tag.Get("computed")
				var fieldExpr string
				if computed != "" {
					fieldExpr = fmt.Sprintf("(%s) %s", strings.ReplaceAll(computed, clause.CurrentTable, tableName), tx.Statement.Quote(f.DBName))
				} else {
					fieldExpr = tableName + "." + tx.Statement.Quote(f.DBName)
				}

				fieldsWithTableName = append(fieldsWithTableName, fieldExpr)
			}
		}

		if override {
			return tx.Select(fieldsWithTableName)
		}

		return tx.Select(tx.Statement.Selects, fieldsWithTableName)
	}
}

func getField(field string, sch *schema.Schema, blacklist *Blacklist) (*schema.Field, *schema.Schema, string) {
	joinName := ""
	s := sch
	if i := strings.LastIndex(field, "."); i != -1 && i+1 < len(field) {
		rel := field[:i]
		field = field[i+1:]
		for _, v := range strings.Split(rel, ".") {
			if blacklist != nil && (sliceutil.ContainsStr(blacklist.RelationsBlacklist, v) || blacklist.IsFinal) {
				return nil, nil, ""
			}
			relation, ok := s.Relationships.Relations[v]
			if !ok || (relation.Type != schema.HasOne && relation.Type != schema.BelongsTo) {
				return nil, nil, ""
			}
			s = relation.FieldSchema
			if blacklist != nil {
				blacklist = blacklist.Relations[v]
			}
		}
		joinName = rel
	}
	if blacklist != nil && sliceutil.ContainsStr(blacklist.FieldsBlacklist, field) {
		return nil, nil, ""
	}
	col, ok := s.FieldsByDBName[field]
	if !ok {
		return nil, nil, ""
	}
	return col, s, joinName
}

func tableFromJoinName(table string, joinName string) string {
	if joinName != "" {
		i := strings.LastIndex(joinName, ".")
		if i != -1 {
			table = joinName[i+1:]
		} else {
			table = joinName
		}
	}
	return table
}
