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
		{value: "string", dataType: DataTypeTextArray, want: "string", wantOk: true},

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
		{value: "1", dataType: DataTypeBoolArray, want: true, wantOk: true},
		{value: "on", dataType: DataTypeBoolArray, want: true, wantOk: true},
		{value: "true", dataType: DataTypeBoolArray, want: true, wantOk: true},
		{value: "yes", dataType: DataTypeBoolArray, want: true, wantOk: true},
		{value: "0", dataType: DataTypeBoolArray, want: false, wantOk: true},
		{value: "off", dataType: DataTypeBoolArray, want: false, wantOk: true},
		{value: "false", dataType: DataTypeBoolArray, want: false, wantOk: true},
		{value: "no", dataType: DataTypeBoolArray, want: false, wantOk: true},
		{value: "not a bool", dataType: DataTypeBoolArray, want: nil, wantOk: false},

		// Float
		{value: "1", dataType: DataTypeFloat, want: 1.0, wantOk: true},
		{value: "1.0", dataType: DataTypeFloat, want: 1.0, wantOk: true},
		{value: "1.23", dataType: DataTypeFloat, want: 1.23, wantOk: true},
		{value: "string", dataType: DataTypeFloat, want: nil, wantOk: false},
		{value: "1", dataType: DataTypeFloatArray, want: 1.0, wantOk: true},
		{value: "1.0", dataType: DataTypeFloatArray, want: 1.0, wantOk: true},
		{value: "1.23", dataType: DataTypeFloatArray, want: 1.23, wantOk: true},
		{value: "string", dataType: DataTypeFloatArray, want: nil, wantOk: false},

		// Int
		{value: "1", dataType: DataTypeInt, want: int64(1), wantOk: true},
		{value: "-2", dataType: DataTypeInt, want: int64(-2), wantOk: true},
		{value: "1.23", dataType: DataTypeInt, want: nil, wantOk: false},
		{value: "string", dataType: DataTypeInt, want: nil, wantOk: false},
		{value: "1", dataType: DataTypeIntArray, want: int64(1), wantOk: true},
		{value: "-2", dataType: DataTypeIntArray, want: int64(-2), wantOk: true},
		{value: "1.23", dataType: DataTypeIntArray, want: nil, wantOk: false},
		{value: "string", dataType: DataTypeIntArray, want: nil, wantOk: false},

		// Uint
		{value: "1", dataType: DataTypeUint, want: uint64(1), wantOk: true},
		{value: "-2", dataType: DataTypeUint, want: nil, wantOk: false},
		{value: "1.23", dataType: DataTypeUint, want: nil, wantOk: false},
		{value: "string", dataType: DataTypeUint, want: nil, wantOk: false},
		{value: "1", dataType: DataTypeUintArray, want: uint64(1), wantOk: true},
		{value: "-2", dataType: DataTypeUintArray, want: nil, wantOk: false},
		{value: "1.23", dataType: DataTypeUintArray, want: nil, wantOk: false},
		{value: "string", dataType: DataTypeUintArray, want: nil, wantOk: false},

		// Time
		{value: "2023-03-23", dataType: DataTypeTime, want: "2023-03-23", wantOk: true},
		{value: "2023-03-23 12:13:24", dataType: DataTypeTime, want: "2023-03-23 12:13:24", wantOk: true},
		{value: "2023-03-23T12:13:24Z", dataType: DataTypeTime, want: "2023-03-23T12:13:24Z", wantOk: true},
		{value: "2022-11-02T09:12:03.081967+01:00", dataType: DataTypeTime, want: "2022-11-02T09:12:03.081967+01:00", wantOk: true},
		{value: "2023-03-23T12:13:24", dataType: DataTypeTime, want: nil, wantOk: false},
		{value: "not a date", dataType: DataTypeTime, want: nil, wantOk: false},
		{value: "1234", dataType: DataTypeTime, want: nil, wantOk: false},
		{value: "2023-03-23", dataType: DataTypeTimeArray, want: "2023-03-23", wantOk: true},
		{value: "2023-03-23 12:13:24", dataType: DataTypeTimeArray, want: "2023-03-23 12:13:24", wantOk: true},
		{value: "2023-03-23T12:13:24Z", dataType: DataTypeTimeArray, want: "2023-03-23T12:13:24Z", wantOk: true},
		{value: "2022-11-02T09:12:03.081967+01:00", dataType: DataTypeTimeArray, want: "2022-11-02T09:12:03.081967+01:00", wantOk: true},
		{value: "2023-03-23T12:13:24", dataType: DataTypeTimeArray, want: nil, wantOk: false},
		{value: "not a date", dataType: DataTypeTimeArray, want: nil, wantOk: false},
		{value: "1234", dataType: DataTypeTimeArray, want: nil, wantOk: false},

		// Unsupported
		{value: "1234", dataType: DataTypeUnsupported, want: nil, wantOk: false},
		{value: "1234", dataType: "CHARACTER VARYING(255)[]", want: nil, wantOk: false},
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

