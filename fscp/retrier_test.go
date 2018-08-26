package fscp

import (
	"errors"
	"testing"
	"time"
)

func TestRetrier(t *testing.T) {
	var failed error
	a := 0

	retrier := &Retrier{
		Operation: func() error {
			a++
			return nil
		},
		OnFailure: func(err error) {
			failed = err
		},
		Period: time.Millisecond,
	}

	retrier.Start()
	defer retrier.Stop()

	time.Sleep(time.Millisecond * 10)

	if b := retrier.Stop(); !b {
		t.Fatalf("true was expected")
	}

	if a < 3 {
		t.Errorf("at least 3 was expected but got %d", a)
	}

	if b := retrier.Stop(); b {
		t.Fatalf("false was expected")
	}

	if failed != nil {
		t.Errorf("expected no error but got: %s", failed)
	}
}

func TestRetrierInitialFailure(t *testing.T) {
	var failed error
	a := 0

	retrier := &Retrier{
		Operation: func() error {
			return errors.New("fail")
		},
		OnFailure: func(err error) {
			failed = err
		},
		Period: time.Millisecond,
	}

	retrier.Start()
	defer retrier.Stop()

	time.Sleep(time.Millisecond * 10)

	if b := retrier.Stop(); !b {
		t.Fatalf("true was expected")
	}

	if a != 0 {
		t.Errorf("0 was expected but got %d", a)
	}

	if b := retrier.Stop(); b {
		t.Fatalf("false was expected")
	}

	if failed == nil {
		t.Errorf("expected an error")
	}
}

func TestRetrierFailure(t *testing.T) {
	var failed error
	a := 0

	retrier := &Retrier{
		Operation: func() error {
			a++

			if a >= 2 {
				return errors.New("fail")
			}

			return nil
		},
		OnFailure: func(err error) {
			failed = err
		},
		Period: time.Millisecond,
	}

	retrier.Start()
	defer retrier.Stop()

	time.Sleep(time.Millisecond * 10)

	if b := retrier.Stop(); b {
		t.Fatalf("false was expected")
	}

	if a != 2 {
		t.Errorf("2 was expected but got %d", a)
	}

	if b := retrier.Stop(); b {
		t.Fatalf("false was expected")
	}

	if failed == nil {
		t.Errorf("expected an error")
	}
}
