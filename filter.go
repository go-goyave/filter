package filter

import (
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
func (f *Filter) Scope(settings *Settings, sch *schema.Schema) func(*gorm.DB) *gorm.DB {
	blacklist := &settings.Blacklist
	field := f.Field
	joinName := ""
	s := sch
	if i := strings.LastIndex(f.Field, "."); i != -1 && i+1 < len(f.Field) {
		rel := f.Field[:i]
		field = f.Field[i+1:]
		for _, v := range strings.Split(rel, ".") {
			if blacklist != nil && sliceutil.ContainsStr(blacklist.RelationsBlacklist, v) {
				return nil
			}
			relation, ok := s.Relationships.Relations[v]
			if !ok || (relation.Type != schema.HasOne && relation.Type != schema.BelongsTo) {
				return nil
			}
			s = relation.FieldSchema
			if blacklist != nil {
				blacklist = blacklist.Relations[v]
			}
		}
		joinName = rel
	}
	if blacklist != nil && sliceutil.ContainsStr(blacklist.FieldsBlacklist, field) {
		return nil
	}
	col, ok := s.FieldsByDBName[field]
	if !ok {
		return nil
	}
	return func(tx *gorm.DB) *gorm.DB {
		if joinName != "" {
			if err := tx.Statement.Parse(tx.Statement.Model); err != nil {
				tx.AddError(err)
				return tx
			}
			tx = join(tx, joinName, sch)
		}

		tableName := tx.Statement.Quote(s.Table) + "."
		return f.Operator.Function(tx, f, tableName+tx.Statement.Quote(field), col.DataType)
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

func join(tx *gorm.DB, joinName string, schema *schema.Schema) *gorm.DB {

	var lastTable string
	joins := make([]clause.Join, 0, strings.Count(joinName, ".")+1)
	for _, rel := range strings.Split(joinName, ".") {
		lastTable = schema.Table
		relation := schema.Relationships.Relations[rel]
		schema = relation.FieldSchema
		exprs := make([]clause.Expression, len(relation.References))
		for idx, ref := range relation.References {
			if ref.OwnPrimaryKey {
				exprs[idx] = clause.Eq{
					Column: clause.Column{Table: lastTable, Name: ref.PrimaryKey.DBName},
					Value:  clause.Column{Table: schema.Table, Name: ref.ForeignKey.DBName},
				}
			} else {
				if ref.PrimaryValue == "" {
					exprs[idx] = clause.Eq{
						Column: clause.Column{Table: lastTable, Name: ref.ForeignKey.DBName},
						Value:  clause.Column{Table: schema.Table, Name: ref.PrimaryKey.DBName},
					}
				} else {
					exprs[idx] = clause.Eq{
						Column: clause.Column{Table: schema.Table, Name: ref.ForeignKey.DBName},
						Value:  ref.PrimaryValue,
					}
				}
			}
		}
		joins = append(joins, clause.Join{
			Type:  clause.LeftJoin,
			Table: clause.Table{Name: schema.Table},
			ON:    clause.Where{Exprs: exprs},
		})
	}
	return tx.Clauses(clause.From{Joins: joins})
}

func selectScope(schema *schema.Schema, fields []string, override bool) func(*gorm.DB) *gorm.DB {
	return func(tx *gorm.DB) *gorm.DB {

		if fields == nil {
			return tx
		}

		var fieldsWithTableName []string
		if len(fields) == 0 {
			fieldsWithTableName = []string{"1"}
		} else {
			fieldsWithTableName = make([]string, 0, len(fields))
			tableName := tx.Statement.Quote(schema.Table) + "."
			for _, f := range fields {
				fieldsWithTableName = append(fieldsWithTableName, tableName+tx.Statement.Quote(f))
			}
		}

		if override {
			return tx.Select(fieldsWithTableName)
		}

		return tx.Select(tx.Statement.Selects, fieldsWithTableName)
	}
}
