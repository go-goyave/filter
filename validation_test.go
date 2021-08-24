package filter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"goyave.dev/goyave/v3/validation"
)

func TestApplyValidation(t *testing.T) {
	set := validation.RuleSet{}
	ApplyValidation(set)

	assert.Contains(t, set, "filter")
	assert.Contains(t, set, "filter[]")
	assert.Contains(t, set, "or")
	assert.Contains(t, set, "or[]")
	assert.Contains(t, set, "sort")
	assert.Contains(t, set, "sort[]")
	assert.Contains(t, set, "join")
	assert.Contains(t, set, "join[]")
	assert.Contains(t, set, "fields")
	assert.Contains(t, set, "page")
	assert.Contains(t, set, "per_page")
}

func TestApplyValidationRules(t *testing.T) {
	set := &validation.Rules{Fields: validation.FieldMap{}}
	ApplyValidationRules(set)

	assert.Contains(t, set.Fields, "filter")
	assert.Contains(t, set.Fields, "filter[]")
	assert.Contains(t, set.Fields, "or")
	assert.Contains(t, set.Fields, "or[]")
	assert.Contains(t, set.Fields, "sort")
	assert.Contains(t, set.Fields, "sort[]")
	assert.Contains(t, set.Fields, "join")
	assert.Contains(t, set.Fields, "join[]")
	assert.Contains(t, set.Fields, "fields")
	assert.Contains(t, set.Fields, "page")
	assert.Contains(t, set.Fields, "per_page")
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
		assert.Equal(t, "Missing operator", err.Error())
	}

	f, err = ParseFilter("||$eq")
	assert.Nil(t, f)
	if assert.NotNil(t, err) {
		assert.Equal(t, "Invalid filter syntax", err.Error())
	}

	f, err = ParseFilter("field||$notanoperator")
	assert.Nil(t, f)
	if assert.NotNil(t, err) {
		assert.Equal(t, "Unknown operator: \"$notanoperator\"", err.Error())
	}

	f, err = ParseFilter("field||$eq||,")
	assert.Nil(t, f)
	if assert.NotNil(t, err) {
		assert.Equal(t, "Invalid filter syntax", err.Error())
	}

	f, err = ParseFilter("field||$eq")
	assert.Nil(t, f)
	if assert.NotNil(t, err) {
		assert.Equal(t, "Operator \"$eq\" requires at least 1 argument(s)", err.Error())
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
		assert.Equal(t, "Invalid sort syntax", err.Error())
	}

	s, err = ParseSort(",DESC")
	assert.Nil(t, s)
	if assert.NotNil(t, err) {
		assert.Equal(t, "Invalid sort syntax", err.Error())
	}

	s, err = ParseSort("name,")
	assert.Nil(t, s)
	if assert.NotNil(t, err) {
		assert.Equal(t, "Invalid sort syntax", err.Error())
	}

	s, err = ParseSort("name,notanorder")
	assert.Nil(t, s)
	if assert.NotNil(t, err) {
		assert.Equal(t, "Invalid sort order \"NOTANORDER\"", err.Error())
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
		assert.Equal(t, "Invalid join syntax", err.Error())
	}

	j, err = ParseJoin("||field1,field2")
	assert.Nil(t, j)
	if assert.NotNil(t, err) {
		assert.Equal(t, "Invalid join syntax", err.Error())
	}
}

func TestValidateFilter(t *testing.T) {
	ctx := &validation.Context{
		Value: "field||$eq||value1,value2",
		Rule: &validation.Rule{
			Name:   "filter",
			Params: []string{},
		},
	}
	expected := &Filter{
		Field:    "field",
		Operator: Operators["$eq"],
		Args:     []string{"value1", "value2"},
		Or:       false,
	}
	assert.True(t, validateFilter(ctx))
	assert.Equal(t, expected, ctx.Value)

	ctx.Value = "field||$eq||value1,value2"
	ctx.Rule.Params = []string{"or"}
	expected.Or = true
	assert.True(t, validateFilter(ctx))
	assert.Equal(t, expected, ctx.Value)

	ctx.Value = 5
	assert.False(t, validateFilter(ctx))
	assert.Equal(t, 5, ctx.Value)

	ctx.Value = ""
	assert.False(t, validateFilter(ctx))
	assert.Equal(t, "", ctx.Value)
}

func TestValidateSort(t *testing.T) {
	ctx := &validation.Context{
		Value: "name,ASC",
		Rule: &validation.Rule{
			Name:   "sort",
			Params: []string{},
		},
	}
	expected := &Sort{
		Field: "name",
		Order: SortAscending,
	}
	assert.True(t, validateSort(ctx))
	assert.Equal(t, expected, ctx.Value)

	ctx.Value = 5
	assert.False(t, validateSort(ctx))
	assert.Equal(t, 5, ctx.Value)

	ctx.Value = ""
	assert.False(t, validateSort(ctx))
	assert.Equal(t, "", ctx.Value)
}

func TestValidateJoin(t *testing.T) {
	ctx := &validation.Context{
		Value: "relation||field1,field2",
		Rule: &validation.Rule{
			Name:   "join",
			Params: []string{},
		},
	}
	expected := &Join{
		Relation: "relation",
		Fields:   []string{"field1", "field2"},
	}
	assert.True(t, validateJoin(ctx))
	assert.Equal(t, expected, ctx.Value)

	ctx.Value = 5
	assert.False(t, validateJoin(ctx))
	assert.Equal(t, 5, ctx.Value)

	ctx.Value = ""
	assert.False(t, validateJoin(ctx))
	assert.Equal(t, "", ctx.Value)
}
