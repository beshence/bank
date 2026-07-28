package webrtc

import (
	"bytes"
	"net/http"
	"net/http/httptest"
)

func HandleEndpoints(router http.Handler, req Request) Response {
	httpReq, err := http.NewRequest(
		req.Method,
		"/api"+req.Path,
		bytes.NewReader(req.Body),
	)

	if err != nil {
		return Response{
			ID: req.ID,
		}
	}

	for key, value := range req.Headers {
		httpReq.Header.Set(
			key,
			value,
		)
	}

	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, httpReq)

	headers := make(map[string]string)

	for key, values := range recorder.Header() {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}

	return Response{
		ID:   req.ID,
		Body: recorder.Body.Bytes(),
	}
}
