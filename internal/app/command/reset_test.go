package command

import (
	"io"
	"testing"
)

func TestRunWithConcurrencyLimitKeepsFiveRemovalsActive(t *testing.T) {
	started := make(chan int, 6)
	releases := make([]chan struct{}, 6)
	for index := range releases {
		releases[index] = make(chan struct{})
	}
	result := make(chan error, 1)

	go func() {
		_, err := runWithConcurrencyLimit([]int{0, 1, 2, 3, 4, 5}, resetRemoveConcurrency, func(item int, _ io.Writer, _ io.Writer) error {
			started <- item
			<-releases[item]
			return nil
		}, io.Discard, io.Discard)
		result <- err
	}()

	for range resetRemoveConcurrency {
		<-started
	}
	select {
	case item := <-started:
		t.Fatalf("operation %d started before a slot was freed", item)
	default:
	}
	releases[0] <- struct{}{}
	if item := <-started; item != 5 {
		t.Fatalf("started item = %d, want 5", item)
	}
	for index := 1; index < len(releases); index++ {
		releases[index] <- struct{}{}
	}
	if err := <-result; err != nil {
		t.Fatalf("runWithConcurrencyLimit() error = %v", err)
	}
}
