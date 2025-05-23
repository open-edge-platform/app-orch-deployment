// Code generated by go-swagger; DO NOT EDIT.

package volumes

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"
	"net/http"
	"time"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	cr "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"

	"github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/skupper/client/generated/libpod/models"
)

// NewVolumeCreateLibpodParams creates a new VolumeCreateLibpodParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewVolumeCreateLibpodParams() *VolumeCreateLibpodParams {
	return &VolumeCreateLibpodParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewVolumeCreateLibpodParamsWithTimeout creates a new VolumeCreateLibpodParams object
// with the ability to set a timeout on a request.
func NewVolumeCreateLibpodParamsWithTimeout(timeout time.Duration) *VolumeCreateLibpodParams {
	return &VolumeCreateLibpodParams{
		timeout: timeout,
	}
}

// NewVolumeCreateLibpodParamsWithContext creates a new VolumeCreateLibpodParams object
// with the ability to set a context for a request.
func NewVolumeCreateLibpodParamsWithContext(ctx context.Context) *VolumeCreateLibpodParams {
	return &VolumeCreateLibpodParams{
		Context: ctx,
	}
}

// NewVolumeCreateLibpodParamsWithHTTPClient creates a new VolumeCreateLibpodParams object
// with the ability to set a custom HTTPClient for a request.
func NewVolumeCreateLibpodParamsWithHTTPClient(client *http.Client) *VolumeCreateLibpodParams {
	return &VolumeCreateLibpodParams{
		HTTPClient: client,
	}
}

/*
VolumeCreateLibpodParams contains all the parameters to send to the API endpoint

	for the volume create libpod operation.

	Typically these are written to a http.Request.
*/
type VolumeCreateLibpodParams struct {

	/* Create.

	   attributes for creating a volume
	*/
	Create *models.VolumeCreateOptions

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the volume create libpod params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *VolumeCreateLibpodParams) WithDefaults() *VolumeCreateLibpodParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the volume create libpod params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *VolumeCreateLibpodParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the volume create libpod params
func (o *VolumeCreateLibpodParams) WithTimeout(timeout time.Duration) *VolumeCreateLibpodParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the volume create libpod params
func (o *VolumeCreateLibpodParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the volume create libpod params
func (o *VolumeCreateLibpodParams) WithContext(ctx context.Context) *VolumeCreateLibpodParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the volume create libpod params
func (o *VolumeCreateLibpodParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the volume create libpod params
func (o *VolumeCreateLibpodParams) WithHTTPClient(client *http.Client) *VolumeCreateLibpodParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the volume create libpod params
func (o *VolumeCreateLibpodParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithCreate adds the create to the volume create libpod params
func (o *VolumeCreateLibpodParams) WithCreate(create *models.VolumeCreateOptions) *VolumeCreateLibpodParams {
	o.SetCreate(create)
	return o
}

// SetCreate adds the create to the volume create libpod params
func (o *VolumeCreateLibpodParams) SetCreate(create *models.VolumeCreateOptions) {
	o.Create = create
}

// WriteToRequest writes these params to a swagger request
func (o *VolumeCreateLibpodParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error
	if o.Create != nil {
		if err := r.SetBodyParam(o.Create); err != nil {
			return err
		}
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
