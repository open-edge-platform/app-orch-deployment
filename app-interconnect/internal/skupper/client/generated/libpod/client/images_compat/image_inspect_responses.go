// Code generated by go-swagger; DO NOT EDIT.

package images_compat

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"
	"fmt"
	"io"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"

	"github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/skupper/client/generated/libpod/models"
)

// ImageInspectReader is a Reader for the ImageInspect structure.
type ImageInspectReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ImageInspectReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewImageInspectOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 404:
		result := NewImageInspectNotFound()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 500:
		result := NewImageInspectInternalServerError()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		return nil, runtime.NewAPIError("response status code does not match any response statuses defined for this endpoint in the swagger spec", response, response.Code())
	}
}

// NewImageInspectOK creates a ImageInspectOK with default headers values
func NewImageInspectOK() *ImageInspectOK {
	return &ImageInspectOK{}
}

/*
ImageInspectOK describes a response with status code 200, with default header values.

Inspect response
*/
type ImageInspectOK struct {
	Payload *ImageInspectOKBody
}

// IsSuccess returns true when this image inspect o k response has a 2xx status code
func (o *ImageInspectOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this image inspect o k response has a 3xx status code
func (o *ImageInspectOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this image inspect o k response has a 4xx status code
func (o *ImageInspectOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this image inspect o k response has a 5xx status code
func (o *ImageInspectOK) IsServerError() bool {
	return false
}

// IsCode returns true when this image inspect o k response a status code equal to that given
func (o *ImageInspectOK) IsCode(code int) bool {
	return code == 200
}

func (o *ImageInspectOK) Error() string {
	return fmt.Sprintf("[GET /images/{name}/json][%d] imageInspectOK  %+v", 200, o.Payload)
}

func (o *ImageInspectOK) String() string {
	return fmt.Sprintf("[GET /images/{name}/json][%d] imageInspectOK  %+v", 200, o.Payload)
}

func (o *ImageInspectOK) GetPayload() *ImageInspectOKBody {
	return o.Payload
}

func (o *ImageInspectOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(ImageInspectOKBody)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewImageInspectNotFound creates a ImageInspectNotFound with default headers values
func NewImageInspectNotFound() *ImageInspectNotFound {
	return &ImageInspectNotFound{}
}

/*
ImageInspectNotFound describes a response with status code 404, with default header values.

No such image
*/
type ImageInspectNotFound struct {
	Payload *ImageInspectNotFoundBody
}

// IsSuccess returns true when this image inspect not found response has a 2xx status code
func (o *ImageInspectNotFound) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this image inspect not found response has a 3xx status code
func (o *ImageInspectNotFound) IsRedirect() bool {
	return false
}

// IsClientError returns true when this image inspect not found response has a 4xx status code
func (o *ImageInspectNotFound) IsClientError() bool {
	return true
}

// IsServerError returns true when this image inspect not found response has a 5xx status code
func (o *ImageInspectNotFound) IsServerError() bool {
	return false
}

// IsCode returns true when this image inspect not found response a status code equal to that given
func (o *ImageInspectNotFound) IsCode(code int) bool {
	return code == 404
}

func (o *ImageInspectNotFound) Error() string {
	return fmt.Sprintf("[GET /images/{name}/json][%d] imageInspectNotFound  %+v", 404, o.Payload)
}

func (o *ImageInspectNotFound) String() string {
	return fmt.Sprintf("[GET /images/{name}/json][%d] imageInspectNotFound  %+v", 404, o.Payload)
}

func (o *ImageInspectNotFound) GetPayload() *ImageInspectNotFoundBody {
	return o.Payload
}

func (o *ImageInspectNotFound) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(ImageInspectNotFoundBody)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewImageInspectInternalServerError creates a ImageInspectInternalServerError with default headers values
func NewImageInspectInternalServerError() *ImageInspectInternalServerError {
	return &ImageInspectInternalServerError{}
}

/*
ImageInspectInternalServerError describes a response with status code 500, with default header values.

Internal server error
*/
type ImageInspectInternalServerError struct {
	Payload *ImageInspectInternalServerErrorBody
}

// IsSuccess returns true when this image inspect internal server error response has a 2xx status code
func (o *ImageInspectInternalServerError) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this image inspect internal server error response has a 3xx status code
func (o *ImageInspectInternalServerError) IsRedirect() bool {
	return false
}

// IsClientError returns true when this image inspect internal server error response has a 4xx status code
func (o *ImageInspectInternalServerError) IsClientError() bool {
	return false
}

// IsServerError returns true when this image inspect internal server error response has a 5xx status code
func (o *ImageInspectInternalServerError) IsServerError() bool {
	return true
}

// IsCode returns true when this image inspect internal server error response a status code equal to that given
func (o *ImageInspectInternalServerError) IsCode(code int) bool {
	return code == 500
}

func (o *ImageInspectInternalServerError) Error() string {
	return fmt.Sprintf("[GET /images/{name}/json][%d] imageInspectInternalServerError  %+v", 500, o.Payload)
}

func (o *ImageInspectInternalServerError) String() string {
	return fmt.Sprintf("[GET /images/{name}/json][%d] imageInspectInternalServerError  %+v", 500, o.Payload)
}

func (o *ImageInspectInternalServerError) GetPayload() *ImageInspectInternalServerErrorBody {
	return o.Payload
}

func (o *ImageInspectInternalServerError) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(ImageInspectInternalServerErrorBody)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

/*
ImageInspectInternalServerErrorBody image inspect internal server error body
swagger:model ImageInspectInternalServerErrorBody
*/
type ImageInspectInternalServerErrorBody struct {

	// API root cause formatted for automated parsing
	// Example: API root cause
	Because string `json:"cause,omitempty"`

	// human error message, formatted for a human to read
	// Example: human error message
	Message string `json:"message,omitempty"`

	// http response code
	ResponseCode int64 `json:"response,omitempty"`
}

// Validate validates this image inspect internal server error body
func (o *ImageInspectInternalServerErrorBody) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this image inspect internal server error body based on context it is used
func (o *ImageInspectInternalServerErrorBody) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (o *ImageInspectInternalServerErrorBody) MarshalBinary() ([]byte, error) {
	if o == nil {
		return nil, nil
	}
	return swag.WriteJSON(o)
}

// UnmarshalBinary interface implementation
func (o *ImageInspectInternalServerErrorBody) UnmarshalBinary(b []byte) error {
	var res ImageInspectInternalServerErrorBody
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*o = res
	return nil
}

/*
ImageInspectNotFoundBody image inspect not found body
swagger:model ImageInspectNotFoundBody
*/
type ImageInspectNotFoundBody struct {

	// API root cause formatted for automated parsing
	// Example: API root cause
	Because string `json:"cause,omitempty"`

	// human error message, formatted for a human to read
	// Example: human error message
	Message string `json:"message,omitempty"`

	// http response code
	ResponseCode int64 `json:"response,omitempty"`
}

// Validate validates this image inspect not found body
func (o *ImageInspectNotFoundBody) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this image inspect not found body based on context it is used
func (o *ImageInspectNotFoundBody) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (o *ImageInspectNotFoundBody) MarshalBinary() ([]byte, error) {
	if o == nil {
		return nil, nil
	}
	return swag.WriteJSON(o)
}

// UnmarshalBinary interface implementation
func (o *ImageInspectNotFoundBody) UnmarshalBinary(b []byte) error {
	var res ImageInspectNotFoundBody
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*o = res
	return nil
}

/*
ImageInspectOKBody image inspect o k body
swagger:model ImageInspectOKBody
*/
type ImageInspectOKBody struct {

	// architecture
	Architecture string `json:"Architecture,omitempty"`

	// author
	Author string `json:"Author,omitempty"`

	// comment
	Comment string `json:"Comment,omitempty"`

	// config
	Config *models.Config `json:"Config,omitempty"`

	// container
	Container string `json:"Container,omitempty"`

	// container config
	ContainerConfig *models.Config `json:"ContainerConfig,omitempty"`

	// created
	Created string `json:"Created,omitempty"`

	// docker version
	DockerVersion string `json:"DockerVersion,omitempty"`

	// graph driver
	GraphDriver *models.GraphDriverData `json:"GraphDriver,omitempty"`

	// ID
	ID string `json:"Id,omitempty"`

	// metadata
	Metadata *models.ImageMetadata `json:"Metadata,omitempty"`

	// os
	Os string `json:"Os,omitempty"`

	// os version
	OsVersion string `json:"OsVersion,omitempty"`

	// parent
	Parent string `json:"Parent,omitempty"`

	// repo digests
	RepoDigests []string `json:"RepoDigests"`

	// repo tags
	RepoTags []string `json:"RepoTags"`

	// root f s
	RootFS *models.RootFS `json:"RootFS,omitempty"`

	// size
	Size int64 `json:"Size,omitempty"`

	// variant
	Variant string `json:"Variant,omitempty"`

	// virtual size
	VirtualSize int64 `json:"VirtualSize,omitempty"`
}

// Validate validates this image inspect o k body
func (o *ImageInspectOKBody) Validate(formats strfmt.Registry) error {
	var res []error

	if err := o.validateConfig(formats); err != nil {
		res = append(res, err)
	}

	if err := o.validateContainerConfig(formats); err != nil {
		res = append(res, err)
	}

	if err := o.validateGraphDriver(formats); err != nil {
		res = append(res, err)
	}

	if err := o.validateMetadata(formats); err != nil {
		res = append(res, err)
	}

	if err := o.validateRootFS(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (o *ImageInspectOKBody) validateConfig(formats strfmt.Registry) error {
	if swag.IsZero(o.Config) { // not required
		return nil
	}

	if o.Config != nil {
		if err := o.Config.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("imageInspectOK" + "." + "Config")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("imageInspectOK" + "." + "Config")
			}
			return err
		}
	}

	return nil
}

func (o *ImageInspectOKBody) validateContainerConfig(formats strfmt.Registry) error {
	if swag.IsZero(o.ContainerConfig) { // not required
		return nil
	}

	if o.ContainerConfig != nil {
		if err := o.ContainerConfig.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("imageInspectOK" + "." + "ContainerConfig")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("imageInspectOK" + "." + "ContainerConfig")
			}
			return err
		}
	}

	return nil
}

func (o *ImageInspectOKBody) validateGraphDriver(formats strfmt.Registry) error {
	if swag.IsZero(o.GraphDriver) { // not required
		return nil
	}

	if o.GraphDriver != nil {
		if err := o.GraphDriver.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("imageInspectOK" + "." + "GraphDriver")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("imageInspectOK" + "." + "GraphDriver")
			}
			return err
		}
	}

	return nil
}

