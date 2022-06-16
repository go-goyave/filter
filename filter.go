package filter

import (
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"goyave.dev/goyave/v4/util/sliceutil"
)

// Filter structured representation of a filter query.
type Filter struct {
	Field    string
	Operator *Operator
	Args     []string
	Or       bool
}

// Scope returns the GORM scope to use in order to apply this filter.
func (f *Filter) Scope(settings *Settings, sch *schema.Schema) (func(*gorm.DB) *gorm.DB, func(*gorm.DB) *gorm.DB) {
	blacklist := &settings.Blacklist
	field := f.Field
	joinName := ""
	s := sch
	if i := strings.LastIndex(f.Field, "."); i != -1 && i+1 < len(f.Field) {
		rel := f.Field[:i]
		field = f.Field[i+1:]
		for _, v := range strings.Split(rel, ".") {
			if blacklist != nil && sliceutil.ContainsStr(blacklist.RelationsBlacklist, v) {
				return nil, nil
			}
			relation, ok := s.Relationships.Relations[v]
			if !ok || (relation.Type != schema.HasOne && relation.Type != schema.BelongsTo) {
				return nil, nil
			}
			s = relation.FieldSchema
			if blacklist != nil {
				blacklist = blacklist.Relations[v]
			}
		}
		joinName = rel
	}
	if blacklist != nil && sliceutil.ContainsStr(blacklist.FieldsBlacklist, field) {
		return nil, nil
	}
	col, ok := s.FieldsByDBName[field]
	if !ok {
		return nil, nil
	}

	joinScope := func(tx *gorm.DB) *gorm.DB {
		if joinName != "" {
			if err := tx.Statement.Parse(tx.Statement.Model); err != nil {
				tx.AddError(err)
				return tx
			}
			tx = join(tx, joinName, sch)
		}

		return tx
	}

	conditionScope := func(tx *gorm.DB) *gorm.DB {
		table := s.Table
		if joinName != "" {
			i := strings.LastIndex(joinName, ".")
			if i != -1 {
				table = joinName[i+1:]
			} else {
				table = joinName
			}
		}
		tableName := tx.Statement.Quote(table) + "."
		return f.Operator.Function(tx, f, tableName+tx.Statement.Quote(field), col.DataType)
	}

	return joinScope, conditionScope
}

// Where applies a condition to given transaction, automatically taking the "Or"
// filter value into account.
func (f *Filter) Where(tx *gorm.DB, query string, args ...interface{}) *gorm.DB {
	if f.Or {
		return tx.Or(query, args...)
	}
	return tx.Where(query, args...)
}
