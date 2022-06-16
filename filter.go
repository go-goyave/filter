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

func join(tx *gorm.DB, joinName string, sch *schema.Schema) *gorm.DB { // TODO move this to join.go

	var lastTable string
	var relation *schema.Relationship
	joins := make([]clause.Join, 0, strings.Count(joinName, ".")+1)
	for _, rel := range strings.Split(joinName, ".") {
		lastTable = sch.Table
		if relation != nil {
			lastTable = relation.Name
		}
		relation = sch.Relationships.Relations[rel]
		sch = relation.FieldSchema
		exprs := make([]clause.Expression, len(relation.References))
		for idx, ref := range relation.References {
			if ref.OwnPrimaryKey {
				exprs[idx] = clause.Eq{
					Column: clause.Column{Table: lastTable, Name: ref.PrimaryKey.DBName},
					Value:  clause.Column{Table: relation.Name, Name: ref.ForeignKey.DBName},
				}
			} else {
				if ref.PrimaryValue == "" {
					exprs[idx] = clause.Eq{
						Column: clause.Column{Table: lastTable, Name: ref.ForeignKey.DBName},
						Value:  clause.Column{Table: relation.Name, Name: ref.PrimaryKey.DBName},
					}
				} else {
					exprs[idx] = clause.Eq{
						Column: clause.Column{Table: relation.Name, Name: ref.ForeignKey.DBName},
						Value:  ref.PrimaryValue,
					}
				}
			}
		}
		j := clause.Join{
			Type:  clause.LeftJoin,
			Table: clause.Table{Name: sch.Table, Alias: relation.Name},
			ON:    clause.Where{Exprs: exprs},
		}
		if !joinExists(tx.Statement, j) {
			joins = append(joins, j)
		}
	}
	if c, ok := tx.Statement.Clauses["FROM"]; ok {
		from := c.Expression.(clause.From)
		from.Joins = append(from.Joins, joins...)
		c.Expression = from
		tx.Statement.Clauses["FROM"] = c
		return tx
	}
	return tx.Clauses(clause.From{Joins: joins})
}

func selectScope(table string, fields []string, override bool) func(*gorm.DB) *gorm.DB { // TODO move this to settings
	return func(tx *gorm.DB) *gorm.DB {

		if fields == nil {
			return tx
		}

		var fieldsWithTableName []string
		if len(fields) == 0 {
			fieldsWithTableName = []string{"1"}
		} else {
			fieldsWithTableName = make([]string, 0, len(fields))
			tableName := tx.Statement.Quote(table) + "."
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
