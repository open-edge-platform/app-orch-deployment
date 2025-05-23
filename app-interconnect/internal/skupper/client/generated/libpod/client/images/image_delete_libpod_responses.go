// Code generated by go-swagger; DO NOT EDIT.

package images

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"

	"github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/skupper/client/generated/libpod/models"
)

// ImageDeleteLibpodReader is a Reader for the ImageDeleteLibpod structure.
type ImageDeleteLibpodReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ImageDeleteLibpodReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewImageDeleteLibpodOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 400:
		result := NewImageDeleteLibpodBadRequest()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 404:
		result := NewImageDeleteLibpodNotFound()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 409:
		result := NewImageDeleteLibpodConflict()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 500:
		result := NewImageDeleteLibpodInternalServerError()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		return nil, runtime.NewAPIError("response status code does not match any response statuses defined for this endpoint in the swagger spec", response, response.Code())
	}
}

// NewImageDeleteLibpodOK creates a ImageDeleteLibpodOK with default headers values
func NewImageDeleteLibpodOK() *ImageDeleteLibpodOK {
	return &ImageDeleteLibpodOK{}
}

/*
ImageDeleteLibpodOK describes a response with status code 200, with default header values.

Remove response
*/
type ImageDeleteLibpodOK struct {
	Payload *models.LibpodImagesRemoveReport
}

