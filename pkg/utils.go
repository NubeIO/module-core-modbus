package pkg

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/NubeIO/lib-utils-go/boolean"
	"github.com/NubeIO/lib-utils-go/float"
	"github.com/NubeIO/lib-utils-go/integer"
	"github.com/NubeIO/lib-utils-go/nstring"
	"github.com/NubeIO/module-core-modbus/smod"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/datatype"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"
	log "github.com/sirupsen/logrus"
)

type Operation struct {
	UnitId       uint8  `json:"unit_id"`     // device addr
	ObjectType   string `json:"object_type"` // read_coil
	op           uint
	Addr         uint16  `json:"addr"`
	ZeroMode     bool    `json:"zero_mode"`
	Length       uint16  `json:"length"`
	IsCoil       bool    `json:"is_coil"`
	IsHoldingReg bool    `json:"is_holding_register"`
	WriteValue   float64 `json:"write_value"`
	Encoding     string  `json:"object_encoding"` // BEB_LEW
	coil         uint16
	u16          uint16
	u32          uint32
	f32          float32
	u64          uint64
	f64          float64
}

func pointWrite(pnt *model.Point) (out float64) {
	out = float.NonNil(pnt.WriteValue)
	log.Infof("modbus-write: pointWrite() ObjectType: %s  Addr: %d WriteValue: %v", pnt.ObjectType, integer.NonNil(pnt.AddressID), out)
	return
}

func writeCoilPayload(in float64) (out uint16) {
	if in > 0 {
		out = 0xFF00
	} else {
		out = 0x0000
	}
	return
}

func pointAddress(pnt *model.Point, zeroMode bool) uint16 {
	address := integer.NonNil(pnt.AddressID)
	// zeroMode will subtract 1 from the register address, so address 1 will be address 0 if set to true
	if !zeroMode {
		return uint16(address) - 1
	} else {
		return uint16(address)
	}
}

