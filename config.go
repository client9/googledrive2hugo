package googledrive2hugo

import (
	"fmt"

	"github.com/client9/shconfig"
)

var sample = `
add-class table "table table-sm"
add-class blockquote "pl-3 lines-dense"
add-class pre "p-1 pl-3 lines-dense"
add-class h1 "h2 mb-3" # no top margin
add-class h2 "h4 mt-4 mb-4"
add-class h3  "img-fluid"
add-class "div:has(img)" "container pl-0"
link-relative "https://www.client9.com"
remove-empty-tags
unsmart-code
narrow-tags
check-punc
`

func Parse(text string) ([]Runner, error) {
	root := &conf{}
	if err := shconfig.Parse(root, text); err != nil {
		return nil, err
	}
	return root.Runner, nil
}

var confmap = map[string]func([]string) (Runner, error){
	"add-class":         configAddClass,
	"link-relative":     configLinkRelative,
	"remove-empty-tags": configRemoveEmpty,
	"unsmart-code":      configUnsmartCode,
	"narrow-tags":       configNarrowTags,
	"check-punc":        configCheckPunc,
}

func configAddClass(args []string) (Runner, error) {
	check := &AddClassAttr{}
	err := shconfig.RequireString2(args, check.Init)
	return check, err
}

func configRemoveEmpty(args []string) (Runner, error) {
	check := &RemoveEmptyTag{}
	err := shconfig.RequireString0(args, check.Init)
	return check, err
}

func configCheckPunc(args []string) (Runner, error) {
	check := &Punc{}
	err := shconfig.RequireString0(args, check.Init)
	return check, err
}

func configUnsmartCode(args []string) (Runner, error) {
	check := &UnsmartCode{}
	err := shconfig.RequireString0(args, check.Init)
	return check, err
}

func configNarrowTags(args []string) (Runner, error) {
	check := &NarrowTag{}
	err := shconfig.RequireString0(args, check.Init)
	return check, err
}

func configLinkRelative(args []string) (Runner, error) {
	check := &LinkRelative{}
	err := shconfig.RequireString1(args, check.Init)
	return check, err
}

type conf struct {
	Runner []Runner
}

func (r *conf) ConfCall(args []string) error {
	fn, ok := confmap[args[0]]
	if !ok {
		return fmt.Errorf("command %q not found", args[0])
	}
	check, err := fn(args)
	if err != nil {
		return err
	}
	if check != nil {
		r.Runner = append(r.Runner, check)
	}
	return nil
}

func (r *conf) ConfObject(args []string) (shconfig.Dispatcher, error) {
	return nil, fmt.Errorf("no config objects")
}
