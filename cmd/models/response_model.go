package models

type Response struct {
	Status int `json:"status"`
	ErrorMsg string `json:"error_msg,omitempty"`
	Body any `json:"body"`
}