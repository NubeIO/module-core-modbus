package pkg

import (
	"errors"
	"fmt"
	"github.com/NubeIO/nubeio-rubix-lib-helpers-go/pkg/nils"
	"github.com/NubeIO/nubeio-rubix-lib-helpers-go/pkg/times/utilstime"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/pkg/v1/model"
	argspkg "github.com/NubeIO/rubix-os/args"
	"github.com/NubeIO/rubix-os/interfaces"
	"github.com/NubeIO/rubix-os/module/shared/pollqueue"
	"github.com/NubeIO/rubix-os/utils/array"
	"github.com/NubeIO/rubix-os/utils/boolean"
	"github.com/NubeIO/rubix-os/utils/float"
	"github.com/NubeIO/rubix-os/utils/writemode"
	"go.bug.st/serial"
	"strings"
	"time"
)

var path = "modbus"

func (m *Module) addNetwork(body *model.Network) (network *model.Network, err error) {
	m.modbusDebugMsg("addNetwork(): ", body.Name)
	body.HasPollingStatistics = true

	network, err = m.grpcMarshaller.CreateNetwork(body)
	if err != nil {
		m.modbusErrorMsg("addNetwork(): failed to create modbus network: ", body.Name)
		return nil, errors.New("failed to create modbus network")
	}

	if boolean.IsTrue(network.Enable) {
		conf := m.GetConfig().(*Config)
		pollQueueConfig := pollqueue.Config{
			EnablePolling: conf.EnablePolling,
			LogLevel:      conf.LogLevel,
		}
		pollManager := NewPollManager(
			&pollQueueConfig,
			m.grpcMarshaller,
			network.UUID,
			network.Name,
			m.pluginUUID,
			m.moduleName,
			float.NonNil(network.MaxPollRate),
		)
		pollManager.StartPolling()
		m.NetworkPollManagers = append(m.NetworkPollManagers, pollManager)
	} else {
		err = m.networkUpdateErr(
			network,
			"network disabled",
			model.MessageLevel.Warning,
			model.CommonFaultCode.NetworkError,
		)
		err = m.grpcMarshaller.SetErrorsForAllDevicesOnNetwork(
			network.UUID,
			"network disabled",
			model.MessageLevel.Warning,
			model.CommonFaultCode.NetworkError,
			true,
		)
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
		err = m.deviceUpdateErr(device, "device disabled", model.MessageLevel.Warning, model.CommonFaultCode.DeviceError)
		err = m.grpcMarshaller.SetErrorsForAllPointsOnDevice(device.UUID, "device disabled", model.MessageLevel.Warning, model.CommonFaultCode.DeviceError)
	}

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
		body.WritePollRequired = boolean.NewTrue()
		body.EnableWriteable = boolean.NewTrue()
	} else {
		body = writemode.ResetWriteableProperties(body)
	}
	body.ReadPollRequired = boolean.NewTrue()

	isTypeBool := checkForBooleanType(body.ObjectType, body.DataType)
	body.IsTypeBool = nils.NewBool(isTypeBool)

	isOutput := checkForOutputType(body.ObjectType)
	body.IsOutput = nils.NewBool(isOutput)

	// point, err = m.grpcMarshaller.CreatePoint(body, true)
	point, err = m.grpcMarshaller.CreatePoint(body)
	if err != nil {
		return nil, err
	}
	point, err = m.grpcMarshaller.UpdatePoint(point.UUID, point)
	if point == nil || err != nil {
		m.modbusDebugMsg("addPoint(): failed to create modbus point: ", body.Name)
		return nil, errors.New(fmt.Sprint("failed to create modbus point. err: ", err))
	}
	m.modbusDebugMsg(fmt.Sprintf("addPoint(): %+v\n", point))

	dev, err := m.grpcMarshaller.GetDevice(point.DeviceUUID, argspkg.Args{})
	if err != nil || dev == nil {
		m.modbusDebugMsg("addPoint(): bad response from GetDevice()")
		return nil, err
	}

	netPollMan, err := m.getNetworkPollManagerByUUID(dev.NetworkUUID)
	if netPollMan == nil || err != nil {
		m.modbusDebugMsg("addPoint(): cannot find NetworkPollManager for network: ", dev.NetworkUUID)
		return
	}

	if boolean.IsTrue(point.Enable) {
		netPollMan.PollQueue.RemovePollingPointByPointUUID(point.UUID)
		// DO POLLING ENABLE ACTIONS FOR POINT
		pp := pollqueue.NewPollingPoint(point.UUID, point.DeviceUUID, dev.NetworkUUID, netPollMan.FFPluginUUID)
		// This will perform the queue re-add actions based on Point WriteMode.
		// TODO: Check function of pointUpdate argument.
		netPollMan.PollingPointCompleteNotification(
			pp,
			false,
			false,
			0,
			true,
			true,
			pollqueue.NORMAL_RETRY,
			false,
			false,
			true,
		)
	} else {
		err = m.pointUpdateErr(point, "point disabled", model.MessageLevel.Warning, model.CommonFaultCode.PointError)
	}

	return point, nil
}

