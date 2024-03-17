package pkg

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/NubeIO/lib-module-go/nmodule"
	"github.com/NubeIO/lib-utils-go/boolean"
	"github.com/NubeIO/module-core-modbus/pollqueue"
	"github.com/NubeIO/nubeio-rubix-lib-helpers-go/pkg/nils"
	"github.com/NubeIO/nubeio-rubix-lib-helpers-go/pkg/times/utilstime"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/datatype"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/dto"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/nargs"
)

func (m *Module) addNetwork(body *model.Network) (network *model.Network, err error) {
	m.modbusDebugMsg("addNetwork(): ", body.Name)
	body.HasPollingStatistics = true

	network, err = m.grpcMarshaller.CreateNetwork(body)
	if err != nil {
		m.modbusErrorMsg("addNetwork(): failed to create modbus network: ", body.Name)
		return nil, errors.New("failed to create modbus network")
	}

	if boolean.IsTrue(network.Enable) {
		m.initiatePolling(m.pollingContext, network)
	} else {
		err = m.networkUpdateErr(network, "network disabled", dto.MessageLevel.Warning, dto.CommonFaultCode.NetworkError)
		err = m.grpcMarshaller.UpdateNetworkDescendantsErrors(network.UUID, "network disabled", dto.MessageLevel.Warning, dto.CommonFaultCode.NetworkError, true)
	}
	return network, nil
}

func (m *Module) addDevice(body *model.Device) (device *model.Device, err error) {
	if body == nil {
		m.modbusDebugMsg("addDevice(): nil device object")
		return nil, errors.New("empty device body, no device created")
	}
	m.modbusDebugMsg("addDevice(): ", body.Name)
	device, err = m.grpcMarshaller.CreateDevice(body)
	if device == nil || err != nil {
		m.modbusDebugMsg("addDevice(): failed to create modbus device: ", body.Name)
		return nil, errors.New("failed to create modbus device")
	}

	m.modbusDebugMsg("addDevice(): ", body.UUID)

	if boolean.IsFalse(device.Enable) {
		err = m.deviceUpdateErr(device, "device disabled", dto.MessageLevel.Warning, dto.CommonFaultCode.DeviceError)
		err = m.grpcMarshaller.UpdateDeviceDescendantsErrors(device.UUID, "device disabled", dto.MessageLevel.Warning, dto.CommonFaultCode.DeviceError)
	}

	netPollMan, err := m.getNetworkPollManagerByUUID(device.NetworkUUID)
	if netPollMan == nil || err != nil {
		m.modbusDebugMsg("addPoint(): cannot find NetworkPollManager for network: ", device.NetworkUUID)
		return
	}
	netPollMan.SetDevicePollRateDurations(device)

	// NOTHING TO DO ON DEVICE CREATED
	return device, nil
}

func (m *Module) addPoint(body *model.Point) (point *model.Point, err error) {
	if body == nil {
		m.modbusDebugMsg("addPoint(): nil point object")
		return nil, errors.New("empty point body, no point created")
	}
	m.modbusDebugMsg("addPoint(): ", body.Name)

	if isWriteable(body.WriteMode, body.ObjectType) {
		body.EnableWriteable = boolean.NewTrue()
		if pollqueue.PollOnStartCheck(body) {
			body.WritePollRequired = boolean.NewTrue()
		}
	} else {
		body = resetWriteableProperties(body)
	}
	body.ReadPollRequired = boolean.NewTrue()

	if *body.AddressID < 1 || *body.AddressID > 65535 {
		return nil, errors.New("register must be between 1 and 65535")
	}

	isTypeBool := checkForBooleanType(body.ObjectType, body.DataType)
	body.IsTypeBool = nils.NewBool(isTypeBool)

	isOutput := checkForOutputType(body.ObjectType)
	body.IsOutput = nils.NewBool(isOutput)

	point, err = m.grpcMarshaller.CreatePoint(body)
	if err != nil {
		return nil, err
	}
	point, err = m.grpcMarshaller.UpdatePoint(point.UUID, point)
	if err != nil {
		return nil, errors.New(fmt.Sprint("failed to create modbus point. err: ", err))
	}

	dev, err := m.grpcMarshaller.GetDevice(point.DeviceUUID)
	if err != nil {
		return nil, err
	}

	netPollMan, err := m.getNetworkPollManagerByUUID(dev.NetworkUUID)
	if err != nil {
		return nil, err
	}

	if boolean.IsTrue(point.Enable) {
		netPollMan.PollQueue.RemovePollingPointByPointUUID(point.UUID)
		pp := pollqueue.NewPollingPoint(point.UUID, point.DeviceUUID, dev.NetworkUUID)
		if pollqueue.PollOnStartCheck(point) {
			netPollMan.PollingPointCompleteNotification(pp, point, false, false, 0, true, true, pollqueue.NORMAL_RETRY, false)
		} else {
			netPollMan.PollingPointCompleteNotification(pp, point, true, true, 0, true, false, pollqueue.NORMAL_RETRY, true)
		}
	} else {
		err = m.internalPointUpdateErr(point, "point disabled", dto.MessageLevel.Warning, dto.CommonFaultCode.PointError)
	}
	return point, nil
}

