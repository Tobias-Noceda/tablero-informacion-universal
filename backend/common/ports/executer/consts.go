package executer

import "time"

const (
	REQUEST_TIMEOUT  = 10 * time.Second
	QUERY_TIMEOUT    = 250 * time.Millisecond
	MAX_PAYLOAD_SIZE = 2 << 20 // 2mb
)