func (m *Module) updateNetwork(body *model.Network) (network *model.Network, err error) {
	m.modbusDebugMsg("updateNetwork(): ", body.UUID)
	if body == nil {
		m.modbusDebugMsg("updateNetwork():  nil network object")
		return
	}

	// Indicates that ui should display polling statistics
	body.HasPollingStatistics = true

	if boolean.IsFalse(body.Enable) {
		body.CommonFault.InFault = true
		body.CommonFault.MessageLevel = model.MessageLevel.Warning
		body.CommonFault.MessageCode = model.CommonFaultCode.NetworkError
		body.CommonFault.Message = "network disabled"
		body.CommonFault.LastFail = time.Now().UTC()
	} else {
		body.CommonFault.InFault = false
		body.CommonFault.MessageLevel = model.MessageLevel.Info
		body.CommonFault.MessageCode = model.CommonFaultCode.Ok
		body.CommonFault.Message = ""
		body.CommonFault.LastOk = time.Now().UTC()
	}

	network, err = m.grpcMarshaller.UpdateNetwork(body.UUID, body)
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
		m.grpcMarshaller.SetErrorsForAllDevicesOnNetwork(
			network.UUID,
			"network disabled",
			model.MessageLevel.Warning,
			model.CommonFaultCode.DeviceError,
			true,
		)
	} else if restartPolling || (boolean.IsTrue(network.Enable) && netPollMan.Enable == false) {
		if restartPolling {
			netPollMan.StopPolling()
		}
		// DO POLLING Enable ACTIONS
		netPollMan.StartPolling()
		m.grpcMarshaller.ClearErrorsForAllDevicesOnNetwork(network.UUID, true)
	}

	network, err = m.grpcMarshaller.UpdateNetwork(body.UUID, network)
	if err != nil || network == nil {
		return nil, err
	}
	return network, nil
}

