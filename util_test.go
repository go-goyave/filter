package filter

import (
	"fmt"
	"testing"
	"time"

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
		want     any
		dataType DataType
		value    string
		wantOk   bool
	}{
		// String
		{value: "string", dataType: DataTypeText, want: "string", wantOk: true},
		{value: "string", dataType: DataTypeTextArray, want: "string", wantOk: true},

		// Enum
		{value: "string", dataType: DataTypeEnum, want: "string", wantOk: true},
		{value: "string", dataType: DataTypeEnumArray, want: "string", wantOk: true},

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

		// Float32
		{value: "1", dataType: DataTypeFloat32, want: 1.0, wantOk: true},
		{value: "1.0", dataType: DataTypeFloat32, want: 1.0, wantOk: true},
		{value: "1.23", dataType: DataTypeFloat32, want: 1.2300000190734863, wantOk: true}, // Precision loss
		{value: "string", dataType: DataTypeFloat32, want: 0.0, wantOk: false},
		{value: "1", dataType: DataTypeFloat32Array, want: 1.0, wantOk: true},
		{value: "1.0", dataType: DataTypeFloat32Array, want: 1.0, wantOk: true},
		{value: "1.23", dataType: DataTypeFloat32Array, want: 1.2300000190734863, wantOk: true}, // Precision loss
		{value: "string", dataType: DataTypeFloat32Array, want: 0.0, wantOk: false},

		// Float64
		{value: "1", dataType: DataTypeFloat64, want: 1.0, wantOk: true},
		{value: "1.0", dataType: DataTypeFloat64, want: 1.0, wantOk: true},
		{value: "1.23", dataType: DataTypeFloat64, want: 1.23, wantOk: true},
		{value: "string", dataType: DataTypeFloat64, want: 0.0, wantOk: false},
		{value: "1", dataType: DataTypeFloat64Array, want: 1.0, wantOk: true},
		{value: "1.0", dataType: DataTypeFloat64Array, want: 1.0, wantOk: true},
		{value: "1.23", dataType: DataTypeFloat64Array, want: 1.23, wantOk: true},
		{value: "string", dataType: DataTypeFloat64Array, want: 0.0, wantOk: false},

		// Int8
		{value: "1", dataType: DataTypeInt8, want: int64(1), wantOk: true},
		{value: "-2", dataType: DataTypeInt8, want: int64(-2), wantOk: true},
		{value: "128", dataType: DataTypeInt8, want: int64(0), wantOk: false},
		{value: "-129", dataType: DataTypeInt8, want: int64(0), wantOk: false},
		{value: "1.23", dataType: DataTypeInt8, want: int64(0), wantOk: false},
		{value: "string", dataType: DataTypeInt8, want: int64(0), wantOk: false},
		{value: "1", dataType: DataTypeInt8Array, want: int64(1), wantOk: true},
		{value: "-2", dataType: DataTypeInt8Array, want: int64(-2), wantOk: true},
		{value: "1.23", dataType: DataTypeInt8Array, want: int64(0), wantOk: false},
		{value: "string", dataType: DataTypeInt8Array, want: int64(0), wantOk: false},

		// Int16
		{value: "1", dataType: DataTypeInt16, want: int64(1), wantOk: true},
		{value: "-2", dataType: DataTypeInt16, want: int64(-2), wantOk: true},
		{value: "32768", dataType: DataTypeInt16, want: int64(0), wantOk: false},
		{value: "-32769", dataType: DataTypeInt16, want: int64(0), wantOk: false},
		{value: "1.23", dataType: DataTypeInt16, want: int64(0), wantOk: false},
		{value: "string", dataType: DataTypeInt16, want: int64(0), wantOk: false},
		{value: "1", dataType: DataTypeInt16Array, want: int64(1), wantOk: true},
		{value: "-2", dataType: DataTypeInt16Array, want: int64(-2), wantOk: true},
		{value: "1.23", dataType: DataTypeInt16Array, want: int64(0), wantOk: false},
		{value: "string", dataType: DataTypeInt16Array, want: int64(0), wantOk: false},

		// Int32
		{value: "1", dataType: DataTypeInt32, want: int64(1), wantOk: true},
		{value: "-2", dataType: DataTypeInt32, want: int64(-2), wantOk: true},
		{value: "2147483648", dataType: DataTypeInt32, want: int64(0), wantOk: false},
		{value: "-2147483649", dataType: DataTypeInt32, want: int64(0), wantOk: false},
		{value: "1.23", dataType: DataTypeInt32, want: int64(0), wantOk: false},
		{value: "string", dataType: DataTypeInt32, want: int64(0), wantOk: false},
		{value: "1", dataType: DataTypeInt32Array, want: int64(1), wantOk: true},
		{value: "-2", dataType: DataTypeInt32Array, want: int64(-2), wantOk: true},
		{value: "1.23", dataType: DataTypeInt32Array, want: int64(0), wantOk: false},
		{value: "string", dataType: DataTypeInt32Array, want: int64(0), wantOk: false},

		// Int64
		{value: "1", dataType: DataTypeInt64, want: int64(1), wantOk: true},
		{value: "-2", dataType: DataTypeInt64, want: int64(-2), wantOk: true},
		{value: "1.23", dataType: DataTypeInt64, want: int64(0), wantOk: false},
		{value: "string", dataType: DataTypeInt64, want: int64(0), wantOk: false},
		{value: "1", dataType: DataTypeInt64Array, want: int64(1), wantOk: true},
		{value: "-2", dataType: DataTypeInt64Array, want: int64(-2), wantOk: true},
		{value: "1.23", dataType: DataTypeInt64Array, want: int64(0), wantOk: false},
		{value: "string", dataType: DataTypeInt64Array, want: int64(0), wantOk: false},

		// Uint8
		{value: "1", dataType: DataTypeUint8, want: uint64(1), wantOk: true},
		{value: "256", dataType: DataTypeUint8, want: uint64(0), wantOk: false},
		{value: "-2", dataType: DataTypeUint8, want: uint64(0), wantOk: false},
		{value: "1.23", dataType: DataTypeUint8, want: uint64(0), wantOk: false},
		{value: "string", dataType: DataTypeUint8, want: uint64(0), wantOk: false},
		{value: "1", dataType: DataTypeUint8Array, want: uint64(1), wantOk: true},
		{value: "-2", dataType: DataTypeUint8Array, want: uint64(0), wantOk: false},
		{value: "1.23", dataType: DataTypeUint8Array, want: uint64(0), wantOk: false},
		{value: "string", dataType: DataTypeUint8Array, want: uint64(0), wantOk: false},

		// Uint16
		{value: "1", dataType: DataTypeUint16, want: uint64(1), wantOk: true},
		{value: "65536", dataType: DataTypeUint16, want: uint64(0), wantOk: false},
		{value: "-2", dataType: DataTypeUint16, want: uint64(0), wantOk: false},
		{value: "1.23", dataType: DataTypeUint16, want: uint64(0), wantOk: false},
		{value: "string", dataType: DataTypeUint16, want: uint64(0), wantOk: false},
		{value: "1", dataType: DataTypeUint16Array, want: uint64(1), wantOk: true},
		{value: "-2", dataType: DataTypeUint16Array, want: uint64(0), wantOk: false},
		{value: "1.23", dataType: DataTypeUint16Array, want: uint64(0), wantOk: false},
		{value: "string", dataType: DataTypeUint16Array, want: uint64(0), wantOk: false},

		// Uint32
		{value: "1", dataType: DataTypeUint32, want: uint64(1), wantOk: true},
		{value: "4294967296", dataType: DataTypeUint32, want: uint64(0), wantOk: false},
		{value: "-2", dataType: DataTypeUint32, want: uint64(0), wantOk: false},
		{value: "1.23", dataType: DataTypeUint32, want: uint64(0), wantOk: false},
		{value: "string", dataType: DataTypeUint32, want: uint64(0), wantOk: false},
		{value: "1", dataType: DataTypeUint32Array, want: uint64(1), wantOk: true},
		{value: "-2", dataType: DataTypeUint32Array, want: uint64(0), wantOk: false},
		{value: "1.23", dataType: DataTypeUint32Array, want: uint64(0), wantOk: false},
		{value: "string", dataType: DataTypeUint32Array, want: uint64(0), wantOk: false},

		// Uint64
		{value: "1", dataType: DataTypeUint64, want: uint64(1), wantOk: true},
		{value: "-2", dataType: DataTypeUint64, want: uint64(0), wantOk: false},
		{value: "1.23", dataType: DataTypeUint64, want: uint64(0), wantOk: false},
		{value: "string", dataType: DataTypeUint64, want: uint64(0), wantOk: false},
		{value: "1", dataType: DataTypeUint64Array, want: uint64(1), wantOk: true},
		{value: "-2", dataType: DataTypeUint64Array, want: uint64(0), wantOk: false},
		{value: "1.23", dataType: DataTypeUint64Array, want: uint64(0), wantOk: false},
		{value: "string", dataType: DataTypeUint64Array, want: uint64(0), wantOk: false},

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
		want     any
		dataType DataType
		value    []string
		wantOk   bool
	}{
		{value: []string{"a", "b"}, dataType: DataTypeText, want: []any{"a", "b"}, wantOk: true},
		{value: []string{"3", "4"}, dataType: DataTypeInt64, want: []any{int64(3), int64(4)}, wantOk: true},
		{value: []string{"a", "2"}, dataType: DataTypeInt64, want: []any(nil), wantOk: false},
	}

	for _, c := range cases {
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
		model any
		want  DataType
	}{
		{desc: "gorm type string", model: struct {
			Field string `gorm:"type:string"`
		}{}, want: DataTypeText},
		{desc: "gorm type bool", model: struct {
			Field string `gorm:"type:bool"`
		}{}, want: DataTypeBool},
		{desc: "gorm type int", model: struct {
			Field string `gorm:"type:int"`
		}{}, want: DataTypeInt64},
		{desc: "gorm type uint", model: struct {
			Field string `gorm:"type:uint"`
		}{}, want: DataTypeUint64},
		{desc: "gorm type float", model: struct {
			Field string `gorm:"type:float"`
		}{}, want: DataTypeFloat64},
		{desc: "gorm type time", model: struct {
			Field string `gorm:"type:time"`
		}{}, want: DataTypeTime},
		{desc: "gorm type bytes", model: struct {
			Field string `gorm:"type:bytes"`
		}{}, want: DataTypeUnsupported},
		{desc: "gorm type custom", model: struct {
			Field string `gorm:"type:CHARACTER VARYING(255)"`
		}{}, want: DataTypeUnsupported},

		{desc: "gorm auto type string", model: struct {
			Field string
		}{}, want: DataTypeText},
		{desc: "gorm auto type bool", model: struct {
			Field bool
		}{}, want: DataTypeBool},
		{desc: "gorm auto type int8", model: struct {
			Field int8
		}{}, want: DataTypeInt8},
		{desc: "gorm auto type int16", model: struct {
			Field int16
		}{}, want: DataTypeInt16},
		{desc: "gorm auto type int32", model: struct {
			Field int32
		}{}, want: DataTypeInt32},
		{desc: "gorm auto type int64", model: struct {
			Field int64
		}{}, want: DataTypeInt64},
		{desc: "gorm auto type uint8", model: struct {
			Field uint8
		}{}, want: DataTypeUint8},
		{desc: "gorm auto type uint16", model: struct {
			Field uint16
		}{}, want: DataTypeUint16},
		{desc: "gorm auto type uint32", model: struct {
			Field uint32
		}{}, want: DataTypeUint32},
		{desc: "gorm auto type uint64", model: struct {
			Field uint64
		}{}, want: DataTypeUint64},
		{desc: "gorm auto type float32", model: struct {
			Field float32
		}{}, want: DataTypeFloat32},
		{desc: "gorm auto type float64", model: struct {
			Field float64
		}{}, want: DataTypeFloat64},
		{desc: "gorm auto type time", model: struct {
			Field time.Time
		}{}, want: DataTypeTime},

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
		{desc: "filter type enum", model: struct {
			Field string `filterType:"enum"`
		}{}, want: DataTypeEnum},
		{desc: "filter type enum array", model: struct {
			Field string `filterType:"enum[]"`
		}{}, want: DataTypeEnumArray},
		{desc: "filter type bool", model: struct {
			Field string `filterType:"bool"`
		}{}, want: DataTypeBool},
		{desc: "filter type bool array", model: struct {
			Field string `filterType:"bool[]"`
		}{}, want: DataTypeBoolArray},
		{desc: "filter type float", model: struct {
			Field string `filterType:"float64"`
		}{}, want: DataTypeFloat64},
		{desc: "filter type float array", model: struct {
			Field string `filterType:"float64[]"`
		}{}, want: DataTypeFloat64Array},
		{desc: "filter type int", model: struct {
			Field string `filterType:"int64"`
		}{}, want: DataTypeInt64},
		{desc: "filter type int array", model: struct {
			Field string `filterType:"int64[]"`
		}{}, want: DataTypeInt64Array},
		{desc: "filter type uint", model: struct {
			Field string `filterType:"uint64"`
		}{}, want: DataTypeUint64},
		{desc: "filter type uint array", model: struct {
			Field string `filterType:"uint64[]"`
		}{}, want: DataTypeUint64Array},
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
		t.Run(c.desc, func(t *testing.T) {
			model, err := parseModel(openDryRunDB(t), c.model)
			if !assert.NoError(t, err) {
				return
			}

			getDataType(model.Fields[0])
		})
	}
}
