package errors

type Error interface {
	error
	CodeString() string
	Unwrap() error
	WithMetadata(key string, value interface{}) *BaseError
}

type BaseError struct {
	Code    string
	Message string
	Details map[string]interface{}
	Cause   error
}

func (e *BaseError) Error() string {

	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}

	return e.Message
}

func (e *BaseError) CodeString() string {
	return e.Code
}

func (e *BaseError) Unwrap() error {
	return e.Cause
}

func (e *BaseError) WithMetadata(key string, value interface{}) *BaseError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

func (e *BaseError) Metadata() map[string]interface{} {
	return e.Details
}

type InitializationError struct {
	*BaseError
}

func NewInitializationError(message string, cause error) *InitializationError {
	return &InitializationError{
		BaseError: &BaseError{
			Message: message,
			Code:    "INITIALIZATION_ERROR",
			Cause:   cause,
		},
	}
}
