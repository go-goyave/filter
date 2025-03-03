package filter

import (
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/samber/lo"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
	"goyave.dev/goyave/v5/database"
	"goyave.dev/goyave/v5/util/errors"
	"goyave.dev/goyave/v5/util/typeutil"
)

// Request DTO for a filter query. Any non-present option will be ignored.
type Request struct {
	Search  typeutil.Undefined[string]
	Filter  typeutil.Undefined[[]*Filter]
	Or      typeutil.Undefined[[]*Filter]
	Sort    typeutil.Undefined[[]*Sort]
	Join    typeutil.Undefined[[]*Join]
	Fields  typeutil.Undefined[[]string]
	Page    typeutil.Undefined[int]
	PerPage typeutil.Undefined[int]
}

// NewRequest creates a filter request from an HTTP request's query.
// Uses the following entries in the query, expected to be validated:
//   - search
//   - filter
//   - or
//   - sort
//   - join
//   - fields
//   - page (current page index,can be changed via the QueryParamPage variable)
//   - per_page (size of each page,can be changed via the QueryParamPerPage variable)
//
// If a field in the query doesn't match the expected type (non-validated) for the
// filtering option, it will be ignored without an error.
func NewRequest(query map[string]any) *Request {
	r := &Request{}
	if search, ok := query["search"].(string); ok {
		r.Search = typeutil.NewUndefined(search)
	}
	if filter, ok := query["filter"].([]*Filter); ok {
		r.Filter = typeutil.NewUndefined(filter)
	}
	if or, ok := query["or"].([]*Filter); ok {
		r.Or = typeutil.NewUndefined(or)
	}
	if sort, ok := query["sort"].([]*Sort); ok {
		r.Sort = typeutil.NewUndefined(sort)
	}
	if join, ok := query["join"].([]*Join); ok {
		r.Join = typeutil.NewUndefined(join)
	}
	if fields, ok := query["fields"].([]string); ok {
		r.Fields = typeutil.NewUndefined(fields)
	}
	if page, ok := query[QueryParamPage].(int); ok {
		r.Page = typeutil.NewUndefined(page)
	}
	if perPage, ok := query[QueryParamPerPage].(int); ok {
		r.PerPage = typeutil.NewUndefined(perPage)
	}
	return r
}

// Settings settings to disable certain features and/or blacklist fields
// and relations.
// The generic type is the pointer type of the model.
type Settings[T any] struct {

	// DefaultSort if not nil and not empty, and if the request is not providing any
	// sort, the request will be sorted according to the `*Sort` defined in this slice.
	// If `DisableSort` is enabled, this has no effect.
	DefaultSort []*Sort

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

	// CaseInsensitiveSort if true, the sort will wrap the value in `LOWER()` if it's a string,
	// resulting in `ORDER BY LOWER(column)`.
	CaseInsensitiveSort bool
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
	// QueryParamPage the name of current page index in pagination
	QueryParamPage = "page"
	// QueryParamPerPage the name of the data size field for each page in pagination
	QueryParamPerPage = "per_page"
	// DefaultPageSize the default pagination page size if the "per_page" query param
	// isn't provided.
	DefaultPageSize = 10

	modelCache = &sync.Map{}
)

func parseModel(db *gorm.DB, model any) (*schema.Schema, error) {
	return schema.Parse(model, modelCache, db.NamingStrategy)
}

// Scope using the default FilterSettings. See `FilterSettings.Scope()` for more details.
func Scope[T any](db *gorm.DB, request *Request, dest *[]T) (*database.Paginator[T], error) {
	return (&Settings[T]{}).Scope(db, request, dest)
}

// ScopeUnpaginated using the default FilterSettings. See `FilterSettings.ScopeUnpaginated()` for more details.
func ScopeUnpaginated[T any](db *gorm.DB, request *Request, dest *[]T) *gorm.DB {
	return (&Settings[T]{}).ScopeUnpaginated(db, request, dest)
}

// Scope apply all filters, sorts and joins defined in the request's data to the given `*gorm.DB`
// and process pagination. Returns the resulting `*database.Paginator`.
// The given request is expected to be validated using `ApplyValidation`.
func (s *Settings[T]) Scope(db *gorm.DB, request *Request, dest *[]T) (*database.Paginator[T], error) {
	page := request.Page.Default(1)
	pageSize := request.PerPage.Default(DefaultPageSize)

	var paginator *database.Paginator[T]
	err := db.Transaction(func(tx *gorm.DB) error {
		tx, schema, hasJoins := s.scopeCommon(tx, request, dest)

		paginator = database.NewPaginator(tx, page, pageSize, dest)
		err := paginator.UpdatePageInfo()
		if err != nil {
			return errors.New(err)
		}
		paginator.DB = s.scopeSort(paginator.DB, request, schema)
		if fieldsDB := s.scopeFields(paginator.DB, request, schema, hasJoins); fieldsDB != nil {
			paginator.DB = fieldsDB
		} else {
			return errors.New(paginator.DB.Error)
		}

		return paginator.Find()
	})

	return paginator, err
}

