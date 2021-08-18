package filter

import (
	"reflect"
	"strings"

	"gorm.io/gorm"
)

var (
	identityCache = make(map[string]*modelIdentity, 10)
)

type modelIdentity struct {
	Columns   map[string]column
	Relations map[string]*modelIdentity
}

type column struct {
	Name string
	Tag  reflect.StructTag
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
func (i *modelIdentity) cleanColumns(columns []string) []string {
	for j := 0; j < len(columns); j++ {
		if _, ok := i.Columns[columns[j]]; !ok {
			columns = append(columns[:j], columns[j+1:]...)
			j--
		}
	}

	return columns
}

func parseModel(db *gorm.DB, model interface{}) *modelIdentity {
	t := reflect.TypeOf(model)
	return parseIdentity(db, t, []reflect.Type{t})
}

func parseIdentity(db *gorm.DB, t reflect.Type, parents []reflect.Type) *modelIdentity {
	for t.Kind() == reflect.Ptr || t.Kind() == reflect.Slice {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct || checkCycle(t, parents) {
		return nil
	}
	identifier := t.PkgPath() + "|" + t.String()
	if cached, ok := identityCache[identifier]; ok {
		return cached
	}
	identity := &modelIdentity{
		Columns:   make(map[string]column, 10),
		Relations: make(map[string]*modelIdentity, 5),
	}
	count := t.NumField()

	for i := 0; i < count; i++ {
		field := t.Field(i)
		fieldType := field.Type
		if fieldType.Kind() == reflect.Ptr {
			fieldType = fieldType.Elem()
		}

		switch fieldType.Kind() {
		case reflect.Struct:
			parents = append(parents, t)
			if field.Anonymous {
				// Promoted fields
				i := parseIdentity(db, fieldType, parents)
				if i == nil {
					continue
				}
				identity.promote(i, "")
			} else if i := parseIdentity(db, fieldType, parents); i != nil {
				// FIXME some structures are not relations (sql.NullTime)
				if prefix, ok := getEmbeddedInfo(field); ok {
					identity.promote(i, prefix)
				} else {
					// "belongs to" / "has one" relation
					identity.Relations[field.Name] = i
				}
			}
		case reflect.Slice:
			// "has many" relation
			parents = append(parents, t)
			if i := parseIdentity(db, fieldType.Elem(), parents); i != nil {
				identity.Relations[field.Name] = i
			}
		default:
			identity.Columns[columnName(db, field)] = column{
				Name: field.Name,
				Tag:  field.Tag,
			}
		}
	}

	identityCache[identifier] = identity
	return identity
}

func columnName(db *gorm.DB, field reflect.StructField) string {
	for _, t := range strings.Split(field.Tag.Get("gorm"), ";") { // Check for gorm column name override
		if strings.HasPrefix(t, "column") {
			i := strings.Index(t, ":")
			if i == -1 || i+1 >= len(t) {
				// Invalid syntax, fallback to auto-naming
				break
			}
			return strings.TrimSpace(t[i+1:])
		}
	}

	return db.NamingStrategy.ColumnName("", field.Name)
}

func getEmbeddedInfo(field reflect.StructField) (string, bool) {
	embedded := false
	embeddedPrefix := ""
	for _, t := range strings.Split(field.Tag.Get("gorm"), ";") { // Check for gorm column name override
		if t == "embedded" {
			embedded = true
		} else if strings.HasPrefix(t, "embeddedPrefix") {
			i := strings.Index(t, ":")
			if i == -1 || i+1 >= len(t) {
				continue
			}
			embeddedPrefix = strings.TrimSpace(t[i+1:])
		}
	}

	return embeddedPrefix, embedded
}

func checkCycle(t reflect.Type, parents []reflect.Type) bool {
	for _, v := range parents {
		if t == v {
			return true
		}
	}
	return false
}