func (m *Module) updateNetwork(uuid string, body *model.Network) (network *model.Network, err error) {
	m.modbusDebugMsg("updateNetwork(): ", uuid)
	if body == nil {
		m.modbusDebugMsg("updateNetwork():  nil network object")
		return
	}

	// indicates that ui should display polling statistics
	body.HasPollingStatistics = true

	if boolean.IsFalse(body.Enable) {
		body.CommonFault.InFault = true
		body.CommonFault.MessageLevel = dto.MessageLevel.Warning
		body.CommonFault.MessageCode = dto.CommonFaultCode.NetworkError
		body.CommonFault.Message = "network disabled"
		body.CommonFault.LastFail = time.Now().UTC()
	} else {
		body.CommonFault.InFault = false
		body.CommonFault.MessageLevel = dto.MessageLevel.Info
		body.CommonFault.MessageCode = dto.CommonFaultCode.Ok
		body.CommonFault.Message = ""
		body.CommonFault.LastOk = time.Now().UTC()
	}

	network, err = m.grpcMarshaller.UpdateNetwork(uuid, body)
	if err != nil || network == nil {
		return nil, err
	}

	restartPolling := false
	if body.MaxPollRate != network.MaxPollRate {
		restartPolling = true
	}

	netPollMan, err := m.getNetworkPollManagerByUUID(network.UUID)
	if netPollMan == nil || err != nil {
		m.modbusDebugMsg("updateNetwork(): cannot find NetworkPollManager for network: ", network.UUID)
		return
	}

	if netPollMan.NetworkName != network.Name {
		netPollMan.NetworkName = network.Name
	}

	if boolean.IsFalse(network.Enable) && netPollMan.Enable == true {
		// DO POLLING DISABLE ACTIONS
		netPollMan.StopPolling()
		m.grpcMarshaller.UpdateNetworkDescendantsErrors(network.UUID, "network disabled", dto.MessageLevel.Warning, dto.CommonFaultCode.DeviceError, true)
	} else if restartPolling || (boolean.IsTrue(network.Enable) && netPollMan.Enable == false) {
		if restartPolling {
			netPollMan.StopPolling()
		}
		// DO POLLING Enable ACTIONS
		netPollMan.StartPolling()
		m.grpcMarshaller.ClearNetworkDescendantsErrors(network.UUID, true)
	}

	network, err = m.grpcMarshaller.UpdateNetwork(uuid, network)
	if err != nil || network == nil {
		return nil, err
	}
	return network, nil
}

