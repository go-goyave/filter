package filter

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
	"goyave.dev/goyave/v4/util/sliceutil"
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
	var b []string
	if blacklist != nil {
		b = blacklist.FieldsBlacklist
	}
	columns := cleanColumns(rel.FieldSchema, fields, b)

	return func(tx *gorm.DB) *gorm.DB {
		if rel.FieldSchema.Table == "" {
			tx.AddError(fmt.Errorf("Relation %q is anonymous, could not get table name", relationName))
			return tx
		}
		if columns != nil {
			for _, k := range rel.FieldSchema.PrimaryFieldDBNames {
				if !sliceutil.ContainsStr(columns, k) && (blacklist == nil || !sliceutil.ContainsStr(blacklist.FieldsBlacklist, k)) {
					columns = append(columns, k)
				}
			}
			for _, backwardsRelation := range rel.FieldSchema.Relationships.Relations {
				if backwardsRelation.FieldSchema == rel.Schema && backwardsRelation.Type == schema.BelongsTo {
					for _, ref := range backwardsRelation.References {
						if !sliceutil.ContainsStr(columns, ref.ForeignKey.DBName) && (blacklist == nil || !sliceutil.ContainsStr(blacklist.FieldsBlacklist, ref.ForeignKey.DBName)) {
							columns = append(columns, ref.ForeignKey.DBName)
						}
					}
				}
			}
		}

		return tx.Preload(relationName, selectScope(rel.FieldSchema.Table, columns, true))
	}
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
	return false
}
