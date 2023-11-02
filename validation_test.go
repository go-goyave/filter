package filter

import (
	"fmt"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"goyave.dev/goyave/v5/validation"
)

func TestApplyValidation(t *testing.T) {
	set := Validation(nil)

	expectedFields := []string{"filter", "filter[]", "or", "or[]", "sort", "sort[]", "join", "join[]", "fields", "page", "per_page", "search"}
	assert.True(t, lo.EveryBy(set, func(f *validation.FieldRules) bool {
		return lo.Contains(expectedFields, f.Path)
	}))
}

func TestParseFilter(t *testing.T) {
	f, err := ParseFilter("field||$eq||value1,value2")
	assert.Nil(t, err)
	if assert.NotNil(t, f) {
		assert.Equal(t, "field", f.Field)
		assert.Equal(t, Operators["$eq"], f.Operator)
		assert.Equal(t, []string{"value1", "value2"}, f.Args)
	}

	f, err = ParseFilter(" field || $eq || value1 , value2 ")
	assert.Nil(t, err)
	if assert.NotNil(t, f) {
		assert.Equal(t, "field", f.Field)
		assert.Equal(t, Operators["$eq"], f.Operator)
		assert.Equal(t, []string{"value1", "value2"}, f.Args)
	}

	f, err = ParseFilter("field")
	assert.Nil(t, f)
	if assert.NotNil(t, err) {
		assert.Equal(t, "missing operator", err.Error())
	}

	f, err = ParseFilter("||$eq")
	assert.Nil(t, f)
	if assert.NotNil(t, err) {
		assert.Equal(t, "invalid filter syntax", err.Error())
	}

	f, err = ParseFilter("field||$notanoperator")
	assert.Nil(t, f)
	if assert.NotNil(t, err) {
		assert.Equal(t, "unknown operator: \"$notanoperator\"", err.Error())
	}

	f, err = ParseFilter("field||$eq||,")
	assert.Nil(t, f)
	if assert.NotNil(t, err) {
		assert.Equal(t, "invalid filter syntax", err.Error())
	}

	f, err = ParseFilter("field||$eq")
	assert.Nil(t, f)
	if assert.NotNil(t, err) {
		assert.Equal(t, "operator \"$eq\" requires at least 1 argument(s)", err.Error())
	}
}

func TestParseSort(t *testing.T) {
	s, err := ParseSort("name,ASC")
	assert.Nil(t, err)
	if assert.NotNil(t, s) {
		assert.Equal(t, "name", s.Field)
		assert.Equal(t, SortAscending, s.Order)
	}

	s, err = ParseSort("name,asc")
	assert.Nil(t, err)
	if assert.NotNil(t, s) {
		assert.Equal(t, "name", s.Field)
		assert.Equal(t, SortAscending, s.Order)
	}

	s, err = ParseSort("name,desc")
	assert.Nil(t, err)
	if assert.NotNil(t, s) {
		assert.Equal(t, "name", s.Field)
		assert.Equal(t, SortDescending, s.Order)
	}
	s, err = ParseSort(" name , desc ")
	assert.Nil(t, err)
	if assert.NotNil(t, s) {
		assert.Equal(t, "name", s.Field)
		assert.Equal(t, SortDescending, s.Order)
	}

	s, err = ParseSort("name")
	assert.Nil(t, s)
	if assert.NotNil(t, err) {
		assert.Equal(t, "invalid sort syntax", err.Error())
	}

	s, err = ParseSort(",DESC")
	assert.Nil(t, s)
	if assert.NotNil(t, err) {
		assert.Equal(t, "invalid sort syntax", err.Error())
	}

	s, err = ParseSort("name,")
	assert.Nil(t, s)
	if assert.NotNil(t, err) {
		assert.Equal(t, "invalid sort syntax", err.Error())
	}

	s, err = ParseSort("name,notanorder")
	assert.Nil(t, s)
	if assert.NotNil(t, err) {
		assert.Equal(t, "invalid sort order \"NOTANORDER\"", err.Error())
	}
}

