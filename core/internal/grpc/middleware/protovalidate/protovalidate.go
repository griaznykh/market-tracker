package protovalidate

import (
	"context"
	"errors"
	"fmt"

	"buf.build/go/protovalidate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// UnaryServerInterceptor returns a new unary server interceptor that validates incoming messages.
func UnaryServerInterceptor(validator protovalidate.Validator, opts ...option) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp interface{}, err error) {
		o := evaluateOpts(opts)
		switch msg := req.(type) {
		case proto.Message:
			if o.shouldIgnoreMessage(msg.ProtoReflect().Type()) {
				break
			}
			if err = validator.Validate(msg); err != nil {
				return nil, handleErr(err)
			}
		default:
			return nil, errors.New("unsupported message type")
		}

		return handler(ctx, req)
	}
}

// StreamServerInterceptor returns a new streaming server interceptor that validates incoming messages.
func StreamServerInterceptor(validator protovalidate.Validator, opts ...option) grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		ctx := stream.Context()

		wrapped := wrapServerStream(stream)
		wrapped.wrappedContext = ctx
		wrapped.validator = validator
		wrapped.options = evaluateOpts(opts)

		return handler(srv, wrapped)
	}
}

func (w *wrappedServerStream) RecvMsg(m interface{}) error {
	if err := w.ServerStream.RecvMsg(m); err != nil {
		return err
	}

	//nolint:errcheck
	msg := m.(proto.Message)
	if w.options.shouldIgnoreMessage(msg.ProtoReflect().Type()) {
		return nil
	}
	if err := w.validator.Validate(msg); err != nil {
		return handleErr(err)
	}

	return nil
}

func handleErr(err error) error {
	var valErr *protovalidate.ValidationError

	// if this is a validation error, add field violations
	if ok := errors.As(err, &valErr); ok {
		violations := valErr.ToProto().Violations

		st := NewErrStatus().WithCode(codes.InvalidArgument)

		for _, v := range violations {
			fp := ""
			if v.Field != nil {
				fp = protovalidate.FieldPathString(v.Field)
			}
			st.WithFieldViolation(fp, fmt.Sprintf(
				"%s [rule_id:%s]",
				func() string {
					if v.Message != nil {
						return *v.Message
					}
					return ""
				}(),
				func() string {
					if v.RuleId != nil {
						return *v.RuleId
					}
					return ""
				}(),
			))
		}
		return st.Err()
	}

	return status.Error(codes.InvalidArgument, err.Error())
}

// wrappedServerStream is a thin wrapper around grpc.ServerStream that allows modifying context.
type wrappedServerStream struct {
	grpc.ServerStream
	// wrappedContext is the wrapper's own Context. You can assign it.
	wrappedContext context.Context

	validator protovalidate.Validator
	options   *options
}

// Context returns the wrapper's WrappedContext, overwriting the nested grpc.ServerStream.Context()
func (w *wrappedServerStream) Context() context.Context {
	return w.wrappedContext
}

// wrapServerStream returns a ServerStream that has the ability to overwrite context.
func wrapServerStream(stream grpc.ServerStream) *wrappedServerStream {
	return &wrappedServerStream{ServerStream: stream, wrappedContext: stream.Context()}
}
