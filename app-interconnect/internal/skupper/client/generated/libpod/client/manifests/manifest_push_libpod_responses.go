// Code generated by go-swagger; DO NOT EDIT.

package manifests

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

// ManifestPushLibpodReader is a Reader for the ManifestPushLibpod structure.
type ManifestPushLibpodReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ManifestPushLibpodReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewManifestPushLibpodOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 400:
		result := NewManifestPushLibpodBadRequest()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 404:
		result := NewManifestPushLibpodNotFound()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 500:
		result := NewManifestPushLibpodInternalServerError()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		return nil, runtime.NewAPIError("response status code does not match any response statuses defined for this endpoint in the swagger spec", response, response.Code())
	}
}

// NewManifestPushLibpodOK creates a ManifestPushLibpodOK with default headers values
func NewManifestPushLibpodOK() *ManifestPushLibpodOK {
	return &ManifestPushLibpodOK{}
}

/*
ManifestPushLibpodOK describes a response with status code 200, with default header values.

ManifestPushLibpodOK manifest push libpod o k
*/
type ManifestPushLibpodOK struct {
	Payload *models.IDResponse
}

// IsSuccess returns true when this manifest push libpod o k response has a 2xx status code
func (o *ManifestPushLibpodOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this manifest push libpod o k response has a 3xx status code
func (o *ManifestPushLibpodOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this manifest push libpod o k response has a 4xx status code
func (o *ManifestPushLibpodOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this manifest push libpod o k response has a 5xx status code
func (o *ManifestPushLibpodOK) IsServerError() bool {
	return false
}

// IsCode returns true when this manifest push libpod o k response a status code equal to that given
func (o *ManifestPushLibpodOK) IsCode(code int) bool {
	return code == 200
}

func (o *ManifestPushLibpodOK) Error() string {
	return fmt.Sprintf("[POST /libpod/manifests/{name}/registry/{destination}][%d] manifestPushLibpodOK  %+v", 200, o.Payload)
}

func (o *ManifestPushLibpodOK) String() string {
	return fmt.Sprintf("[POST /libpod/manifests/{name}/registry/{destination}][%d] manifestPushLibpodOK  %+v", 200, o.Payload)
}

func (o *ManifestPushLibpodOK) GetPayload() *models.IDResponse {
	return o.Payload
}

func (o *ManifestPushLibpodOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.IDResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewManifestPushLibpodBadRequest creates a ManifestPushLibpodBadRequest with default headers values
func NewManifestPushLibpodBadRequest() *ManifestPushLibpodBadRequest {
	return &ManifestPushLibpodBadRequest{}
}

/*
ManifestPushLibpodBadRequest describes a response with status code 400, with default header values.

Bad parameter in request
*/
type ManifestPushLibpodBadRequest struct {
	Payload *ManifestPushLibpodBadRequestBody
}

// IsSuccess returns true when this manifest push libpod bad request response has a 2xx status code
func (o *ManifestPushLibpodBadRequest) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this manifest push libpod bad request response has a 3xx status code
func (o *ManifestPushLibpodBadRequest) IsRedirect() bool {
	return false
}

// IsClientError returns true when this manifest push libpod bad request response has a 4xx status code
func (o *ManifestPushLibpodBadRequest) IsClientError() bool {
	return true
}

// IsServerError returns true when this manifest push libpod bad request response has a 5xx status code
func (o *ManifestPushLibpodBadRequest) IsServerError() bool {
	return false
}

// IsCode returns true when this manifest push libpod bad request response a status code equal to that given
func (o *ManifestPushLibpodBadRequest) IsCode(code int) bool {
	return code == 400
}

func (o *ManifestPushLibpodBadRequest) Error() string {
	return fmt.Sprintf("[POST /libpod/manifests/{name}/registry/{destination}][%d] manifestPushLibpodBadRequest  %+v", 400, o.Payload)
}

func (o *ManifestPushLibpodBadRequest) String() string {
	return fmt.Sprintf("[POST /libpod/manifests/{name}/registry/{destination}][%d] manifestPushLibpodBadRequest  %+v", 400, o.Payload)
}

func (o *ManifestPushLibpodBadRequest) GetPayload() *ManifestPushLibpodBadRequestBody {
	return o.Payload
}

func (o *ManifestPushLibpodBadRequest) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(ManifestPushLibpodBadRequestBody)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewManifestPushLibpodNotFound creates a ManifestPushLibpodNotFound with default headers values
func NewManifestPushLibpodNotFound() *ManifestPushLibpodNotFound {
	return &ManifestPushLibpodNotFound{}
}

/*
ManifestPushLibpodNotFound describes a response with status code 404, with default header values.

No such manifest
*/
type ManifestPushLibpodNotFound struct {
	Payload *ManifestPushLibpodNotFoundBody
}

// IsSuccess returns true when this manifest push libpod not found response has a 2xx status code
func (o *ManifestPushLibpodNotFound) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this manifest push libpod not found response has a 3xx status code
func (o *ManifestPushLibpodNotFound) IsRedirect() bool {
	return false
}

// IsClientError returns true when this manifest push libpod not found response has a 4xx status code
func (o *ManifestPushLibpodNotFound) IsClientError() bool {
	return true
}

// IsServerError returns true when this manifest push libpod not found response has a 5xx status code
func (o *ManifestPushLibpodNotFound) IsServerError() bool {
	return false
}

// IsCode returns true when this manifest push libpod not found response a status code equal to that given
func (o *ManifestPushLibpodNotFound) IsCode(code int) bool {
	return code == 404
}

func (o *ManifestPushLibpodNotFound) Error() string {
	return fmt.Sprintf("[POST /libpod/manifests/{name}/registry/{destination}][%d] manifestPushLibpodNotFound  %+v", 404, o.Payload)
}

func (o *ManifestPushLibpodNotFound) String() string {
	return fmt.Sprintf("[POST /libpod/manifests/{name}/registry/{destination}][%d] manifestPushLibpodNotFound  %+v", 404, o.Payload)
}

func (o *ManifestPushLibpodNotFound) GetPayload() *ManifestPushLibpodNotFoundBody {
	return o.Payload
}

func (o *ManifestPushLibpodNotFound) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(ManifestPushLibpodNotFoundBody)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewManifestPushLibpodInternalServerError creates a ManifestPushLibpodInternalServerError with default headers values
func NewManifestPushLibpodInternalServerError() *ManifestPushLibpodInternalServerError {
	return &ManifestPushLibpodInternalServerError{}
}

/*
ManifestPushLibpodInternalServerError describes a response with status code 500, with default header values.

Internal server error
*/
type ManifestPushLibpodInternalServerError struct {
	Payload *ManifestPushLibpodInternalServerErrorBody
}

// IsSuccess returns true when this manifest push libpod internal server error response has a 2xx status code
func (o *ManifestPushLibpodInternalServerError) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this manifest push libpod internal server error response has a 3xx status code
func (o *ManifestPushLibpodInternalServerError) IsRedirect() bool {
	return false
}

// IsClientError returns true when this manifest push libpod internal server error response has a 4xx status code
func (o *ManifestPushLibpodInternalServerError) IsClientError() bool {
	return false
}

// IsServerError returns true when this manifest push libpod internal server error response has a 5xx status code
func (o *ManifestPushLibpodInternalServerError) IsServerError() bool {
	return true
}

// IsCode returns true when this manifest push libpod internal server error response a status code equal to that given
func (o *ManifestPushLibpodInternalServerError) IsCode(code int) bool {
	return code == 500
}

func (o *ManifestPushLibpodInternalServerError) Error() string {
	return fmt.Sprintf("[POST /libpod/manifests/{name}/registry/{destination}][%d] manifestPushLibpodInternalServerError  %+v", 500, o.Payload)
}

func (o *ManifestPushLibpodInternalServerError) String() string {
	return fmt.Sprintf("[POST /libpod/manifests/{name}/registry/{destination}][%d] manifestPushLibpodInternalServerError  %+v", 500, o.Payload)
}

func (o *ManifestPushLibpodInternalServerError) GetPayload() *ManifestPushLibpodInternalServerErrorBody {
	return o.Payload
}

func (o *ManifestPushLibpodInternalServerError) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(ManifestPushLibpodInternalServerErrorBody)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

/*
ManifestPushLibpodBadRequestBody manifest push libpod bad request body
swagger:model ManifestPushLibpodBadRequestBody
*/
type ManifestPushLibpodBadRequestBody struct {

	// API root cause formatted for automated parsing
	// Example: API root cause
	Because string `json:"cause,omitempty"`

	// human error message, formatted for a human to read
	// Example: human error message
	Message string `json:"message,omitempty"`

	// http response code
	ResponseCode int64 `json:"response,omitempty"`
}

// Validate validates this manifest push libpod bad request body
func (o *ManifestPushLibpodBadRequestBody) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this manifest push libpod bad request body based on context it is used
func (o *ManifestPushLibpodBadRequestBody) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (o *ManifestPushLibpodBadRequestBody) MarshalBinary() ([]byte, error) {
	if o == nil {
		return nil, nil
	}
	return swag.WriteJSON(o)
}

// UnmarshalBinary interface implementation
func (o *ManifestPushLibpodBadRequestBody) UnmarshalBinary(b []byte) error {
	var res ManifestPushLibpodBadRequestBody
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*o = res
	return nil
}

/*
ManifestPushLibpodInternalServerErrorBody manifest push libpod internal server error body
swagger:model ManifestPushLibpodInternalServerErrorBody
*/
type ManifestPushLibpodInternalServerErrorBody struct {

	// API root cause formatted for automated parsing
	// Example: API root cause
	Because string `json:"cause,omitempty"`

	// human error message, formatted for a human to read
	// Example: human error message
	Message string `json:"message,omitempty"`

	// http response code
	ResponseCode int64 `json:"response,omitempty"`
}

// Validate validates this manifest push libpod internal server error body
func (o *ManifestPushLibpodInternalServerErrorBody) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this manifest push libpod internal server error body based on context it is used
func (o *ManifestPushLibpodInternalServerErrorBody) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (o *ManifestPushLibpodInternalServerErrorBody) MarshalBinary() ([]byte, error) {
	if o == nil {
		return nil, nil
	}
	return swag.WriteJSON(o)
}

// UnmarshalBinary interface implementation
func (o *ManifestPushLibpodInternalServerErrorBody) UnmarshalBinary(b []byte) error {
	var res ManifestPushLibpodInternalServerErrorBody
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*o = res
	return nil
}

/*
ManifestPushLibpodNotFoundBody manifest push libpod not found body
swagger:model ManifestPushLibpodNotFoundBody
*/
type ManifestPushLibpodNotFoundBody struct {

	// API root cause formatted for automated parsing
	// Example: API root cause
	Because string `json:"cause,omitempty"`

	// human error message, formatted for a human to read
	// Example: human error message
	Message string `json:"message,omitempty"`

	// http response code
	ResponseCode int64 `json:"response,omitempty"`
}

// Validate validates this manifest push libpod not found body
func (o *ManifestPushLibpodNotFoundBody) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this manifest push libpod not found body based on context it is used
func (o *ManifestPushLibpodNotFoundBody) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (o *ManifestPushLibpodNotFoundBody) MarshalBinary() ([]byte, error) {
	if o == nil {
		return nil, nil
	}
	return swag.WriteJSON(o)
}

// UnmarshalBinary interface implementation
func (o *ManifestPushLibpodNotFoundBody) UnmarshalBinary(b []byte) error {
	var res ManifestPushLibpodNotFoundBody
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*o = res
	return nil
}
