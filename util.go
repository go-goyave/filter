package filter

import (
	"gorm.io/gorm/schema"
	"goyave.dev/goyave/v4/util/sliceutil"
)

func cleanColumns(sch *schema.Schema, columns []string, blacklist []string) []*schema.Field {
	fields := make([]*schema.Field, 0, len(columns))
	for _, c := range columns {
		f, ok := sch.FieldsByDBName[c]
		if ok && !sliceutil.ContainsStr(blacklist, c) {
			fields = append(fields, f)
		}
	}

	return fields
}

func addPrimaryKeys(schema *schema.Schema, fields []string) []string {
	for _, k := range schema.PrimaryFieldDBNames {
		if !sliceutil.ContainsStr(fields, k) {
			fields = append(fields, k)
		}
	}
	return fields
}

func addForeignKeys(sch *schema.Schema, fields []string) []string {
	for _, r := range sch.Relationships.Relations {
		if r.Type == schema.HasOne || r.Type == schema.BelongsTo {
			for _, ref := range r.References {
				if !sliceutil.ContainsStr(fields, ref.ForeignKey.DBName) {
					fields = append(fields, ref.ForeignKey.DBName)
				}
			}
		}
	}
	return fields
}

func columnsContain(fields []*schema.Field, field *schema.Field) bool {
	for _, f := range fields {
		if f.DBName == field.DBName {
			return true
		}
	}
	return false
}
