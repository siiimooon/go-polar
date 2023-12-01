package main

import (
	"context"
	"fmt"
	"github.com/siiimooon/bluetooth"
	"github.com/siiimooon/go-polar/pkg/h10"
)

func main() {
	sensorReader := h10.New(&bluetooth.Device{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sink := make(chan h10.HeartRateMeasurement, 1)
	go func() {
		err := sensorReader.StreamHeartRate(ctx, sink)
		if err != nil {
			panic(fmt.Errorf("failed at streaming hr: %w", err))
		}
	}()

	for {
		select {
		case measurement := <-sink:
			fmt.Println(measurement.String())
		}
	}
}
