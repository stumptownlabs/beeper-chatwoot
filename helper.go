package main

import (
	_ "strconv"
	"time"

	"github.com/sethvargo/go-retry"
	log "github.com/sirupsen/logrus"
)

func DoRetry(description string, fn func() (interface{}, error)) (interface{}, error) {
	var err error
	b, err := retry.NewFibonacci(1 * time.Second)
	if err != nil {
		panic(err)
	}
	b = retry.WithMaxRetries(5, b)
	for {
		log.Info("trying: ", description)
		var val interface{}
		val, err = fn()
		if err == nil {
			log.Info(description, " succeeded")
			return val, nil
		}
		nextDuration, stop := b.Next()
		log.Debugf("  %s failed. Retrying in %f seconds. Error: %+v", description, nextDuration.Seconds(), err)
		if stop {
			log.Debugf("  %s failed. Retry limit reached. Will not retry.", description)
			break
		}
		time.Sleep(nextDuration)
	}
	return nil, err
}
