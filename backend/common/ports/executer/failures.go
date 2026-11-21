package executer

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/Secreto31126/tesis/common/infrastructure"
)

// transportFailure tells a source that took too long from one that could
// not be reached at all: a refused connection, an unknown host, an address
// the SSRF guard blocks or too many redirects.
func transportFailure(err error) error {
	var network net.Error
	if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &network) && network.Timeout()) {
		return fmt.Errorf("%w: %v", infrastructure.ErrSourceTimeout, err)
	}
	return fmt.Errorf("%w: %v", infrastructure.ErrSourceUnreachable, err)
}

// readFailure is a body that could not be decoded; one cut short by the
// deadline is still a timeout.
func readFailure(err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("%w: %v", infrastructure.ErrSourceTimeout, err)
	}
	return fmt.Errorf("%w: %v", infrastructure.ErrSourceUnreadable, err)
}

func queryFailure(err error) error {
	return fmt.Errorf("%w: %v", infrastructure.ErrQueryFailed, err)
}
