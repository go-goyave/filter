package filter

import (
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm/schema"
	"goyave.dev/goyave/v4/util/sliceutil"
)

// DataType is determined by the `filterType` struct tag (see `DataType` for available options).
// If not given, uses GORM's general DataType. Raw database data types are not supported so it is
// recommended to always specify a `filterType` in this scenario.
type DataType string

// IsArray returns true if this data type is an array.
func (d DataType) IsArray() bool {
	return strings.HasSuffix(string(d), "[]")
}

// Supported DataTypes
const (
	DataTypeText      DataType = "text"
	DataTypeTextArray DataType = "text[]"

	DataTypeBool      DataType = "bool"
	DataTypeBoolArray DataType = "bool[]"

	DataTypeInt8       DataType = "int8"
	DataTypeInt8Array  DataType = "int8[]"
	DataTypeInt16      DataType = "int16"
	DataTypeInt16Array DataType = "int16[]"
	DataTypeInt32      DataType = "int32"
	DataTypeInt32Array DataType = "int32[]"
	DataTypeInt64      DataType = "int64"
	DataTypeInt64Array DataType = "int64[]"

	DataTypeUint8       DataType = "uint8"
	DataTypeUint8Array  DataType = "uint8[]"
	DataTypeUint16      DataType = "uint16"
	DataTypeUint16Array DataType = "uint16[]"
	DataTypeUint32      DataType = "uint32"
	DataTypeUint32Array DataType = "uint32[]"
	DataTypeUint64      DataType = "uint64"
	DataTypeUint64Array DataType = "uint64[]"

	DataTypeFloat32      DataType = "float32"
	DataTypeFloat32Array DataType = "float32[]"
	DataTypeFloat64      DataType = "float64"
	DataTypeFloat64Array DataType = "float64[]"

	DataTypeTime      DataType = "time"
	DataTypeTimeArray DataType = "time[]"

	// DataTypeUnsupported all fields with this tag will be ignored in filters and search.
	DataTypeUnsupported DataType = "-"
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

func getDataType(field *schema.Field) DataType {
	fromTag := DataType(strings.ToLower(field.Tag.Get("filterType")))
	switch fromTag {
	case DataTypeText, DataTypeTextArray,
		DataTypeBool, DataTypeBoolArray,
		DataTypeFloat32, DataTypeFloat32Array,
		DataTypeFloat64, DataTypeFloat64Array,
		DataTypeInt8, DataTypeInt16, DataTypeInt32, DataTypeInt64,
		DataTypeInt8Array, DataTypeInt16Array, DataTypeInt32Array, DataTypeInt64Array,
		DataTypeUint8, DataTypeUint16, DataTypeUint32, DataTypeUint64,
		DataTypeUint8Array, DataTypeUint16Array, DataTypeUint32Array, DataTypeUint64Array,
		DataTypeTime, DataTypeTimeArray,
		DataTypeUnsupported:
		return fromTag
	case "":
		switch field.GORMDataType {
		case schema.String:
			return DataTypeText
		case schema.Bool:
			return DataTypeBool
		case schema.Float:
			switch field.Size {
			case 32:
				return DataTypeFloat32
			case 64:
				return DataTypeFloat64
			}
		case schema.Int:
			switch field.Size {
			case 8:
				return DataTypeInt8
			case 16:
				return DataTypeInt16
			case 32:
				return DataTypeInt32
			case 64:
				return DataTypeInt64
			}
		case schema.Uint:
			switch field.Size {
			case 8:
				return DataTypeUint8
			case 16:
				return DataTypeUint16
			case 32:
				return DataTypeUint32
			case 64:
				return DataTypeUint64
			}
		case schema.Time:
			return DataTypeTime
		}
	}
	return DataTypeUnsupported
}

// ConvertToSafeType convert the string argument to a safe type that
// matches the column's data type. Returns false if the input could not
// be converted.
func ConvertToSafeType(arg string, dataType DataType) (interface{}, bool) {
	switch dataType {
	case DataTypeText, DataTypeTextArray:
		return arg, true
	case DataTypeBool, DataTypeBoolArray:
		switch arg {
		case "1", "on", "true", "yes":
			return true, true
		case "0", "off", "false", "no":
			return false, true
		}
		return nil, false
	case DataTypeFloat32, DataTypeFloat32Array:
		return validateFloat(arg, 32)
	case DataTypeFloat64, DataTypeFloat64Array:
		return validateFloat(arg, 64)
	case DataTypeInt8, DataTypeInt8Array:
		return validateInt(arg, 8)
	case DataTypeInt16, DataTypeInt16Array:
		return validateInt(arg, 16)
	case DataTypeInt32, DataTypeInt32Array:
		return validateInt(arg, 32)
	case DataTypeInt64, DataTypeInt64Array:
		return validateInt(arg, 64)
	case DataTypeUint8, DataTypeUint8Array:
		return validateUint(arg, 8)
	case DataTypeUint16, DataTypeUint16Array:
		return validateUint(arg, 16)
	case DataTypeUint32, DataTypeUint32Array:
		return validateUint(arg, 32)
	case DataTypeUint64, DataTypeUint64Array:
		return validateUint(arg, 64)
	case DataTypeTime, DataTypeTimeArray:
		if validateTime(arg) {
			return arg, true
		}
	}
	return nil, false
}

func validateInt(arg string, bitSize int) (int64, bool) {
	i, err := strconv.ParseInt(arg, 10, bitSize)
	if err != nil {
		return 0, false
	}
	return i, true
}

func validateUint(arg string, bitSize int) (uint64, bool) {
	i, err := strconv.ParseUint(arg, 10, bitSize)
	if err != nil {
		return 0, false
	}
	return i, true
}

func validateFloat(arg string, bitSize int) (float64, bool) {
	i, err := strconv.ParseFloat(arg, bitSize)
	if err != nil {
		return 0, false
	}
	return i, true
}

func validateTime(timeStr string) bool {
	for _, format := range []string{time.RFC3339, time.RFC3339Nano, "2006-01-02 15:04:05", "2006-01-02"} {
		_, err := time.Parse(format, timeStr)
		if err == nil {
			return true
		}
	}

	return false
}

// ConvertArgsToSafeType converts a slice of string arguments to safe type
// that matches the column's data type in the same way as `ConvertToSafeType`.
// If any of the values in the given slice could not be converted, returns false.
func ConvertArgsToSafeType(args []string, dataType DataType) ([]interface{}, bool) {
	result := make([]interface{}, 0, len(args))
	for _, arg := range args {
		a, ok := ConvertToSafeType(arg, dataType)
		if !ok {
			return nil, false
		}
		result = append(result, a)
	}
	return result, true
}
