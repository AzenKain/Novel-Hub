package services

import (
	"context"
	"sync"
	"testing"

	"novelhub/pkg/cache"
	"novelhub/pkg/constants"
)

// Verify reads the attempt counter, compares, then writes it back.
func TestOTPBruteForceCannotOutrunTheAttemptLimit(t *testing.T) {
	store := NewOTPStore(cache.NewRamCache())
	ctx := context.Background()
	code, err := store.Issue(ctx, OTPPurposePasswordReset, "victim@example.com")
	if err != nil {
		t.Fatal(err)
	}

	const guesses = 200
	var start, done sync.WaitGroup
	start.Add(1)
	accepted := 0
	var mu sync.Mutex
	for i := range guesses {
		done.Add(1)
		go func(i int) {
			defer done.Done()
			start.Wait()
			wrong := "1" + string(rune('0'+i%10)) + "0000"
			if wrong == code {
				return
			}
			if _, err := store.Verify(ctx, OTPPurposePasswordReset, "victim@example.com", wrong); err == nil {
				mu.Lock()
				accepted++
				mu.Unlock()
			}
		}(i)
	}
	start.Done()
	done.Wait()

	if accepted > 0 {
		t.Fatalf("%d wrong codes were accepted", accepted)
	}
	if _, err := store.Verify(ctx, OTPPurposePasswordReset, "victim@example.com", code); err == nil {
		t.Fatalf("the correct code still worked after %d wrong guesses against a limit of %d",
			guesses, constants.OTPMaxAttempts)
	}
}

// The same read-compare-write window lets every concurrent holder of a correct code past the digest check before any of them deletes it, so one emailed code mints one ticket per racer and each ticket is independently good for a password reset.
func TestOneCodeMintsExactlyOneTicket(t *testing.T) {
	store := NewOTPStore(cache.NewRamCache())
	ctx := context.Background()
	code, err := store.Issue(ctx, OTPPurposePasswordReset, "victim@example.com")
	if err != nil {
		t.Fatal(err)
	}

	const racers = 64
	var start, done sync.WaitGroup
	start.Add(1)
	tickets := make([]string, racers)
	for i := range racers {
		done.Add(1)
		go func(i int) {
			defer done.Done()
			start.Wait()
			if ticket, err := store.Verify(ctx, OTPPurposePasswordReset, "victim@example.com", code); err == nil {
				tickets[i] = ticket
			}
		}(i)
	}
	start.Done()
	done.Wait()

	unique := make(map[string]bool)
	for _, ticket := range tickets {
		if ticket != "" {
			unique[ticket] = true
		}
	}
	if len(unique) != 1 {
		t.Fatalf("one code minted %d distinct reset tickets, want 1", len(unique))
	}
}

// Consume is a read-then-delete, so concurrent resets on one ticket all see it present and all proceed.
func TestOneTicketIsConsumedOnce(t *testing.T) {
	ctx := context.Background()
	const (
		rounds = 300
		racers = 64
	)
	extra := 0

	for range rounds {
		store := NewOTPStore(cache.NewRamCache())
		code, err := store.Issue(ctx, OTPPurposePasswordReset, "victim@example.com")
		if err != nil {
			t.Fatal(err)
		}
		ticket, err := store.Verify(ctx, OTPPurposePasswordReset, "victim@example.com", code)
		if err != nil {
			t.Fatal(err)
		}

		var start, done sync.WaitGroup
		start.Add(1)
		consumed := 0
		var mu sync.Mutex
		for range racers {
			done.Add(1)
			go func() {
				defer done.Done()
				start.Wait()
				if err := store.Consume(ctx, OTPPurposePasswordReset, "victim@example.com", ticket); err == nil {
					mu.Lock()
					consumed++
					mu.Unlock()
				}
			}()
		}
		start.Done()
		done.Wait()

		if consumed != 1 {
			extra += consumed - 1
		}
	}

	if extra != 0 {
		t.Fatalf("%d extra consumes across %d rounds — one ticket resets the password more than once", extra, rounds)
	}
}
