package filter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm/schema"
)

func TestCleanColumns(t *testing.T) {
	sch := &schema.Schema{
		FieldsByDBName: map[string]*schema.Field{
			"id":   {},
			"name": {},
		},
	}
	assert.Equal(t, []*schema.Field{sch.FieldsByDBName["name"]}, cleanColumns(sch, []string{"id", "test", "name", "notacolumn"}, []string{"name"}))
}

func TestAddPrimaryKeys(t *testing.T) {
	schema := &schema.Schema{
		PrimaryFieldDBNames: []string{"id_1", "id_2"},
	}

	fields := []string{"id_2"}
	fields = addPrimaryKeys(schema, fields)
	assert.Equal(t, []string{"id_2", "id_1"}, fields)
}

func TestAddForeignKeys(t *testing.T) {
	schema := &schema.Schema{
		Relationships: schema.Relationships{
			Relations: map[string]*schema.Relationship{
				"Many": {
					Type: schema.Many2Many,
				},
				"HasMany": {
					Type: schema.HasMany,
				},
				"HasOne": {
					Type: schema.HasOne,
					References: []*schema.Reference{
						{ForeignKey: &schema.Field{DBName: "child_id"}},
					},
				},
				"BelongsTo": {
					Type: schema.BelongsTo,
					References: []*schema.Reference{
						{ForeignKey: &schema.Field{DBName: "parent_id"}},
					},
				},
			},
		},
	}
	fields := []string{"id"}
	fields = addForeignKeys(schema, fields)
	assert.ElementsMatch(t, []string{"id", "child_id", "parent_id"}, fields)
}
