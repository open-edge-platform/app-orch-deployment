// Code generated by protoc-gen-validate. DO NOT EDIT.
// source: resource/v2/endpoint_resource.proto

package resourcev2

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"google.golang.org/protobuf/types/known/anypb"
)

// ensure the imports are used
var (
	_ = bytes.MinRead
	_ = errors.New("")
	_ = fmt.Print
	_ = utf8.UTFMax
	_ = (*regexp.Regexp)(nil)
	_ = (*strings.Reader)(nil)
	_ = net.IPv4len
	_ = time.Duration(0)
	_ = (*url.URL)(nil)
	_ = (*mail.Address)(nil)
	_ = anypb.Any{}
	_ = sort.Sort
)

// define the regex for a UUID once up-front
var _endpoint_resource_uuidPattern = regexp.MustCompile("^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$")

// Validate checks the field values on AppEndpoint with the rules defined in
// the proto definition for this message. If any rules are violated, the first
// error encountered is returned, or nil if there are no violations.
func (m *AppEndpoint) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on AppEndpoint with the rules defined in
// the proto definition for this message. If any rules are violated, the
// result is a list of violation errors wrapped in AppEndpointMultiError, or
// nil if none found.
func (m *AppEndpoint) ValidateAll() error {
	return m.validate(true)
}

func (m *AppEndpoint) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	if err := m._validateUuid(m.GetId()); err != nil {
		err = AppEndpointValidationError{
			field:  "Id",
			reason: "value must be a valid UUID",
			cause:  err,
		}
		if !all {
			return err
		}
		errors = append(errors, err)
	}

	if l := utf8.RuneCountInString(m.GetName()); l < 1 || l > 40 {
		err := AppEndpointValidationError{
			field:  "Name",
			reason: "value length must be between 1 and 40 runes, inclusive",
		}
		if !all {
			return err
		}
		errors = append(errors, err)
	}

	if !_AppEndpoint_Name_Pattern.MatchString(m.GetName()) {
		err := AppEndpointValidationError{
			field:  "Name",
			reason: "value does not match regex pattern \"^[a-z0-9][a-z0-9-]{0,38}[a-z0-9]{0,1}$\"",
		}
		if !all {
			return err
		}
		errors = append(errors, err)
	}

	for idx, item := range m.GetFqdns() {
		_, _ = idx, item

		if all {
			switch v := interface{}(item).(type) {
			case interface{ ValidateAll() error }:
				if err := v.ValidateAll(); err != nil {
					errors = append(errors, AppEndpointValidationError{
						field:  fmt.Sprintf("Fqdns[%v]", idx),
						reason: "embedded message failed validation",
						cause:  err,
					})
				}
			case interface{ Validate() error }:
				if err := v.Validate(); err != nil {
					errors = append(errors, AppEndpointValidationError{
						field:  fmt.Sprintf("Fqdns[%v]", idx),
						reason: "embedded message failed validation",
						cause:  err,
					})
				}
			}
		} else if v, ok := interface{}(item).(interface{ Validate() error }); ok {
			if err := v.Validate(); err != nil {
				return AppEndpointValidationError{
					field:  fmt.Sprintf("Fqdns[%v]", idx),
					reason: "embedded message failed validation",
					cause:  err,
				}
			}
		}

	}

	for idx, item := range m.GetPorts() {
		_, _ = idx, item

		if all {
			switch v := interface{}(item).(type) {
			case interface{ ValidateAll() error }:
				if err := v.ValidateAll(); err != nil {
					errors = append(errors, AppEndpointValidationError{
						field:  fmt.Sprintf("Ports[%v]", idx),
						reason: "embedded message failed validation",
						cause:  err,
					})
				}
			case interface{ Validate() error }:
				if err := v.Validate(); err != nil {
					errors = append(errors, AppEndpointValidationError{
						field:  fmt.Sprintf("Ports[%v]", idx),
						reason: "embedded message failed validation",
						cause:  err,
					})
				}
			}
		} else if v, ok := interface{}(item).(interface{ Validate() error }); ok {
			if err := v.Validate(); err != nil {
				return AppEndpointValidationError{
					field:  fmt.Sprintf("Ports[%v]", idx),
					reason: "embedded message failed validation",
					cause:  err,
				}
			}
		}

	}

	if all {
		switch v := interface{}(m.GetEndpointStatus()).(type) {
		case interface{ ValidateAll() error }:
			if err := v.ValidateAll(); err != nil {
				errors = append(errors, AppEndpointValidationError{
					field:  "EndpointStatus",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		case interface{ Validate() error }:
			if err := v.Validate(); err != nil {
				errors = append(errors, AppEndpointValidationError{
					field:  "EndpointStatus",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		}
	} else if v, ok := interface{}(m.GetEndpointStatus()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return AppEndpointValidationError{
				field:  "EndpointStatus",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	if len(errors) > 0 {
		return AppEndpointMultiError(errors)
	}

	return nil
}

func (m *AppEndpoint) _validateUuid(uuid string) error {
	if matched := _endpoint_resource_uuidPattern.MatchString(uuid); !matched {
		return errors.New("invalid uuid format")
	}

	return nil
}

// AppEndpointMultiError is an error wrapping multiple validation errors
// returned by AppEndpoint.ValidateAll() if the designated constraints aren't met.
type AppEndpointMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m AppEndpointMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m AppEndpointMultiError) AllErrors() []error { return m }

// AppEndpointValidationError is the validation error returned by
// AppEndpoint.Validate if the designated constraints aren't met.
type AppEndpointValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e AppEndpointValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e AppEndpointValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e AppEndpointValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e AppEndpointValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e AppEndpointValidationError) ErrorName() string { return "AppEndpointValidationError" }

// Error satisfies the builtin error interface
func (e AppEndpointValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sAppEndpoint.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = AppEndpointValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = AppEndpointValidationError{}

var _AppEndpoint_Name_Pattern = regexp.MustCompile("^[a-z0-9][a-z0-9-]{0,38}[a-z0-9]{0,1}$")

// Validate checks the field values on Fqdn with the rules defined in the proto
// definition for this message. If any rules are violated, the first error
// encountered is returned, or nil if there are no violations.
func (m *Fqdn) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on Fqdn with the rules defined in the
// proto definition for this message. If any rules are violated, the result is
// a list of violation errors wrapped in FqdnMultiError, or nil if none found.
func (m *Fqdn) ValidateAll() error {
	return m.validate(true)
}

func (m *Fqdn) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	if err := m._validateHostname(m.GetFqdn()); err != nil {
		err = FqdnValidationError{
			field:  "Fqdn",
			reason: "value must be a valid hostname",
			cause:  err,
		}
		if !all {
			return err
		}
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return FqdnMultiError(errors)
	}

	return nil
}

func (m *Fqdn) _validateHostname(host string) error {
	s := strings.ToLower(strings.TrimSuffix(host, "."))

	if len(host) > 253 {
		return errors.New("hostname cannot exceed 253 characters")
	}

	for _, part := range strings.Split(s, ".") {
		if l := len(part); l == 0 || l > 63 {
			return errors.New("hostname part must be non-empty and cannot exceed 63 characters")
		}

		if part[0] == '-' {
			return errors.New("hostname parts cannot begin with hyphens")
		}

		if part[len(part)-1] == '-' {
			return errors.New("hostname parts cannot end with hyphens")
		}

		for _, r := range part {
			if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
				return fmt.Errorf("hostname parts can only contain alphanumeric characters or hyphens, got %q", string(r))
			}
		}
	}

	return nil
}

// FqdnMultiError is an error wrapping multiple validation errors returned by
// Fqdn.ValidateAll() if the designated constraints aren't met.
type FqdnMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m FqdnMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m FqdnMultiError) AllErrors() []error { return m }

// FqdnValidationError is the validation error returned by Fqdn.Validate if the
// designated constraints aren't met.
type FqdnValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e FqdnValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e FqdnValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e FqdnValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e FqdnValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e FqdnValidationError) ErrorName() string { return "FqdnValidationError" }

// Error satisfies the builtin error interface
func (e FqdnValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sFqdn.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = FqdnValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = FqdnValidationError{}

// Validate checks the field values on Port with the rules defined in the proto
// definition for this message. If any rules are violated, the first error
// encountered is returned, or nil if there are no violations.
func (m *Port) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on Port with the rules defined in the
// proto definition for this message. If any rules are violated, the result is
// a list of violation errors wrapped in PortMultiError, or nil if none found.
func (m *Port) ValidateAll() error {
	return m.validate(true)
}

func (m *Port) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	if l := utf8.RuneCountInString(m.GetName()); l < 1 || l > 40 {
		err := PortValidationError{
			field:  "Name",
			reason: "value length must be between 1 and 40 runes, inclusive",
		}
		if !all {
			return err
		}
		errors = append(errors, err)
	}

	if !_Port_Name_Pattern.MatchString(m.GetName()) {
		err := PortValidationError{
			field:  "Name",
			reason: "value does not match regex pattern \"^[a-z0-9][a-z0-9-]{0,38}[a-z0-9]{0,1}$\"",
		}
		if !all {
			return err
		}
		errors = append(errors, err)
	}

	// no validation rules for Value

	// no validation rules for Protocol

	// no validation rules for ServiceProxyUrl

	if len(errors) > 0 {
		return PortMultiError(errors)
	}

	return nil
}

// PortMultiError is an error wrapping multiple validation errors returned by
// Port.ValidateAll() if the designated constraints aren't met.
type PortMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m PortMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m PortMultiError) AllErrors() []error { return m }

// PortValidationError is the validation error returned by Port.Validate if the
// designated constraints aren't met.
type PortValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e PortValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e PortValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e PortValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e PortValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e PortValidationError) ErrorName() string { return "PortValidationError" }

