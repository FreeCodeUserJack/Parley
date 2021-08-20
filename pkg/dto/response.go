package dto

import (
	"fmt"
	"reflect"
	"sort"
)

type Response struct {
	Message string `json:"message"`
	Id      string `json:"id"`
}

// Response fields can change but reflect can pull them out
var responseFields []string

func init() {
	typ := reflect.TypeOf(Response{})
	responseFields = make([]string, typ.NumField())

	for i := 0; i < typ.NumField(); i++ {
		responseFields[i] = typ.Field(i).Name
	}

	sort.Strings(responseFields)
}

func (r *Response) String() string {
	return fmt.Sprintf("Message: %s, Id: %s", r.Message, r.Id)
}

func (r *Response) Format(state fmt.State, verb rune) {
	switch verb {
	case 's', 'q':
		val := r.String()
		if verb == 'q' {
			val = fmt.Sprintf("%q", val)
		}
		fmt.Fprint(state, val)
	case 'v':
		if state.Flag('#') {
			fmt.Fprintf(state, "%T", r)
		}
		fmt.Fprint(state, "{")
		val := reflect.ValueOf(*r)
		for i, name := range responseFields {
			if state.Flag('#') || state.Flag('+') {
				fmt.Fprintf(state, "%s:", name)
			}
			fld := val.FieldByName(name)
			// can perform additional logic here dependent on the field - e.g. mask a password
			fmt.Fprint(state, fld)

			if i < len(responseFields)-1 {
				fmt.Fprint(state, " ")
			}
		}
		fmt.Fprint(state, "}")
	}
}
