package filter

import (
	"gorm.io/gorm/schema"
	"goyave.dev/goyave/v4/util/sliceutil"
)

func cleanColumns(schema *schema.Schema, columns []string, blacklist []string) []string {
	for j := 0; j < len(columns); j++ {
		_, ok := schema.FieldsByDBName[columns[j]]
		if !ok || sliceutil.ContainsStr(blacklist, columns[j]) {
			columns = append(columns[:j], columns[j+1:]...)
			j--
		}
	}

	return columns
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
