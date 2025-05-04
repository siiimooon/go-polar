package h10

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
)

// decodeHeartRateData decodes the heart rate payload from the Polar H10.
func decodeHeartRateData(data []byte) HeartRateMeasurement {
	mapRr1024ToRrMs := func(rrsRaw int) int {
		return int(float64(rrsRaw) / 1024.0 * 1000.0)
	}

	hrFormat := int(data[0]) & 0x01
	sensorContact := int(data[0])&0x06>>1 == 0x03
	contactSupported := int(data[0])&0x04 != 0
	energyExpended := int(data[0]) & 0x08 >> 3
	rrPresent := int(data[0]) & 0x10 >> 4
	var hrValue int
	if hrFormat == 1 {
		hrValue = int(data[1])&0xFF + (int(data[2]) << 8)
	} else {
		hrValue = int(data[1]) & 0x000000FF
	}
	offset := hrFormat + 2
	energy := 0
	if energyExpended == 1 {
		energy = (int(data[offset]) & 0xFF) + (int(data[offset+1]) & 0xFF << 8)
		offset += 2
	}
	rrs := make([]int, 0)
	rrsMs := make([]int, 0)
	if rrPresent == 1 {
		dataLen := len(data)
		for offset < dataLen {
			rrValue := (int(data[offset]) & 0xFF) + (int(data[offset+1]) & 0xFF << 8)
			offset += 2
			rrs = append(rrs, rrValue)
			rrsMs = append(rrsMs, mapRr1024ToRrMs(rrValue))
		}
	}
	return HeartRateMeasurement{
		hrValue:                hrValue,
		sensorContact:          sensorContact,
		energy:                 energy,
		rrs:                    rrs,
		rrsMs:                  rrsMs,
		sensorContactSupported: contactSupported,
		rrPresent:              rrPresent == 1,
	}
}

// decodeECGData decodes the ECG payload from the Polar H10.
func decodeECGData(data []byte) (ECGMeasurement, error) {
	offset := ECG_SAMPLING_STEP

	sampleType := data[0]
	sampleIsECG := sampleType == 0x00
	frameType := data[9]
	frameTypeIsEcg := frameType == 0x00
	sampleTimestampBytes := data[1:9]
	sampleTimestamp := binary.LittleEndian.Uint64(sampleTimestampBytes) / 1e6
	samples := data[10:]
	magnitudeOfThree := len(samples)%3 == 0
	if !sampleIsECG {
		return ECGMeasurement{}, fmt.Errorf("expected sample type ecg: %v", sampleType)
	}
	if !magnitudeOfThree {
		return ECGMeasurement{}, fmt.Errorf("number of samples not a factor of 3: %v", len(samples)%3)
	}
	if !frameTypeIsEcg {
		return ECGMeasurement{}, fmt.Errorf("expected frame type ecg: %v", frameType)
	}

	ecgSamples := make([]int, 0)
	for offset < len(samples) {
		sampleBytes := samples[offset : offset+ECG_SAMPLING_STEP]
		offset += ECG_SAMPLING_STEP
		sample := int(int32(sampleBytes[0]) | int32(sampleBytes[1])<<8 | int32(int8(sampleBytes[2]))<<16)
		ecgSamples = append(ecgSamples, sample)
	}

	return ECGMeasurement{
		timestamp: sampleTimestamp,
		samples:   ecgSamples,
	}, nil
}

func suppressCancellationError(err error) error {
	if errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}
