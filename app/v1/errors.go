package v1

import (
	"strings"

	"github.com/osuthailand/api/common"
)

// Boilerplate errors
var (
	Err500     = common.SimpleResponse(500, "Uh oh... Seems like Aoba did something bad to API... Please try again! If it's broken... Please tell me in the Discord!")
	ErrBadJSON = common.SimpleResponse(400, "Your JSON for this request is invalid.")
)

// ErrMissingField generates a response to a request when some fields in the JSON are missing.
func ErrMissingField(missingFields ...string) common.CodeMessager {
	return common.ResponseBase{
		Code:    422, // http://stackoverflow.com/a/10323055/5328069
		Message: "Missing parameters: " + strings.Join(missingFields, ", ") + ".",
	}
}
