package h10

import (
	"fmt"
	"github.com/siiimooon/bluetooth"
)

// Reader is a struct that represents a Polar H10 heart rate monitor.
type Reader struct {
	device *bluetooth.Device
}

// HeartRateMeasurement is a struct that represents a heart rate measurement.
type HeartRateMeasurement struct {
	hrValue                int
	sensorContact          bool
	energy                 int
	rrs                    []int
	rrsMs                  []int
	sensorContactSupported bool
	rrPresent              bool
}

func (receiver HeartRateMeasurement) GetHeartRate() int {
	return receiver.hrValue
}

func (receiver HeartRateMeasurement) GetRRIntervals() []int {
	return receiver.rrs
}

func (receiver HeartRateMeasurement) String() string {
	return fmt.Sprintf("Heart rate: %v, RR interval(s): %v", receiver.hrValue, receiver.rrs)
}

// ECGMeasurement is a struct that represents an ECG measurement.
type ECGMeasurement struct {
	timestamp uint64
	samples   []int
}

func (receiver ECGMeasurement) GetSamples() []int {
	return receiver.samples
}

func (receiver ECGMeasurement) GetTimestamp() uint64 {
	return receiver.timestamp
}

func (receiver ECGMeasurement) String() string {
	return fmt.Sprintf("Timestamp: %v, Samples: %v", receiver.GetTimestamp(), receiver.GetSamples())
}