// IsSuccess returns true when this image delete libpod o k response has a 2xx status code
func (o *ImageDeleteLibpodOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this image delete libpod o k response has a 3xx status code
func (o *ImageDeleteLibpodOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this image delete libpod o k response has a 4xx status code
func (o *ImageDeleteLibpodOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this image delete libpod o k response has a 5xx status code
func (o *ImageDeleteLibpodOK) IsServerError() bool {
	return false
}

// IsCode returns true when this image delete libpod o k response a status code equal to that given
func (o *ImageDeleteLibpodOK) IsCode(code int) bool {
	return code == 200
}

func (o *ImageDeleteLibpodOK) Error() string {
	return fmt.Sprintf("[DELETE /libpod/images/{name}][%d] imageDeleteLibpodOK  %+v", 200, o.Payload)
}

func (o *ImageDeleteLibpodOK) String() string {
	return fmt.Sprintf("[DELETE /libpod/images/{name}][%d] imageDeleteLibpodOK  %+v", 200, o.Payload)
}

func (o *ImageDeleteLibpodOK) GetPayload() *models.LibpodImagesRemoveReport {
	return o.Payload
}

func (o *ImageDeleteLibpodOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.LibpodImagesRemoveReport)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewImageDeleteLibpodBadRequest creates a ImageDeleteLibpodBadRequest with default headers values
func NewImageDeleteLibpodBadRequest() *ImageDeleteLibpodBadRequest {
	return &ImageDeleteLibpodBadRequest{}
}

/*
ImageDeleteLibpodBadRequest describes a response with status code 400, with default header values.

Bad parameter in request
*/
type ImageDeleteLibpodBadRequest struct {
	Payload *ImageDeleteLibpodBadRequestBody
}

// IsSuccess returns true when this image delete libpod bad request response has a 2xx status code
func (o *ImageDeleteLibpodBadRequest) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this image delete libpod bad request response has a 3xx status code
func (o *ImageDeleteLibpodBadRequest) IsRedirect() bool {
	return false
}

// IsClientError returns true when this image delete libpod bad request response has a 4xx status code
func (o *ImageDeleteLibpodBadRequest) IsClientError() bool {
	return true
}

// IsServerError returns true when this image delete libpod bad request response has a 5xx status code
func (o *ImageDeleteLibpodBadRequest) IsServerError() bool {
	return false
}

// IsCode returns true when this image delete libpod bad request response a status code equal to that given
func (o *ImageDeleteLibpodBadRequest) IsCode(code int) bool {
	return code == 400
}

func (o *ImageDeleteLibpodBadRequest) Error() string {
	return fmt.Sprintf("[DELETE /libpod/images/{name}][%d] imageDeleteLibpodBadRequest  %+v", 400, o.Payload)
}

func (o *ImageDeleteLibpodBadRequest) String() string {
	return fmt.Sprintf("[DELETE /libpod/images/{name}][%d] imageDeleteLibpodBadRequest  %+v", 400, o.Payload)
}

func (o *ImageDeleteLibpodBadRequest) GetPayload() *ImageDeleteLibpodBadRequestBody {
	return o.Payload
}

func (o *ImageDeleteLibpodBadRequest) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(ImageDeleteLibpodBadRequestBody)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewImageDeleteLibpodNotFound creates a ImageDeleteLibpodNotFound with default headers values
func NewImageDeleteLibpodNotFound() *ImageDeleteLibpodNotFound {
	return &ImageDeleteLibpodNotFound{}
}

/*
ImageDeleteLibpodNotFound describes a response with status code 404, with default header values.

No such image
*/
type ImageDeleteLibpodNotFound struct {
	Payload *ImageDeleteLibpodNotFoundBody
}

// IsSuccess returns true when this image delete libpod not found response has a 2xx status code
func (o *ImageDeleteLibpodNotFound) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this image delete libpod not found response has a 3xx status code
func (o *ImageDeleteLibpodNotFound) IsRedirect() bool {
	return false
}

// IsClientError returns true when this image delete libpod not found response has a 4xx status code
func (o *ImageDeleteLibpodNotFound) IsClientError() bool {
	return true
}

// IsServerError returns true when this image delete libpod not found response has a 5xx status code
func (o *ImageDeleteLibpodNotFound) IsServerError() bool {
	return false
}

// IsCode returns true when this image delete libpod not found response a status code equal to that given
func (o *ImageDeleteLibpodNotFound) IsCode(code int) bool {
	return code == 404
}

func (o *ImageDeleteLibpodNotFound) Error() string {
	return fmt.Sprintf("[DELETE /libpod/images/{name}][%d] imageDeleteLibpodNotFound  %+v", 404, o.Payload)
}

func (o *ImageDeleteLibpodNotFound) String() string {
	return fmt.Sprintf("[DELETE /libpod/images/{name}][%d] imageDeleteLibpodNotFound  %+v", 404, o.Payload)
}

func (o *ImageDeleteLibpodNotFound) GetPayload() *ImageDeleteLibpodNotFoundBody {
	return o.Payload
}

func (o *ImageDeleteLibpodNotFound) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(ImageDeleteLibpodNotFoundBody)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewImageDeleteLibpodConflict creates a ImageDeleteLibpodConflict with default headers values
func NewImageDeleteLibpodConflict() *ImageDeleteLibpodConflict {
	return &ImageDeleteLibpodConflict{}
}

/*
ImageDeleteLibpodConflict describes a response with status code 409, with default header values.

Conflict error in operation
*/
type ImageDeleteLibpodConflict struct {
	Payload *ImageDeleteLibpodConflictBody
}

// IsSuccess returns true when this image delete libpod conflict response has a 2xx status code
func (o *ImageDeleteLibpodConflict) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this image delete libpod conflict response has a 3xx status code
func (o *ImageDeleteLibpodConflict) IsRedirect() bool {
	return false
}

// IsClientError returns true when this image delete libpod conflict response has a 4xx status code
func (o *ImageDeleteLibpodConflict) IsClientError() bool {
	return true
}

// IsServerError returns true when this image delete libpod conflict response has a 5xx status code
func (o *ImageDeleteLibpodConflict) IsServerError() bool {
	return false
}

// IsCode returns true when this image delete libpod conflict response a status code equal to that given
func (o *ImageDeleteLibpodConflict) IsCode(code int) bool {
	return code == 409
}

func (o *ImageDeleteLibpodConflict) Error() string {
	return fmt.Sprintf("[DELETE /libpod/images/{name}][%d] imageDeleteLibpodConflict  %+v", 409, o.Payload)
}

func (o *ImageDeleteLibpodConflict) String() string {
	return fmt.Sprintf("[DELETE /libpod/images/{name}][%d] imageDeleteLibpodConflict  %+v", 409, o.Payload)
}

func (o *ImageDeleteLibpodConflict) GetPayload() *ImageDeleteLibpodConflictBody {
	return o.Payload
}

func (o *ImageDeleteLibpodConflict) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(ImageDeleteLibpodConflictBody)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewImageDeleteLibpodInternalServerError creates a ImageDeleteLibpodInternalServerError with default headers values
func NewImageDeleteLibpodInternalServerError() *ImageDeleteLibpodInternalServerError {
	return &ImageDeleteLibpodInternalServerError{}
}

/*
ImageDeleteLibpodInternalServerError describes a response with status code 500, with default header values.

Internal server error
*/
type ImageDeleteLibpodInternalServerError struct {
	Payload *ImageDeleteLibpodInternalServerErrorBody
}

// IsSuccess returns true when this image delete libpod internal server error response has a 2xx status code
func (o *ImageDeleteLibpodInternalServerError) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this image delete libpod internal server error response has a 3xx status code
func (o *ImageDeleteLibpodInternalServerError) IsRedirect() bool {
	return false
}

// IsClientError returns true when this image delete libpod internal server error response has a 4xx status code
func (o *ImageDeleteLibpodInternalServerError) IsClientError() bool {
	return false
}

// IsServerError returns true when this image delete libpod internal server error response has a 5xx status code
func (o *ImageDeleteLibpodInternalServerError) IsServerError() bool {
	return true
}

// IsCode returns true when this image delete libpod internal server error response a status code equal to that given
func (o *ImageDeleteLibpodInternalServerError) IsCode(code int) bool {
	return code == 500
}

func (o *ImageDeleteLibpodInternalServerError) Error() string {
	return fmt.Sprintf("[DELETE /libpod/images/{name}][%d] imageDeleteLibpodInternalServerError  %+v", 500, o.Payload)
}

func (o *ImageDeleteLibpodInternalServerError) String() string {
	return fmt.Sprintf("[DELETE /libpod/images/{name}][%d] imageDeleteLibpodInternalServerError  %+v", 500, o.Payload)
}

func (o *ImageDeleteLibpodInternalServerError) GetPayload() *ImageDeleteLibpodInternalServerErrorBody {
	return o.Payload
}

func (o *ImageDeleteLibpodInternalServerError) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(ImageDeleteLibpodInternalServerErrorBody)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

/*
ImageDeleteLibpodBadRequestBody image delete libpod bad request body
swagger:model ImageDeleteLibpodBadRequestBody
*/
type ImageDeleteLibpodBadRequestBody struct {

	// API root cause formatted for automated parsing
	// Example: API root cause
	Because string `json:"cause,omitempty"`

	// human error message, formatted for a human to read
	// Example: human error message
	Message string `json:"message,omitempty"`

	// http response code
	ResponseCode int64 `json:"response,omitempty"`
}

// Validate validates this image delete libpod bad request body
func (o *ImageDeleteLibpodBadRequestBody) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this image delete libpod bad request body based on context it is used
func (o *ImageDeleteLibpodBadRequestBody) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (o *ImageDeleteLibpodBadRequestBody) MarshalBinary() ([]byte, error) {
	if o == nil {
		return nil, nil
	}
	return swag.WriteJSON(o)
}

// UnmarshalBinary interface implementation
func (o *ImageDeleteLibpodBadRequestBody) UnmarshalBinary(b []byte) error {
	var res ImageDeleteLibpodBadRequestBody
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*o = res
	return nil
}

/*
ImageDeleteLibpodConflictBody image delete libpod conflict body
swagger:model ImageDeleteLibpodConflictBody
*/
type ImageDeleteLibpodConflictBody struct {

	// API root cause formatted for automated parsing
	// Example: API root cause
	Because string `json:"cause,omitempty"`

	// human error message, formatted for a human to read
	// Example: human error message
	Message string `json:"message,omitempty"`

	// http response code
	ResponseCode int64 `json:"response,omitempty"`
}

// Validate validates this image delete libpod conflict body
func (o *ImageDeleteLibpodConflictBody) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this image delete libpod conflict body based on context it is used
func (o *ImageDeleteLibpodConflictBody) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (o *ImageDeleteLibpodConflictBody) MarshalBinary() ([]byte, error) {
	if o == nil {
		return nil, nil
	}
	return swag.WriteJSON(o)
}

// UnmarshalBinary interface implementation
func (o *ImageDeleteLibpodConflictBody) UnmarshalBinary(b []byte) error {
	var res ImageDeleteLibpodConflictBody
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*o = res
	return nil
}

/*
ImageDeleteLibpodInternalServerErrorBody image delete libpod internal server error body
swagger:model ImageDeleteLibpodInternalServerErrorBody
*/
type ImageDeleteLibpodInternalServerErrorBody struct {

	// API root cause formatted for automated parsing
	// Example: API root cause
	Because string `json:"cause,omitempty"`

	// human error message, formatted for a human to read
	// Example: human error message
	Message string `json:"message,omitempty"`

	// http response code
	ResponseCode int64 `json:"response,omitempty"`
}

// Validate validates this image delete libpod internal server error body
func (o *ImageDeleteLibpodInternalServerErrorBody) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this image delete libpod internal server error body based on context it is used
func (o *ImageDeleteLibpodInternalServerErrorBody) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (o *ImageDeleteLibpodInternalServerErrorBody) MarshalBinary() ([]byte, error) {
	if o == nil {
		return nil, nil
	}
	return swag.WriteJSON(o)
}

// UnmarshalBinary interface implementation
func (o *ImageDeleteLibpodInternalServerErrorBody) UnmarshalBinary(b []byte) error {
	var res ImageDeleteLibpodInternalServerErrorBody
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*o = res
	return nil
}

/*
ImageDeleteLibpodNotFoundBody image delete libpod not found body
swagger:model ImageDeleteLibpodNotFoundBody
*/
type ImageDeleteLibpodNotFoundBody struct {

	// API root cause formatted for automated parsing
	// Example: API root cause
	Because string `json:"cause,omitempty"`

	// human error message, formatted for a human to read
	// Example: human error message
	Message string `json:"message,omitempty"`

	// http response code
	ResponseCode int64 `json:"response,omitempty"`
}

// Validate validates this image delete libpod not found body
func (o *ImageDeleteLibpodNotFoundBody) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this image delete libpod not found body based on context it is used
func (o *ImageDeleteLibpodNotFoundBody) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (o *ImageDeleteLibpodNotFoundBody) MarshalBinary() ([]byte, error) {
	if o == nil {
		return nil, nil
	}
	return swag.WriteJSON(o)
}

// UnmarshalBinary interface implementation
func (o *ImageDeleteLibpodNotFoundBody) UnmarshalBinary(b []byte) error {
	var res ImageDeleteLibpodNotFoundBody
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*o = res
	return nil
}
