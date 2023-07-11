package pkg

import (
	"context"
	"errors"
	"fmt"
	"github.com/NubeIO/module-core-modbus/smod"
	"github.com/NubeIO/nubeio-rubix-lib-helpers-go/pkg/nils"
	"github.com/NubeIO/nubeio-rubix-lib-helpers-go/pkg/times/utilstime"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/pkg/v1/model"
	argspkg "github.com/NubeIO/rubix-os/args"
	"github.com/NubeIO/rubix-os/module/shared/pollqueue"
	"github.com/NubeIO/rubix-os/src/poller"
	"github.com/NubeIO/rubix-os/utils/boolean"
	"github.com/NubeIO/rubix-os/utils/float"
	"github.com/NubeIO/rubix-os/utils/nurl"
	"github.com/NubeIO/rubix-os/utils/writemode"
	"math"
	"strconv"
	"time"
)

type polling struct {
	enable        bool
	loopDelay     time.Duration
	delayNetworks time.Duration
	delayDevices  time.Duration
	delayPoints   time.Duration
	isRunning     bool
}

var poll poller.Poller // TODO: Do we need standalone

func (m *Module) getNetworkPollManagerByUUID(netUUID string) (*pollqueue.NetworkPollManager, error) {
	for _, netPollMan := range m.NetworkPollManagers {
		if netPollMan.FFNetworkUUID == netUUID {
			return netPollMan, nil
		}
	}
	return nil, errors.New("modbus getNetworkPollManagerByUUID(): couldn't find NetworkPollManager")
}