func (m *Module) updateDevice(body *model.Device) (device *model.Device, err error) {
	m.modbusDebugMsg("updateDevice(): ", body.UUID)
	if body == nil {
		m.modbusDebugMsg("updateDevice(): nil device object")
		return
	}

	if boolean.IsFalse(body.Enable) {
		body.CommonFault.InFault = true
		body.CommonFault.MessageLevel = model.MessageLevel.Warning
		body.CommonFault.MessageCode = model.CommonFaultCode.DeviceError
		body.CommonFault.Message = "device disabled"
		body.CommonFault.LastFail = time.Now().UTC()
	} else {
		body.CommonFault.InFault = false
		body.CommonFault.MessageLevel = model.MessageLevel.Info
		body.CommonFault.MessageCode = model.CommonFaultCode.Ok
		body.CommonFault.LastOk = time.Now().UTC()
	}

	device, err = m.grpcMarshaller.UpdateDevice(body.UUID, body)
	if err != nil || device == nil {
		return nil, err
	}

	if boolean.IsTrue(device.Enable) { // If Enabled we need to GetDevice so we get Points
		device, err = m.grpcMarshaller.GetDevice(device.UUID, argspkg.Args{})
		if err != nil || device == nil {
			return nil, err
		}
	}

	netPollMan, err := m.getNetworkPollManagerByUUID(device.NetworkUUID)
	if netPollMan == nil || err != nil {
		m.modbusDebugMsg("updateDevice(): cannot find NetworkPollManager for network: ", device.NetworkUUID)
		return
	}
	if boolean.IsFalse(device.Enable) && netPollMan.PollQueue.CheckIfActiveDevicesListIncludes(device.UUID) {
		// DO POLLING DISABLE ACTIONS FOR DEVICE
		m.grpcMarshaller.SetErrorsForAllPointsOnDevice(
			device.UUID,
			"device disabled",
			model.MessageLevel.Warning,
			model.CommonFaultCode.DeviceError,
		)
		netPollMan.PollQueue.RemovePollingPointByDeviceUUID(device.UUID)

	} else if boolean.IsTrue(device.Enable) && !netPollMan.PollQueue.CheckIfActiveDevicesListIncludes(device.UUID) {
		// DO POLLING ENABLE ACTIONS FOR DEVICE
		err = m.grpcMarshaller.ClearErrorsForAllPointsOnDevice(device.UUID)
		if err != nil {
			m.modbusDebugMsg("updateDevice(): error on ClearErrorsForAllPointsOnDevice(): ", err)
		}
		for _, pnt := range device.Points {
			if boolean.IsTrue(pnt.Enable) {
				pp := pollqueue.NewPollingPoint(pnt.UUID, pnt.DeviceUUID, device.NetworkUUID, netPollMan.FFPluginUUID)
				// This will perform the queue re-add actions based on Point WriteMode.
				// TODO: check function of pointUpdate argument.
				netPollMan.PollingPointCompleteNotification(
					pp,
					false,
					false,
					0,
					true,
					true,
					pollqueue.NORMAL_RETRY,
					false,
					false,
					true,
				)
			}
		}

	} else if boolean.IsTrue(device.Enable) {
		// TODO: Currently on every device update, all device points are removed, and re-added.
		device.CommonFault.InFault = false
		device.CommonFault.MessageLevel = model.MessageLevel.Info
		device.CommonFault.MessageCode = model.CommonFaultCode.Ok
		device.CommonFault.Message = ""
		device.CommonFault.LastOk = time.Now().UTC()
		netPollMan.PollQueue.RemovePollingPointByDeviceUUID(device.UUID)
		for _, pnt := range device.Points {
			if boolean.IsTrue(pnt.Enable) {
				pp := pollqueue.NewPollingPoint(pnt.UUID, pnt.DeviceUUID, device.NetworkUUID, netPollMan.FFPluginUUID)
				// This will perform the queue re-add actions based on Point WriteMode.
				// TODO: check function of pointUpdate argument.
				netPollMan.PollingPointCompleteNotification(
					pp,
					false,
					false,
					0,
					true,
					true,
					pollqueue.NORMAL_RETRY,
					false,
					false,
					true,
				)
			}
		}
	}

	// TODO: NEED TO ACCOUNT FOR OTHER CHANGES ON DEVICE.
	//  It would be useful to have a way to know if the device polling rates were changed.
	device, err = m.grpcMarshaller.UpdateDevice(device.UUID, device)
	if err != nil {
		return nil, err
	}
	return device, nil
}