// Error satisfies the builtin error interface
func (e PortValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sPort.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = PortValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = PortValidationError{}

var _Port_Name_Pattern = regexp.MustCompile("^[a-z0-9][a-z0-9-]{0,38}[a-z0-9]{0,1}$")

// Validate checks the field values on EndpointStatus with the rules defined in
// the proto definition for this message. If any rules are violated, the first
// error encountered is returned, or nil if there are no violations.
func (m *EndpointStatus) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on EndpointStatus with the rules defined
// in the proto definition for this message. If any rules are violated, the
// result is a list of violation errors wrapped in EndpointStatusMultiError,
// or nil if none found.
func (m *EndpointStatus) ValidateAll() error {
	return m.validate(true)
}

func (m *EndpointStatus) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	// no validation rules for State

	if len(errors) > 0 {
		return EndpointStatusMultiError(errors)
	}

	return nil
}

// EndpointStatusMultiError is an error wrapping multiple validation errors
// returned by EndpointStatus.ValidateAll() if the designated constraints
// aren't met.
type EndpointStatusMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m EndpointStatusMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m EndpointStatusMultiError) AllErrors() []error { return m }

// EndpointStatusValidationError is the validation error returned by
// EndpointStatus.Validate if the designated constraints aren't met.
type EndpointStatusValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e EndpointStatusValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e EndpointStatusValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e EndpointStatusValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e EndpointStatusValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e EndpointStatusValidationError) ErrorName() string { return "EndpointStatusValidationError" }

// Error satisfies the builtin error interface
func (e EndpointStatusValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sEndpointStatus.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = EndpointStatusValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = EndpointStatusValidationError{}
