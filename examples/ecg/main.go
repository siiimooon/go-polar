package main

import (
	"context"
	"fmt"

	"github.com/siiimooon/go-polar/pkg/h10"
	"tinygo.org/x/bluetooth"
)

func main() {
	sensorReader := h10.New(&bluetooth.Device{})

	sink := make(chan h10.ECGMeasurement, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		err := sensorReader.StreamECG(ctx, sink)
		if err != nil {
			panic(fmt.Errorf("error occured during ecg streaming from Polar H10: %w", err))
		}
	}()

	for {
		select {
		case measurement := <-sink:
			fmt.Println(measurement.GetSamples())
		}
	}
}
