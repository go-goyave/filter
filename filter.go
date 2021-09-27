package filter

import (
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
	"goyave.dev/goyave/v4/helper"
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
	blacklist := &settings.Blacklist
	field := f.Field
	joinName := ""
	m := modelIdentity
	if i := strings.LastIndex(f.Field, "."); i != -1 && i+1 < len(f.Field) {
		rel := f.Field[:i]
		field = f.Field[i+1:]
		for _, v := range strings.Split(rel, ".") {
			if blacklist != nil && helper.ContainsStr(blacklist.RelationsBlacklist, v) {
				return nil
			}
			relation, ok := m.Relations[v]
			if !ok || relation.Type != schema.HasOne {
				return nil
			}
			m = relation.modelIdentity
			if blacklist != nil {
				blacklist = blacklist.Relations[v]
			}
		}
		joinName = rel
	}
	if blacklist != nil && helper.ContainsStr(blacklist.FieldsBlacklist, field) {
		return nil
	}
	col, ok := m.Columns[field]
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

		// Skip the query if someone try to filter on non boolean column type
		if f.Operator.Name == "$istrue" || f.Operator.Name == "$isfalse" {
			if col.Type != "bool" {
				return tx
			}
		}

		tableName := tx.Statement.Quote(m.TableName) + "."
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

	lastTable := clause.CurrentTable
	joins := make([]clause.Join, 0, strings.Count(joinName, ".")+1)
	schema := tx.Statement.Schema
	for _, rel := range strings.Split(joinName, ".") {
		modelIdentity = modelIdentity.Relations[rel].modelIdentity
		relation := schema.Relationships.Relations[rel]
		exprs := make([]clause.Expression, len(relation.References))
		for idx, ref := range relation.References {
			if ref.OwnPrimaryKey {
				exprs[idx] = clause.Eq{
					Column: clause.Column{Table: lastTable, Name: ref.PrimaryKey.DBName},
					Value:  clause.Column{Table: modelIdentity.TableName, Name: ref.ForeignKey.DBName},
				}
			} else {
				if ref.PrimaryValue == "" {
					exprs[idx] = clause.Eq{
						Column: clause.Column{Table: lastTable, Name: ref.ForeignKey.DBName},
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
		joins = append(joins, clause.Join{
			Type:  clause.LeftJoin,
			Table: clause.Table{Name: modelIdentity.TableName},
			ON:    clause.Where{Exprs: exprs},
		})
		lastTable = modelIdentity.TableName
		schema = relation.FieldSchema
	}
	return tx.Clauses(clause.From{Joins: joins})
}

func selectScope(modelIdentity *modelIdentity, fields []string, override bool) func(*gorm.DB) *gorm.DB {
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

		if override {
			return tx.Select(tx.Statement.Selects, fieldsWithTableName)
		}

		return tx.Select(fieldsWithTableName)
	}
}