// ScopeUnpaginated apply all filters, sorts and joins defined in the request's data to the given `*gorm.DB`
// without any pagination.
// Returns the `*gorm.DB` result, which can be used to check for database errors.
// The records will be added in the given `dest` slice.
// The given request is expected to be validated using `ApplyValidation`.
func (s *Settings[T]) ScopeUnpaginated(db *gorm.DB, request *Request, dest *[]T) *gorm.DB {
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
func (s *Settings[T]) scopeCommon(db *gorm.DB, request *Request, dest any) (*gorm.DB, *schema.Schema, bool) {
	schema, err := parseModel(db, dest)
	if err != nil {
		panic(errors.New(err))
	}

	db = db.Model(dest)
	db = s.applyFilters(db, request, schema)

	hasJoins := false
	if !s.DisableJoin && request.Join.Present {
		joins := request.Join.Val
		selectCache := map[string][]string{}
		for _, j := range joins {
			hasJoins = true
			j.selectCache = selectCache
			if s := j.Scopes(s.Blacklist, schema); s != nil {
				db = db.Scopes(s...)
			}
		}
	}

	if !s.DisableSearch && request.Search.Present {
		if search := s.applySearch(request.Search.Val, schema); search != nil {
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

func (s *Settings[T]) scopeFields(db *gorm.DB, request *Request, schema *schema.Schema, hasJoins bool) *gorm.DB {
	if !s.DisableFields && request.Fields.Present {
		fields := slices.Clone(request.Fields.Val)
		if hasJoins {
			if len(schema.PrimaryFieldDBNames) == 0 {
				db.AddError(errors.New("could not find primary key. Add `gorm:\"primaryKey\"` to your model"))
				return nil
			}
			fields = addPrimaryKeys(schema, fields)
			fields = addForeignKeys(schema, fields)
		}
		return db.Scopes(selectScope(schema.Table, cleanColumns(schema, fields, s.FieldsBlacklist), false))
	}
	return db.Scopes(selectScope(schema.Table, getSelectableFields(&s.Blacklist, schema), false))
}

func (s *Settings[T]) scopeSort(db *gorm.DB, request *Request, schema *schema.Schema) *gorm.DB {
	var sorts []*Sort
	if !request.Sort.Present {
		sorts = s.DefaultSort
	} else {
		sorts = request.Sort.Val
	}

	if !s.DisableSort {
		for _, sort := range sorts {
			if scope := sort.Scope(s.Blacklist, schema, s.CaseInsensitiveSort); scope != nil {
				db = db.Scopes(scope)
			}
		}
	}
	return db
}

func (s *Settings[T]) applyFilters(db *gorm.DB, request *Request, schema *schema.Schema) *gorm.DB {
	if s.DisableFilter {
		return db
	}
	filterScopes := make([]func(*gorm.DB) *gorm.DB, 0, 2)
	joinScopes := make([]func(*gorm.DB) *gorm.DB, 0, 2)

	andLen := len(request.Filter.Default([]*Filter{}))
	orLen := len(request.Or.Default([]*Filter{}))
	mixed := orLen > 1 && andLen > 0

	for _, filters := range []typeutil.Undefined[[]*Filter]{request.Filter, request.Or} {
		if filters.Present {
			group := make([]func(*gorm.DB) *gorm.DB, 0, 4)
			for _, f := range filters.Val {
				if mixed {
					f = &Filter{
						Field:    f.Field,
						Operator: f.Operator,
						Args:     f.Args,
						Or:       false,
					}
				}
				joinScope, conditionScope := f.Scope(s.Blacklist, schema)
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
	if len(joinScopes) > 0 {
		db = db.Scopes(joinScopes...)
	}
	if len(filterScopes) > 0 {
		db = db.Scopes(groupFilters(filterScopes, true))
	}
	return db
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

func (s *Settings[T]) applySearch(query string, schema *schema.Schema) *Search {
	// Note: the search condition is not in a group condition (parenthesis)
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

func getSelectableFields(blacklist *Blacklist, sch *schema.Schema) []*schema.Field {
	b := []string{}
	if blacklist != nil && blacklist.FieldsBlacklist != nil {
		b = blacklist.FieldsBlacklist
	}
	columns := make([]*schema.Field, 0, len(sch.DBNames))
	for _, f := range sch.DBNames {
		if !lo.Contains(b, f) {
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
			// Use two filler values so Gorm doesn't attempt to Scan into a []int64
			fieldsWithTableName = []string{"1", "2"}
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
			if blacklist != nil && (lo.Contains(blacklist.RelationsBlacklist, v) || blacklist.IsFinal) {
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
	if blacklist != nil && lo.Contains(blacklist.FieldsBlacklist, field) {
		return nil, nil, ""
	}
	col := s.LookUpField(field)
	if col == nil {
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
