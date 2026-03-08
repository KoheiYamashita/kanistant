package tools

import (
	"fmt"
	"regexp"
)

var (
	phoneNumberRe = regexp.MustCompile(`^[0-9+\-() #*]+$`)
	emailRe       = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
)

func init() {
	registerCategoryValidator(validateCommunicationParams,
		"dial", "compose_sms", "compose_email")
}

func validatePhoneNumber(phone string) error {
	if !phoneNumberRe.MatchString(phone) {
		return fmt.Errorf("invalid phone number: only digits, +, -, (), spaces, #, * are allowed")
	}
	return nil
}

func validateEmail(email string) error {
	if !emailRe.MatchString(email) {
		return fmt.Errorf("invalid email address: %s", email)
	}
	return nil
}

func validateCommunicationParams(action string, args map[string]interface{}) (map[string]interface{}, error) {
	params := make(map[string]interface{})

	switch action {
	case "dial":
		phone := toString(args["phone_number"])
		if phone == "" {
			return nil, fmt.Errorf("dial requires phone_number")
		}
		if err := validatePhoneNumber(phone); err != nil {
			return nil, err
		}
		params["phone_number"] = phone

	case "compose_sms":
		phone := toString(args["phone_number"])
		if phone == "" {
			return nil, fmt.Errorf("compose_sms requires phone_number")
		}
		if err := validatePhoneNumber(phone); err != nil {
			return nil, err
		}
		params["phone_number"] = phone
		if v := toString(args["message"]); v != "" {
			params["message"] = v
		}

	case "compose_email":
		to := toString(args["to"])
		if to == "" {
			return nil, fmt.Errorf("compose_email requires to")
		}
		if err := validateEmail(to); err != nil {
			return nil, err
		}
		params["to"] = to
		if v := toString(args["subject"]); v != "" {
			params["subject"] = v
		}
		if v := toString(args["body"]); v != "" {
			params["body"] = v
		}
	}

	return params, nil
}
