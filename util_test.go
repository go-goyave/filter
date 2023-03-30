package filter

import (
	"fmt"
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

func TestConvertToSafeType(t *testing.T) {
	cases := []struct {
		want     interface{}
		dataType DataType
		value    string
		wantOk   bool
	}{
		// String
		{value: "string", dataType: DataTypeText, want: "string", wantOk: true},

		// Bool
		{value: "1", dataType: DataTypeBool, want: true, wantOk: true},
		{value: "on", dataType: DataTypeBool, want: true, wantOk: true},
		{value: "true", dataType: DataTypeBool, want: true, wantOk: true},
		{value: "yes", dataType: DataTypeBool, want: true, wantOk: true},
		{value: "0", dataType: DataTypeBool, want: false, wantOk: true},
		{value: "off", dataType: DataTypeBool, want: false, wantOk: true},
		{value: "false", dataType: DataTypeBool, want: false, wantOk: true},
		{value: "no", dataType: DataTypeBool, want: false, wantOk: true},
		{value: "not a bool", dataType: DataTypeBool, want: nil, wantOk: false},

		// Float
		{value: "1", dataType: DataTypeFloat, want: 1.0, wantOk: true},
		{value: "1.0", dataType: DataTypeFloat, want: 1.0, wantOk: true},
		{value: "1.23", dataType: DataTypeFloat, want: 1.23, wantOk: true},
		{value: "string", dataType: DataTypeFloat, want: nil, wantOk: false},

		// Int
		{value: "1", dataType: DataTypeInt, want: int64(1), wantOk: true},
		{value: "-2", dataType: DataTypeInt, want: int64(-2), wantOk: true},
		{value: "1.23", dataType: DataTypeInt, want: nil, wantOk: false},
		{value: "string", dataType: DataTypeInt, want: nil, wantOk: false},

		// Uint
		{value: "1", dataType: DataTypeUint, want: uint64(1), wantOk: true},
		{value: "-2", dataType: DataTypeUint, want: nil, wantOk: false},
		{value: "1.23", dataType: DataTypeUint, want: nil, wantOk: false},
		{value: "string", dataType: DataTypeUint, want: nil, wantOk: false},

		// Time
		{value: "2023-03-23", dataType: DataTypeTime, want: "2023-03-23", wantOk: true},
		{value: "2023-03-23 12:13:24", dataType: DataTypeTime, want: "2023-03-23 12:13:24", wantOk: true},
		{value: "2023-03-23T12:13:24Z", dataType: DataTypeTime, want: "2023-03-23T12:13:24Z", wantOk: true},
		{value: "2023-03-23T12:13:24", dataType: DataTypeTime, want: nil, wantOk: false},
		{value: "not a date", dataType: DataTypeTime, want: nil, wantOk: false},
		{value: "1234", dataType: DataTypeTime, want: nil, wantOk: false},

		// Unsupported
		{value: "1234", dataType: DataTypeUnsupported, want: nil, wantOk: false},
	}

	for _, c := range cases {
		c := c
		t.Run(fmt.Sprintf("%s_%s", c.value, c.dataType), func(t *testing.T) {
			val, ok := ConvertToSafeType(c.value, c.dataType)
			assert.Equal(t, c.want, val)
			assert.Equal(t, c.wantOk, ok)
		})
	}
}
