package filter

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"goyave.dev/goyave/v4/helper"
)

// Join structured representation of a join query.
type Join struct {
	selectCache map[string][]string
	Relation    string
	Fields      []string
}

// Scopes returns the GORM scopes to use in order to apply this joint.
func (j *Join) Scopes(settings *Settings, modelIdentity *modelIdentity) []func(*gorm.DB) *gorm.DB {
	scopes := j.applyRelation(modelIdentity, &settings.Blacklist, j.Relation, 0, make([]func(*gorm.DB) *gorm.DB, 0, strings.Count(j.Relation, ".")+1))
	if scopes != nil {
		return scopes
	}
	return nil
}

func (j *Join) applyRelation(modelIdentity *modelIdentity, blacklist *Blacklist, relationName string, startIndex int, scopes []func(*gorm.DB) *gorm.DB) []func(*gorm.DB) *gorm.DB {
	if blacklist != nil && blacklist.IsFinal {
		return nil
	}
	trimmedRelationName := relationName[startIndex:]
	i := strings.Index(trimmedRelationName, ".")
	if i == -1 {
		if blacklist != nil {
			if helper.ContainsStr(blacklist.RelationsBlacklist, trimmedRelationName) {
				return nil
			}
			blacklist = blacklist.Relations[trimmedRelationName]
		}

		r, ok := modelIdentity.Relations[trimmedRelationName]
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
		if helper.ContainsStr(blacklist.RelationsBlacklist, name) {
			return nil
		}
		b = blacklist.Relations[name]
	}
	r, ok := modelIdentity.Relations[name]
	if !ok {
		return nil
	}
	n := relationName[:startIndex+i]
	fields := []string{}
	if f, ok := j.selectCache[n]; ok {
		fields = f
	}
	scopes = append(scopes, joinScope(n, r, fields, b))

	return j.applyRelation(r.modelIdentity, b, relationName, startIndex+i+1, scopes)
}

func joinScope(relationName string, relationIdentity *relation, fields []string, blacklist *Blacklist) func(*gorm.DB) *gorm.DB {
	var b []string
	if blacklist != nil {
		b = blacklist.FieldsBlacklist
	}
	columns := relationIdentity.cleanColumns(fields, b)

	return func(tx *gorm.DB) *gorm.DB {
		if columns != nil {
			if len(relationIdentity.PrimaryKeys) == 0 {
				tx.AddError(fmt.Errorf("Could not find %q relation's primary key. Add `gorm:\"primaryKey\"` to your model", relationName))
				return tx
			}
			for _, k := range relationIdentity.PrimaryKeys {
				if !helper.ContainsStr(columns, k) && (blacklist == nil || !helper.ContainsStr(blacklist.FieldsBlacklist, k)) {
					columns = append(columns, k)
				}
			}
			if relationIdentity.Type == schema.HasMany {
				for _, v := range relationIdentity.ForeignKeys {
					if !helper.ContainsStr(columns, v) && (blacklist == nil || !helper.ContainsStr(blacklist.FieldsBlacklist, v)) {
						columns = append(columns, v)
					}
				}
			}
		}

		return tx.Preload(relationName, selectScope(relationIdentity.modelIdentity, columns, func(tx *gorm.DB, fields []string) *gorm.DB {
			return tx.Select(fields)
		}))
	}
}
