package main

import "net/http"

func canRetryRequest(req *http.Request) bool {
	switch req.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace: // not PUT, POST, DELETE
		return true
	}

	if _, ok := req.Header["x-idempotency-key"]; ok {
		return true
	}

	return false
}
