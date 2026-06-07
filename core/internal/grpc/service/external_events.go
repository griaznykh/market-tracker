package service

import (
	"fmt"
	"time"

	"google.golang.org/grpc"

	pb "lib/schema/gen/go/api/core/v1"
)

func (s *ExternalService) SubscribeOnEvents(
	req *pb.SubscribeOnEventsRequest,
	stream grpc.ServerStreamingServer[pb.SubscribeOnEventsResponse],
) error {
	ctx := stream.Context()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	sub, err := s.collector.Subscribe(ctx, req.Provider, req.Ticker)
	if err != nil {
		return fmt.Errorf("unable to subscribe: %w", err)
	}

	defer func() {
		s.collector.Unsubscribe(req.Provider, req.Ticker, sub)
	}()

	for {
		select {
		case <-ctx.Done():
			// stream is closing
			return ctx.Err()

		case data := <-sub:
			envelope := &pb.SubscribeOnEventsResponse{Event: &pb.Event{Name: pb.EventName_EVENT_NAME_TICK, Tick: &pb.Tick{Provider: data.Provider, Ticker: data.Ticker, Price: data.Price}}}

			if err := stream.Send(envelope); err != nil {
				return fmt.Errorf("unable to send event to the stream: %w", err)
			}

		case <-ticker.C:
			envelope := &pb.SubscribeOnEventsResponse{
				Event: &pb.Event{Name: pb.EventName_EVENT_NAME_KEEP_ALIVE},
			}
			if err := stream.Send(envelope); err != nil {
				return fmt.Errorf("unable to send event to the stream: %w", err)
			}
		}
	}
}
