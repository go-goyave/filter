package filter

import (
	"fmt"
	"strings"

	"goyave.dev/goyave/v5"
	"goyave.dev/goyave/v5/lang"
	v "goyave.dev/goyave/v5/validation"
)

// Separator the separator used when parsing the query
var Separator = "||"

func init() {
	lang.SetDefaultValidationRule("filter.array", "The filter format is invalid.")
	lang.SetDefaultValidationRule("join.array", "The join format is invalid.")
	lang.SetDefaultValidationRule("sort.array", "The sort format is invalid.")
}

// FilterValidator checks the `filter` format and converts it to `*Filter` struct.
type FilterValidator struct {
	v.BaseValidator
	Or bool
}

// Validate checks the field under validation satisfies this validator's criteria.
func (v *FilterValidator) Validate(ctx *v.Context) bool {
	if _, ok := ctx.Value.(*Filter); ok {
		return true
	}
	str, ok := ctx.Value.(string)
	if !ok {
		return false
	}
	f, err := ParseFilter(str)
	if err != nil {
		return false
	}
	f.Or = v.Or
	ctx.Value = f
	return true
}

// Name returns the string name of the validator.
func (v *FilterValidator) Name() string { return "filter" }

// IsType returns true
func (v *FilterValidator) IsType() bool { return true }

// SortValidator checks the `sort` format and converts it to `*Sort` struct.
type SortValidator struct {
	v.BaseValidator
}

// Validate checks the field under validation satisfies this validator's criteria.
func (v *SortValidator) Validate(ctx *v.Context) bool {
	if _, ok := ctx.Value.(*Sort); ok {
		return true
	}
	str, ok := ctx.Value.(string)
	if !ok {
		return false
	}
	sort, err := ParseSort(str)
	if err != nil {
		return false
	}
	ctx.Value = sort
	return true
}

// Name returns the string name of the validator.
func (v *SortValidator) Name() string { return "sort" }

// IsType returns true
func (v *SortValidator) IsType() bool { return true }

// JoinValidator checks the `sort` format and converts it to `*Join` struct.
type JoinValidator struct {
	v.BaseValidator
}

// Validate checks the field under validation satisfies this validator's criteria.
func (v *JoinValidator) Validate(ctx *v.Context) bool {
	if _, ok := ctx.Value.(*Join); ok {
		return true
	}
	str, ok := ctx.Value.(string)
	if !ok {
		return false
	}
	join, err := ParseJoin(str)
	if err != nil {
		return false
	}
	ctx.Value = join
	return true
}

// Name returns the string name of the validator.
func (v *JoinValidator) Name() string { return "join" }

// IsType returns true
func (v *JoinValidator) IsType() bool { return true }

// Validation returns a new RuleSet for query validation.
func Validation(_ *goyave.Request) v.RuleSet {
	return v.RuleSet{
		{Path: "filter", Rules: v.List{v.Array()}},
		{Path: "filter[]", Rules: v.List{&FilterValidator{}}},
		{Path: "or", Rules: v.List{v.Array()}},
		{Path: "or[]", Rules: v.List{&FilterValidator{Or: true}}},
		{Path: "sort", Rules: v.List{v.Array()}},
		{Path: "sort[]", Rules: v.List{&SortValidator{}}},
		{Path: "join", Rules: v.List{v.Array()}},
		{Path: "join[]", Rules: v.List{&JoinValidator{}}},
		{Path: "page", Rules: v.List{v.Int(), v.Min(1)}},
		{Path: "per_page", Rules: v.List{v.Int(), v.Between(1, 500)}},
		{Path: "search", Rules: v.List{v.String(), v.Max(255)}},
		{Path: "fields", Rules: v.List{v.String()}},
	}
}

// ParseFilter parse a string in format "field||$operator||value" and return
// a Filter struct. The filter string must satisfy the used operator's "RequiredArguments"
// constraint, otherwise an error is returned.
func ParseFilter(filter string) (*Filter, error) {
	res := &Filter{}
	f := filter
	op := ""

	index := strings.Index(f, Separator)
	if index == -1 {
		return nil, fmt.Errorf("missing operator")
	}
	res.Field = strings.TrimSpace(f[:index])
	if res.Field == "" {
		return nil, fmt.Errorf("invalid filter syntax")
	}
	f = f[index+2:]

	index = strings.Index(f, Separator)
	if index == -1 {
		index = len(f)
	}
	op = strings.TrimSpace(f[:index])
	operator, ok := Operators[op]
	if !ok {
		return nil, fmt.Errorf("unknown operator: %q", f[:index])
	}
	res.Operator = operator

	if index < len(f) {
		f = f[index+2:]
		for paramIndex := strings.Index(f, ","); paramIndex < len(f); paramIndex = strings.Index(f, ",") {
			if paramIndex == -1 {
				paramIndex = len(f)
			}
			p := strings.TrimSpace(f[:paramIndex])
			if p == "" {
				return nil, fmt.Errorf("invalid filter syntax")
			}
			res.Args = append(res.Args, p)
			if paramIndex+1 >= len(f) {
				break
			}
			f = f[paramIndex+1:]
		}
	}

	if len(res.Args) < int(res.Operator.RequiredArguments) {
		return nil, fmt.Errorf("operator %q requires at least %d argument(s)", op, res.Operator.RequiredArguments)
	}

	return res, nil
}

// ParseSort parse a string in format "name,ASC" and return a Sort struct.
// The element after the comma (sort order) must have a value allowing it to be
// converted to SortOrder, otherwise an error is returned.
func ParseSort(sort string) (*Sort, error) {
	commaIndex := strings.Index(sort, ",")
	if commaIndex == -1 {
		return nil, fmt.Errorf("invalid sort syntax")
	}

	fieldName := strings.TrimSpace(sort[:commaIndex])
	order := strings.TrimSpace(strings.ToUpper(sort[commaIndex+1:]))
	if fieldName == "" || order == "" {
		return nil, fmt.Errorf("invalid sort syntax")
	}

	if order != string(SortAscending) && order != string(SortDescending) {
		return nil, fmt.Errorf("invalid sort order %q", order)
	}

	s := &Sort{
		Field: fieldName,
		Order: SortOrder(order),
	}
	return s, nil
}

// ParseJoin parse a string in format "relation||field1,field2,..." and return
// a Join struct.
func ParseJoin(join string) (*Join, error) {
	separatorIndex := strings.Index(join, Separator)
	if separatorIndex == -1 {
		separatorIndex = len(join)
	}

	relation := strings.TrimSpace(join[:separatorIndex])
	if relation == "" {
		return nil, fmt.Errorf("invalid join syntax")
	}

	var fields []string
	if separatorIndex+2 < len(join) {
		fields = strings.Split(join[separatorIndex+2:], ",")
		for i, f := range fields {
			f = strings.TrimSpace(f)
			if f == "" {
				return nil, fmt.Errorf("invalid join syntax")
			}
			fields[i] = f
		}
	} else {
		fields = nil
	}

	j := &Join{
		Relation: relation,
		Fields:   fields,
	}
	return j, nil
}
