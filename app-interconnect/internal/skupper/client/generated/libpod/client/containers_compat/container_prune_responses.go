// Code generated by go-swagger; DO NOT EDIT.

package containers_compat

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

// ContainerPruneReader is a Reader for the ContainerPrune structure.
type ContainerPruneReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ContainerPruneReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewContainerPruneOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 500:
		result := NewContainerPruneInternalServerError()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		return nil, runtime.NewAPIError("response status code does not match any response statuses defined for this endpoint in the swagger spec", response, response.Code())
	}
}

// NewContainerPruneOK creates a ContainerPruneOK with default headers values
func NewContainerPruneOK() *ContainerPruneOK {
	return &ContainerPruneOK{}
}

/*
ContainerPruneOK describes a response with status code 200, with default header values.

Prune containers
*/
type ContainerPruneOK struct {
	Payload []*models.ContainersPruneReport
}

// IsSuccess returns true when this container prune o k response has a 2xx status code
func (o *ContainerPruneOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this container prune o k response has a 3xx status code
func (o *ContainerPruneOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this container prune o k response has a 4xx status code
func (o *ContainerPruneOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this container prune o k response has a 5xx status code
func (o *ContainerPruneOK) IsServerError() bool {
	return false
}

// IsCode returns true when this container prune o k response a status code equal to that given
func (o *ContainerPruneOK) IsCode(code int) bool {
	return code == 200
}

func (o *ContainerPruneOK) Error() string {
	return fmt.Sprintf("[POST /containers/prune][%d] containerPruneOK  %+v", 200, o.Payload)
}

func (o *ContainerPruneOK) String() string {
	return fmt.Sprintf("[POST /containers/prune][%d] containerPruneOK  %+v", 200, o.Payload)
}

func (o *ContainerPruneOK) GetPayload() []*models.ContainersPruneReport {
	return o.Payload
}

func (o *ContainerPruneOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewContainerPruneInternalServerError creates a ContainerPruneInternalServerError with default headers values
func NewContainerPruneInternalServerError() *ContainerPruneInternalServerError {
	return &ContainerPruneInternalServerError{}
}

/*
ContainerPruneInternalServerError describes a response with status code 500, with default header values.

Internal server error
*/
type ContainerPruneInternalServerError struct {
	Payload *ContainerPruneInternalServerErrorBody
}

// IsSuccess returns true when this container prune internal server error response has a 2xx status code
func (o *ContainerPruneInternalServerError) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this container prune internal server error response has a 3xx status code
func (o *ContainerPruneInternalServerError) IsRedirect() bool {
	return false
}

// IsClientError returns true when this container prune internal server error response has a 4xx status code
func (o *ContainerPruneInternalServerError) IsClientError() bool {
	return false
}

// IsServerError returns true when this container prune internal server error response has a 5xx status code
func (o *ContainerPruneInternalServerError) IsServerError() bool {
	return true
}

// IsCode returns true when this container prune internal server error response a status code equal to that given
func (o *ContainerPruneInternalServerError) IsCode(code int) bool {
	return code == 500
}

func (o *ContainerPruneInternalServerError) Error() string {
	return fmt.Sprintf("[POST /containers/prune][%d] containerPruneInternalServerError  %+v", 500, o.Payload)
}

func (o *ContainerPruneInternalServerError) String() string {
	return fmt.Sprintf("[POST /containers/prune][%d] containerPruneInternalServerError  %+v", 500, o.Payload)
}

func (o *ContainerPruneInternalServerError) GetPayload() *ContainerPruneInternalServerErrorBody {
	return o.Payload
}

func (o *ContainerPruneInternalServerError) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(ContainerPruneInternalServerErrorBody)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

/*
ContainerPruneInternalServerErrorBody container prune internal server error body
swagger:model ContainerPruneInternalServerErrorBody
*/
type ContainerPruneInternalServerErrorBody struct {

	// API root cause formatted for automated parsing
	// Example: API root cause
	Because string `json:"cause,omitempty"`

	// human error message, formatted for a human to read
	// Example: human error message
	Message string `json:"message,omitempty"`

	// http response code
	ResponseCode int64 `json:"response,omitempty"`
}

// Validate validates this container prune internal server error body
func (o *ContainerPruneInternalServerErrorBody) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this container prune internal server error body based on context it is used
func (o *ContainerPruneInternalServerErrorBody) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (o *ContainerPruneInternalServerErrorBody) MarshalBinary() ([]byte, error) {
	if o == nil {
		return nil, nil
	}
	return swag.WriteJSON(o)
}

// UnmarshalBinary interface implementation
func (o *ContainerPruneInternalServerErrorBody) UnmarshalBinary(b []byte) error {
	var res ContainerPruneInternalServerErrorBody
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*o = res
	return nil
}
