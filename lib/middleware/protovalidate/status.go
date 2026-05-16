package protovalidate

import (
	"errors"
	"fmt"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/protoadapt"
)

// StatusFromError retrieves the *status.Status from the provided error.  It'll
// attempt to unwrap the *status.Error, which is something status.FromError
// does not do.
//
// code taken from https://github.com/hashicorp/boundary/blob/main/internal/daemon/metric/instrument_handlers.go#L167-L181
func StatusFromError(err error) (*status.Status, bool) {
	if s, ok := status.FromError(err); ok {
		return s, true
	}

	type gRPCStatus interface {
		GRPCStatus() *status.Status
	}
	var unwrappedStatus gRPCStatus
	if ok := errors.As(err, &unwrappedStatus); ok {
		return unwrappedStatus.GRPCStatus(), true
	}

	return nil, false
}

// StatusWithErrDetails is a shorthand for adding error details to the gRPC status message.
// The method panics on unexpected error.
func StatusWithErrDetails(st *status.Status, details ...proto.Message) *status.Status {
	if len(details) == 0 {
		return st
	}

	var err error

	// https://github.com/grpc/grpc-go/issues/5316
	adapted := make([]protoadapt.MessageV1, 0, len(details))

	for _, d := range details {
		adapted = append(adapted, protoadapt.MessageV1Of(d))
	}

	st, err = st.WithDetails(adapted...)
	if err != nil {
		panic(fmt.Sprintf("unexpected error attaching metadata: %v", err))
	}

	return st
}

// ErrStatus is a handy wrapper for constructing grpc error status
// with error details.
type ErrStatus struct {
	st *status.Status

	resources         []*errdetails.ResourceInfo
	errInfo           []*errdetails.ErrorInfo
	fieldViolations   []*errdetails.BadRequest_FieldViolation
	preCondViolations []*errdetails.PreconditionFailure_Violation
	quotaViolations   []*errdetails.QuotaFailure_Violation
}

// NewErrStatus returns new instance of ErrStatus.
func NewErrStatus() *ErrStatus {
	return &ErrStatus{}
}

// GRPCStatus converts collected data to a valid gRPC status.
func (es *ErrStatus) GRPCStatus() *status.Status {
	if es.st == nil {
		es.st = status.New(codes.Unknown, codes.Unknown.String())
	}
	if len(es.st.Details()) == 0 {
		// add details

		var details []proto.Message

		for _, d := range es.resources {
			details = append(details, d)
		}
		for _, d := range es.errInfo {
			details = append(details, d)
		}
		if len(es.fieldViolations) > 0 {
			details = append(details, &errdetails.BadRequest{FieldViolations: es.fieldViolations})
		}
		if len(es.preCondViolations) > 0 {
			details = append(details, &errdetails.PreconditionFailure{Violations: es.preCondViolations})
		}
		if len(es.quotaViolations) > 0 {
			details = append(details, &errdetails.QuotaFailure{Violations: es.quotaViolations})
		}

		es.st = StatusWithErrDetails(es.st, details...)
	}

	return es.st
}

// Err converts collected data to gRPC status error.
func (es *ErrStatus) Err() error {
	return es.GRPCStatus().Err()
}

// HasFieldViolations returns true if collected data contains
// field violations.
func (es *ErrStatus) HasFieldViolations() bool {
	return len(es.fieldViolations) > 0
}

// HasPreconditionViolations returns true if collected data contains
// precondition violations.
func (es *ErrStatus) HasPreconditionViolations() bool {
	return len(es.preCondViolations) > 0
}

// HasQuotaViolations returns true if collected data contains
// quota violations.
func (es *ErrStatus) HasQuotaViolations() bool {
	return len(es.quotaViolations) > 0
}

// HasViolations returns true if collected data contains any violations.
func (es *ErrStatus) HasViolations() bool {
	return es.HasFieldViolations() ||
		es.HasPreconditionViolations() ||
		es.HasQuotaViolations()
}

// WithCode sets the given status code and its default message as a base gRPC
// status.
func (es *ErrStatus) WithCode(code codes.Code) *ErrStatus {
	es.st = status.New(code, code.String())
	return es
}

// WithCodeMsg sets the given status code and message as a base gRPC status.
func (es *ErrStatus) WithCodeMsg(code codes.Code, msg string) *ErrStatus {
	es.st = status.New(code, msg)
	return es
}

// ErrWithCode same as WithCode but also converts the status to error.
func (es *ErrStatus) ErrWithCode(code codes.Code) error {
	return es.WithCode(code).Err()
}

// ErrWithCodeMsg same as WithCodeMsg but also converts the status to error.
func (es *ErrStatus) ErrWithCodeMsg(code codes.Code, msg string) error {
	return es.WithCodeMsg(code, msg).Err()
}

// WithErrorInfo adds error info to the status.
func (es *ErrStatus) WithErrorInfo(reason, domain string, md map[string]string) *ErrStatus {
	es.errInfo = append(es.errInfo, &errdetails.ErrorInfo{
		Reason:   reason,
		Domain:   domain,
		Metadata: md,
	})
	return es
}

// WithResourceInfo adds the given ResourceInfo to the status.
func (es *ErrStatus) WithResourceInfo(info *errdetails.ResourceInfo) *ErrStatus {
	es.resources = append(es.resources, info)
	return es
}

// WithResource is a convenience method for adding resource type and name as
// ResourceInfo to the status.
func (es *ErrStatus) WithResource(resType, resName string) *ErrStatus {
	es.resources = append(es.resources, &errdetails.ResourceInfo{
		ResourceType: resType,
		ResourceName: resName,
	})
	return es
}

// WithFieldViolation adds field violation to the status.
func (es *ErrStatus) WithFieldViolation(field, desc string) *ErrStatus {
	es.fieldViolations = append(es.fieldViolations, &errdetails.BadRequest_FieldViolation{
		Field:       field,
		Description: desc,
	})
	return es
}

// WithPreconditionViolation adds precondition violation to the status.
func (es *ErrStatus) WithPreconditionViolation(vType, subj, desc string) *ErrStatus {
	es.preCondViolations = append(es.preCondViolations, &errdetails.PreconditionFailure_Violation{
		Type:        vType,
		Subject:     subj,
		Description: desc,
	})
	return es
}

// WithQuotaViolation adds quota violation to the status.
func (es *ErrStatus) WithQuotaViolation(subj, desc string) *ErrStatus {
	es.quotaViolations = append(es.quotaViolations, &errdetails.QuotaFailure_Violation{
		Subject:     subj,
		Description: desc,
	})
	return es
}
