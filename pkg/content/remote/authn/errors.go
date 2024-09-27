package authn

import (
	"fmt"

	"github.com/octohelm/courier/pkg/statuserror"
)

type ErrUnauthorized struct {
	statuserror.Unauthorized
	Reason error
}

func (e *ErrUnauthorized) Error() string {
	return fmt.Sprintf("Unauthorized: %s", e.Reason)
}
