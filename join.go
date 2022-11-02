package filter

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
	"goyave.dev/goyave/v4/util/sliceutil"
)

var (
	joinRegex = regexp.MustCompile("(?i)((LEFT|RIGHT|FULL)\\s+)?((OUTER|INNER)\\s+)?JOIN\\s+[\"'`]?(?P<TableName>\\w+)[\"'`]?\\s+((AS\\s+)?[\"'`]?(?P<Alias>\\w+)[\"'`]?)?\\s*ON")
)

// Join structured representation of a join query.
type Join struct {
	selectCache map[string][]string
	Relation    string
	Fields      []string
}

// Scopes returns the GORM scopes to use in order to apply this joint.
func (j *Join) Scopes(settings *Settings, schema *schema.Schema) []func(*gorm.DB) *gorm.DB {
	scopes := j.applyRelation(schema, &settings.Blacklist, j.Relation, 0, make([]func(*gorm.DB) *gorm.DB, 0, strings.Count(j.Relation, ".")+1))
	if scopes != nil {
		return scopes
	}
	return nil
}

func (j *Join) applyRelation(schema *schema.Schema, blacklist *Blacklist, relationName string, startIndex int, scopes []func(*gorm.DB) *gorm.DB) []func(*gorm.DB) *gorm.DB {
	if blacklist != nil && blacklist.IsFinal {
		return nil
	}
	trimmedRelationName := relationName[startIndex:]
	i := strings.Index(trimmedRelationName, ".")
	if i == -1 {
		if blacklist != nil {
			if sliceutil.ContainsStr(blacklist.RelationsBlacklist, trimmedRelationName) {
				return nil
			}
			blacklist = blacklist.Relations[trimmedRelationName]
		}

		r, ok := schema.Relationships.Relations[trimmedRelationName]
		if !ok {
			return nil
		}

		j.selectCache[relationName] = j.Fields
		return append(scopes, joinScope(relationName, r, j.Fields, blacklist))
	}

	if startIndex+i+1 >= len(relationName) {
		return nil
	}

	name := trimmedRelationName[:i]
	var b *Blacklist
	if blacklist != nil {
		if sliceutil.ContainsStr(blacklist.RelationsBlacklist, name) {
			return nil
		}
		b = blacklist.Relations[name]
	}
	r, ok := schema.Relationships.Relations[name]
	if !ok {
		return nil
	}
	n := relationName[:startIndex+i]
	fields := []string{}
	if f, ok := j.selectCache[n]; ok {
		fields = f
	}
	scopes = append(scopes, joinScope(n, r, fields, b))

	return j.applyRelation(r.FieldSchema, b, relationName, startIndex+i+1, scopes)
}

func joinScope(relationName string, rel *schema.Relationship, fields []string, blacklist *Blacklist) func(*gorm.DB) *gorm.DB {
	var columns []*schema.Field
	if fields == nil {
		columns = getSelectableFields(blacklist, rel.FieldSchema)
	} else {
		var b []string
		if blacklist != nil {
			b = blacklist.FieldsBlacklist
		}
		columns = cleanColumns(rel.FieldSchema, fields, b)
	}

	return func(tx *gorm.DB) *gorm.DB {
		if rel.FieldSchema.Table == "" {
			tx.AddError(fmt.Errorf("Relation %q is anonymous, could not get table name", relationName))
			return tx
		}
		if columns != nil {
			for _, primaryField := range rel.FieldSchema.PrimaryFields {
				if !columnsContain(columns, primaryField) && (blacklist == nil || !sliceutil.ContainsStr(blacklist.FieldsBlacklist, primaryField.DBName)) {
					columns = append(columns, primaryField)
				}
			}
			for _, backwardsRelation := range rel.FieldSchema.Relationships.Relations {
				if backwardsRelation.FieldSchema == rel.Schema && backwardsRelation.Type == schema.BelongsTo {
					for _, ref := range backwardsRelation.References {
						if !columnsContain(columns, ref.ForeignKey) && (blacklist == nil || !sliceutil.ContainsStr(blacklist.FieldsBlacklist, ref.ForeignKey.DBName)) {
							columns = append(columns, ref.ForeignKey)
						}
					}
				}
			}
		}

		return tx.Preload(relationName, selectScope(rel.FieldSchema.Table, columns, true))
	}
}

func join(tx *gorm.DB, joinName string, sch *schema.Schema) *gorm.DB {

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
			findStatementJoin(tx.Statement, relation, &j)
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

func joinExists(stmt *gorm.Statement, join clause.Join) bool {
	if c, ok := stmt.Clauses["FROM"]; ok {
		from := c.Expression.(clause.From)
		c.Expression = from
		for _, j := range from.Joins {
			if j.Table == join.Table {
				return true
			}
		}
	}
	for _, j := range stmt.Joins {
		groups := joinRegex.FindStringSubmatch(j.Name)
		if groups != nil {
			tableName := groups[joinRegex.SubexpIndex("TableName")]
			aliasName := groups[joinRegex.SubexpIndex("Alias")]
			tableMatch := tableName == join.Table.Name
			aliasMatch := aliasName == "" || aliasName == join.Table.Alias
			if tableMatch && aliasMatch {
				return true
			}
		}
	}
	return false
}

// findStatementJoin finds a matching join in the given statement,
// adds its conditions (if any) to the given clause.Join and removes
// the matched join from Statement.Joins.
// This is used to avoid duplicate joins that produce ambiguous column names.
func findStatementJoin(stmt *gorm.Statement, relation *schema.Relationship, join *clause.Join) {
	for i, j := range stmt.Joins {
		if j.Name == join.Table.Alias {
			on := join.ON
			if j.On != nil {
				on.Exprs = append(on.Exprs, *j.On)
			}
			join.ON = on
			stmt.Joins = append(stmt.Joins[:i], stmt.Joins[i+1:]...)

			for _, s := range relation.FieldSchema.DBNames {
				stmt.Selects = append(stmt.Selects, fmt.Sprintf("%s.%s AS %s", quoteString(stmt, relation.Name), quoteString(stmt, s), quoteString(stmt, relation.Name+"__"+s)))
			}
			return
		}
	}
}

func quoteString(stmt *gorm.Statement, str string) string {
	writer := bytes.NewBufferString("")
	stmt.DB.Dialector.QuoteTo(writer, str)
	return writer.String()
}