func TestParseJoin(t *testing.T) {
	j, err := ParseJoin("relation||field1,field2")
	assert.Nil(t, err)
	if assert.NotNil(t, j) {
		assert.Equal(t, "relation", j.Relation)
		assert.Equal(t, []string{"field1", "field2"}, j.Fields)
	}

	j, err = ParseJoin(" relation || field1 , field2 ")
	assert.Nil(t, err)
	if assert.NotNil(t, j) {
		assert.Equal(t, "relation", j.Relation)
		assert.Equal(t, []string{"field1", "field2"}, j.Fields)
	}

	j, err = ParseJoin("relation")
	assert.Nil(t, err)
	if assert.NotNil(t, j) {
		assert.Equal(t, "relation", j.Relation)
		assert.Nil(t, j.Fields)
	}

	j, err = ParseJoin("relation||field1")
	assert.Nil(t, err)
	if assert.NotNil(t, j) {
		assert.Equal(t, "relation", j.Relation)
		assert.Equal(t, []string{"field1"}, j.Fields)
	}

	j, err = ParseJoin("relation||")
	assert.Nil(t, err)
	if assert.NotNil(t, j) {
		assert.Equal(t, "relation", j.Relation)
		assert.Nil(t, j.Fields)
	}

	j, err = ParseJoin("relation||,")
	assert.Nil(t, j)
	if assert.NotNil(t, err) {
		assert.Equal(t, "invalid join syntax", err.Error())
	}

	j, err = ParseJoin("||field1,field2")
	assert.Nil(t, j)
	if assert.NotNil(t, err) {
		assert.Equal(t, "invalid join syntax", err.Error())
	}
}

func TestValidateFilter(t *testing.T) {

	t.Run("Constructor", func(t *testing.T) {
		v := &FilterValidator{}
		assert.NotNil(t, v)
		assert.Equal(t, "goyave-filter-filter", v.Name())
		assert.True(t, v.IsType())
		assert.False(t, v.IsTypeDependent())
		assert.Empty(t, v.MessagePlaceholders(&validation.Context{}))
	})

	cases := []struct {
		value     any
		wantValue *Filter
		or        bool
		want      bool
	}{
		{
			value: "field||$eq||value1,value2",
			or:    false,
			want:  true,
			wantValue: &Filter{
				Field:    "field",
				Operator: Operators["$eq"],
				Args:     []string{"value1", "value2"},
				Or:       false,
			},
		},
		{
			value: "field||$eq||value1,value2",
			or:    true,
			want:  true,
			wantValue: &Filter{
				Field:    "field",
				Operator: Operators["$eq"],
				Args:     []string{"value1", "value2"},
				Or:       true,
			},
		},
		{
			value: 5,
			want:  false,
		},
		{
			value: "",
			want:  false,
		},
		{
			value: &Filter{
				Field:    "field",
				Operator: Operators["$eq"],
				Args:     []string{"value1", "value2"},
				Or:       false,
			},
			want: true,
			wantValue: &Filter{
				Field:    "field",
				Operator: Operators["$eq"],
				Args:     []string{"value1", "value2"},
				Or:       false,
			},
		},
	}

	for _, c := range cases {
		c := c
		t.Run(fmt.Sprintf("Validate_%v_%t", c.value, c.want), func(t *testing.T) {
			v := &FilterValidator{Or: c.or}
			ctx := &validation.Context{
				Value: c.value,
			}
			assert.Equal(t, c.want, v.Validate(ctx))
			if c.wantValue != nil {
				assert.Equal(t, c.wantValue, ctx.Value)
			}
		})
	}
}

