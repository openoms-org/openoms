package smsapi

// SendSMSRequest represents a request to send an SMS message.
type SendSMSRequest struct {
	To      string `json:"to"`
	Message string `json:"message"`
	From    string `json:"from,omitempty"`
}

// SendSMSResponse represents the response from the SMSAPI.pl API after sending an SMS.
type SendSMSResponse struct {
	Count int         `json:"count"`
	List  []SMSResult `json:"list"`
}

// SMSResult represents a single SMS delivery result.
type SMSResult struct {
	ID     string  `json:"id"`
	Points float64 `json:"points"`
	Number string  `json:"number"`
	Status string  `json:"status"`
	Error  *int    `json:"error"`
}

// APIError represents an error response from the SMSAPI.pl API.
type APIError struct {
	StatusCode int
	ErrorCode  int    `json:"error"`
	Message    string `json:"message"`
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return "smsapi: " + e.Message
	}
	return "smsapi: unexpected status " + httpStatusText(e.StatusCode)
}

func httpStatusText(code int) string {
	switch code {
	case 400:
		return "400 Bad Request"
	case 401:
		return "401 Unauthorized"
	case 403:
		return "403 Forbidden"
	case 404:
		return "404 Not Found"
	case 500:
		return "500 Internal Server Error"
	default:
		return "unknown error"
	}
}
