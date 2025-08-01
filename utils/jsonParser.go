package utils

import (
	"encoding/json"
	"net/http"
)

func ParseJSON[T any](r *http.Request) (T, error) {
    var data T
    err := json.NewDecoder(r.Body).Decode(&data)
    return data, err
}