func (m *Module) updateDevice(uuid string, body *model.Device) (device *model.Device, err error) {
	m.modbusDebugMsg("updateDevice(): ", uuid)
	if body == nil {
		m.modbusDebugMsg("updateDevice(): nil device object")
		return
	}

	if boolean.IsFalse(body.Enable) {
		body.CommonFault.InFault = true
		body.CommonFault.MessageLevel = dto.MessageLevel.Warning
		body.CommonFault.MessageCode = dto.CommonFaultCode.DeviceError
		body.CommonFault.Message = "device disabled"
		body.CommonFault.LastFail = time.Now().UTC()
	} else {
		body.CommonFault.InFault = false
		body.CommonFault.MessageLevel = dto.MessageLevel.Info
		body.CommonFault.MessageCode = dto.CommonFaultCode.Ok
		body.CommonFault.LastOk = time.Now().UTC()
	}

	device, err = m.grpcMarshaller.UpdateDevice(uuid, body)
	if err != nil || device == nil {
		return nil, err
	}

	if boolean.IsTrue(device.Enable) { // If Enabled we need to GetDevice so we get Points
		device, err = m.grpcMarshaller.GetDevice(device.UUID, &nmodule.Opts{Args: &nargs.Args{WithPoints: true}})
		if err != nil || device == nil {
			return nil, err
		}
	}

	netPollMan, err := m.getNetworkPollManagerByUUID(device.NetworkUUID)
	if netPollMan == nil || err != nil {
		m.modbusDebugMsg("updateDevice(): cannot find NetworkPollManager for network: ", device.NetworkUUID)
		return
	}
	if boolean.IsFalse(device.Enable) {
		// DO POLLING DISABLE ACTIONS FOR DEVICE
		m.grpcMarshaller.UpdateDeviceDescendantsErrors(device.UUID, "device disabled", dto.MessageLevel.Warning, dto.CommonFaultCode.DeviceError)
		netPollMan.PollQueue.RemovePollingPointByDeviceUUID(device.UUID)

	} else if boolean.IsTrue(device.Enable) {
		// DO POLLING ENABLE ACTIONS FOR DEVICE
		err = m.grpcMarshaller.ClearDeviceDescendantsErrors(device.UUID)
		if err != nil {
			m.modbusDebugMsg("updateDevice(): error on ClearErrorsForAllPointsOnDevice(): ", err)
		}
		for _, pnt := range device.Points {
			if boolean.IsTrue(pnt.Enable) {
				pp := pollqueue.NewPollingPoint(pnt.UUID, pnt.DeviceUUID, device.NetworkUUID)
				if pollqueue.PollOnStartCheck(pnt) {
					netPollMan.PollingPointCompleteNotification(pp, pnt, false, false, 0, true, true, pollqueue.NORMAL_RETRY, false)
				} else {
					netPollMan.PollingPointCompleteNotification(pp, pnt, true, true, 0, true, false, pollqueue.NORMAL_RETRY, true)
				}
			}
		}
	}
	netPollMan.SetDevicePollRateDurations(device)
	// TODO: NEED TO ACCOUNT FOR OTHER CHANGES ON DEVICE.
	//  It would be useful to have a way to know if the device polling rates were changed.
	device, err = m.grpcMarshaller.UpdateDevice(device.UUID, device)
	if err != nil {
		return nil, err
	}
	return device, nil
}

