// Code generated by go-swagger; DO NOT EDIT.

package containers

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

// ContainerPruneLibpodReader is a Reader for the ContainerPruneLibpod structure.
type ContainerPruneLibpodReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ContainerPruneLibpodReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewContainerPruneLibpodOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 500:
		result := NewContainerPruneLibpodInternalServerError()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		return nil, runtime.NewAPIError("response status code does not match any response statuses defined for this endpoint in the swagger spec", response, response.Code())
	}
}

// NewContainerPruneLibpodOK creates a ContainerPruneLibpodOK with default headers values
func NewContainerPruneLibpodOK() *ContainerPruneLibpodOK {
	return &ContainerPruneLibpodOK{}
}

/*
ContainerPruneLibpodOK describes a response with status code 200, with default header values.

Prune containers
*/
type ContainerPruneLibpodOK struct {
	Payload []*models.LibpodContainersPruneReport
}

// IsSuccess returns true when this container prune libpod o k response has a 2xx status code
func (o *ContainerPruneLibpodOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this container prune libpod o k response has a 3xx status code
func (o *ContainerPruneLibpodOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this container prune libpod o k response has a 4xx status code
func (o *ContainerPruneLibpodOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this container prune libpod o k response has a 5xx status code
func (o *ContainerPruneLibpodOK) IsServerError() bool {
	return false
}

// IsCode returns true when this container prune libpod o k response a status code equal to that given
func (o *ContainerPruneLibpodOK) IsCode(code int) bool {
	return code == 200
}

func (o *ContainerPruneLibpodOK) Error() string {
	return fmt.Sprintf("[POST /libpod/containers/prune][%d] containerPruneLibpodOK  %+v", 200, o.Payload)
}

func (o *ContainerPruneLibpodOK) String() string {
	return fmt.Sprintf("[POST /libpod/containers/prune][%d] containerPruneLibpodOK  %+v", 200, o.Payload)
}

func (o *ContainerPruneLibpodOK) GetPayload() []*models.LibpodContainersPruneReport {
	return o.Payload
}

func (o *ContainerPruneLibpodOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewContainerPruneLibpodInternalServerError creates a ContainerPruneLibpodInternalServerError with default headers values
func NewContainerPruneLibpodInternalServerError() *ContainerPruneLibpodInternalServerError {
	return &ContainerPruneLibpodInternalServerError{}
}

/*
ContainerPruneLibpodInternalServerError describes a response with status code 500, with default header values.

Internal server error
*/
type ContainerPruneLibpodInternalServerError struct {
	Payload *ContainerPruneLibpodInternalServerErrorBody
}

// IsSuccess returns true when this container prune libpod internal server error response has a 2xx status code
func (o *ContainerPruneLibpodInternalServerError) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this container prune libpod internal server error response has a 3xx status code
func (o *ContainerPruneLibpodInternalServerError) IsRedirect() bool {
	return false
}

// IsClientError returns true when this container prune libpod internal server error response has a 4xx status code
func (o *ContainerPruneLibpodInternalServerError) IsClientError() bool {
	return false
}

// IsServerError returns true when this container prune libpod internal server error response has a 5xx status code
func (o *ContainerPruneLibpodInternalServerError) IsServerError() bool {
	return true
}

// IsCode returns true when this container prune libpod internal server error response a status code equal to that given
func (o *ContainerPruneLibpodInternalServerError) IsCode(code int) bool {
	return code == 500
}

func (o *ContainerPruneLibpodInternalServerError) Error() string {
	return fmt.Sprintf("[POST /libpod/containers/prune][%d] containerPruneLibpodInternalServerError  %+v", 500, o.Payload)
}

func (o *ContainerPruneLibpodInternalServerError) String() string {
	return fmt.Sprintf("[POST /libpod/containers/prune][%d] containerPruneLibpodInternalServerError  %+v", 500, o.Payload)
}

func (o *ContainerPruneLibpodInternalServerError) GetPayload() *ContainerPruneLibpodInternalServerErrorBody {
	return o.Payload
}

func (o *ContainerPruneLibpodInternalServerError) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(ContainerPruneLibpodInternalServerErrorBody)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

/*
ContainerPruneLibpodInternalServerErrorBody container prune libpod internal server error body
swagger:model ContainerPruneLibpodInternalServerErrorBody
*/
type ContainerPruneLibpodInternalServerErrorBody struct {

	// API root cause formatted for automated parsing
	// Example: API root cause
	Because string `json:"cause,omitempty"`

	// human error message, formatted for a human to read
	// Example: human error message
	Message string `json:"message,omitempty"`

	// http response code
	ResponseCode int64 `json:"response,omitempty"`
}

// Validate validates this container prune libpod internal server error body
func (o *ContainerPruneLibpodInternalServerErrorBody) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this container prune libpod internal server error body based on context it is used
func (o *ContainerPruneLibpodInternalServerErrorBody) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (o *ContainerPruneLibpodInternalServerErrorBody) MarshalBinary() ([]byte, error) {
	if o == nil {
		return nil, nil
	}
	return swag.WriteJSON(o)
}

// UnmarshalBinary interface implementation
func (o *ContainerPruneLibpodInternalServerErrorBody) UnmarshalBinary(b []byte) error {
	var res ContainerPruneLibpodInternalServerErrorBody
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*o = res
	return nil
}