func (m *Module) ModbusPolling() error {
	poll = poller.New()
	var counter = 0
	f := func() (bool, error) {
		counter++
		m.modbusDebugMsg("LOOP COUNT: ", counter)

		if len(m.NetworkPollManagers) == 0 {
			m.modbusDebugMsg("NO MODBUS NETWORKS FOUND")
			time.Sleep(15000 * time.Millisecond)
		}

		for _, netPollMan := range m.NetworkPollManagers {
			if netPollMan.PortUnavailableTimeout != nil {
				m.modbusDebugMsg("modbus port unavailable. polling paused.")
				continue
			}

			pollStartTime := time.Now()

			net, err := m.grpcMarshaller.GetNetwork(netPollMan.FFNetworkUUID, argspkg.Args{})
			if err != nil || net == nil || net.PluginConfId != m.pluginUUID {
				m.modbusDebugMsg("MODBUS NETWORK NOT FOUND")
				continue
			}

			if !boolean.IsTrue(net.Enable) {
				m.modbusDebugMsg(fmt.Sprintf("NETWORK DISABLED: NAME: %s", net.Name))
				continue
			}

			pp, callback := netPollMan.GetNextPollingPoint()
			if pp == nil {
				continue
			}

			if pp.FFNetworkUUID != net.UUID {
				m.modbusErrorMsg("PollingPoint FFNetworkUUID does not match the Network UUID")
				netPollMan.PollingFinished(
					pp,
					pollStartTime,
					false,
					false,
					false,
					false,
					pollqueue.DELAYED_RETRY,
					callback,
				)
				continue
			}
			netPollMan.PrintPollQueuePointUUIDs()
			netPollMan.PrintPollingPointDebugInfo(pp)

			dev, err := m.grpcMarshaller.GetDevice(pp.FFDeviceUUID, argspkg.Args{})
			if dev == nil || err != nil {
				m.modbusErrorMsg("could not find deviceID:", pp.FFDeviceUUID)
				netPollMan.PollingFinished(
					pp,
					pollStartTime,
					false,
					false,
					false,
					true,
					pollqueue.DELAYED_RETRY,
					callback,
				)
				continue
			}
			if boolean.IsFalse(dev.Enable) {
				m.modbusErrorMsg("device is disabled.")
				m.grpcMarshaller.SetErrorsForAllPointsOnDevice(
					dev.UUID,
					"device disabled",
					model.MessageLevel.Warning,
					model.CommonFaultCode.DeviceError,
				)
				netPollMan.PollingFinished(
					pp,
					pollStartTime,
					false,
					false,
					false,
					true,
					pollqueue.NEVER_RETRY,
					callback,
				)
				continue
			}
			if dev.AddressId <= 0 || dev.AddressId >= 255 {
				m.modbusErrorMsg("address is not valid. modbus addresses must be between 1 and 254")
				m.grpcMarshaller.SetErrorsForAllPointsOnDevice(
					dev.UUID,
					"address out of range",
					model.MessageLevel.Critical,
					model.CommonFaultCode.ConfigError,
				)
				netPollMan.PollingFinished(
					pp,
					pollStartTime,
					false,
					false,
					false,
					true,
					pollqueue.NEVER_RETRY,
					callback,
				)
				continue
			}

			pnt, err := m.grpcMarshaller.GetPoint(pp.FFPointUUID, argspkg.Args{})
			if pnt == nil || err != nil {
				m.modbusErrorMsg("could not find pointID: ", pp.FFPointUUID)
				netPollMan.PollingFinished(
					pp,
					pollStartTime,
					false,
					false,
					false,
					true,
					pollqueue.DELAYED_RETRY,
					callback,
				)
				continue
			}

			m.printPointDebugInfo(pnt)

			if pnt.Priority == nil {
				m.modbusErrorMsg("ModbusPolling: HAD TO ADD PRIORITY ARRAY")
				pnt.Priority = &model.Priority{}
			}

			if !boolean.IsTrue(pnt.Enable) {
				m.modbusErrorMsg("point is disabled.")
				netPollMan.PollingFinished(
					pp,
					pollStartTime,
					false,
					false,
					false,
					true,
					pollqueue.NEVER_RETRY,
					callback,
				)
				continue
			}

			m.modbusPollingMsg(fmt.Sprintf("NEXT POLL DRAWN! : Network: %s, Device: %s, Point: %s, Priority: %s, Device-Add: %d, Point-Add: %d, Point Type: %s, WriteRequired: %t, ReadRequired: %t", net.Name, dev.Name, pnt.Name, pnt.PollPriority, dev.AddressId, *pnt.AddressID, pnt.ObjectType, boolean.IsTrue(pnt.WritePollRequired), boolean.IsTrue(pnt.ReadPollRequired)))

			if boolean.IsFalse(pnt.WritePollRequired) && boolean.IsFalse(pnt.ReadPollRequired) {
				m.modbusDebugMsg("polling not required on this point")
				netPollMan.PollingFinished(
					pp,
					pollStartTime,
					false,
					false,
					false,
					true,
					pollqueue.NORMAL_RETRY,
					callback,
				)
				continue
			}

			writemode.SetPriorityArrayModeBasedOnWriteMode(pnt) // Ensures the point PointPriorityArrayMode is set correctly

			// SETUP MODBUS CLIENT CONNECTION
			var mbClient smod.ModbusClient
			mbClient, err = m.setClient(net, dev, true)
			if err != nil {
				m.modbusErrorMsg(fmt.Sprintf("failed to set client error: %v, network name: %s", err, net.Name))
				if mbClient.PortUnavailable {
					netPollMan.PortUnavailable()
					unpauseFunc := func() {
						netPollMan.PortAvailable()
					}
					netPollMan.PortUnavailableTimeout = time.AfterFunc(10*time.Second, unpauseFunc)
				}
				netPollMan.PollingFinished(
					pp,
					pollStartTime,
					false,
					false,
					false,
					false,
					pollqueue.NORMAL_RETRY,
					callback,
				)
				continue
			}
			if net.TransportType == model.TransType.Serial || net.TransportType == model.TransType.LoRa {
				if dev.AddressId >= 1 {
					mbClient.RTUClientHandler.SlaveID = byte(dev.AddressId)
				}
			} else if dev.TransportType == model.TransType.IP {
				url, err := nurl.JoinIPPort(nurl.Parts{Host: dev.Host, Port: strconv.Itoa(dev.Port)})
				if err != nil {
					m.modbusErrorMsg("failed to validate device IP", url)
					netPollMan.PollingFinished(pp, pollStartTime, false, false, false, false, pollqueue.DELAYED_RETRY, callback)
					continue
				}
				mbClient.TCPClientHandler.Address = url
				mbClient.TCPClientHandler.SlaveID = byte(dev.AddressId)
			} else {
				m.modbusDebugMsg(fmt.Sprintf("failed to validate device and network %v %s", err, dev.Name))
				netPollMan.PollingFinished(pp, pollStartTime, false, false, false, false, pollqueue.DELAYED_RETRY, callback)
				continue
			}

			var readResponseValue float64
			var writeResponseValue float64
			var bitwiseResponseValue float64
			var bitwiseWriteValueFloat float64
			var bitwiseWriteValueBool bool
			var readResponse interface{}
			var writeResponse interface{}

			bitwiseType := boolean.IsTrue(pnt.IsBitwise) && pnt.BitwiseIndex != nil && *pnt.BitwiseIndex >= 0

			readSuccess := false
			if boolean.IsTrue(pnt.ReadPollRequired) && (boolean.IsFalse(pnt.WritePollRequired) || (bitwiseType && boolean.IsTrue(pnt.WritePollRequired))) { // DO READ IF REQUIRED
				readResponse, readResponseValue, err = m.networkRead(mbClient, pnt)
				if err != nil {
					err = m.pointUpdateErr(pnt, err.Error(), model.MessageLevel.Fail, model.CommonFaultCode.PointError)
					netPollMan.PollingFinished(pp, pollStartTime, false, false, true, false, pollqueue.IMMEDIATE_RETRY, callback)
					continue
				}
				if bitwiseType {
					var bitValue bool
					bitValue, err = getBitFromFloat64(readResponseValue, *pnt.BitwiseIndex)
					if err != nil {
						m.modbusDebugMsg("Bitwise Error: ", err)
						err = m.pointUpdateErr(pnt, err.Error(), model.MessageLevel.Fail, model.CommonFaultCode.PointError)
						netPollMan.PollingFinished(pp, pollStartTime, false, false, true, false, pollqueue.DELAYED_RETRY, callback)
						continue
					}
					if bitValue {
						bitwiseResponseValue = float64(1)
					} else {
						bitwiseResponseValue = float64(0)
					}
				}
				readSuccess = true
				m.modbusPollingMsg(fmt.Sprintf("READ-RESPONSE: responseValue %f, point UUID: %s, response: %+v ", readResponseValue, pnt.UUID, readResponse))
			}

			writeSuccess := false
			if writemode.IsWriteable(pnt.WriteMode) && boolean.IsTrue(pnt.WritePollRequired) { // DO WRITE IF REQUIRED
				if pnt.WriteValue != nil {
					if readSuccess {
						m.modbusDebugMsg(netPollMan.MaxPollRate.String(), " delay between read and write.")
						time.Sleep(netPollMan.MaxPollRate)
					}
					if bitwiseType {
						if !readSuccess || math.Mod(readResponseValue, 1) != 0 {
							err = m.pointUpdateErr(pnt, "read fail: bitwise point needs successful read before write", model.MessageLevel.Fail, model.CommonFaultCode.PointError)
							netPollMan.PollingFinished(pp, pollStartTime, false, false, true, false, pollqueue.DELAYED_RETRY, callback)
							continue
						}
						// Set appropriate writeValue for the bitwise type.  This value is the readResponseValue with the bit index modified
						if *pnt.WriteValue == 1 {
							bitwiseWriteValueBool = true
							bitwiseWriteValueFloat = float64(setBit(int(readResponseValue), uint(*pnt.BitwiseIndex)))
						} else if *pnt.WriteValue == 0 {
							bitwiseWriteValueBool = false
							bitwiseWriteValueFloat = float64(clearBit(int(readResponseValue), uint(*pnt.BitwiseIndex)))
						}
						pnt.WriteValue = float.New(bitwiseWriteValueFloat)
					}
					writeResponse, writeResponseValue, err = m.networkWrite(mbClient, pnt)
					if err != nil {
						err = m.pointUpdateErr(pnt, err.Error(), model.MessageLevel.Fail, model.CommonFaultCode.PointWriteError)
						netPollMan.PollingFinished(pp, pollStartTime, false, false, true, false, pollqueue.IMMEDIATE_RETRY, callback)
						continue
					}
					if bitwiseType {
						if bitwiseWriteValueBool {
							writeResponseValue = float64(1)
						} else {
							writeResponseValue = float64(0)
						}
					}
					writeSuccess = true

					m.modbusPollingMsg(fmt.Sprintf("WRITE-RESPONSE: responseValue %f, point UUID: %s, response: %+v", writeResponseValue, pnt.UUID, writeResponse))
				} else {
					writeSuccess = true // Successful because there is no value to write. Otherwise, the point will short cycle.
					m.modbusDebugMsg("modbus write point error: no value in priority array to write")
				}
			}

			var newValue float64
			if writeSuccess {
				newValue = writeResponseValue
			} else if readSuccess {
				if bitwiseType {
					newValue = bitwiseResponseValue
				} else {
					newValue = readResponseValue
				}
			} else {
				newValue = float.NonNil(pnt.PresentValue)
			}

			isChange := !float.ComparePtrValues(pnt.OriginalValue, &newValue)
			if isChange { // No change so just complete the polling (no point update required)
				if err != nil {
					netPollMan.PollingFinished(pp, pollStartTime, writeSuccess, readSuccess, true, false, pollqueue.NORMAL_RETRY, callback)
					continue
				}
			}

			// Update point in DB if required
			// For write_once and write_always type, write value should become present value
			writeValueToPresentVal := (pnt.WriteMode == model.WriteOnce || pnt.WriteMode == model.WriteAlways) && writeSuccess && pnt.WriteValue != nil

			if readSuccess || writeSuccess || writeValueToPresentVal {
				if writeValueToPresentVal {
					readSuccess = true
				}

				// This resets IsTypeBool to correct setting (fixes issues from points created before this was fixed in updatePoint())
				isTypeBool := checkForBooleanType(pnt.ObjectType, pnt.DataType)
				pnt.IsTypeBool = nils.NewBool(isTypeBool)

				pnt.CommonFault.InFault = false
				pnt.CommonFault.MessageLevel = model.MessageLevel.Info
				pnt.CommonFault.MessageCode = model.CommonFaultCode.PointWriteOk
				pnt.CommonFault.Message = fmt.Sprintf("last-updated: %s", utilstime.TimeStamp())
				pnt.CommonFault.LastOk = time.Now().UTC()
				m.pointUpdate(pnt, newValue, readSuccess || writeSuccess)
			}

			// This callback function triggers the PollManager to evaluate whether the point should be re-added to the PollQueue (Never, Immediately, or after the Poll Rate Delay)
			netPollMan.PollingFinished(pp, pollStartTime, writeSuccess, readSuccess, true, false, pollqueue.NORMAL_RETRY, callback)
		}
		return false, nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.pollingCancel = cancel
	go poll.GoPoll(ctx, f)
	return nil
}
