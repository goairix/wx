package transport

import "time"

// RetryPolicy controls retry attempts and retryable HTTP status codes. A
// request must also permit retries through its request.RetryMode and method.
type RetryPolicy struct {
	MaxAttempts int
	Backoff     func(attempt int) time.Duration
	RetryStatus map[int]bool
}

const defaultMaxAttempts = 1

func (p RetryPolicy) normalized() RetryPolicy {
	if p.MaxAttempts <= 0 {
		p.MaxAttempts = defaultMaxAttempts
	}
	if p.Backoff == nil {
		p.Backoff = defaultBackoff
	}
	// A nil map means the standard transient statuses. An explicitly empty map
	// disables status retries, which is useful when callers want strict control.
	if p.RetryStatus == nil {
		p.RetryStatus = map[int]bool{
			429: true,
			500: true,
			502: true,
			503: true,
			504: true,
		}
	}
	return p
}

func defaultBackoff(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	d := 10 * time.Millisecond
	for i := 1; i < attempt; i++ {
		d *= 2
		if d >= time.Second {
			return time.Second
		}
	}
	return d
}
