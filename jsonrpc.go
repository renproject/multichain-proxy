package proxy

import (
	"encoding/json"
	"io"
	"net/http"
)

type JSONRPCRequest struct {
	Version string          `json:"version"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

func CopyResponse(dst http.ResponseWriter, res *http.Response) {
	CopyHeader(dst.Header(), res.Header)
	dst.WriteHeader(res.StatusCode)
	io.Copy(dst, res.Body)
}

func CopyHeader(dst, src http.Header) {
	for k, vs := range src {
		for _, v := range vs {
			dst.Add(k, v)
		}
	}
}

func WriteError(dst http.ResponseWriter, id interface{}, err error) error {
	dst.Header().Set("content-type", "application/json")
	dst.WriteHeader(http.StatusBadRequest)
	return json.NewEncoder(dst).Encode(map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"error":   err.Error(),
	})
}
