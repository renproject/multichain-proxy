package util

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
)

func WriteError(dst http.ResponseWriter, id interface{}, err error) error {
	dst.Header().Set("content-type", "application/json")
	dst.WriteHeader(http.StatusBadRequest)
	return json.NewEncoder(dst).Encode(map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"error":   err.Error(),
	})
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

func ConvertEnv2Map(env string) map[string]bool {
	envList := strings.Split(os.Getenv(env), ",")
	envMap := map[string]bool{}
	if len(envList) == 0 {
		envMap = nil
	} else {
		for i := range envList {
			envMap[strings.TrimSpace(envList[i])] = true
		}
	}
	return envMap
}
