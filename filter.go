package filter

import (
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
	"goyave.dev/goyave/v3/helper"
)

// Filter structured representation of a filter query.
type Filter struct {
	Field    string
	Operator *Operator
	Args     []string
	Or       bool
}

// Scope returns the GORM scope to use in order to apply this filter.
func (f *Filter) Scope(settings *Settings, modelIdentity *modelIdentity) func(*gorm.DB) *gorm.DB {
	blacklist := settings.Blacklist
	field := f.Field
	joinName := ""
	if i := strings.LastIndex(f.Field, "."); i != -1 && i+1 < len(f.Field) {
		rel := f.Field[:i]
		field = f.Field[i+1:]
		for _, v := range strings.Split(rel, ".") {
			if helper.ContainsStr(blacklist.RelationsBlacklist, v) {
				return nil
			}
			relation, ok := modelIdentity.Relations[v]
			if !ok || relation.Type != schema.HasOne {
				return nil
			}
			modelIdentity = relation.modelIdentity
		}
		joinName = rel
	}
	if helper.ContainsStr(blacklist.FieldsBlacklist, field) {
		return nil
	}
	_, ok := modelIdentity.Columns[field]
	if !ok {
		return nil
	}
	return func(tx *gorm.DB) *gorm.DB {
		if joinName != "" {
			if err := tx.Statement.Parse(tx.Statement.Model); err != nil {
				tx.AddError(err)
				return tx
			}
			tx = join(tx, joinName, modelIdentity)
		}
		tableName := tx.Statement.Quote(modelIdentity.TableName) + "."
		return f.Operator.Function(tx, f, tableName+tx.Statement.Quote(field))
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

func join(tx *gorm.DB, joinName string, modelIdentity *modelIdentity) *gorm.DB {

	relation := tx.Statement.Schema.Relationships.Relations[joinName]
	exprs := make([]clause.Expression, len(relation.References))
	for idx, ref := range relation.References {
		if ref.OwnPrimaryKey {
			exprs[idx] = clause.Eq{
				Column: clause.Column{Table: clause.CurrentTable, Name: ref.PrimaryKey.DBName},
				Value:  clause.Column{Table: modelIdentity.TableName, Name: ref.ForeignKey.DBName},
			}
		} else {
			if ref.PrimaryValue == "" {
				exprs[idx] = clause.Eq{
					Column: clause.Column{Table: clause.CurrentTable, Name: ref.ForeignKey.DBName},
					Value:  clause.Column{Table: modelIdentity.TableName, Name: ref.PrimaryKey.DBName},
				}
			} else {
				exprs[idx] = clause.Eq{
					Column: clause.Column{Table: modelIdentity.TableName, Name: ref.ForeignKey.DBName},
					Value:  ref.PrimaryValue,
				}
			}
		}
	}
	return tx.Clauses(clause.From{Joins: []clause.Join{
		{
			Type:  clause.LeftJoin,
			Table: clause.Table{Name: modelIdentity.TableName},
			ON:    clause.Where{Exprs: exprs},
		},
	}})
	// TODO test nested relations
	// TODO test what happens if there are multiple joins
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
			tableName := tx.Statement.Quote(modelIdentity.TableName) + "."
			for _, f := range fields {
				fieldsWithTableName = append(fieldsWithTableName, tableName+tx.Statement.Quote(f))
			}
		}
		return tx.Select(fieldsWithTableName)
	}
}
