package filter

import (
	"fmt"
	"strings"

	"goyave.dev/goyave/v3/validation"
)

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

	// TODO add language entries
}

func validateFilter(ctx *validation.Context) bool {
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
	set["per_page"] = validation.List{"integer", "min:1"}
}

// TODO apply validation to validation.Rules too

// ParseFilter parse a string in format "field||$operator||value" and return
// a Filter struct. The filter string must satisfy the used operator's "RequiredArguments"
// constraint, otherwise an error is returned.
func ParseFilter(filter string) (*Filter, error) {
	res := &Filter{}
	f := filter
	op := ""

	index := strings.Index(f, "||")
	if index == -1 {
		return nil, fmt.Errorf("Missing operator")
	}
	res.Field = f[:index]
	f = f[index+2:]

	index = strings.Index(f, "||")
	if index == -1 {
		index = len(f)
	}
	op = f[:index]
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
			res.Args = append(res.Args, f[:paramIndex])
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

	fieldName := sort[:commaIndex]
	order := strings.ToUpper(sort[commaIndex+1:])

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
	commaIndex := strings.Index(join, "||")
	if commaIndex == -1 {
		j := &Join{
			Relation: join,
			Fields:   []string{},
		}
		return j, nil
	}

	relation := join[:commaIndex]

	var fields []string
	if commaIndex+2 < len(join) {
		fields = strings.Split(join[commaIndex+2:], ",")
	} else {
		fields = []string{}
	}

	j := &Join{
		Relation: relation,
		Fields:   fields,
	}
	return j, nil
}