func TestValidateSort(t *testing.T) {

	t.Run("Constructor", func(t *testing.T) {
		v := &SortValidator{}
		assert.NotNil(t, v)
		assert.Equal(t, "goyave-filter-sort", v.Name())
		assert.True(t, v.IsType())
		assert.False(t, v.IsTypeDependent())
		assert.Empty(t, v.MessagePlaceholders(&validation.Context{}))
	})

	cases := []struct {
		value     any
		wantValue *Sort
		want      bool
	}{
		{
			value: "name,ASC",
			want:  true,
			wantValue: &Sort{
				Field: "name",
				Order: SortAscending,
			},
		},
		{
			value: "name,DESC",
			want:  true,
			wantValue: &Sort{
				Field: "name",
				Order: SortDescending,
			},
		},
		{
			value: "name",
			want:  false,
		},
		{
			value: 5,
			want:  false,
		},
		{
			value: "",
			want:  false,
		},
		{
			value: &Sort{
				Field: "name",
				Order: SortAscending,
			},
			want: true,
			wantValue: &Sort{
				Field: "name",
				Order: SortAscending,
			},
		},
	}

	for _, c := range cases {
		c := c
		t.Run(fmt.Sprintf("Validate_%v_%t", c.value, c.want), func(t *testing.T) {
			v := &SortValidator{}
			ctx := &validation.Context{
				Value: c.value,
			}
			assert.Equal(t, c.want, v.Validate(ctx))
			if c.wantValue != nil {
				assert.Equal(t, c.wantValue, ctx.Value)
			}
		})
	}
}

func TestValidateJoin(t *testing.T) {

	t.Run("Constructor", func(t *testing.T) {
		v := &JoinValidator{}
		assert.NotNil(t, v)
		assert.Equal(t, "goyave-filter-join", v.Name())
		assert.True(t, v.IsType())
		assert.False(t, v.IsTypeDependent())
		assert.Empty(t, v.MessagePlaceholders(&validation.Context{}))
	})

	cases := []struct {
		value     any
		wantValue *Join
		want      bool
	}{
		{
			value: "relation||field1,field2",
			want:  true,
			wantValue: &Join{
				Relation: "relation",
				Fields:   []string{"field1", "field2"},
			},
		},
		{
			value: "relation",
			want:  true,
			wantValue: &Join{
				Relation: "relation",
			},
		},
		{
			value: 5,
			want:  false,
		},
		{
			value: "",
			want:  false,
		},
		{
			value: &Join{
				Relation: "relation",
				Fields:   []string{"field1", "field2"},
			},
			want: true,
			wantValue: &Join{
				Relation: "relation",
				Fields:   []string{"field1", "field2"},
			},
		},
	}

	for _, c := range cases {
		c := c
		t.Run(fmt.Sprintf("Validate_%v_%t", c.value, c.want), func(t *testing.T) {
			v := &JoinValidator{}
			ctx := &validation.Context{
				Value: c.value,
			}
			assert.Equal(t, c.want, v.Validate(ctx))
			if c.wantValue != nil {
				assert.Equal(t, c.wantValue, ctx.Value)
			}
		})
	}
}

func TestValidateFields(t *testing.T) {

	t.Run("Constructor", func(t *testing.T) {
		v := &FieldsValidator{}
		assert.NotNil(t, v)
		assert.Equal(t, "goyave-filter-fields", v.Name())
		assert.True(t, v.IsType())
		assert.False(t, v.IsTypeDependent())
		assert.Empty(t, v.MessagePlaceholders(&validation.Context{}))
	})

	cases := []struct {
		value     any
		wantValue any
		want      bool
		invalid   bool
	}{
		{
			value:     []string{"field a", "field b  ", "  field c"},
			want:      true,
			wantValue: []string{"field a", "field b", "field c"},
		},
		{
			value:     "field a,field b   ,    field c",
			want:      true,
			wantValue: []string{"field a", "field b", "field c"},
		},
		{
			value:     123,
			invalid:   true,
			want:      true,
			wantValue: 123,
		},
	}

	for _, c := range cases {
		c := c
		t.Run(fmt.Sprintf("Validate_%v_%t", c.value, c.want), func(t *testing.T) {
			v := &FieldsValidator{}
			ctx := &validation.Context{
				Value:   c.value,
				Invalid: c.invalid,
			}
			assert.Equal(t, c.want, v.Validate(ctx))
			if c.wantValue != nil {
				assert.Equal(t, c.wantValue, ctx.Value)
			}
		})
	}
}