func (m *Module) updatePoint(body *model.Point) (point *model.Point, err error) {
	m.modbusDebugMsg("updatePoint(): ", body.UUID)
	if body == nil {
		m.modbusDebugMsg("updatePoint(): nil point object")
		return
	}

	if isWriteable(body.WriteMode, body.ObjectType) {
		body.WritePollRequired = boolean.NewTrue()
		body.EnableWriteable = boolean.NewTrue()
	} else {
		body = writemode.ResetWriteableProperties(body)
	}

	isTypeBool := checkForBooleanType(body.ObjectType, body.DataType)
	body.IsTypeBool = nils.NewBool(isTypeBool)

	m.modbusDebugMsg(fmt.Sprintf("updatePoint() body: %+v\n", body))
	m.modbusDebugMsg(fmt.Sprintf("updatePoint() priority: %+v\n", body.Priority))

	if boolean.IsFalse(body.Enable) {
		body.CommonFault.InFault = true
		body.CommonFault.MessageLevel = model.MessageLevel.Fail
		body.CommonFault.MessageCode = model.CommonFaultCode.PointError
		body.CommonFault.Message = "point disabled"
		body.CommonFault.LastFail = time.Now().UTC()
	}
	body.CommonFault.InFault = false
	body.CommonFault.MessageLevel = model.MessageLevel.Info
	body.CommonFault.MessageCode = model.CommonFaultCode.PointWriteOk
	body.CommonFault.Message = fmt.Sprintf("last-updated: %s", utilstime.TimeStamp())
	body.CommonFault.LastOk = time.Now().UTC()
	point, err = m.grpcMarshaller.UpdatePoint(body.UUID, body)
	if err != nil || point == nil {
		m.modbusErrorMsg("updatePoint(): bad response from UpdatePoint() err:", err)
		return nil, err
	}

	dev, err := m.grpcMarshaller.GetDevice(point.DeviceUUID, argspkg.Args{})
	if err != nil || dev == nil {
		m.modbusErrorMsg("updatePoint(): bad response from GetDevice()")
		return nil, err
	}

	netPollMan, err := m.getNetworkPollManagerByUUID(dev.NetworkUUID)
	if netPollMan == nil || err != nil {
		m.modbusErrorMsg("updatePoint(): cannot find NetworkPollManager for network: ", dev.NetworkUUID)
		_ = m.pointUpdateErr(
			point,
			"cannot find NetworkPollManager for network",
			model.MessageLevel.Fail,
			model.CommonFaultCode.SystemError,
		)
		return
	}

	if boolean.IsTrue(point.Enable) && boolean.IsTrue(dev.Enable) {
		netPollMan.PollQueue.RemovePollingPointByPointUUID(point.UUID)
		// DO POLLING ENABLE ACTIONS FOR POINT
		// TODO: Review these steps to check that UpdatePollingPointByUUID might work better?
		pp := pollqueue.NewPollingPoint(point.UUID, point.DeviceUUID, dev.NetworkUUID, netPollMan.FFPluginUUID)
		// This will perform the queue re-add actions based on Point WriteMode.
		// TODO: Check function of pointUpdate argument.
		netPollMan.PollingPointCompleteNotification(
			pp,
			false,
			false,
			0,
			true,
			true,
			pollqueue.NORMAL_RETRY,
			false,
			false,
			true,
		)
	} else {
		// DO POLLING DISABLE ACTIONS FOR POINT
		netPollMan.PollQueue.RemovePollingPointByPointUUID(point.UUID)
	}
	return point, nil
}

