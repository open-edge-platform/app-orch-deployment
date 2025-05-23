// Code generated by go-swagger; DO NOT EDIT.

package pods

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

// PodDeleteLibpodReader is a Reader for the PodDeleteLibpod structure.
type PodDeleteLibpodReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *PodDeleteLibpodReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewPodDeleteLibpodOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 400:
		result := NewPodDeleteLibpodBadRequest()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 404:
		result := NewPodDeleteLibpodNotFound()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 500:
		result := NewPodDeleteLibpodInternalServerError()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		return nil, runtime.NewAPIError("response status code does not match any response statuses defined for this endpoint in the swagger spec", response, response.Code())
	}
}

// NewPodDeleteLibpodOK creates a PodDeleteLibpodOK with default headers values
func NewPodDeleteLibpodOK() *PodDeleteLibpodOK {
	return &PodDeleteLibpodOK{}
}

/*
PodDeleteLibpodOK describes a response with status code 200, with default header values.

Rm pod
*/
type PodDeleteLibpodOK struct {
	Payload *models.PodRmReport
}

// IsSuccess returns true when this pod delete libpod o k response has a 2xx status code
func (o *PodDeleteLibpodOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this pod delete libpod o k response has a 3xx status code
func (o *PodDeleteLibpodOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this pod delete libpod o k response has a 4xx status code
func (o *PodDeleteLibpodOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this pod delete libpod o k response has a 5xx status code
func (o *PodDeleteLibpodOK) IsServerError() bool {
	return false
}

// IsCode returns true when this pod delete libpod o k response a status code equal to that given
func (o *PodDeleteLibpodOK) IsCode(code int) bool {
	return code == 200
}

func (o *PodDeleteLibpodOK) Error() string {
	return fmt.Sprintf("[DELETE /libpod/pods/{name}][%d] podDeleteLibpodOK  %+v", 200, o.Payload)
}

func (o *PodDeleteLibpodOK) String() string {
	return fmt.Sprintf("[DELETE /libpod/pods/{name}][%d] podDeleteLibpodOK  %+v", 200, o.Payload)
}

func (o *PodDeleteLibpodOK) GetPayload() *models.PodRmReport {
	return o.Payload
}

func (o *PodDeleteLibpodOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.PodRmReport)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewPodDeleteLibpodBadRequest creates a PodDeleteLibpodBadRequest with default headers values
func NewPodDeleteLibpodBadRequest() *PodDeleteLibpodBadRequest {
	return &PodDeleteLibpodBadRequest{}
}

/*
PodDeleteLibpodBadRequest describes a response with status code 400, with default header values.

Bad parameter in request
*/
type PodDeleteLibpodBadRequest struct {
	Payload *PodDeleteLibpodBadRequestBody
}

// IsSuccess returns true when this pod delete libpod bad request response has a 2xx status code
func (o *PodDeleteLibpodBadRequest) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this pod delete libpod bad request response has a 3xx status code
func (o *PodDeleteLibpodBadRequest) IsRedirect() bool {
	return false
}

// IsClientError returns true when this pod delete libpod bad request response has a 4xx status code
func (o *PodDeleteLibpodBadRequest) IsClientError() bool {
	return true
}

// IsServerError returns true when this pod delete libpod bad request response has a 5xx status code
func (o *PodDeleteLibpodBadRequest) IsServerError() bool {
	return false
}

// IsCode returns true when this pod delete libpod bad request response a status code equal to that given
func (o *PodDeleteLibpodBadRequest) IsCode(code int) bool {
	return code == 400
}

func (o *PodDeleteLibpodBadRequest) Error() string {
	return fmt.Sprintf("[DELETE /libpod/pods/{name}][%d] podDeleteLibpodBadRequest  %+v", 400, o.Payload)
}

func (o *PodDeleteLibpodBadRequest) String() string {
	return fmt.Sprintf("[DELETE /libpod/pods/{name}][%d] podDeleteLibpodBadRequest  %+v", 400, o.Payload)
}

func (o *PodDeleteLibpodBadRequest) GetPayload() *PodDeleteLibpodBadRequestBody {
	return o.Payload
}

func (o *PodDeleteLibpodBadRequest) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(PodDeleteLibpodBadRequestBody)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewPodDeleteLibpodNotFound creates a PodDeleteLibpodNotFound with default headers values
func NewPodDeleteLibpodNotFound() *PodDeleteLibpodNotFound {
	return &PodDeleteLibpodNotFound{}
}

/*
PodDeleteLibpodNotFound describes a response with status code 404, with default header values.

No such pod
*/
type PodDeleteLibpodNotFound struct {
	Payload *PodDeleteLibpodNotFoundBody
}

// IsSuccess returns true when this pod delete libpod not found response has a 2xx status code
func (o *PodDeleteLibpodNotFound) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this pod delete libpod not found response has a 3xx status code
func (o *PodDeleteLibpodNotFound) IsRedirect() bool {
	return false
}

// IsClientError returns true when this pod delete libpod not found response has a 4xx status code
func (o *PodDeleteLibpodNotFound) IsClientError() bool {
	return true
}

// IsServerError returns true when this pod delete libpod not found response has a 5xx status code
func (o *PodDeleteLibpodNotFound) IsServerError() bool {
	return false
}

// IsCode returns true when this pod delete libpod not found response a status code equal to that given
func (o *PodDeleteLibpodNotFound) IsCode(code int) bool {
	return code == 404
}

func (o *PodDeleteLibpodNotFound) Error() string {
	return fmt.Sprintf("[DELETE /libpod/pods/{name}][%d] podDeleteLibpodNotFound  %+v", 404, o.Payload)
}

func (o *PodDeleteLibpodNotFound) String() string {
	return fmt.Sprintf("[DELETE /libpod/pods/{name}][%d] podDeleteLibpodNotFound  %+v", 404, o.Payload)
}

func (o *PodDeleteLibpodNotFound) GetPayload() *PodDeleteLibpodNotFoundBody {
	return o.Payload
}

func (o *PodDeleteLibpodNotFound) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(PodDeleteLibpodNotFoundBody)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewPodDeleteLibpodInternalServerError creates a PodDeleteLibpodInternalServerError with default headers values
func NewPodDeleteLibpodInternalServerError() *PodDeleteLibpodInternalServerError {
	return &PodDeleteLibpodInternalServerError{}
}

/*
PodDeleteLibpodInternalServerError describes a response with status code 500, with default header values.

Internal server error
*/
type PodDeleteLibpodInternalServerError struct {
	Payload *PodDeleteLibpodInternalServerErrorBody
}

// IsSuccess returns true when this pod delete libpod internal server error response has a 2xx status code
func (o *PodDeleteLibpodInternalServerError) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this pod delete libpod internal server error response has a 3xx status code
func (o *PodDeleteLibpodInternalServerError) IsRedirect() bool {
	return false
}

// IsClientError returns true when this pod delete libpod internal server error response has a 4xx status code
func (o *PodDeleteLibpodInternalServerError) IsClientError() bool {
	return false
}

// IsServerError returns true when this pod delete libpod internal server error response has a 5xx status code
func (o *PodDeleteLibpodInternalServerError) IsServerError() bool {
	return true
}

// IsCode returns true when this pod delete libpod internal server error response a status code equal to that given
func (o *PodDeleteLibpodInternalServerError) IsCode(code int) bool {
	return code == 500
}

func (o *PodDeleteLibpodInternalServerError) Error() string {
	return fmt.Sprintf("[DELETE /libpod/pods/{name}][%d] podDeleteLibpodInternalServerError  %+v", 500, o.Payload)
}

func (o *PodDeleteLibpodInternalServerError) String() string {
	return fmt.Sprintf("[DELETE /libpod/pods/{name}][%d] podDeleteLibpodInternalServerError  %+v", 500, o.Payload)
}

func (o *PodDeleteLibpodInternalServerError) GetPayload() *PodDeleteLibpodInternalServerErrorBody {
	return o.Payload
}

func (o *PodDeleteLibpodInternalServerError) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(PodDeleteLibpodInternalServerErrorBody)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

/*
PodDeleteLibpodBadRequestBody pod delete libpod bad request body
swagger:model PodDeleteLibpodBadRequestBody
*/
type PodDeleteLibpodBadRequestBody struct {

	// API root cause formatted for automated parsing
	// Example: API root cause
	Because string `json:"cause,omitempty"`

	// human error message, formatted for a human to read
	// Example: human error message
	Message string `json:"message,omitempty"`

	// http response code
	ResponseCode int64 `json:"response,omitempty"`
}

// Validate validates this pod delete libpod bad request body
func (o *PodDeleteLibpodBadRequestBody) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this pod delete libpod bad request body based on context it is used
func (o *PodDeleteLibpodBadRequestBody) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (o *PodDeleteLibpodBadRequestBody) MarshalBinary() ([]byte, error) {
	if o == nil {
		return nil, nil
	}
	return swag.WriteJSON(o)
}

// UnmarshalBinary interface implementation
func (o *PodDeleteLibpodBadRequestBody) UnmarshalBinary(b []byte) error {
	var res PodDeleteLibpodBadRequestBody
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*o = res
	return nil
}

/*
PodDeleteLibpodInternalServerErrorBody pod delete libpod internal server error body
swagger:model PodDeleteLibpodInternalServerErrorBody
*/
type PodDeleteLibpodInternalServerErrorBody struct {

	// API root cause formatted for automated parsing
	// Example: API root cause
	Because string `json:"cause,omitempty"`

	// human error message, formatted for a human to read
	// Example: human error message
	Message string `json:"message,omitempty"`

	// http response code
	ResponseCode int64 `json:"response,omitempty"`
}

// Validate validates this pod delete libpod internal server error body
func (o *PodDeleteLibpodInternalServerErrorBody) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this pod delete libpod internal server error body based on context it is used
func (o *PodDeleteLibpodInternalServerErrorBody) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (o *PodDeleteLibpodInternalServerErrorBody) MarshalBinary() ([]byte, error) {
	if o == nil {
		return nil, nil
	}
	return swag.WriteJSON(o)
}

// UnmarshalBinary interface implementation
func (o *PodDeleteLibpodInternalServerErrorBody) UnmarshalBinary(b []byte) error {
	var res PodDeleteLibpodInternalServerErrorBody
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*o = res
	return nil
}

/*
PodDeleteLibpodNotFoundBody pod delete libpod not found body
swagger:model PodDeleteLibpodNotFoundBody
*/
type PodDeleteLibpodNotFoundBody struct {

	// API root cause formatted for automated parsing
	// Example: API root cause
	Because string `json:"cause,omitempty"`

	// human error message, formatted for a human to read
	// Example: human error message
	Message string `json:"message,omitempty"`

	// http response code
	ResponseCode int64 `json:"response,omitempty"`
}

// Validate validates this pod delete libpod not found body
func (o *PodDeleteLibpodNotFoundBody) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this pod delete libpod not found body based on context it is used
func (o *PodDeleteLibpodNotFoundBody) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (o *PodDeleteLibpodNotFoundBody) MarshalBinary() ([]byte, error) {
	if o == nil {
		return nil, nil
	}
	return swag.WriteJSON(o)
}

// UnmarshalBinary interface implementation
func (o *PodDeleteLibpodNotFoundBody) UnmarshalBinary(b []byte) error {
	var res PodDeleteLibpodNotFoundBody
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*o = res
	return nil
}