func (m *Module) networkRequest(mbClient *smod.ModbusClient, pnt *model.Point, doWrite bool) (response interface{}, responseValue float64, err error) {
	mbClient.Debug = true
	objectEncoding := pnt.ObjectEncoding                      // beb_lew
	dataType := nstring.NewString(pnt.DataType).ToSnakeCase() // eg: int16, uint16
	address := pointAddress(pnt, mbClient.DeviceZeroMode)     // register address
	length := integer.NonNil(pnt.AddressLength)               // modbus register length

	objectType := nstring.NewString(pnt.ObjectType).ToSnakeCase() // eg: readCoil, read_coil, writeCoil
	objectType = convertOldObjectType(objectType)

	switch objectEncoding {
	case string(datatype.ByteOrderLebBew):
		mbClient.SetEncoding(smod.LittleEndian, smod.HighWordFirst)
	case string(datatype.ByteOrderLebLew):
		mbClient.SetEncoding(smod.LittleEndian, smod.LowWordFirst)
	case string(datatype.ByteOrderBebLew):
		mbClient.SetEncoding(smod.BigEndian, smod.LowWordFirst)
	case string(datatype.ByteOrderBebBew):
		mbClient.SetEncoding(smod.BigEndian, smod.HighWordFirst)
	default:
		mbClient.SetEncoding(smod.BigEndian, smod.LowWordFirst)
	}
	if length <= 0 { // Make sure length is > 0
		length = 1
	}
	var writeValue float64
	if doWrite {
		writeValue = pointWrite(pnt)
	}

	if doWrite {
		m.modbusDebugMsg("modbus-write: ObjectType: %s  Addr: %d WriteValue: %v\n", objectType, address, writeValue)
	} else {
		m.modbusDebugMsg("modbus-read: ObjectType: %s  Addr: %d", objectType, address)
	}

	switch objectType {
	// COILS
	case string(datatype.ObjTypeCoil):
		if doWrite {
			return mbClient.WriteCoil(address, writeCoilPayload(writeValue))
		} else {
			return mbClient.ReadCoils(address, uint16(length))
		}

	// READ DISCRETE INPUTS
	case string(datatype.ObjTypeDiscreteInput):
		return mbClient.ReadDiscreteInputs(address, uint16(length))

	// READ HOLDINGS
	case string(datatype.ObjTypeHoldingRegister):
		if doWrite {
			if dataType == string(datatype.TypeUint16) || dataType == string(datatype.TypeInt16) {
				return mbClient.WriteSingleRegister(address, uint16(writeValue))
			} else if dataType == string(datatype.TypeUint32) || dataType == string(datatype.TypeInt32) {
				return mbClient.WriteDoubleRegister(address, uint32(writeValue))
			} else if dataType == string(datatype.TypeUint64) || dataType == string(datatype.TypeInt64) {
				return mbClient.WriteQuadRegister(address, uint64(writeValue))
			} else if dataType == string(datatype.TypeFloat32) {
				return mbClient.WriteFloat32(address, writeValue)
			} else if dataType == string(datatype.TypeFloat64) {
				return mbClient.WriteFloat32(address, writeValue)
			}
		} else {
			if dataType == string(datatype.TypeUint16) || dataType == string(datatype.TypeInt16) {
				return mbClient.ReadHoldingRegisters(address, uint16(length), dataType)
			} else if dataType == string(datatype.TypeUint32) || dataType == string(datatype.TypeInt32) {
				return mbClient.ReadHoldingRegisters(address, uint16(length), dataType)
			} else if dataType == string(datatype.TypeUint64) || dataType == string(datatype.TypeInt64) {
				return mbClient.ReadHoldingRegisters(address, uint16(length), dataType)
			} else if dataType == string(datatype.TypeFloat32) {
				return mbClient.ReadFloat32(address, smod.HoldingRegister)
			} else if dataType == string(datatype.TypeFloat64) {
				return mbClient.ReadFloat32(address, smod.HoldingRegister)
			}
		}

	// READ INPUT REGISTERS
	case string(datatype.ObjTypeInputRegister):
		if dataType == string(datatype.TypeUint16) || dataType == string(datatype.TypeInt16) {
			return mbClient.ReadInputRegisters(address, uint16(length), dataType)
		} else if dataType == string(datatype.TypeUint32) || dataType == string(datatype.TypeInt32) {
			return mbClient.ReadInputRegisters(address, uint16(length), dataType)
		} else if dataType == string(datatype.TypeUint64) || dataType == string(datatype.TypeInt64) {
			return mbClient.ReadInputRegisters(address, uint16(length), dataType)
		} else if dataType == string(datatype.TypeFloat32) {
			return mbClient.ReadFloat32(address, smod.InputRegister)
		} else if dataType == string(datatype.TypeFloat64) {
			return mbClient.ReadFloat32(address, smod.InputRegister)
		}

	}

	return nil, 0, nil
}