func (m *Module) updatePoint(uuid string, body *model.Point) (point *model.Point, err error) {
	m.modbusDebugMsg("updatePoint(): ", uuid)
	if body == nil {
		m.modbusDebugMsg("updatePoint(): nil point object")
		return
	}

	if isWriteable(body.WriteMode, body.ObjectType) {
		body.WritePollRequired = boolean.NewTrue()
		body.EnableWriteable = boolean.NewTrue()
	} else {
		body = resetWriteableProperties(body)
	}

	if *body.AddressID < 1 || *body.AddressID > 65535 {
		return nil, errors.New("register must be between 1 and 65535")
	}

	isTypeBool := checkForBooleanType(body.ObjectType, body.DataType)
	body.IsTypeBool = nils.NewBool(isTypeBool)

	m.modbusDebugMsg(fmt.Sprintf("updatePoint() body: %+v\n", body))
	m.modbusDebugMsg(fmt.Sprintf("updatePoint() priority: %+v\n", body.Priority))

	if boolean.IsFalse(body.Enable) {
		body.CommonFault.InFault = true
		body.CommonFault.MessageLevel = dto.MessageLevel.Fail
		body.CommonFault.MessageCode = dto.CommonFaultCode.PointError
		body.CommonFault.Message = "point disabled"
		body.CommonFault.LastFail = time.Now().UTC()
	}
	body.CommonFault.InFault = false
	body.CommonFault.MessageLevel = dto.MessageLevel.Info
	body.CommonFault.MessageCode = dto.CommonFaultCode.PointWriteOk
	body.CommonFault.Message = fmt.Sprintf("last-updated: %s", utilstime.TimeStamp())
	body.CommonFault.LastOk = time.Now().UTC()
	point, err = m.grpcMarshaller.UpdatePoint(uuid, body)
	if err != nil || point == nil {
		m.modbusErrorMsg("updatePoint(): bad response from UpdatePoint() err:", err)
		return nil, err
	}

	dev, err := m.grpcMarshaller.GetDevice(point.DeviceUUID)
	if err != nil || dev == nil {
		m.modbusErrorMsg("updatePoint(): bad response from GetDevice()")
		return nil, err
	}

	netPollMan, err := m.getNetworkPollManagerByUUID(dev.NetworkUUID)
	if netPollMan == nil || err != nil {
		m.modbusErrorMsg("updatePoint(): cannot find NetworkPollManager for network: ", dev.NetworkUUID)
		_ = m.internalPointUpdateErr(point, "cannot find NetworkPollManager for network", dto.MessageLevel.Fail, dto.CommonFaultCode.SystemError)
		return
	}

	if boolean.IsTrue(point.Enable) && boolean.IsTrue(dev.Enable) {
		netPollMan.PollQueue.RemovePollingPointByPointUUID(point.UUID)
		pp := pollqueue.NewPollingPoint(point.UUID, point.DeviceUUID, dev.NetworkUUID)
		if pollqueue.PollOnStartCheck(point) {
			netPollMan.PollingPointCompleteNotification(pp, point, false, false, 0, true, true, pollqueue.NORMAL_RETRY, false)
		} else {
			netPollMan.PollingPointCompleteNotification(pp, point, true, true, 0, true, false, pollqueue.NORMAL_RETRY, true)
		}
	} else {
		netPollMan.PollQueue.RemovePollingPointByPointUUID(point.UUID)
	}
	return point, nil
}

func (m *Module) writePoint(pntUUID string, body *dto.PointWriter) (point *model.Point, err error) {
	m.modbusDebugMsg("writePoint(): ", pntUUID)
	if body == nil {
		m.modbusDebugMsg("writePoint(): nil point object")
		return
	}

	pnt, err := m.grpcMarshaller.PointWrite(pntUUID, body)
	if err != nil {
		m.modbusDebugMsg("writePoint(): bad response from WritePoint(), ", err)
		return nil, err
	}

	point = &pnt.Point
	dev, err := m.grpcMarshaller.GetDevice(point.DeviceUUID)
	if err != nil || dev == nil {
		m.modbusDebugMsg("writePoint(): bad response from GetDevice()")
		return nil, err
	}

	netPollMan, err := m.getNetworkPollManagerByUUID(dev.NetworkUUID)
	if netPollMan == nil || err != nil {
		m.modbusDebugMsg("writePoint(): cannot find NetworkPollManager for network: ", dev.NetworkUUID)
		_ = m.internalPointUpdateErr(point, err.Error(), dto.MessageLevel.Fail, dto.CommonFaultCode.SystemError)
		return nil, err
	}

	if boolean.IsTrue(point.Enable) {
		if pnt.IsWriteValueChange || point.WriteMode == datatype.WriteOnceReadOnce || point.WriteMode == datatype.WriteOnce || (point.WriteMode == datatype.WriteOnceThenRead && *point.WriteValue != *point.OriginalValue) { // if the write value has changed, we need to re-add the point so that it is polled asap (if required)
			if IsWriteable(point.WriteMode) {
				point.WritePollRequired = boolean.NewTrue()
			} else {
				point.WritePollRequired = boolean.NewFalse()
			}
			if point.WriteMode != datatype.WriteAlways && point.WriteMode != datatype.WriteOnce {
				point.ReadPollRequired = boolean.NewTrue()
			} else {
				point.ReadPollRequired = boolean.NewFalse()
			}
			point.CommonFault.InFault = false
			point.CommonFault.MessageLevel = dto.MessageLevel.Info
			point.CommonFault.MessageCode = dto.CommonFaultCode.PointWriteOk
			point.CommonFault.Message = fmt.Sprintf("last-updated: %s", utilstime.TimeStamp())
			point.CommonFault.LastOk = time.Now().UTC()
			point, err = m.grpcMarshaller.UpdatePoint(point.UUID, point)
			if err != nil || point == nil {
				m.modbusDebugMsg("writePoint(): bad response from UpdatePoint() err:", err)
				m.internalPointUpdateErr(point, fmt.Sprint("writePoint(): bad response from UpdatePoint() err:", err), dto.MessageLevel.Fail, dto.CommonFaultCode.SystemError)
				return point, err
			}
			pp := netPollMan.PollQueue.RemovePollingPointByPointUUID(point.UUID)
			if pp != nil { // this most likely fails when the device is disabled
				// pp.PollPriority = model.PRIORITY_ASAP   // TODO: THIS NEEDS TO BE IMPLEMENTED SO THAT ONLY MANUAL WRITES ARE PROMOTED TO ASAP PRIORITY
				netPollMan.PollingPointCompleteNotification(pp, point, false, false, 0, true, false, pollqueue.IMMEDIATE_RETRY, false) // This will perform the queue re-add actions based on Point WriteMode. TODO: check function of pointUpdate argument.
			}
		}
	}
	return point, nil
}

