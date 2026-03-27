package httpapi

import (
	"strings"

	"creditflow/services/bff/internal/backend"
)

func maskCustomer(customer *backend.Customer) *backend.Customer {
	if customer == nil {
		return nil
	}

	masked := *customer
	masked.CPF = maskCPF(masked.CPF)
	masked.Email = maskEmail(masked.Email)
	masked.Phone = maskPhone(masked.Phone)
	return &masked
}

func maskNotifications(notifications []backend.Notification) []backend.Notification {
	if len(notifications) == 0 {
		return notifications
	}

	masked := make([]backend.Notification, len(notifications))
	for index, notification := range notifications {
		masked[index] = notification
		masked[index].Recipient = maskEmail(notification.Recipient)
	}
	return masked
}

func maskCPF(value string) string {
	digits := onlyDigits(value)
	if len(digits) <= 4 {
		return digits
	}

	return strings.Repeat("*", len(digits)-4) + digits[len(digits)-4:]
}

func maskPhone(value string) string {
	digits := onlyDigits(value)
	if len(digits) <= 4 {
		return digits
	}

	return strings.Repeat("*", len(digits)-4) + digits[len(digits)-4:]
}

func maskEmail(value string) string {
	trimmed := strings.TrimSpace(value)
	parts := strings.Split(trimmed, "@")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return trimmed
	}

	local := parts[0]
	if len(local) == 1 {
		return "*" + "@" + parts[1]
	}

	return local[:1] + strings.Repeat("*", len(local)-1) + "@" + parts[1]
}

func onlyDigits(value string) string {
	var builder strings.Builder
	for _, char := range value {
		if char >= '0' && char <= '9' {
			builder.WriteRune(char)
		}
	}
	return builder.String()
}