func (m *Module) networkWrite(mbClient *smod.ModbusClient, pnt *model.Point) (response interface{}, responseValue float64, err error) {
	if pnt.WriteValue == nil {
		return nil, 0, errors.New("modbus-write: point has no WriteValue")
	}
	mbClient.Debug = true
	objectEncoding := pnt.ObjectEncoding                      // beb_lew
	dataType := nstring.NewString(pnt.DataType).ToSnakeCase() // eg: int16, uint16
	address := pointAddress(pnt, mbClient.DeviceZeroMode)     // register address

	objectType := nstring.NewString(pnt.ObjectType).ToSnakeCase() // eg: readCoil, read_coil, writeCoil
	objectType = convertOldObjectType(objectType)

	switch objectEncoding {
	case string(datatype.ByteOrderLebBew):
		mbClient.SetEncoding(smod.LittleEndian, smod.HighWordFirst)
	case string(datatype.ByteOrderLebLew):
		mbClient.SetEncoding(smod.LittleEndian, smod.LowWordFirst)
	case string(datatype.ByteOrderBebLew):
		mbClient.SetEncoding(smod.BigEndian, smod.LowWordFirst)
	case string(datatype.ByteOrderBebBew):
		mbClient.SetEncoding(smod.BigEndian, smod.HighWordFirst)
	default:
		mbClient.SetEncoding(smod.BigEndian, smod.LowWordFirst)
	}

	writeValue := *pnt.WriteValue

	m.modbusPollingMsg(fmt.Sprintf("WRITE-POLL: ObjectType: %s  Addr: %d WriteValue: %v", objectType, address, writeValue))

	switch objectType {
	// WRITE COILS
	case string(datatype.ObjTypeCoil):
		return mbClient.WriteCoil(address, writeCoilPayload(writeValue))

	// WRITE HOLDINGS
	case string(datatype.ObjTypeHoldingRegister):
		if dataType == string(datatype.TypeUint16) || dataType == string(datatype.TypeInt16) {
			return mbClient.WriteSingleRegister(address, uint16(writeValue))
		} else if dataType == string(datatype.TypeUint32) || dataType == string(datatype.TypeInt32) {
			return mbClient.WriteDoubleRegister(address, uint32(writeValue))
		} else if dataType == string(datatype.TypeUint64) || dataType == string(datatype.TypeInt64) {
			return mbClient.WriteQuadRegister(address, uint64(writeValue))
		} else if dataType == string(datatype.TypeFloat32) {
			return mbClient.WriteFloat32(address, writeValue)
		} else if dataType == string(datatype.TypeFloat64) {
			return mbClient.WriteFloat32(address, writeValue)
		}
	}

	return nil, 0, errors.New("modbus-write: dataType is not recognized")
}

func (m *Module) networkRead(mbClient *smod.ModbusClient, pnt *model.Point) (response interface{}, responseValue float64, err error) {
	mbClient.Debug = true
	objectEncoding := pnt.ObjectEncoding                      // beb_lew
	dataType := nstring.NewString(pnt.DataType).ToSnakeCase() // eg: int16, uint16
	address := pointAddress(pnt, mbClient.DeviceZeroMode)     // register address
	length := integer.NonNil(pnt.AddressLength)               // modbus register length
	length = 1

	objectType := nstring.NewString(pnt.ObjectType).ToSnakeCase() // eg: readCoil, read_coil, writeCoil
	objectType = convertOldObjectType(objectType)

	switch objectEncoding {
	case string(datatype.ByteOrderLebBew):
		mbClient.SetEncoding(smod.LittleEndian, smod.HighWordFirst)
	case string(datatype.ByteOrderLebLew):
		mbClient.SetEncoding(smod.LittleEndian, smod.LowWordFirst)
	case string(datatype.ByteOrderBebLew):
		mbClient.SetEncoding(smod.BigEndian, smod.LowWordFirst)
	case string(datatype.ByteOrderBebBew):
		mbClient.SetEncoding(smod.BigEndian, smod.HighWordFirst)
	default:
		mbClient.SetEncoding(smod.BigEndian, smod.LowWordFirst)
	}

	m.modbusDebugMsg(fmt.Sprintf("modbus-read: ObjectType: %s  Addr: %d", objectType, address))

	switch objectType {
	// COILS
	case string(datatype.ObjTypeCoil):
		return mbClient.ReadCoils(address, uint16(length))

	// READ DISCRETE INPUTS
	case string(datatype.ObjTypeDiscreteInput):
		return mbClient.ReadDiscreteInputs(address, uint16(length))

	// READ INPUT REGISTERS
	case string(datatype.ObjTypeInputRegister):
		if dataType == string(datatype.TypeUint16) || dataType == string(datatype.TypeInt16) {
			length = 1
			return mbClient.ReadInputRegisters(address, uint16(length), dataType)
		} else if dataType == string(datatype.TypeUint32) || dataType == string(datatype.TypeInt32) || dataType == string(datatype.TypeMod10U32) {
			length = 2
			return mbClient.ReadInputRegisters(address, uint16(length), dataType)
		} else if dataType == string(datatype.TypeUint64) || dataType == string(datatype.TypeInt64) {
			length = 4
			return mbClient.ReadInputRegisters(address, uint16(length), dataType)
		} else if dataType == string(datatype.TypeFloat32) {
			return mbClient.ReadFloat32(address, smod.InputRegister)
		} else if dataType == string(datatype.TypeFloat64) {
			return mbClient.ReadFloat64(address, smod.InputRegister)
		}

	// READ HOLDINGS
	case string(datatype.ObjTypeHoldingRegister):
		if dataType == string(datatype.TypeUint16) || dataType == string(datatype.TypeInt16) {
			length = 1
			return mbClient.ReadHoldingRegisters(address, uint16(length), dataType)
		} else if dataType == string(datatype.TypeUint32) || dataType == string(datatype.TypeInt32) || dataType == string(datatype.TypeMod10U32) {
			length = 2
			return mbClient.ReadHoldingRegisters(address, uint16(length), dataType)
		} else if dataType == string(datatype.TypeUint64) || dataType == string(datatype.TypeInt64) {
			length = 4
			return mbClient.ReadHoldingRegisters(address, uint16(length), dataType)
		} else if dataType == string(datatype.TypeFloat32) {
			return mbClient.ReadFloat32(address, smod.HoldingRegister)
		} else if dataType == string(datatype.TypeFloat64) {
			return mbClient.ReadFloat64(address, smod.HoldingRegister)
		}

	}

	return nil, 0, errors.New("modbus-read: dataType is not recognized")
}

