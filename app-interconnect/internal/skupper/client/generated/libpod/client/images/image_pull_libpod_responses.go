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

// ImagePullLibpodReader is a Reader for the ImagePullLibpod structure.
type ImagePullLibpodReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ImagePullLibpodReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewImagePullLibpodOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 400:
		result := NewImagePullLibpodBadRequest()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 500:
		result := NewImagePullLibpodInternalServerError()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		return nil, runtime.NewAPIError("response status code does not match any response statuses defined for this endpoint in the swagger spec", response, response.Code())
	}
}

// NewImagePullLibpodOK creates a ImagePullLibpodOK with default headers values
func NewImagePullLibpodOK() *ImagePullLibpodOK {
	return &ImagePullLibpodOK{}
}

/*
ImagePullLibpodOK describes a response with status code 200, with default header values.

Pull response
*/
type ImagePullLibpodOK struct {
	Payload *models.LibpodImagesPullReport
}

// IsSuccess returns true when this image pull libpod o k response has a 2xx status code
func (o *ImagePullLibpodOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this image pull libpod o k response has a 3xx status code
func (o *ImagePullLibpodOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this image pull libpod o k response has a 4xx status code
func (o *ImagePullLibpodOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this image pull libpod o k response has a 5xx status code
func (o *ImagePullLibpodOK) IsServerError() bool {
	return false
}

// IsCode returns true when this image pull libpod o k response a status code equal to that given
func (o *ImagePullLibpodOK) IsCode(code int) bool {
	return code == 200
}

func (o *ImagePullLibpodOK) Error() string {
	return fmt.Sprintf("[POST /libpod/images/pull][%d] imagePullLibpodOK  %+v", 200, o.Payload)
}

func (o *ImagePullLibpodOK) String() string {
	return fmt.Sprintf("[POST /libpod/images/pull][%d] imagePullLibpodOK  %+v", 200, o.Payload)
}

func (o *ImagePullLibpodOK) GetPayload() *models.LibpodImagesPullReport {
	return o.Payload
}

func (o *ImagePullLibpodOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.LibpodImagesPullReport)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewImagePullLibpodBadRequest creates a ImagePullLibpodBadRequest with default headers values
func NewImagePullLibpodBadRequest() *ImagePullLibpodBadRequest {
	return &ImagePullLibpodBadRequest{}
}

/*
ImagePullLibpodBadRequest describes a response with status code 400, with default header values.

Bad parameter in request
*/
type ImagePullLibpodBadRequest struct {
	Payload *ImagePullLibpodBadRequestBody
}

// IsSuccess returns true when this image pull libpod bad request response has a 2xx status code
func (o *ImagePullLibpodBadRequest) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this image pull libpod bad request response has a 3xx status code
func (o *ImagePullLibpodBadRequest) IsRedirect() bool {
	return false
}

// IsClientError returns true when this image pull libpod bad request response has a 4xx status code
func (o *ImagePullLibpodBadRequest) IsClientError() bool {
	return true
}

// IsServerError returns true when this image pull libpod bad request response has a 5xx status code
func (o *ImagePullLibpodBadRequest) IsServerError() bool {
	return false
}

// IsCode returns true when this image pull libpod bad request response a status code equal to that given
func (o *ImagePullLibpodBadRequest) IsCode(code int) bool {
	return code == 400
}

func (o *ImagePullLibpodBadRequest) Error() string {
	return fmt.Sprintf("[POST /libpod/images/pull][%d] imagePullLibpodBadRequest  %+v", 400, o.Payload)
}

func (o *ImagePullLibpodBadRequest) String() string {
	return fmt.Sprintf("[POST /libpod/images/pull][%d] imagePullLibpodBadRequest  %+v", 400, o.Payload)
}

func (o *ImagePullLibpodBadRequest) GetPayload() *ImagePullLibpodBadRequestBody {
	return o.Payload
}

func (o *ImagePullLibpodBadRequest) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(ImagePullLibpodBadRequestBody)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewImagePullLibpodInternalServerError creates a ImagePullLibpodInternalServerError with default headers values
func NewImagePullLibpodInternalServerError() *ImagePullLibpodInternalServerError {
	return &ImagePullLibpodInternalServerError{}
}

/*
ImagePullLibpodInternalServerError describes a response with status code 500, with default header values.

Internal server error
*/
type ImagePullLibpodInternalServerError struct {
	Payload *ImagePullLibpodInternalServerErrorBody
}

// IsSuccess returns true when this image pull libpod internal server error response has a 2xx status code
func (o *ImagePullLibpodInternalServerError) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this image pull libpod internal server error response has a 3xx status code
func (o *ImagePullLibpodInternalServerError) IsRedirect() bool {
	return false
}

// IsClientError returns true when this image pull libpod internal server error response has a 4xx status code
func (o *ImagePullLibpodInternalServerError) IsClientError() bool {
	return false
}

// IsServerError returns true when this image pull libpod internal server error response has a 5xx status code
func (o *ImagePullLibpodInternalServerError) IsServerError() bool {
	return true
}

// IsCode returns true when this image pull libpod internal server error response a status code equal to that given
func (o *ImagePullLibpodInternalServerError) IsCode(code int) bool {
	return code == 500
}

func (o *ImagePullLibpodInternalServerError) Error() string {
	return fmt.Sprintf("[POST /libpod/images/pull][%d] imagePullLibpodInternalServerError  %+v", 500, o.Payload)
}

func (o *ImagePullLibpodInternalServerError) String() string {
	return fmt.Sprintf("[POST /libpod/images/pull][%d] imagePullLibpodInternalServerError  %+v", 500, o.Payload)
}

func (o *ImagePullLibpodInternalServerError) GetPayload() *ImagePullLibpodInternalServerErrorBody {
	return o.Payload
}

func (o *ImagePullLibpodInternalServerError) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(ImagePullLibpodInternalServerErrorBody)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

/*
ImagePullLibpodBadRequestBody image pull libpod bad request body
swagger:model ImagePullLibpodBadRequestBody
*/
type ImagePullLibpodBadRequestBody struct {

	// API root cause formatted for automated parsing
	// Example: API root cause
	Because string `json:"cause,omitempty"`

	// human error message, formatted for a human to read
	// Example: human error message
	Message string `json:"message,omitempty"`

	// http response code
	ResponseCode int64 `json:"response,omitempty"`
}

// Validate validates this image pull libpod bad request body
func (o *ImagePullLibpodBadRequestBody) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this image pull libpod bad request body based on context it is used
func (o *ImagePullLibpodBadRequestBody) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (o *ImagePullLibpodBadRequestBody) MarshalBinary() ([]byte, error) {
	if o == nil {
		return nil, nil
	}
	return swag.WriteJSON(o)
}

// UnmarshalBinary interface implementation
func (o *ImagePullLibpodBadRequestBody) UnmarshalBinary(b []byte) error {
	var res ImagePullLibpodBadRequestBody
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*o = res
	return nil
}

/*
ImagePullLibpodInternalServerErrorBody image pull libpod internal server error body
swagger:model ImagePullLibpodInternalServerErrorBody
*/
type ImagePullLibpodInternalServerErrorBody struct {

	// API root cause formatted for automated parsing
	// Example: API root cause
	Because string `json:"cause,omitempty"`

	// human error message, formatted for a human to read
	// Example: human error message
	Message string `json:"message,omitempty"`

	// http response code
	ResponseCode int64 `json:"response,omitempty"`
}

// Validate validates this image pull libpod internal server error body
func (o *ImagePullLibpodInternalServerErrorBody) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this image pull libpod internal server error body based on context it is used
func (o *ImagePullLibpodInternalServerErrorBody) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (o *ImagePullLibpodInternalServerErrorBody) MarshalBinary() ([]byte, error) {
	if o == nil {
		return nil, nil
	}
	return swag.WriteJSON(o)
}

// UnmarshalBinary interface implementation
func (o *ImagePullLibpodInternalServerErrorBody) UnmarshalBinary(b []byte) error {
	var res ImagePullLibpodInternalServerErrorBody
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*o = res
	return nil
}
