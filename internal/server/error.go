package server

import (
	"encoding/json"
	"net/http"
)

type ErrorResponse struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func NewErrorResponse(status int, msg string) ErrorResponse {
	er := ErrorResponse{}
	er.Error.Code = status
	er.Error.Message = msg
	return er
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(NewErrorResponse(status, msg))
}
