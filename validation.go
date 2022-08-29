package filter

import (
	"fmt"
	"strings"

	"goyave.dev/goyave/v4/lang"
	"goyave.dev/goyave/v4/validation"
)

// Separator the separator used when parsing the query
var Separator = "||"

func init() {
	validation.AddRule("filter", &validation.RuleDefinition{
		Function: validateFilter,
	})
	validation.AddRule("sort", &validation.RuleDefinition{
		Function: validateSort,
	})
	validation.AddRule("join", &validation.RuleDefinition{
		Function: validateJoin,
	})
	lang.SetDefaultValidationRule("filter.array", "The filter format is invalid.")
	lang.SetDefaultValidationRule("join.array", "The join format is invalid.")
	lang.SetDefaultValidationRule("sort.array", "The sort format is invalid.")
}

func validateFilter(ctx *validation.Context) bool {
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
	if len(ctx.Rule.Params) > 0 && ctx.Rule.Params[0] == "or" {
		f.Or = true
	}
	ctx.Value = f
	return true
}

func validateSort(ctx *validation.Context) bool {
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

func validateJoin(ctx *validation.Context) bool {
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

// ApplyValidation add all fields used by the filter module to the given RuleSet.
func ApplyValidation(set validation.RuleSet) {
	set["filter"] = validation.List{"array"}
	set["filter[]"] = validation.List{"filter"}
	set["or"] = validation.List{"array"}
	set["or[]"] = validation.List{"filter:or"}
	set["sort"] = validation.List{"array"}
	set["sort[]"] = validation.List{"sort"}
	set["join"] = validation.List{"array"}
	set["join[]"] = validation.List{"join"}
	set["fields"] = validation.List{"string"}
	set["page"] = validation.List{"integer", "min:1"}
	set["per_page"] = validation.List{"integer", "between:1,500"}
	set["search"] = validation.List{"string"}
}

// ApplyValidationRules add all fields used by the filter module to the given *Rules.
func ApplyValidationRules(set *validation.Rules) {
	set.Fields["filter"] = &validation.Field{Rules: []*validation.Rule{{Name: "array"}}}
	set.Fields["filter[]"] = &validation.Field{Rules: []*validation.Rule{{Name: "filter"}}}
	set.Fields["or"] = &validation.Field{Rules: []*validation.Rule{{Name: "array"}}}
	set.Fields["or[]"] = &validation.Field{Rules: []*validation.Rule{{Name: "filter", Params: []string{"or"}}}}
	set.Fields["sort"] = &validation.Field{Rules: []*validation.Rule{{Name: "array"}}}
	set.Fields["sort[]"] = &validation.Field{Rules: []*validation.Rule{{Name: "sort"}}}
	set.Fields["join"] = &validation.Field{Rules: []*validation.Rule{{Name: "array"}}}
	set.Fields["join[]"] = &validation.Field{Rules: []*validation.Rule{{Name: "join"}}}
	set.Fields["fields"] = &validation.Field{Rules: []*validation.Rule{{Name: "string"}}}
	set.Fields["page"] = &validation.Field{Rules: []*validation.Rule{{Name: "integer"}, {Name: "min", Params: []string{"1"}}}}
	set.Fields["per_page"] = &validation.Field{Rules: []*validation.Rule{{Name: "integer"}, {Name: "between", Params: []string{"1", "500"}}}}
	set.Fields["search"] = &validation.Field{Rules: []*validation.Rule{{Name: "string"}}}
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
		return nil, fmt.Errorf("Missing operator")
	}
	res.Field = strings.TrimSpace(f[:index])
	if res.Field == "" {
		return nil, fmt.Errorf("Invalid filter syntax")
	}
	f = f[index+2:]

	index = strings.Index(f, Separator)
	if index == -1 {
		index = len(f)
	}
	op = strings.TrimSpace(f[:index])
	operator, ok := Operators[op]
	if !ok {
		return nil, fmt.Errorf("Unknown operator: %q", f[:index])
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
				return nil, fmt.Errorf("Invalid filter syntax")
			}
			res.Args = append(res.Args, p)
			if paramIndex+1 >= len(f) {
				break
			}
			f = f[paramIndex+1:]
		}
	}

	if len(res.Args) < int(res.Operator.RequiredArguments) {
		return nil, fmt.Errorf("Operator %q requires at least %d argument(s)", op, res.Operator.RequiredArguments)
	}

	return res, nil
}

// ParseSort parse a string in format "name,ASC" and return a Sort struct.
// The element after the comma (sort order) must have a value allowing it to be
// converted to SortOrder, otherwise an error is returned.
func ParseSort(sort string) (*Sort, error) {
	commaIndex := strings.Index(sort, ",")
	if commaIndex == -1 {
		return nil, fmt.Errorf("Invalid sort syntax")
	}

	fieldName := strings.TrimSpace(sort[:commaIndex])
	order := strings.TrimSpace(strings.ToUpper(sort[commaIndex+1:]))
	if fieldName == "" || order == "" {
		return nil, fmt.Errorf("Invalid sort syntax")
	}

	if order != string(SortAscending) && order != string(SortDescending) {
		return nil, fmt.Errorf("Invalid sort order %q", order)
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
		return nil, fmt.Errorf("Invalid join syntax")
	}

	var fields []string
	if separatorIndex+2 < len(join) {
		fields = strings.Split(join[separatorIndex+2:], ",")
		for i, f := range fields {
			f = strings.TrimSpace(f)
			if f == "" {
				return nil, fmt.Errorf("Invalid join syntax")
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