func TestConvertArgsToSafeType(t *testing.T) {

	// No need for exhaustive testing here since it's already done by TestConvertToSafeType
	cases := []struct {
		want     interface{}
		dataType DataType
		value    []string
		wantOk   bool
	}{
		{value: []string{"a", "b"}, dataType: DataTypeText, want: []interface{}{"a", "b"}, wantOk: true},
		{value: []string{"3", "4"}, dataType: DataTypeInt, want: []interface{}{int64(3), int64(4)}, wantOk: true},
		{value: []string{"a", "2"}, dataType: DataTypeInt, want: []interface{}(nil), wantOk: false},
	}

	for _, c := range cases {
		c := c
		t.Run(fmt.Sprintf("%s_%s", c.value, c.dataType), func(t *testing.T) {
			val, ok := ConvertArgsToSafeType(c.value, c.dataType)
			assert.Equal(t, c.want, val)
			assert.Equal(t, c.wantOk, ok)
		})
	}
}

func TestGetDataType(t *testing.T) {
	cases := []struct {
		desc  string
		model interface{}
		want  DataType
	}{
		{desc: "no specified type", model: struct{ Field string }{}, want: DataTypeText},
		{desc: "gorm type string", model: struct {
			Field string `gorm:"type:string"`
		}{}, want: DataTypeText},
		{desc: "gorm type bool", model: struct {
			Field string `gorm:"type:bool"`
		}{}, want: DataTypeBool},
		{desc: "gorm type int", model: struct {
			Field string `gorm:"type:int"`
		}{}, want: DataTypeInt},
		{desc: "gorm type uint", model: struct {
			Field string `gorm:"type:uint"`
		}{}, want: DataTypeUint},
		{desc: "gorm type float", model: struct {
			Field string `gorm:"type:float"`
		}{}, want: DataTypeFloat},
		{desc: "gorm type time", model: struct {
			Field string `gorm:"type:time"`
		}{}, want: DataTypeTime},
		{desc: "gorm type bytes", model: struct {
			Field string `gorm:"type:bytes"`
		}{}, want: DataTypeUnsupported},
		{desc: "gorm type custom", model: struct {
			Field string `gorm:"type:CHARACTER VARYING(255)"`
		}{}, want: DataTypeUnsupported},

		{desc: "filter type unsupported", model: struct {
			Field string `filterType:"-"`
		}{}, want: DataTypeUnsupported},
		{desc: "filter type invalid", model: struct {
			Field string `filterType:"invalid"`
		}{}, want: DataTypeUnsupported},
		{desc: "filter type text", model: struct {
			Field string `filterType:"text"`
		}{}, want: DataTypeText},
		{desc: "filter type text array", model: struct {
			Field string `filterType:"text[]"`
		}{}, want: DataTypeTextArray},
		{desc: "filter type bool", model: struct {
			Field string `filterType:"bool"`
		}{}, want: DataTypeBool},
		{desc: "filter type bool array", model: struct {
			Field string `filterType:"bool[]"`
		}{}, want: DataTypeBoolArray},
		{desc: "filter type float", model: struct {
			Field string `filterType:"float"`
		}{}, want: DataTypeFloat},
		{desc: "filter type float array", model: struct {
			Field string `filterType:"float[]"`
		}{}, want: DataTypeFloatArray},
		{desc: "filter type int", model: struct {
			Field string `filterType:"int"`
		}{}, want: DataTypeInt},
		{desc: "filter type int array", model: struct {
			Field string `filterType:"int[]"`
		}{}, want: DataTypeIntArray},
		{desc: "filter type uint", model: struct {
			Field string `filterType:"uint"`
		}{}, want: DataTypeUint},
		{desc: "filter type uint array", model: struct {
			Field string `filterType:"uint[]"`
		}{}, want: DataTypeUintArray},
		{desc: "filter type time", model: struct {
			Field string `filterType:"time"`
		}{}, want: DataTypeTime},
		{desc: "filter type time array", model: struct {
			Field string `filterType:"time[]"`
		}{}, want: DataTypeTimeArray},

		{desc: "filter type has priority over gorm type", model: struct {
			Field string `gorm:"type:CHARACTER VARYING(255)" filterType:"text"`
		}{}, want: DataTypeText},
	}

	for _, c := range cases {
		c := c
		t.Run(c.desc, func(t *testing.T) {
			model, err := parseModel(openDryRunDB(t), c.model)
			if !assert.NoError(t, err) {
				return
			}

			getDataType(model.Fields[0])
		})
	}
}
