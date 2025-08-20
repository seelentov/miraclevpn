package controller

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

func HandleValidation(errs validator.ValidationErrors, req interface{}) map[string]string {
	errors := make(map[string]string, 0)

	for _, fe := range errs {
		jsonFieldName := getJSONFieldName(req, fe.Field())
		errors[jsonFieldName] = getDetailedValidationMessage(fe)
	}

	return errors
}

func getJSONFieldName(obj interface{}, fieldName string) string {
	t := reflect.TypeOf(obj)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	field, found := t.FieldByName(fieldName)
	if !found {
		return strings.ToLower(fieldName)
	}

	jsonTag := field.Tag.Get("json")
	if jsonTag == "" {
		return strings.ToLower(fieldName)
	}

	// Убираем опции из json тега (например: `json:"name,omitempty"`)
	if commaIndex := strings.Index(jsonTag, ","); commaIndex != -1 {
		return jsonTag[:commaIndex]
	}

	return jsonTag
}

func getDetailedValidationMessage(fe validator.FieldError) string {
	tag := fe.Tag()
	param := fe.Param()

	switch tag {
	case "min":
		return fmt.Sprintf("Минимальное значение: %s", param)
	case "max":
		return fmt.Sprintf("Максимальное значение: %s", param)
	case "len":
		return fmt.Sprintf("Требуемая длина: %s символов", param)
	case "eqfield":
		return fmt.Sprintf("Должно совпадать с полем %s", param)
	case "oneof":
		return fmt.Sprintf("Допустимые значения: %s", strings.Replace(param, " ", ", ", -1))
	case "contains":
		return fmt.Sprintf("Должно содержать: %s", param)
	case "startswith":
		return fmt.Sprintf("Должно начинаться с: %s", param)
	case "endswith":
		return fmt.Sprintf("Должно заканчиваться на: %s", param)
	case "required_if", "required_unless", "required_with", "required_without":
		return "Обязательное поле"
	default:
		return getBasicValidationMessage(tag)
	}
}

func getBasicValidationMessage(tag string) string {
	basicMessages := map[string]string{
		"required": "Обязательное поле",
		"email":    "Некорректный формат email",
		"url":      "Некорректный URL",
		"uuid":     "Некорректный UUID",
		"ip":       "Некорректный IP адрес",
	}

	if msg, ok := basicMessages[tag]; ok {
		return msg
	}

	panic("not implemented")
}
