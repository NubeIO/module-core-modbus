package smod

import (
	"github.com/NubeIO/nubeio-rubix-lib-models-go/datatype"
	"github.com/grid-x/modbus"
	log "github.com/sirupsen/logrus"
	"strings"
)

type RegType uint
type Endianness uint
type WordOrder uint
type Error string

const (
	HoldingRegister RegType = 0
	InputRegister   RegType = 1

	// BigEndian endianness of 16-bit registers
	BigEndian    Endianness = 1
	LittleEndian Endianness = 2

	// HighWordFirst word order of 32-bit registers
	HighWordFirst WordOrder = 1
	LowWordFirst  WordOrder = 2
)

type ModbusClient struct {
	Client           modbus.Client
	RTUClientHandler *modbus.RTUClientHandler
	TCPClientHandler *modbus.TCPClientHandler
	Endianness       Endianness
	WordOrder        WordOrder
	RegType          RegType
	DeviceZeroMode   bool
	Debug            bool
	PortUnavailable  bool
}

// SetEncoding Sets the encoding (endianness and word ordering) of subsequent requests.
func (mc *ModbusClient) SetEncoding(endianness Endianness, wordOrder WordOrder) {
	mc.Endianness = endianness
	mc.WordOrder = wordOrder
}

// ReadCoils Reads multiple coils (function code 01).
func (mc *ModbusClient) ReadCoils(addr uint16, quantity uint16) (raw []byte, out float64, err error) {
	raw, err = mc.Client.ReadCoils(addr, quantity)
	if err != nil {
		log.Errorf("Modbus Polling: [failed to ReadCoils: %v]", err)
		return
	}
	out = float64(raw[0])
	return
}

// ReadDiscreteInputs Reads multiple Discrete Input Registers (function code 02).
func (mc *ModbusClient) ReadDiscreteInputs(addr uint16, quantity uint16) (raw []byte, out float64, err error) {
	raw, err = mc.Client.ReadDiscreteInputs(addr, quantity)
	if err != nil {
		log.Errorf("Modbus Polling: [failed to ReadDiscreteInputs: %v]", err)
		return
	}
	out = float64(raw[0])
	return
}

// ReadInputRegisters Reads multiple Input Registers (function code 02).
func (mc *ModbusClient) ReadInputRegisters(addr uint16, quantity uint16, dataType string) (raw []byte, out float64, err error) {
	raw, err = mc.Client.ReadInputRegisters(addr, quantity)
	if err != nil {
		log.Errorf("Modbus Polling: [failed to ReadInputRegisters: %v]", err)
		return
	}

	switch dataType {
	case string(datatype.TypeInt16):
		// Decode payload bytes as int16s
		decode := bytesToInt16s(mc.Endianness, raw)
		if len(decode) >= 0 {
			out = float64(decode[0])
		}
	case string(datatype.TypeUint16):
		// Decode payload bytes as uint16s
		decode := bytesToUint16s(mc.Endianness, raw)
		if len(decode) >= 0 {
			out = float64(decode[0])
		}
	case string(datatype.TypeInt32):
		// Decode payload bytes as uint16s
		decode := bytesToInt32s(mc.Endianness, mc.WordOrder, raw)
		if len(decode) >= 0 {
			out = float64(decode[0])
		}
	case string(datatype.TypeUint32):
		// Decode payload bytes as uint16s
		decode := bytesToUint32s(mc.Endianness, mc.WordOrder, raw)
		if len(decode) >= 0 {
			out = float64(decode[0])
		}
	case string(datatype.TypeInt64):
		// Decode payload bytes as uint16s
		decode := bytesToInt64s(mc.Endianness, mc.WordOrder, raw)
		if len(decode) >= 0 {
			out = float64(decode[0])
		}
	case string(datatype.TypeUint64):
		// Decode payload bytes as uint16s
		decode := bytesToUint64s(mc.Endianness, mc.WordOrder, raw)
		if len(decode) >= 0 {
			out = float64(decode[0])
		}
	case string(datatype.TypeMod10U32):
		// decode payload bytes as uint16s, then do R2*10,000 + R1
		decode := bytesToMod10_u32(mc.Endianness, mc.WordOrder, raw)
		out = decode[0]
	default:
		// Decode payload bytes as uint16s
		decode := bytesToUint16s(mc.Endianness, raw)
		if len(decode) >= 0 {
			out = float64(decode[0])
		}
	}
	return
}

