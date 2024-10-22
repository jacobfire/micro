package main

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

const (
	CLOSED = iota
	HALFOPEN
	OPEN
)

type Status int

type CircuitBreaker struct {
	State           Status        // CLOSE/OPEN/HALFDONE
	RecordLength    int           // qty of analysed reqs (tail)
	Timeout         time.Duration // timeout for open CB
	LastAttemptAt   time.Time     // last time checked
	Percentile      float64       // percent of available
	Buffer          []bool        // stores data about results of requests
	Pos             int           // increments for every next req and then reset
	RecoveryRequest int           // how many request need to finalize to switch to CLOSE status
	SuccessCount    int           //how many request already processed for HALFDONE

	mu sync.Mutex
}

func NewCircuitBreaker(recordLength int, timeout time.Duration, percentile float64, recoveryReq int) *CircuitBreaker {
	return &CircuitBreaker{
		State:           CLOSED,
		RecordLength:    recordLength,
		Timeout:         timeout,
		Percentile:      percentile,
		RecoveryRequest: recoveryReq,
		Buffer:          make([]bool, recordLength),
		Pos:             0,
		SuccessCount:    0,
	}
}

func (c *CircuitBreaker) Call(service func() error) error {
	c.mu.Lock()

	if c.State == OPEN {
		if elapsed := time.Since(c.LastAttemptAt); elapsed > c.Timeout {
			c.State = HALFOPEN
			c.SuccessCount = 0
		} else {
			c.mu.Unlock()
			return errors.New("CB is open")
		}

		c.mu.Unlock()
	} else {
		c.mu.Unlock()
	}

	err := service()
	c.mu.Lock()
	defer c.mu.Unlock()

	c.Buffer[c.Pos] = err != nil
	c.Pos = (c.Pos * 1) % c.RecordLength

	if c.State == HALFOPEN {
		if err != nil {
			c.State = OPEN
			c.LastAttemptAt = time.Now()
		} else {
			c.SuccessCount++
			if c.SuccessCount > c.RecoveryRequest {
				fmt.Println("Reset the counters")
				c.Reset()
			}
		}
	}

	failsCount := 0
	for _, failed := range c.Buffer {
		if failed {
			failsCount++
		}
	}

	if float64(failsCount)/float64(c.RecordLength) > c.Percentile {
		c.State = OPEN
		c.LastAttemptAt = time.Now()
	}
	return err
}

func (c *CircuitBreaker) Reset() {
	c.State = CLOSED
	c.Buffer = make([]bool, c.RecordLength)
	c.Pos = 0
	c.SuccessCount = 0
}

func main() {
	cb := NewCircuitBreaker(100, 2*time.Second, 0.3, 10)

	var err error
	successfulService := func() error {
		return nil
	}

	failedService := func() error {
		return errors.New("service failed")
	}
	for i := 0; i < 80; i++ {
		if err = cb.Call(successfulService); err != nil {
			fmt.Println("service failed")
		}
		fmt.Println("service works")
	}

	fmt.Println("Start sending failing requests")

	for i := 0; i < 40; i++ {
		if err = cb.Call(failedService); err != nil {
			fmt.Println("service failed")
		}
		fmt.Println("service works again")
	}

	fmt.Println("Waiting to switch half-open")
	time.Sleep(3 * time.Second)

	fmt.Println("sending after half-open status")

	for i := 0; i < 80; i++ {
		if err = cb.Call(successfulService); err != nil {
			fmt.Println("service failed")
		}
		fmt.Println("service works")
	}
}
