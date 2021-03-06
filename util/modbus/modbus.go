package modbus

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/andig/evcc/util"
	"github.com/volkszaehler/mbmd/meters"
	"github.com/volkszaehler/mbmd/meters/rs485"
	"github.com/volkszaehler/mbmd/meters/sunspec"
)

// Settings combine physical modbus configuration and MBMD model
type Settings struct {
	Model      string
	Connection `mapstructure:",squash"`
}

// Connection contains the ModBus connection configuration
type Connection struct {
	ID                  uint8
	URI, Device, Comset string
	Baudrate            int
	RTU                 *bool // indicates RTU over TCP if true
}

var connections map[string]meters.Connection

func registeredConnection(key string, newConn meters.Connection) meters.Connection {
	if connections == nil {
		connections = make(map[string]meters.Connection)
	}

	if conn, ok := connections[key]; ok {
		return conn
	}

	connections[key] = newConn
	return newConn
}

// NewConnection creates physical modbus device from config
func NewConnection(log *util.Logger, uri, device, comset string, baudrate int, rtu bool) (conn meters.Connection) {
	if device != "" {
		conn = registeredConnection(device, meters.NewRTU(device, baudrate, comset))
	}

	if uri != "" {
		if rtu {
			conn = registeredConnection(uri, meters.NewRTUOverTCP(uri))
		} else {
			conn = registeredConnection(uri, meters.NewTCP(uri))
		}
	}

	if conn == nil {
		log.FATAL.Fatalf("config: invalid modbus configuration: need either uri or device")
	}

	return conn
}

// NewDevice creates physical modbus device from config
func NewDevice(log *util.Logger, model string, isRS485 bool) (device meters.Device, err error) {
	if isRS485 {
		device, err = rs485.NewDevice(strings.ToUpper(model))
	} else {
		device = sunspec.NewDevice(strings.ToUpper(model))
	}

	if device == nil {
		log.FATAL.Fatalf("config: invalid modbus configuration: need either uri or device")
	}

	return device, err
}

// IsRS485 determines if model is a known MBMD rs485 device model
func IsRS485(model string) bool {
	for k := range rs485.Producers {
		if strings.ToUpper(model) == k {
			return true
		}
	}
	return false
}

// RS485FindDeviceOp checks is RS485 device supports operation
func RS485FindDeviceOp(device *rs485.RS485, measurement meters.Measurement) (op rs485.Operation, err error) {
	ops := device.Producer().Produce()

	for _, op := range ops {
		if op.IEC61850 == measurement {
			return op, nil
		}
	}

	return op, fmt.Errorf("unsupported measurement: %s", measurement.String())
}

// Register contains the ModBus register configuration
type Register struct {
	Address uint16 // Length  uint16
	Type    string
	Decode  string
}

// RegisterOperation creates a read operation from a register definition
func RegisterOperation(r Register) (rs485.Operation, error) {
	op := rs485.Operation{
		OpCode:  r.Address,
		ReadLen: 2,
	}

	switch strings.ToLower(r.Type) {
	case "holding":
		op.FuncCode = rs485.ReadHoldingReg
	case "input":
		op.FuncCode = rs485.ReadInputReg
	default:
		return rs485.Operation{}, fmt.Errorf("invalid register type: %s", r.Type)
	}

	switch strings.ToLower(r.Decode) {
	case "float32", "ieee754":
		op.Transform = rs485.RTUIeee754ToFloat64
	case "float32s", "ieee754s":
		op.Transform = rs485.RTUIeee754ToFloat64Swapped
	case "float64":
		op.Transform = rs485.RTUUint64ToFloat64
		op.ReadLen = 4
	case "uint16":
		op.Transform = rs485.RTUUint16ToFloat64
		op.ReadLen = 1
	case "uint32":
		op.Transform = rs485.RTUUint32ToFloat64
	case "uint32s":
		op.Transform = rs485.RTUUint32ToFloat64Swapped
	case "uint64":
		op.Transform = rs485.RTUUint64ToFloat64
		op.ReadLen = 4
	case "int16":
		op.Transform = rs485.RTUInt16ToFloat64
		op.ReadLen = 1
	case "int32":
		op.Transform = rs485.RTUInt32ToFloat64
	case "int32s":
		op.Transform = rs485.RTUInt32ToFloat64Swapped
	default:
		return rs485.Operation{}, fmt.Errorf("invalid register decoding: %s", r.Decode)
	}

	return op, nil
}

// SunSpecOperation is a sunspec modbus operation
type SunSpecOperation struct {
	Model, Block int
	Point        string
}

// ParsePoint parses sunspec point from string
func ParsePoint(selector string) (model int, block int, point string, err error) {
	err = fmt.Errorf("invalid point: %s", selector)

	el := strings.Split(selector, ":")
	if len(el) < 2 || len(el) > 3 {
		return
	}

	if model, err = strconv.Atoi(el[0]); err != nil {
		return
	}

	if len(el) == 3 {
		// block is the middle element
		if block, err = strconv.Atoi(el[1]); err != nil {
			return
		}
	}

	point = el[len(el)-1]

	return model, block, point, nil
}

// Operation is a register-based or sunspec modbus operation
type Operation struct {
	MBMD    rs485.Operation
	SunSpec SunSpecOperation
}

// ParseOperation parses an MBMD measurement or SunsSpec point definition into a modbus operation
func ParseOperation(dev meters.Device, measurement string, op *Operation) (err error) {
	// if measurement cannot be parsed it could be SunSpec model/block/point
	if op.MBMD.IEC61850, err = meters.MeasurementString(measurement); err != nil {
		op.SunSpec.Model, op.SunSpec.Block, op.SunSpec.Point, err = ParsePoint(measurement)
		return err
	}

	// for RS485 check if producer supports the measurement
	if dev, ok := dev.(*rs485.RS485); ok {
		op.MBMD, err = RS485FindDeviceOp(dev, op.MBMD.IEC61850)
	}

	return err
}
