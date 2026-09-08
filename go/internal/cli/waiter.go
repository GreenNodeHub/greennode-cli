package cli

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/greennodehub/greennode-cli/internal/client"
)

// WaiterExitCode is the process exit status used when a waiter fails or times
// out, matching the AWS CLI's waiter convention so scripts can distinguish
// "the wait did not succeed" from an ordinary command error.
const WaiterExitCode = 255

// IsPermanentAPIError reports whether err is an HTTP client error that will not
// resolve by polling again (bad request, unauthenticated, forbidden). 404 is
// intentionally excluded — each waiter interprets it for itself, since a
// missing resource means success to a "*-deleted" waiter and failure to an
// "*-active" one.
func IsPermanentAPIError(err error) bool {
	var apiErr *client.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.StatusCode {
		case 400, 401, 403:
			return true
		}
	}
	return false
}

// IsNotFound reports whether err is an HTTP 404.
func IsNotFound(err error) bool {
	var apiErr *client.APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == 404
}

// WaitEvaluator decides, for one poll, whether the waiter is done, has failed,
// or hit a fatal error that should abort polling immediately. status is shown
// in the progress line.
type WaitEvaluator func(result interface{}, err error) (done bool, failed bool, status string, fatal error)

// RunWaiter polls describe() every delay seconds up to maxAttempts times,
// driving the waiter via eval. Progress goes to stderr so it does not pollute
// piped output; on success successMsg is printed to stdout and nil returned.
// On a terminal failure or timeout the process exits with WaiterExitCode.
//
// Shared by every product's `wait` commands — the product supplies describe()
// and the evaluator, which is where the service-specific status handling lives.
func RunWaiter(label, successMsg string, describe func() (interface{}, error), eval WaitEvaluator, delay, maxAttempts int) error {
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		result, err := describe()
		done, failed, status, fatal := eval(result, err)

		// A fatal error (e.g. 403 Forbidden) will never resolve by polling —
		// abort immediately instead of retrying until timeout.
		if fatal != nil {
			fmt.Fprintln(os.Stderr)
			fmt.Fprintf(os.Stderr, "Error waiting for %s: %v\n", label, fatal)
			os.Exit(WaiterExitCode)
		}

		shown := status
		if shown == "" {
			shown = "polling"
		}
		fmt.Fprintf(os.Stderr, "\rWaiting for %s: %s (attempt %d/%d)", label, shown, attempt, maxAttempts)

		if done {
			fmt.Fprintln(os.Stderr)
			fmt.Println(successMsg)
			return nil
		}
		if failed {
			fmt.Fprintln(os.Stderr)
			fmt.Fprintf(os.Stderr, "Waiter failed: %s reached %s\n", label, status)
			os.Exit(WaiterExitCode)
		}

		if attempt < maxAttempts {
			time.Sleep(time.Duration(delay) * time.Second)
		}
	}

	fmt.Fprintln(os.Stderr)
	fmt.Fprintf(os.Stderr, "Waiter timed out after %d attempts\n", maxAttempts)
	os.Exit(WaiterExitCode)
	return nil
}
