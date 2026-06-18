package gossm

import (
	"errors"
	"fmt"
)

// ErrEmptyARN is returned when an empty ARN is passed to Fetch or FetchWithContext.
var ErrEmptyARN = errors.New("gossm: secret ARN must not be empty")

// ErrEmptyKey is returned when an empty key is passed to Fetch or FetchWithContext.
var ErrEmptyKey = errors.New("gossm: secret key must not be empty")

// SecretError wraps an underlying AWS error with the ARN of the secret that caused it.
// Use errors.As to inspect it:
//
//	var se *gossm.SecretError
//	if errors.As(err, &se) {
//		log.Printf("secret %s failed: %v", se.ARN, se.Err)
//	}
type SecretError struct {
	ARN string
	Err error
}

func (e *SecretError) Error() string {
	return fmt.Sprintf("gossm: failed to fetch secret %q: %v", e.ARN, e.Err)
}

func (e *SecretError) Unwrap() error { return e.Err }
