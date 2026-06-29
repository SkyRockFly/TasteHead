package kit

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

func ValidateStruct(validate *validator.Validate, str any) error {
	err := validate.Struct(str)
	if err != nil {
		var validateErrs validator.ValidationErrors
		var str strings.Builder
		if errors.As(err, &validateErrs) {
			for _, e := range validateErrs {
				fmt.Fprintf(&str, "%s failed on '%s'\n,require: %s",
					e.StructNamespace(),
					e.Tag(),
					e.Param())
			}
			return fmt.Errorf("validate req: \n%s", str.String())
		}

		return fmt.Errorf("validate req: %w", err)
	}
	return nil
}