func (o *ImageInspectOKBody) validateMetadata(formats strfmt.Registry) error {
	if swag.IsZero(o.Metadata) { // not required
		return nil
	}

	if o.Metadata != nil {
		if err := o.Metadata.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("imageInspectOK" + "." + "Metadata")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("imageInspectOK" + "." + "Metadata")
			}
			return err
		}
	}

	return nil
}

func (o *ImageInspectOKBody) validateRootFS(formats strfmt.Registry) error {
	if swag.IsZero(o.RootFS) { // not required
		return nil
	}

	if o.RootFS != nil {
		if err := o.RootFS.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("imageInspectOK" + "." + "RootFS")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("imageInspectOK" + "." + "RootFS")
			}
			return err
		}
	}

	return nil
}

// ContextValidate validate this image inspect o k body based on the context it is used
func (o *ImageInspectOKBody) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := o.contextValidateConfig(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := o.contextValidateContainerConfig(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := o.contextValidateGraphDriver(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := o.contextValidateMetadata(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := o.contextValidateRootFS(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (o *ImageInspectOKBody) contextValidateConfig(ctx context.Context, formats strfmt.Registry) error {

	if o.Config != nil {
		if err := o.Config.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("imageInspectOK" + "." + "Config")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("imageInspectOK" + "." + "Config")
			}
			return err
		}
	}

	return nil
}

func (o *ImageInspectOKBody) contextValidateContainerConfig(ctx context.Context, formats strfmt.Registry) error {

	if o.ContainerConfig != nil {
		if err := o.ContainerConfig.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("imageInspectOK" + "." + "ContainerConfig")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("imageInspectOK" + "." + "ContainerConfig")
			}
			return err
		}
	}

	return nil
}

func (o *ImageInspectOKBody) contextValidateGraphDriver(ctx context.Context, formats strfmt.Registry) error {

	if o.GraphDriver != nil {
		if err := o.GraphDriver.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("imageInspectOK" + "." + "GraphDriver")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("imageInspectOK" + "." + "GraphDriver")
			}
			return err
		}
	}

	return nil
}

func (o *ImageInspectOKBody) contextValidateMetadata(ctx context.Context, formats strfmt.Registry) error {

	if o.Metadata != nil {
		if err := o.Metadata.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("imageInspectOK" + "." + "Metadata")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("imageInspectOK" + "." + "Metadata")
			}
			return err
		}
	}

	return nil
}

func (o *ImageInspectOKBody) contextValidateRootFS(ctx context.Context, formats strfmt.Registry) error {

	if o.RootFS != nil {
		if err := o.RootFS.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("imageInspectOK" + "." + "RootFS")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("imageInspectOK" + "." + "RootFS")
			}
			return err
		}
	}

	return nil
}

// MarshalBinary interface implementation
func (o *ImageInspectOKBody) MarshalBinary() ([]byte, error) {
	if o == nil {
		return nil, nil
	}
	return swag.WriteJSON(o)
}

// UnmarshalBinary interface implementation
func (o *ImageInspectOKBody) UnmarshalBinary(b []byte) error {
	var res ImageInspectOKBody
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*o = res
	return nil
}
