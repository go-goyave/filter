package filter

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
	"goyave.dev/goyave/v4"
	"goyave.dev/goyave/v4/database"
	"goyave.dev/goyave/v4/helper"
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
	// DisableSearch ignore the "search" query if true.
	DisableSearch bool

	// FieldsSearch allows search for these fields
	FieldsSearch   []string
	SearchOperator *Operator
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

	if !s.DisableSearch && s.SearchOperator != nil && request.Has("search") {
		query, ok := request.Data["search"].(string)
		if ok {
			if len(s.FieldsSearch) == 0 {
				s.FieldsSearch = s.getSelectableFields(modelIdentity.Columns)
			}

			if len(s.FieldsSearch) > 0 {
				search := &Search{
					Fields: s.FieldsSearch,
					Query:  query,
				}

				scope := search.Scopes(s, modelIdentity)

				if scope != nil {
					db = db.Scopes(scope)
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

	db = db.Model(dest)
	paginator := database.NewPaginator(db, page, pageSize, dest)
	paginator.UpdatePageInfo()

	if !s.DisableSort && request.Has("sort") {
		sorts, ok := request.Data["sort"].([]*Sort)
		if ok {
			for _, sort := range sorts {
				if scope := sort.Scope(s, modelIdentity); scope != nil {
					paginator.DB = paginator.DB.Scopes(scope)
				}
			}
		}
	}

	if !s.DisableFields && request.Has("fields") {
		fields := strings.Split(request.String("fields"), ",")
		if hasJoins {
			if len(modelIdentity.PrimaryKeys) == 0 {
				paginator.DB.AddError(fmt.Errorf("Could not find primary key. Add `gorm:\"primaryKey\"` to your model"))
				return nil, paginator.DB
			}
			fields = modelIdentity.addPrimaryKeys(fields)
			fields = modelIdentity.addForeignKeys(fields)
		}
		paginator.DB = paginator.DB.Scopes(selectScope(modelIdentity, modelIdentity.cleanColumns(fields, s.FieldsBlacklist), true))
	} else {
		paginator.DB = paginator.DB.Scopes(selectScope(modelIdentity, s.getSelectableFields(modelIdentity.Columns), true))
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
	blacklist := []string{}
	if b.FieldsBlacklist != nil {
		blacklist = b.FieldsBlacklist
	}
	columns := make([]string, 0, len(fields))
	for k := range fields {
		if !helper.ContainsStr(blacklist, k) {
			columns = append(columns, k)
		}
	}

	return columns
}
