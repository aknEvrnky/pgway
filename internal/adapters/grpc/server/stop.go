package server

import (
	"time"

	"google.golang.org/grpc"
)

// GracefulStopWithTimeout runs GracefulStop, then Force-stops if streams or
// RPCs have not drained within timeout (e.g. stuck clients).
func GracefulStopWithTimeout(s *grpc.Server, timeout time.Duration) {
	done := make(chan struct{})
	go func() {
		s.GracefulStop()
		close(done)
	}()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-done:
	case <-timer.C:
		s.Stop()
		<-done
	}
}