func SetPriorityArrayModeBasedOnWriteMode(pnt *model.Point) bool {
	switch pnt.WriteMode {
	case datatype.ReadOnce, datatype.ReadOnly:
		pnt.PointPriorityArrayMode = datatype.ReadOnlyNoPriorityArrayRequired
		return true
	case datatype.WriteOnce, datatype.WriteOnceReadOnce, datatype.WriteAlways, datatype.WriteOnceThenRead, datatype.WriteAndMaintain:
		pnt.PointPriorityArrayMode = datatype.PriorityArrayToWriteValue
		return true
	}
	return false
}

func isWriteable(writeMode datatype.WriteMode, objectType string) bool {
	if isWriteableObjectType(objectType) && IsWriteable(writeMode) {
		return true
	} else {
		return false
	}
}

func isWriteableObjectType(objectType string) bool {
	switch objectType {
	case string(datatype.ObjTypeWriteCoil), string(datatype.ObjTypeWriteCoils), string(datatype.ObjTypeCoil):
		return true
	case string(datatype.ObjTypeWriteHolding), string(datatype.ObjTypeWriteHoldings), string(datatype.ObjTypeHoldingRegister):
		return true
	case string(datatype.ObjTypeWriteInt16), string(datatype.ObjTypeWriteUint16):
		return true
	case string(datatype.ObjTypeWriteFloat32):
		return true
	}
	return false
}

func checkForBooleanType(ObjectType, DataType string) (isTypeBool bool) {
	isTypeBool = false
	if DataType == string(datatype.TypeDigital) {
		isTypeBool = true
	}
	switch ObjectType {
	case
		string(datatype.ObjTypeReadCoil),
		string(datatype.ObjTypeReadCoils),
		string(datatype.ObjTypeReadDiscreteInput),
		string(datatype.ObjTypeReadDiscreteInputs),
		string(datatype.ObjTypeWriteCoil),
		string(datatype.ObjTypeWriteCoils):
		isTypeBool = true
	}
	return
}

func checkForOutputType(ObjectType string) (isOutput bool) {
	isOutput = false
	switch ObjectType {
	case
		string(datatype.ObjTypeWriteCoil),
		string(datatype.ObjTypeWriteCoils),
		string(datatype.ObjTypeWriteHolding),
		string(datatype.ObjTypeWriteHoldings),
		string(datatype.ObjTypeWriteInt16),
		string(datatype.ObjTypeWriteUint16),
		string(datatype.ObjTypeWriteFloat32),
		string(datatype.ObjTypeWriteFloat64):
		isOutput = true
	}
	return
}

