package controller

type MessageRes struct {
	Message string `json:"message"`
}

func NewMessageRes(message string) *MessageRes {
	return &MessageRes{
		Message: message,
	}
}
