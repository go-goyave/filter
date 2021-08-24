package filter

import (
	"database/sql"
	"reflect"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"gorm.io/gorm/utils"
	"goyave.dev/goyave/v3/helper"
)

var (
	identityCache = make(map[string]*modelIdentity, 10)
)

type gormTags struct {
	Column         string
	ForeignKey     string
	References     string
	EmbeddedPrefix string
	Embedded       bool
	PrimaryKey     bool
	Ignored        bool
}

type modelIdentity struct {
	Columns     map[string]*column
	Relations   map[string]*relation
	PrimaryKeys []string
}

type relation struct {
	*modelIdentity
	Type        schema.RelationshipType
	Tags        *gormTags
	ForeignKeys []string

	keysProcessed bool
}

type column struct {
	Tags *gormTags
	Name string
}

func (i *modelIdentity) promote(identity *modelIdentity, prefix string) {
	for k, v := range identity.Columns {
		i.Columns[prefix+k] = v
	}
	for k, v := range identity.Relations {
		i.Relations[prefix+k] = v
	}
}

// cleanColumns returns a slice of column names containing only the valid
// column names from the input columns slice.
func (i *modelIdentity) cleanColumns(columns []string, blacklist []string) []string {
	for j := 0; j < len(columns); j++ {
		_, ok := i.Columns[columns[j]]
		if !ok || helper.ContainsStr(blacklist, columns[j]) {
			columns = append(columns[:j], columns[j+1:]...)
			j--
		}
	}

	return columns
}

func (i *modelIdentity) addPrimaryKeys(fields []string) []string {
	for _, k := range i.PrimaryKeys {
		if !helper.ContainsStr(fields, k) {
			fields = append(fields, k)
		}
	}
	return fields
}

func (i *modelIdentity) findColumn(name string) (*column, string) {
	for k, v := range i.Columns {
		if v.Name == name {
			return v, k
		}
	}
	return nil, ""
}

func (r *relation) processKeys(db *gorm.DB) {
	if r.keysProcessed {
		return
	}
	r.keysProcessed = true
	r.ForeignKeys = r.findForeignKeys(db)
	for _, rel := range r.Relations {
		rel.processKeys(db)
	}
}

func (r *relation) findForeignKeys(db *gorm.DB) []string {
	foreignKeys := []string{}

	for k, v := range r.Relations {
		foreignKeys = append(foreignKeys, r.findForeignKey(db, k, v)...)
	}

	return foreignKeys
}

func (r *relation) findForeignKey(db *gorm.DB, name string, rel *relation) []string {
	keys := make([]string, 0, 2)
	if rel.Tags.ForeignKey != "" {
		for _, v := range strings.Split(rel.Tags.ForeignKey, ",") {
			if col, colName := r.findColumn(strings.TrimSpace(v)); col != nil {
				keys = append(keys, columnName(db, col.Tags, colName))
			}
		}
		return keys
	}
	colName := columnName(db, rel.Tags, name) + "_id"
	if col, ok := r.Columns[colName]; ok {
		keys = append(keys, columnName(db, col.Tags, colName))
	}
	return keys
}

func parseModel(db *gorm.DB, model interface{}) *modelIdentity {
	t := reflect.TypeOf(model)
	i := parseIdentity(db, t, []reflect.Type{t})
	if i != nil {
		for _, r := range i.Relations {
			r.processKeys(db)
		}
	}
	return i
}

func parseIdentity(db *gorm.DB, t reflect.Type, parents []reflect.Type) *modelIdentity {
	t = actualType(t)
	if t.Kind() != reflect.Struct {
		return nil
	}
	identifier := t.PkgPath() + "|" + t.String()
	if cached, ok := identityCache[identifier]; ok {
		return cached
	}
	identity := &modelIdentity{
		Columns:     make(map[string]*column, 10),
		Relations:   make(map[string]*relation, 5),
		PrimaryKeys: make([]string, 0, 2),
	}
	identityCache[identifier] = identity
	count := t.NumField()

	for i := 0; i < count; i++ {
		field := t.Field(i)
		fieldType := field.Type
		if fieldType.Kind() == reflect.Ptr {
			fieldType = fieldType.Elem()
		}
		gormTags := parseGormTags(field)
		if gormTags.Ignored {
			continue
		}

		switch fieldType.Kind() {
		case reflect.Struct:
			parents = append(parents, t)
			if field.Anonymous {
				// Promoted fields
				identity.promote(parseIdentity(db, fieldType, parents), "")
			} else if field.Type.Implements(reflect.TypeOf((*sql.Scanner)(nil)).Elem()) {
				// This is not a relation but a field such as sql.NullTime
				identity.Columns[columnName(db, gormTags, field.Name)] = &column{
					Name: field.Name,
					Tags: gormTags,
				}
			} else if i := parseIdentity(db, fieldType, parents); i != nil {
				if gormTags.Embedded {
					identity.promote(i, gormTags.EmbeddedPrefix)
				} else {
					// "belongs to" / "has one" relation
					r := &relation{
						modelIdentity: i,
						Tags:          gormTags,
						Type:          schema.HasOne,
					}
					identity.Relations[field.Name] = r
				}
			}
		case reflect.Slice:
			// "has many" relation
			parents = append(parents, t)
			if i := parseIdentity(db, fieldType.Elem(), parents); i != nil {
				r := &relation{
					modelIdentity: i,
					Tags:          gormTags,
					Type:          schema.HasMany,
				}
				identity.Relations[field.Name] = r
			}
		default:
			colName := columnName(db, gormTags, field.Name)
			if gormTags.PrimaryKey {
				identity.PrimaryKeys = append(identity.PrimaryKeys, colName)
			}
			identity.Columns[colName] = &column{
				Name: field.Name,
				Tags: gormTags,
			}
		}
	}

	if len(identity.PrimaryKeys) == 0 {
		colName := "id"
		if col, ok := identity.Columns[colName]; ok {
			identity.PrimaryKeys = append(identity.PrimaryKeys, columnName(db, col.Tags, colName))
		}
	}
	return identity
}

func actualType(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Ptr || t.Kind() == reflect.Slice {
		t = t.Elem()
	}
	return t
}

func parseGormTags(field reflect.StructField) *gormTags {
	settings := schema.ParseTagSetting(field.Tag.Get("gorm"), ";")
	res := &gormTags{}
	for k, v := range settings {
		switch k {
		case "COLUMN":
			res.Column = strings.TrimSpace(v)
		case "EMBEDDED":
			res.Embedded = true
		case "EMBEDDEDPREFIX":
			res.EmbeddedPrefix = strings.TrimSpace(v)
		case "FOREIGNKEY":
			res.ForeignKey = strings.TrimSpace(v)
		case "REFERENCES":
			res.References = strings.TrimSpace(v)
		case "PRIMARYKEY":
			res.PrimaryKey = utils.CheckTruth(v)
		case "PRIMARY_KEY":
			res.PrimaryKey = utils.CheckTruth(v)
		case "-":
			res.Ignored = true
		}
	}

	return res
}

func columnName(db *gorm.DB, tags *gormTags, fieldName string) string {
	if tags.Column != "" {
		return tags.Column
	}

	return db.NamingStrategy.ColumnName("", fieldName)
}