func getBitsFromFloat64(value float64) (bitArray []bool, err error) {
	if math.Mod(value, 1) != 0 {
		err = errors.New("cannot get bits from floats")
		return
	}
	if value < 0 {
		err = errors.New("cannot get bits from negative numbers")
		return
	}
	buf := make([]byte, binary.MaxVarintLen64)
	length := binary.PutUvarint(buf, uint64(value))
	for j := 0; j < length; j++ {
		bits := buf[j]
		for i := 0; bits > 0; i, bits = i+1, bits>>1 {
			if bits&1 == 1 {
				bitArray = append(bitArray, true)
			} else if bits&1 == 0 {
				bitArray = append(bitArray, false)
			}
		}
	}
	return
}

func getBitFromFloat64(value float64, reqIndex int) (indexValue bool, err error) {
	if math.Mod(value, 1) != 0 {
		err = errors.New("cannot get bits from floats")
		return
	}
	if value < 0 {
		err = errors.New("cannot get bits from negative numbers")
		return
	}
	indexValue = hasBit(int(value), uint(reqIndex))
	return
}

// Sets the bit at pos in the integer n.
func setBit(n int, pos uint) int {
	n |= (1 << pos)
	return n
}

// Clears the bit at pos in n.
func clearBit(n int, pos uint) int {
	mask := ^(1 << pos)
	n &= mask
	return n
}

// Checks the bit at pos in n
func hasBit(n int, pos uint) bool {
	val := n & (1 << pos)
	return (val > 0)
}

// Convert
func convertOldObjectType(objectType string) string {
	switch objectType {
	// COILS
	case string(datatype.ObjTypeReadCoil), string(datatype.ObjTypeReadCoils), string(datatype.ObjTypeWriteCoil), string(datatype.ObjTypeWriteCoils), string(datatype.ObjTypeCoil):
		return string(datatype.ObjTypeCoil)

	// READ DISCRETE INPUTS
	case string(datatype.ObjTypeReadDiscreteInput), string(datatype.ObjTypeReadDiscreteInputs), string(datatype.ObjTypeDiscreteInput):
		return string(datatype.ObjTypeDiscreteInput)

	// READ INPUT REGISTERS
	case string(datatype.ObjTypeReadRegister), string(datatype.ObjTypeReadRegisters), string(datatype.ObjTypeInputRegister):
		return string(datatype.ObjTypeInputRegister)

	// READ HOLDINGS
	case string(datatype.ObjTypeReadHolding), string(datatype.ObjTypeReadHoldings), string(datatype.ObjTypeWriteHolding), string(datatype.ObjTypeWriteHoldings), string(datatype.ObjTypeHoldingRegister):
		return string(datatype.ObjTypeHoldingRegister)

	default:
		fmt.Println("invalid ObjectType: ", objectType)
		return string(datatype.ObjTypeHoldingRegister)
	}
}

func TimeStamp() (hostTime string) {
	hostTime = time.Now().Format(time.Stamp)
	return
}

func IsWriteable(writeMode datatype.WriteMode) bool {
	switch writeMode {
	case datatype.ReadOnce, datatype.ReadOnly:
		return false
	case datatype.WriteOnce, datatype.WriteOnceReadOnce, datatype.WriteAlways, datatype.WriteOnceThenRead, datatype.WriteAndMaintain:
		return true
	default:
		return false
	}
}

func resetWriteableProperties(point *model.Point) *model.Point {
	point.WriteValueOriginal = nil
	point.WriteValue = nil
	point.WritePriority = nil
	point.CurrentPriority = nil
	point.EnableWriteable = boolean.NewFalse()
	point.WritePollRequired = boolean.NewFalse()
	return point
}
