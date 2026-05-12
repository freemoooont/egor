package middleware

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// DecodeStrictJSON reads r.Body into out, rejecting unknown fields and
// returning a *JSONDecodeError on syntactic errors.
func DecodeStrictJSON(r *http.Request, out any) error {
	if r.Body == nil {
		return &JSONDecodeError{Message: "missing request body"}
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		if errors.Is(err, io.EOF) {
			return &JSONDecodeError{Message: "missing request body"}
		}
		return &JSONDecodeError{Message: humanizeJSONErr(err)}
	}
	// Reject extra payload after the first JSON value (e.g. two objects).
	if dec.More() {
		return &JSONDecodeError{Message: "unexpected extra json after request body"}
	}
	return nil
}

func humanizeJSONErr(err error) string {
	var ue *json.UnmarshalTypeError
	if errors.As(err, &ue) {
		return fmt.Sprintf("invalid type for field %q: expected %s", ue.Field, ue.Type.Kind())
	}
	var se *json.SyntaxError
	if errors.As(err, &se) {
		return fmt.Sprintf("invalid json at offset %d: %s", se.Offset, se.Error())
	}
	msg := err.Error()
	if strings.Contains(msg, "json: unknown field") {
		return msg
	}
	return msg
}
