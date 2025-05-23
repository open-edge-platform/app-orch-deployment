// Code generated by go-swagger; DO NOT EDIT.

package manifests

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

// NewManifestAddLibpodParams creates a new ManifestAddLibpodParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewManifestAddLibpodParams() *ManifestAddLibpodParams {
	return &ManifestAddLibpodParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewManifestAddLibpodParamsWithTimeout creates a new ManifestAddLibpodParams object
// with the ability to set a timeout on a request.
func NewManifestAddLibpodParamsWithTimeout(timeout time.Duration) *ManifestAddLibpodParams {
	return &ManifestAddLibpodParams{
		timeout: timeout,
	}
}

// NewManifestAddLibpodParamsWithContext creates a new ManifestAddLibpodParams object
// with the ability to set a context for a request.
func NewManifestAddLibpodParamsWithContext(ctx context.Context) *ManifestAddLibpodParams {
	return &ManifestAddLibpodParams{
		Context: ctx,
	}
}

// NewManifestAddLibpodParamsWithHTTPClient creates a new ManifestAddLibpodParams object
// with the ability to set a custom HTTPClient for a request.
func NewManifestAddLibpodParamsWithHTTPClient(client *http.Client) *ManifestAddLibpodParams {
	return &ManifestAddLibpodParams{
		HTTPClient: client,
	}
}

/*
ManifestAddLibpodParams contains all the parameters to send to the API endpoint

	for the manifest add libpod operation.

	Typically these are written to a http.Request.
*/
type ManifestAddLibpodParams struct {

	/* Name.

	   the name or ID of the manifest
	*/
	Name string

	/* Options.

	   options for creating a manifest
	*/
	Options *models.ManifestAddOptions

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the manifest add libpod params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ManifestAddLibpodParams) WithDefaults() *ManifestAddLibpodParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the manifest add libpod params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ManifestAddLibpodParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the manifest add libpod params
func (o *ManifestAddLibpodParams) WithTimeout(timeout time.Duration) *ManifestAddLibpodParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the manifest add libpod params
func (o *ManifestAddLibpodParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the manifest add libpod params
func (o *ManifestAddLibpodParams) WithContext(ctx context.Context) *ManifestAddLibpodParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the manifest add libpod params
func (o *ManifestAddLibpodParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the manifest add libpod params
func (o *ManifestAddLibpodParams) WithHTTPClient(client *http.Client) *ManifestAddLibpodParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the manifest add libpod params
func (o *ManifestAddLibpodParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithName adds the name to the manifest add libpod params
func (o *ManifestAddLibpodParams) WithName(name string) *ManifestAddLibpodParams {
	o.SetName(name)
	return o
}

// SetName adds the name to the manifest add libpod params
func (o *ManifestAddLibpodParams) SetName(name string) {
	o.Name = name
}

// WithOptions adds the options to the manifest add libpod params
func (o *ManifestAddLibpodParams) WithOptions(options *models.ManifestAddOptions) *ManifestAddLibpodParams {
	o.SetOptions(options)
	return o
}

// SetOptions adds the options to the manifest add libpod params
func (o *ManifestAddLibpodParams) SetOptions(options *models.ManifestAddOptions) {
	o.Options = options
}

// WriteToRequest writes these params to a swagger request
func (o *ManifestAddLibpodParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	// path param name
	if err := r.SetPathParam("name", o.Name); err != nil {
		return err
	}
	if o.Options != nil {
		if err := r.SetBodyParam(o.Options); err != nil {
			return err
		}
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
