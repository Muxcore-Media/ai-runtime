package internal

import (
	"net/http"
	"strings"
	"time"

	"github.com/Muxcore-Media/core/sdk/go/module/netguard"
)

// aiGuardOptions is the Integration profile for an OpenAI-compatible endpoint.
// The default host is public. A household may point the base at a LAN or
// loopback model server. Link-local, cloud metadata, and non-HTTP schemes
// stay refused.
func aiGuardOptions(timeout time.Duration) netguard.Options {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return netguard.Options{
		AllowPrivate:  true,
		AllowLoopback: true,
		Timeout:       timeout,
	}
}

func newGuardedClient(timeout time.Duration) *http.Client {
	return netguard.NewClient(netguard.Integration, aiGuardOptions(timeout))
}

// guardOutboundURL checks raw before a request. An empty URL is left to the
// caller (unset configuration).
func guardOutboundURL(raw string) error {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	return netguard.ValidateURL(raw, netguard.Integration, aiGuardOptions(30*time.Second))
}
