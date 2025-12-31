package root

import "fmt"

type missingHandlerError string

func (e missingHandlerError) Error() string {
	return fmt.Sprintf("missing CLI handler for %s", string(e))
}
