package protovalidate

import (
	"golang.org/x/exp/slices"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type options struct {
	ignoreMessages []protoreflect.MessageType
}

func (o *options) shouldIgnoreMessage(m protoreflect.MessageType) bool {
	return slices.ContainsFunc(o.ignoreMessages, func(t protoreflect.MessageType) bool {
		return m == t
	})
}

func evaluateOpts(opts []option) *options {
	optCopy := &options{}
	for _, o := range opts {
		o(optCopy)
	}
	return optCopy
}

type option func(*options)

// WithIgnoreMessages sets the messages that should be ignored by the validator. Use with
// caution and ensure validation is performed elsewhere.
func WithIgnoreMessages(msgs ...protoreflect.MessageType) option {
	return func(o *options) {
		o.ignoreMessages = msgs
	}
}
