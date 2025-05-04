package h10

import (
	"context"
	"errors"
	"fmt"
	"time"

	"tinygo.org/x/bluetooth"
)

var ErrUnbufferedChannel = errors.New("unbuffered channels are not supported")

// New creates a new Reader from the provided device - which is expected to be a Polar H10 device.
// The device must be connected before creating a new Reader.
// If not a Polar H10 device, the behaviour is undefined.
func New(device *bluetooth.Device) Reader {
	return Reader{
		device: device,
	}
}

// Disconnect disconnects the device.
func (receiver Reader) Disconnect() error {
	return receiver.device.Disconnect()
}

// GetBatteryLevel retrieves the battery level (percentage) of the device.
func (receiver Reader) GetBatteryLevel() (int, error) {
	batteryService, _ := bluetooth.ParseUUID(BATTERY_SERVICE)
	batteryLevelCharacteristic, _ := bluetooth.ParseUUID(BATTERY_CHARACTERISTIC_LEVEL)
	characteristic, err := receiver.retrieveDeviceCharacteristic(batteryService, batteryLevelCharacteristic)
	if err != nil {
		return 0, fmt.Errorf("failed to retrieve device character: %w", err)
	}

	characteristicResponse, err := receiver.readCharacteristic(characteristic)
	if err != nil {
		return 0, fmt.Errorf("failed at reading characteristic: %w", err)
	}
	return int(characteristicResponse[0]), nil
}

// StreamHeartRate streams heart rate data from the device to the provided channel.
func (receiver Reader) StreamHeartRate(ctx context.Context, sink chan HeartRateMeasurement) error {
	if cap(sink) == 0 {
		return ErrUnbufferedChannel
	}

	hrService, _ := bluetooth.ParseUUID(HR_SERVICES)
	hrCharacteristic, _ := bluetooth.ParseUUID(HR_CHARACTERISTIC_MEASUREMENT)

	deviceCharacteristic, err := receiver.retrieveDeviceCharacteristic(hrService, hrCharacteristic)
	if err != nil {
		return fmt.Errorf("failed to retrieve device characteristic: %w", err)
	}
	hrStream := make(chan []byte, 1)

	streamCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	err = streamNotification(streamCtx, deviceCharacteristic, hrStream)

	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return suppressCancellationError(ctx.Err())
		case rawMeasurement := <-hrStream:
			hrMeasurement := decodeHeartRateData(rawMeasurement)
			sink <- hrMeasurement
		}
	}
}

// StreamECG streams ECG data from the device to the provided channel.
func (receiver Reader) StreamECG(ctx context.Context, sink chan ECGMeasurement) error {
	if cap(sink) == 0 {
		return ErrUnbufferedChannel
	}

	pmdService, _ := bluetooth.ParseUUID(PMD_SERVICES)
	pmdCpCharacteristic, _ := bluetooth.ParseUUID(PMD_CHARACTERISTIC_CP)
	pmdDataCharacteristic, _ := bluetooth.ParseUUID(PMD_CHARACTERISTIC_ECG)

	pmdCpDeviceCharacteristic, err := receiver.retrieveDeviceCharacteristic(pmdService, pmdCpCharacteristic)
	if err != nil {
		return fmt.Errorf("failed to retrieve device PMD CP characteristic: %w", err)
	}
	pmdDataDeviceCharacteristic, err := receiver.retrieveDeviceCharacteristic(pmdService, pmdDataCharacteristic)
	if err != nil {
		return fmt.Errorf("failed to retrieve device PMD Data characteristic: %w", err)
	}

	receiver.writeCharacteristic(pmdCpDeviceCharacteristic, ENABLE_ECG)
	time.Sleep(1 * time.Second)
	ecgStream := make(chan []byte, 1)

	streamCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	err = streamNotification(streamCtx, pmdDataDeviceCharacteristic, ecgStream)

	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return suppressCancellationError(ctx.Err())
		case rawEcgMeasurement := <-ecgStream:
			ecgMeasurement, err := decodeECGData(rawEcgMeasurement)
			if err != nil {
				return fmt.Errorf("failed at parsing ecg measurement: %w", err)
			}
			sink <- ecgMeasurement
		}
	}
}

// retrieveDeviceCharacteristic retrieves a device characteristic from a service.
func (receiver Reader) retrieveDeviceCharacteristic(service, characteristic bluetooth.UUID) (bluetooth.DeviceCharacteristic, error) {
	services, err := receiver.device.DiscoverServices([]bluetooth.UUID{service})
	if err != nil {
		return bluetooth.DeviceCharacteristic{}, fmt.Errorf("failed at discovering service %s: %w", service.String(), err)
	}
	for _, service := range services {
		characteristics, err := service.DiscoverCharacteristics([]bluetooth.UUID{characteristic})
		if err != nil {
			return bluetooth.DeviceCharacteristic{}, fmt.Errorf("failed at discovering device characteristic %s: %w", characteristic.String(), err)
		}
		for _, characteristic := range characteristics {
			return characteristic, nil
		}
	}
	return bluetooth.DeviceCharacteristic{}, fmt.Errorf("device characteristic not found")
}

// readCharacteristic reads a response from a device characteristic.
func (receiver Reader) readCharacteristic(characteristic bluetooth.DeviceCharacteristic) ([]byte, error) {
	mtu, err := characteristic.GetMTU()
	if err != nil {
		return nil, fmt.Errorf("failed to obtain MTU of characteristic: %w", err)
	}
	data := make([]byte, mtu)
	dataLen, err := characteristic.Read(data)
	if err != nil {
		return nil, fmt.Errorf("failed to read response from characteristic: %w", err)
	}
	return data[:dataLen], nil
}

// writeCharacteristic writes a request to a device characteristic.
func (receiver Reader) writeCharacteristic(characteristic bluetooth.DeviceCharacteristic, req []byte) {
	// Both return values of DeviceCharacteristic.Write are properties of
	// the input - and gives no indication of operation succession.
	_, _ = characteristic.Write(req)
}

// streamNotification streams notifications from a device characteristic to the provided channel.
func streamNotification(ctx context.Context, characteristic bluetooth.DeviceCharacteristic, sink chan []byte) error {
	err := characteristic.EnableNotifications(func(buf []byte) {
		sink <- buf
	})

	if err != nil {
		return err
	}

	go func() {
		<-ctx.Done()
		characteristic.EnableNotifications(nil)
	}()

	return nil
}