func (m *Module) deleteNetwork(uuid string) (ok bool, err error) {
	m.modbusDebugMsg("deleteNetwork(): ", uuid)
	if len(strings.TrimSpace(uuid)) == 0 {
		m.modbusDebugMsg("deleteNetwork(): uuid is empty")
		return
	}
	found := false
	for index, netPollMan := range m.NetworkPollManagers {
		if netPollMan.FFNetworkUUID == uuid {
			netPollMan.StopPolling()
			// Next remove the NetworkPollManager from the slice in polling instance
			m.NetworkPollManagers[index] = m.NetworkPollManagers[len(m.NetworkPollManagers)-1]
			m.NetworkPollManagers = m.NetworkPollManagers[:len(m.NetworkPollManagers)-1]
			found = true
		}
	}
	if !found {
		m.modbusDebugMsg("deleteNetwork(): cannot find NetworkPollManager for network: ", uuid)
	}
	err = m.grpcMarshaller.DeleteNetwork(uuid)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (m *Module) deleteDevice(body *model.Device) (ok bool, err error) {
	m.modbusDebugMsg("deleteDevice(): ", body.UUID)
	if body == nil {
		m.modbusDebugMsg("deleteDevice(): nil device object")
		return
	}

	netPollMan, err := m.getNetworkPollManagerByUUID(body.NetworkUUID)
	if netPollMan == nil || err != nil {
		m.modbusDebugMsg("deleteDevice(): cannot find NetworkPollManager for network: ", body.NetworkUUID)
		_ = m.deviceUpdateErr(body, "cannot find NetworkPollManager for network", dto.MessageLevel.Fail, dto.CommonFaultCode.SystemError)
		return
	}
	netPollMan.PollQueue.RemovePollingPointByDeviceUUID(body.UUID)
	err = m.grpcMarshaller.DeleteDevice(body.UUID)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (m *Module) deletePoint(body *model.Point) (ok bool, err error) {
	m.modbusDebugMsg("deletePoint(): ", body.UUID)
	if body == nil {
		m.modbusDebugMsg("deletePoint(): nil point object")
		return
	}

	dev, err := m.grpcMarshaller.GetDevice(body.DeviceUUID)
	if err != nil || dev == nil {
		m.modbusDebugMsg("addPoint(): bad response from GetDevice()")
		return false, err
	}

	netPollMan, err := m.getNetworkPollManagerByUUID(dev.NetworkUUID)
	if netPollMan == nil || err != nil {
		m.modbusDebugMsg("addPoint(): cannot find NetworkPollManager for network: ", dev.NetworkUUID)
		_ = m.internalPointUpdateErr(body, "cannot find NetworkPollManager for network", dto.MessageLevel.Fail, dto.CommonFaultCode.SystemError)
		return
	}

	netPollMan.PollQueue.RemovePollingPointByPointUUID(body.UUID)
	err = m.grpcMarshaller.DeletePoint(body.UUID)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (m *Module) internalPointUpdate(point *model.Point, value float64) (*model.Point, error) {
	pointWriter := &dto.PointWriter{
		OriginalValue: &value,
		Message:       fmt.Sprintf("last-updated: %s", utilstime.TimeStamp()),
		Fault:         false,
		PollState:     datatype.PointStatePollOk,
	}
	pnt, err := m.grpcMarshaller.PointWrite(point.UUID, pointWriter)
	if err != nil {
		m.modbusErrorMsg("internalPointUpdate() error: ", err)
		return nil, err
	}
	return &pnt.Point, nil
}

func (m *Module) internalPointUpdateErr(point *model.Point, message string, messageLevel string, messageCode string) error {
	if point == nil {
		return errors.New("point body can not be empty")
	}
	point.CommonFault.InFault = true
	point.CommonFault.MessageLevel = messageLevel
	point.CommonFault.MessageCode = messageCode
	point.CommonFault.Message = fmt.Sprintf("modbus: %s", message)
	point.CommonFault.LastFail = time.Now().UTC()
	err := m.grpcMarshaller.UpdatePointErrors(point.UUID, point)
	if err != nil {
		m.modbusErrorMsg("internalPointUpdateErr()", err)
	}
	return err
}

func (m *Module) deviceUpdateErr(device *model.Device, message string, messageLevel string, messageCode string) error {
	device.CommonFault.InFault = true
	device.CommonFault.MessageLevel = messageLevel
	device.CommonFault.MessageCode = messageCode
	device.CommonFault.Message = fmt.Sprintf("modbus: %s", message)
	device.CommonFault.LastFail = time.Now().UTC()
	err := m.grpcMarshaller.UpdateDeviceErrors(device.UUID, device)
	if err != nil {
		m.modbusErrorMsg(" deviceUpdateErr()", err)
	}
	return err
}

func (m *Module) networkUpdateMessage(network *model.Network, message string, messageLevel string, messageCode string) error {
	network.CommonFault.InFault = false
	network.CommonFault.MessageLevel = messageLevel
	network.CommonFault.MessageCode = messageCode
	network.CommonFault.Message = fmt.Sprintf("%s", message)
	network.CommonFault.LastOk = time.Now().UTC()
	_, err := m.grpcMarshaller.UpdateNetwork(network.UUID, network)
	if err != nil {
		m.modbusErrorMsg(" networkUpdate()", err)
	}
	return err
}

func (m *Module) networkUpdateErr(network *model.Network, message string, messageLevel string, messageCode string) error {
	network.CommonFault.InFault = true
	network.CommonFault.MessageLevel = messageLevel
	network.CommonFault.MessageCode = messageCode
	network.CommonFault.Message = fmt.Sprintf("%s", message)
	network.CommonFault.LastFail = time.Now().UTC()
	err := m.grpcMarshaller.UpdateNetworkErrors(network.UUID, network)
	if err != nil {
		m.modbusErrorMsg(" networkUpdateErr()", err)
	}
	return err
}

func (m *Module) getPollingStats(networkName string) (result *dto.PollQueueStatistics, error error) {
	if len(m.NetworkPollManagers) == 0 {
		return nil, errors.New("couldn't find any plugin network poll managers")
	}
	for _, netPollMan := range m.NetworkPollManagers {
		if netPollMan == nil || netPollMan.NetworkName != networkName {
			continue
		}
		result = netPollMan.GetPollingQueueStatistics()
		return result, nil
	}
	return nil, errors.New(fmt.Sprintf("couldn't find network %s for polling statistics", networkName))
}
