package sender

import "fmt"

func VerifyMessage(code int32) string {
	return fmt.Sprintf("Код для подтверждения регистрации: %d", code)
}