// ReadHoldingRegisters Reads Holding Registers (function code 02).
func (mc *ModbusClient) ReadHoldingRegisters(addr uint16, quantity uint16, dataType string) (raw []byte, out float64, err error) {
	raw, err = mc.Client.ReadHoldingRegisters(addr, quantity)
	if err != nil {
		log.Errorf("Modbus Polling: [failed to ReadHoldingRegisters  addr:%d  quantity:%d error: %v]\n", addr, quantity, err)
		return
	}

	switch dataType {
	case string(datatype.TypeInt16):
		// Decode payload bytes as int16s
		decode := bytesToInt16s(mc.Endianness, raw)
		if len(decode) >= 0 {
			out = float64(decode[0])
		}
	case string(datatype.TypeUint16):
		// Decode payload bytes as uint16s
		decode := bytesToUint16s(mc.Endianness, raw)
		if len(decode) >= 0 {
			out = float64(decode[0])
		}
	case string(datatype.TypeInt32):
		// Decode payload bytes as uint16s
		decode := bytesToInt32s(mc.Endianness, mc.WordOrder, raw)
		if len(decode) >= 0 {
			out = float64(decode[0])
		}
	case string(datatype.TypeUint32):
		// Decode payload bytes as uint16s
		decode := bytesToUint32s(mc.Endianness, mc.WordOrder, raw)
		if len(decode) >= 0 {
			out = float64(decode[0])
		}
	case string(datatype.TypeInt64):
		// Decode payload bytes as uint16s
		decode := bytesToInt64s(mc.Endianness, mc.WordOrder, raw)
		if len(decode) >= 0 {
			out = float64(decode[0])
		}
	case string(datatype.TypeUint64):
		// Decode payload bytes as uint16s
		decode := bytesToUint64s(mc.Endianness, mc.WordOrder, raw)
		if len(decode) >= 0 {
			out = float64(decode[0])
		}
	case string(datatype.TypeMod10U32):
		// decode payload bytes as uint16s, then do R2*10,000 + R1
		decode := bytesToMod10_u32(mc.Endianness, mc.WordOrder, raw)
		out = decode[0]
	default:
		// Decode payload bytes as uint16s
		decode := bytesToUint16s(mc.Endianness, raw)
		if len(decode) >= 0 {
			out = float64(decode[0])
		}
	}
	return
}

// ReadFloat32s Reads multiple 32-bit float registers.
func (mc *ModbusClient) ReadFloat32s(addr uint16, quantity uint16, regType RegType) (raw []float32, err error) {
	var mbPayload []byte
	// Read 2 * quantity uint16 registers, as bytes
	if regType == HoldingRegister {
		mbPayload, err = mc.Client.ReadHoldingRegisters(addr, quantity*2)
		if err != nil {
			return
		}
	} else {
		mbPayload, err = mc.Client.ReadInputRegisters(addr, quantity*2)
		if err != nil {
			return
		}
	}
	// Decode payload bytes as float32s
	raw = bytesToFloat32s(mc.Endianness, mc.WordOrder, mbPayload)
	return
}

// ReadFloat32 Reads a single 32-bit float register.
func (mc *ModbusClient) ReadFloat32(addr uint16, regType RegType) (raw []float32, out float64, err error) {
	raw, err = mc.ReadFloat32s(addr, 1, regType)
	if err != nil {
		log.Errorf("Modbus Polling: [failed to ReadFloat32: %v]", err)
		return
	}
	out = float64(raw[0])
	return
}

