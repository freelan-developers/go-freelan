package fscp

import (
	"sync"
	"time"
)

// A Retrier retries a given operation until it is satisfied.
type Retrier struct {
	Operation func() error
	OnFailure func(error)
	Period    time.Duration
	once      sync.Once
	closed    chan struct{}
}

// Start the retrier.
func (r *Retrier) Start() {
	r.closed = make(chan struct{})

	if err := r.Operation(); err != nil {
		r.OnFailure(err)
		return
	}

	timer := time.NewTimer(r.Period)

	go func() {
		defer timer.Stop()

		for {
			select {
			case <-r.closed:
				if !timer.Stop() {
					<-timer.C
					return
				}
			case <-timer.C:
				if err := r.Operation(); err != nil {
					r.OnFailure(err)
					r.Stop()
					continue
				}

				timer.Reset(r.Period)
			}
		}
	}()
}

// Stop the retrier.
func (r *Retrier) Stop() bool {
	closed := false

	r.once.Do(func() {
		close(r.closed)
		closed = true
	})

	return closed
}
