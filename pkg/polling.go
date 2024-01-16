package pkg

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/NubeIO/lib-module-go/nmodule"
	"github.com/NubeIO/lib-utils-go/boolean"
	"github.com/NubeIO/lib-utils-go/float"
	"github.com/NubeIO/lib-utils-go/integer"
	"github.com/NubeIO/lib-utils-go/nurl"
	"github.com/NubeIO/module-core-modbus/pollqueue"
	"github.com/NubeIO/module-core-modbus/utils/poller"
	"github.com/NubeIO/module-core-modbus/utils/writemode"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/datatype"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/dto"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/nargs"
	log "github.com/sirupsen/logrus"
)

type polling struct {
	enable        bool
	loopDelay     time.Duration
	delayNetworks time.Duration
	delayDevices  time.Duration
	delayPoints   time.Duration
	isRunning     bool
}

var poll poller.Poller

func (m *Module) initiatePolling() {
	poll = poller.New()
	ctx, cancel := context.WithCancel(context.Background())
	m.pollingCancel = cancel
	m.pollCounter = 0
	go poll.Poll(ctx, m.modbusPolling)
}

// modbusPolling TODO: currently Polling loops through each network, grabs one point, and polls it.  Could be improved by having a seperate client/go routine for each of the networks.
func (m *Module) modbusPolling() (bool, error) {
	m.pollCounter++
	m.modbusDebugMsg("LOOP COUNT: ", m.pollCounter)

	if len(m.NetworkPollManagers) == 0 {
		m.modbusDebugMsg("NO MODBUS NETWORKS FOUND")
		time.Sleep(10 * time.Second)
		return false, nil
	}

	for _, netPollMan := range m.NetworkPollManagers { // LOOP THROUGH AND POLL NEXT POINTS IN EACH NETWORK QUEUE
		if netPollMan.PortUnavailableTimeout != nil {
			m.modbusDebugMsg("skipping poll, port unavailable", netPollMan.FFNetworkUUID)
			continue
		}

		net, ok := m.getAndCheckNetwork(netPollMan.FFNetworkUUID)
		if !ok {
			continue
		}

		pp := netPollMan.GetNextPollingPoint()
		if pp == nil {
			m.modbusDebugMsg("skipping poll, no points to poll", net.Name, net.UUID)
			continue
		}
		pollStartTime := time.Now()

		dev, ok, retryMode := m.getAndCheckDevice(pp.FFDeviceUUID)
		if !ok {
			netPollMan.SinglePollFinished(pp, nil, pollStartTime, false, false, true, retryMode)
			continue
		}

		pnt, ok, retryMode := m.getAndCheckPoint(pp.FFPointUUID)
		if !ok {
			netPollMan.SinglePollFinished(pp, nil, pollStartTime, false, false, true, retryMode)
			continue
		}

		m.modbusPollingMsg(fmt.Sprintf("next poll drawn. Network: %s, Device: %s, Point: %s, Priority: %s, Device-Add: %d, Point-Add: %d, Point Type: %s, WriteRequired: %t, ReadRequired: %t", net.Name, dev.Name, pnt.Name, pnt.PollPriority, dev.AddressId, integer.NonNil(pnt.AddressID), pnt.ObjectType, boolean.IsTrue(pnt.WritePollRequired), boolean.IsTrue(pnt.ReadPollRequired)))

		writemode.SetPriorityArrayModeBasedOnWriteMode(pnt) // ensures the point PointPriorityArrayMode is set correctly

		var err error = nil
		mbClient, ok := m.mbClients[net.UUID]
		if !ok {
			mbClient, err = m.createMbClient(netPollMan, net, dev)
			if err != nil {
				m.modbusErrorMsg(fmt.Sprintf("failed to set client error: %v. network name:%s", err, net.Name))
				netPollMan.SinglePollFinished(pp, pnt, pollStartTime, false, false, false, pollqueue.NORMAL_RETRY)
				continue
			}
		}
		if net.TransportType == dto.TransType.Serial || net.TransportType == dto.TransType.LoRa {
			mbClient.RTUClientHandler.SlaveID = byte(dev.AddressId)
		} else if net.TransportType == dto.TransType.IP {
			url, err1 := nurl.JoinIPPort(nurl.Parts{Host: dev.Host, Port: strconv.Itoa(dev.Port)})
			if err1 != nil {
				errMes := fmt.Sprintf("failed to validate device address: %s, %s", url, err1.Error())
				m.modbusErrorMsg(errMes)
				m.updateNetworkMessage(net, "", errors.New(errMes), m.pollCounter)
				netPollMan.SinglePollFinished(pp, pnt, pollStartTime, false, false, false, pollqueue.DELAYED_RETRY)
				continue
			}
			mbClient.TCPClientHandler.Address = url
			mbClient.TCPClientHandler.SlaveID = byte(dev.AddressId)
		} else {
			errMes := fmt.Sprintf("invalid network transport type: %s, net: %s, err: %v", net.TransportType, net.Name, err)
			m.modbusDebugMsg(errMes)
			netPollMan.SinglePollFinished(pp, pnt, pollStartTime, false, false, false, pollqueue.DELAYED_RETRY)
			m.updateNetworkMessage(net, "", errors.New(errMes), m.pollCounter)
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

		// READ POINT
		readSuccess := false
		if boolean.IsTrue(pnt.ReadPollRequired) && (boolean.IsFalse(pnt.WritePollRequired) || (bitwiseType && boolean.IsTrue(pnt.WritePollRequired))) { // DO READ IF REQUIRED
			readResponse, readResponseValue, err = m.networkRead(mbClient, pnt)
			if err != nil {
				err = m.pointUpdateErr(pnt, err.Error(), dto.MessageLevel.Fail, dto.CommonFaultCode.PointError)
				netPollMan.SinglePollFinished(pp, pnt, pollStartTime, false, false, false, pollqueue.IMMEDIATE_RETRY)
				continue
			}
			if bitwiseType {
				var bitValue bool
				bitValue, err = getBitFromFloat64(readResponseValue, *pnt.BitwiseIndex)
				if err != nil {
					m.modbusDebugMsg("Bitwise Error: ", err)
					err = m.pointUpdateErr(pnt, err.Error(), dto.MessageLevel.Fail, dto.CommonFaultCode.PointError)
					netPollMan.SinglePollFinished(pp, pnt, pollStartTime, false, false, false, pollqueue.DELAYED_RETRY)
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

		// WRITE POINT
		writeSuccess := false
		if writemode.IsWriteable(pnt.WriteMode) && boolean.IsTrue(pnt.WritePollRequired) { // DO WRITE IF REQUIRED
			if pnt.WriteValue != nil {
				// TODO: should this be here?????
				if readSuccess {
					if net.MaxPollRate == nil {
						*net.MaxPollRate = 0.03
					}
					sleepTime := time.Second * time.Duration(*net.MaxPollRate)
					m.modbusDebugMsg(sleepTime.String(), " delay between read and write.")
					time.Sleep(sleepTime)
				}
				if bitwiseType {
					if !readSuccess || math.Mod(readResponseValue, 1) != 0 {
						err = m.pointUpdateErr(pnt, "read fail: bitwise point needs successful read before write", dto.MessageLevel.Fail, dto.CommonFaultCode.PointError)
						netPollMan.SinglePollFinished(pp, pnt, pollStartTime, false, false, false, pollqueue.DELAYED_RETRY)
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
					err = m.pointUpdateErr(pnt, err.Error(), dto.MessageLevel.Fail, dto.CommonFaultCode.PointWriteError)
					netPollMan.SinglePollFinished(pp, pnt, pollStartTime, false, false, false, pollqueue.IMMEDIATE_RETRY)
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
				writeSuccess = true // successful because there is no value to write.  Otherwise the point will short cycle.
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

		isChange := !float.ComparePtrValues(pnt.PresentValue, &newValue)
		if isChange {
			// For write_once and write_always type, write value should become present value
			writeValueToPresentVal := (pnt.WriteMode == datatype.WriteOnce || pnt.WriteMode == datatype.WriteAlways) && writeSuccess && pnt.WriteValue != nil
			if readSuccess || writeSuccess || writeValueToPresentVal {
				if writeValueToPresentVal {
					readSuccess = true
				}
				pnt, _ = m.pointUpdate(pnt, newValue, readSuccess || writeSuccess)
			}

			if m.pollCounter == 1 || m.pollCounter%100 == 0 { // give the user some feedback on how the polling has been working
				deviceMessage := fmt.Sprintf("last 100th poll: %s", TimeStamp())
				m.updateNetworkMessage(net, deviceMessage, nil, m.pollCounter)
				if m.pollCounter > 100000 {
					m.pollCounter = 100
				}
				device, err := m.grpcMarshaller.GetDevice(dev.UUID, &nmodule.Opts{Args: &nargs.Args{}})
				if err != nil || device == nil {
					continue
				}
				device.Message = deviceMessage
				device.CommonFault.LastOk = time.Now().UTC()
				m.grpcMarshaller.UpdateDevice(device.UUID, device)
			}
		}

		netPollMan.SinglePollFinished(pp, pnt, pollStartTime, writeSuccess, readSuccess, false, pollqueue.NORMAL_RETRY)

	}
	return false, nil
}

func (m *Module) getNetworkPollManagerByUUID(netUUID string) (*pollqueue.NetworkPollManager, error) {
	for _, netPollMan := range m.NetworkPollManagers {
		if netPollMan.FFNetworkUUID == netUUID {
			return netPollMan, nil
		}
	}
	return nil, errors.New("modbus getNetworkPollManagerByUUID(): couldn't find NetworkPollManager")
}

func (m *Module) getAndCheckNetwork(uuid string) (*model.Network, bool) {
	net, err := m.grpcMarshaller.GetNetwork(uuid)
	if err != nil || net == nil || net.PluginUUID != m.pluginUUID {
		m.modbusErrorMsg("network not found")
		return nil, false
	}
	if !boolean.IsTrue(net.Enable) {
		m.modbusDebugMsg("skipping poll, network disabled", net.Name, net.UUID)
		return nil, false
	}

	return net, true
}

func (m *Module) getAndCheckDevice(uuid string) (*model.Device, bool, pollqueue.PollRetryType) {
	dev, err := m.grpcMarshaller.GetDevice(uuid)
	if dev == nil || err != nil {
		m.modbusErrorMsg("skipping poll, could not find device", uuid)
		return nil, false, pollqueue.DELAYED_RETRY
	}
	if boolean.IsFalse(dev.Enable) {
		m.modbusErrorMsg("skipping poll, device disabled", dev.Name, dev.UUID)
		return nil, false, pollqueue.NEVER_RETRY
	}
	if dev.AddressId <= 0 || dev.AddressId >= 255 {
		m.modbusErrorMsg("skipping poll, invalid device address", dev.Name, dev.UUID)
		return nil, false, pollqueue.NEVER_RETRY
	}
	return dev, true, ""
}

func (m *Module) getAndCheckPoint(uuid string) (*model.Point, bool, pollqueue.PollRetryType) {
	pnt, err := m.grpcMarshaller.GetPoint(uuid, &nmodule.Opts{Args: &nargs.Args{WithPriority: true}})
	if pnt == nil || err != nil {
		m.modbusErrorMsg("could not find pointID: ", uuid)
		return nil, false, pollqueue.DELAYED_RETRY
	}

	m.printPointDebugInfo(pnt)

	if boolean.IsFalse(pnt.Enable) {
		m.modbusErrorMsg("skipping poll, point disabled", pnt.Name, pnt.UUID)
		return nil, false, pollqueue.NEVER_RETRY
	}

	if boolean.IsFalse(pnt.WritePollRequired) && boolean.IsFalse(pnt.ReadPollRequired) {
		m.modbusDebugMsg("skipping poll, polling not required", pnt.Name, pnt.UUID)
		return nil, false, pollqueue.NORMAL_RETRY
	}

	if pnt.Priority == nil {
		pnt.Priority = &model.Priority{}
	}

	return pnt, true, ""
}

func (m *Module) updateNetworkMessage(network *model.Network, message string, err error, loopCount int) {
	if err != nil {
		err = m.networkUpdateErr(network, err.Error(), dto.MessageLevel.Fail, dto.CommonFaultCode.NetworkError)
		if err != nil {
			log.Errorf("modbus failed to update network err: %s", err)
		}
	} else {
		err = m.networkUpdateMessage(network, fmt.Sprintf("%s poll count: %d", message, loopCount), dto.MessageLevel.Normal, dto.CommonFaultCode.Ok)
		if err != nil {
			log.Errorf("modbus failed to update network err: %s", err)
		}
	}

}