// ReadFloat64s Reads multiple 64-bit float registers.
func (mc *ModbusClient) ReadFloat64s(addr uint16, quantity uint16, regType RegType) (raw []float64, err error) {
	var mbPayload []byte
	// Read 2 * quantity uint16 registers, as bytes
	if regType == HoldingRegister {
		mbPayload, err = mc.Client.ReadHoldingRegisters(addr, quantity*2)
		if err != nil {
			return
		}
	} else {
		mbPayload, err = mc.Client.ReadInputRegisters(addr, quantity*2)
		if err != nil {
			return
		}
	}
	// Decode payload bytes as float32s
	raw = bytesToFloat64s(mc.Endianness, mc.WordOrder, mbPayload)
	return
}

// ReadFloat64 Reads a single 64-bit float register.
func (mc *ModbusClient) ReadFloat64(addr uint16, regType RegType) (raw []float64, out float64, err error) {
	raw, err = mc.ReadFloat64s(addr, 1, regType)
	if err != nil {
		log.Errorf("Modbus Polling: [failed to ReadFloat64: %v]", err)
		return
	}
	out = raw[0]
	return
}

// WriteFloat32 Writes a single 32-bit float register.
func (mc *ModbusClient) WriteFloat32(addr uint16, value float64) (raw []byte, out float64, err error) {
	raw, err = mc.Client.WriteMultipleRegisters(addr, 2, float32ToBytes(mc.Endianness, mc.WordOrder, float32(value)))
	if err != nil {
		log.Errorf("Modbus Polling: [failed to WriteFloat32: %v]", err)
		return
	}
	out = value
	return
}

// WriteSingleRegister write one register
func (mc *ModbusClient) WriteSingleRegister(addr uint16, value uint16) (raw []byte, out float64, err error) {
	raw, err = mc.Client.WriteSingleRegister(addr, value)
	if err != nil {
		// This is a small hack for Nube-IO modbus (R-IO_v2.0 to R-IO_v3.1)(ZHT_v0.1 to ZHT_v2.1)
		//  where the value bytes are switched around.
		//  Most other Modbus tools do not check for this error anyway.
		if !strings.Contains(err.Error(), "modbus: response value") {
			log.Errorf("Modbus Polling: [failed to WriteSingleRegister: %v]", err)
			return
		} else {
			err = nil
		}
	}
	out = float64(value)
	return
}

// WriteDoubleRegister Writes to a double register (32bit)
func (mc *ModbusClient) WriteDoubleRegister(addr uint16, value uint32) (raw []byte, out float64, err error) {
	raw, err = mc.Client.WriteMultipleRegisters(addr, 2, uint32ToBytes(mc.Endianness, mc.WordOrder, value))
	if err != nil {
		log.Errorf("Modbus Polling: [failed to WriteDoubleRegister: %v]", err)
		return
	}
	out = float64(value)
	return
}

// WriteQuadRegister Writes to a double register (64bit)
func (mc *ModbusClient) WriteQuadRegister(addr uint16, value uint64) (raw []byte, out float64, err error) {
	raw, err = mc.Client.WriteMultipleRegisters(addr, 4, uint64ToBytes(mc.Endianness, mc.WordOrder, value))
	if err != nil {
		log.Errorf("Modbus Polling: [failed to WriteQuadRegister : %v]", err)
		return
	}
	out = float64(value)
	return
}

// WriteCoil Writes a single coil (function code 05)
func (mc *ModbusClient) WriteCoil(addr uint16, value uint16) (values []byte, out float64, err error) {
	values, err = mc.Client.WriteSingleCoil(addr, value)
	if err != nil {
		log.Errorf("Modbus Polling: [failed to WriteCoil: %v]", err)
		return
	}
	if value == 0 {
		out = 0
	} else {
		out = 1
	}
	return
}
