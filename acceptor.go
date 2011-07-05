package pusher

import (
	"http"
)

// Acceptor is a pre-flight mechanism for a) authenticating incoming
// subscriber/publisher request and b) extracting a channel id from a request.
// The channel id is gIf an acceptor decides that the request should NOT be
// allowed to publish/subscribe, then it must return an empty string.
type Acceptor func(req *http.Request) string

// StaticAcceptor accepts all requests and uses always an static channel id.
func StaticAcceptor(cid string) Acceptor {
	return func(req *http.Request) string {
		return cid
	}
}

// QueryParameterAcceptor accepts all requests and uses always a provided parameter
// form the query as the channel id.
func QueryParameterAcceptor(parameterName string) Acceptor {
	return func(req *http.Request) string {
		return req.FormValue(parameterName)
	}
}