func (m *Module) writePoint(pntUUID string, body *model.PointWriter) (point *model.Point, err error) {
	// TODO: Check for PointWriteByName calls that might not flow through the plugin.
	point = nil
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

	dev, err := m.grpcMarshaller.GetDevice(point.DeviceUUID, argspkg.Args{})
	if err != nil || dev == nil {
		m.modbusDebugMsg("writePoint(): bad response from GetDevice()")
		return nil, err
	}

	netPollMan, err := m.getNetworkPollManagerByUUID(dev.NetworkUUID)
	if netPollMan == nil || err != nil {
		m.modbusDebugMsg("writePoint(): cannot find NetworkPollManager for network: ", dev.NetworkUUID)
		_ = m.pointUpdateErr(point, err.Error(), model.MessageLevel.Fail, model.CommonFaultCode.SystemError)
		return nil, err
	}

	if boolean.IsTrue(point.Enable) {
		// If the write value has changed, we need to re-add the point so that it is polled asap (if required)
		if pnt.IsWriteValueChange ||
			point.WriteMode == model.WriteOnceReadOnce ||
			point.WriteMode == model.WriteOnce ||
			(point.WriteMode == model.WriteOnceThenRead && *point.WriteValue != *point.OriginalValue) {
			pp, _ := netPollMan.PollQueue.RemovePollingPointByPointUUID(point.UUID)
			if pp == nil {
				if netPollMan.PollQueue.OutstandingPollingPoints.GetPollingPointIndexByPointUUID(point.UUID) > -1 {
					if writemode.IsWriteable(point.WriteMode) {
						netPollMan.PollQueue.PointsUpdatedWhilePolling[point.UUID] = true // This triggers a write post at ASAP priority (for writeable points).
						point.WritePollRequired = boolean.NewTrue()
						if point.WriteMode != model.WriteAlways && point.WriteMode != model.WriteOnce {
							point.ReadPollRequired = boolean.NewTrue()
						} else {
							point.ReadPollRequired = boolean.NewFalse()
						}
					} else {
						netPollMan.PollQueue.PointsUpdatedWhilePolling[point.UUID] = false
						point.WritePollRequired = boolean.NewFalse()
					}
					point.CommonFault.InFault = false
					point.CommonFault.MessageLevel = model.MessageLevel.Info
					point.CommonFault.MessageCode = model.CommonFaultCode.PointWriteOk
					point.CommonFault.Message = fmt.Sprintf("last-updated: %s", utilstime.TimeStamp())
					point.CommonFault.LastOk = time.Now().UTC()
					point, err = m.grpcMarshaller.UpdatePoint(point.UUID, point)
					if err != nil || point == nil {
						m.modbusDebugMsg("writePoint(): bad response from UpdatePoint() err:", err)
						_ = m.pointUpdateErr(point, fmt.Sprint("writePoint(): cannot find PollingPoint for point: ", point.UUID), model.MessageLevel.Fail, model.CommonFaultCode.SystemError)
						return point, err
					}
					return point, nil
				} else {
					m.modbusDebugMsg("writePoint(): cannot find PollingPoint for point (could be out for polling: ", point.UUID)
					_ = m.pointUpdateErr(point, "writePoint(): cannot find PollingPoint for point: ", model.MessageLevel.Fail, model.CommonFaultCode.PointWriteError)
					return point, err
				}
			}
			if writemode.IsWriteable(point.WriteMode) {
				point.WritePollRequired = boolean.NewTrue()
			} else {
				point.WritePollRequired = boolean.NewFalse()
			}
			if point.WriteMode != model.WriteAlways && point.WriteMode != model.WriteOnce {
				point.ReadPollRequired = boolean.NewTrue()
			} else {
				point.ReadPollRequired = boolean.NewFalse()
			}
			point.CommonFault.InFault = false
			point.CommonFault.MessageLevel = model.MessageLevel.Info
			point.CommonFault.MessageCode = model.CommonFaultCode.PointWriteOk
			point.CommonFault.Message = fmt.Sprintf("last-updated: %s", utilstime.TimeStamp())
			point.CommonFault.LastOk = time.Now().UTC()
			point, err = m.grpcMarshaller.UpdatePoint(point.UUID, point)
			if err != nil || point == nil {
				m.modbusDebugMsg("writePoint(): bad response from UpdatePoint() err:", err)
				_ = m.pointUpdateErr(point, fmt.Sprint("writePoint(): bad response from UpdatePoint() err:", err), model.MessageLevel.Fail, model.CommonFaultCode.SystemError)
				return point, err
			}

			// pp.PollPriority = model.PRIORITY_ASAP   // TODO: THIS NEEDS TO BE IMPLEMENTED SO THAT ONLY MANUAL WRITES ARE PROMOTED TO ASAP PRIORITY

			// This will perform the queue re-add actions based on Point WriteMode.
			// TODO: Check function of pointUpdate argument.
			netPollMan.PollingPointCompleteNotification(
				pp,
				false,
				false,
				0,
				true,
				false,
				pollqueue.IMMEDIATE_RETRY,
				false,
				false,
				true,
			)
			// netPollMan.PollQueue.AddPollingPoint(pp)
			// netPollMan.PollQueue.UpdatePollingPointByPointUUID(point.UUID, model.PRIORITY_ASAP)

			/*
				netPollMan.PollQueue.RemovePollingPointByPointUUID(body.UUID)
				//DO POLLING ENABLE ACTIONS FOR POINT
				// TODO: Review these steps to check that UpdatePollingPointByUUID might work better?
				pp := pollqueue.NewPollingPoint(body.UUID, body.DeviceUUID, dev.NetworkUUID, netPollMan.FFPluginUUID)
				netPollMan.PollingPointCompleteNotification(pp, false, false, 0, true, true) // This will perform the queue re-add actions based on Point WriteMode. TODO: check function of pointUpdate argument.
				//netPollMan.PollQueue.AddPollingPoint(pp)
				//netPollMan.SetPointPollRequiredFlagsBasedOnWriteMode(pnt)
			*/
		}
	} else {
		// DO POLLING DISABLE ACTIONS FOR POINT
		netPollMan.PollQueue.RemovePollingPointByPointUUID(pntUUID)
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
		_ = m.deviceUpdateErr(body, "cannot find NetworkPollManager for network", model.MessageLevel.Fail, model.CommonFaultCode.SystemError)
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

	dev, err := m.grpcMarshaller.GetDevice(body.DeviceUUID, argspkg.Args{})
	if err != nil || dev == nil {
		m.modbusDebugMsg("addPoint(): bad response from GetDevice()")
		return false, err
	}

	netPollMan, err := m.getNetworkPollManagerByUUID(dev.NetworkUUID)
	if netPollMan == nil || err != nil {
		m.modbusDebugMsg("addPoint(): cannot find NetworkPollManager for network: ", dev.NetworkUUID)
		_ = m.pointUpdateErr(body, "cannot find NetworkPollManager for network", model.MessageLevel.Fail, model.CommonFaultCode.SystemError)
		return
	}

	netPollMan.PollQueue.RemovePollingPointByPointUUID(body.UUID)
	otherPointsOnSameDeviceExist := netPollMan.PollQueue.CheckPollingQueueForDevUUID(body.DeviceUUID)
	if !otherPointsOnSameDeviceExist {
		netPollMan.PollQueue.RemoveDeviceFromActiveDevicesList(body.DeviceUUID)
	}
	err = m.grpcMarshaller.DeletePoint(body.UUID)
	if err != nil {
		return false, err
	}
	return true, nil
}

// THE FOLLOWING FUNCTIONS ARE CALLED FROM WITHIN THE PLUGIN
func (m *Module) pointUpdate(point *model.Point, value float64, readSuccess bool) (*model.Point, error) {
	if readSuccess {
		point.OriginalValue = float.New(value)
	}
	opts := interfaces.UpdatePointOpts{
		WriteValue: true,
	}
	_, err := m.grpcMarshaller.UpdatePoint(point.UUID, point, opts)
	if err != nil {
		m.modbusDebugMsg("MODBUS UPDATE POINT pointUpdate() error: ", err)
		return nil, err
	}
	return point, nil
}

func (m *Module) pointUpdateErr(point *model.Point, message string, messageLevel string, messageCode string) error {
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
		m.modbusErrorMsg(" pointUpdateErr()", err)
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

func (m *Module) listSerialPorts() (*array.Array, error) {
	ports, err := serial.GetPortsList()
	p := array.NewArray()
	for _, port := range ports {
		p.Add(port)
	}
	return p, err
}

func (m *Module) getPollingStats(networkName string) (result *model.PollQueueStatistics, error error) {
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
