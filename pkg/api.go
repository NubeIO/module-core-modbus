package pkg

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/NubeIO/lib-schema/modbuschema"
	"github.com/NubeIO/nubeio-rubix-lib-helpers-go/pkg/uurl"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/pkg/v1/model"
	argspkg "github.com/NubeIO/rubix-os/args"
	"github.com/NubeIO/rubix-os/module/common"
	"strings"
)

const (
	listSerial          = "/list/serial"
	jsonSchemaNetwork   = "/schema/json/network"
	jsonSchemaDevice    = "/schema/json/device"
	jsonSchemaPoint     = "/schema/json/point"
	pointOperation      = "/modbus/point/operation"
	wizardTcp           = "/modbus/wizard/tcp"
	wizardSerial        = "/modbus/wizard/serial"
	networkPollingStats = "/polling/stats/network/"
)

type Scan struct {
	Start  uint32 `json:"start"`
	Count  uint32 `json:"count"`
	IsCoil bool   `json:"is_coil"`
}

type Body struct {
	Network       *model.Network
	Device        *model.Device
	Point         *model.Point
	Client        `json:"client"`
	Operation     `json:"request_body"`
	Scan          `json:"scan"`
	ReturnArray   bool  `json:"return_array"`
	IsSerial      bool  `json:"is_serial"`
	DeviceAddress uint8 `json:"device_address"`
}

type wizard struct {
	IP            string `json:"ip"`
	Port          int    `json:"port"`
	SerialPort    string `json:"serial_port"`
	BaudRate      uint   `json:"baud_rate"`
	DeviceAddr    uint   `json:"device_addr"`
	WizardVersion uint   `json:"wizard_version"`
	NameArg       string `json:"name_arg"`
	AddArg        uint   `json:"add_arg"`
}

func (m *Module) Get(path string) ([]byte, error) {
	if path == jsonSchemaNetwork {
		return json.Marshal(modbuschema.GetNetworkSchema())
	} else if path == jsonSchemaDevice {
		return json.Marshal(modbuschema.GetDeviceSchema())
	} else if path == jsonSchemaPoint {
		return json.Marshal(modbuschema.GetPointSchema())
	} else if path == listSerial {
		serial, err := m.listSerialPorts()
		if err != nil {
			return nil, err
		}
		return json.Marshal(serial)
	} else if strings.Contains(path, networkPollingStats) {
		name := strings.TrimPrefix(path, networkPollingStats)
		stats, err := m.getPollingStats(name)
		if err != nil {
			return nil, err
		}
		return json.Marshal(stats)
	}
	return nil, errors.New("not found")
}

func (m *Module) Post(path string, body []byte) ([]byte, error) {
	if path == common.NetworksURL {
		var network *model.Network
		err := json.Unmarshal(body, &network)
		if err != nil {
			return nil, err
		}
		net, err := m.addNetwork(network)
		if err != nil {
			return nil, err
		}
		return json.Marshal(net)
	} else if path == common.DevicesURL {
		var device *model.Device
		err := json.Unmarshal(body, &device)
		if err != nil {
			return nil, err
		}
		dev, err := m.addDevice(device)
		if err != nil {
			return nil, err
		}
		return json.Marshal(dev)
	} else if path == common.PointsURL {
		var point *model.Point
		err := json.Unmarshal(body, &point)
		if err != nil {
			return nil, err
		}
		pnt, err := m.addPoint(point)
		if err != nil {
			return nil, err
		}
		return json.Marshal(pnt)
	} else if path == pointOperation {
		var dto Body
		err := json.Unmarshal(body, &dto)
		if err != nil {
			return nil, err
		}
		netType := dto.Network.TransportType
		mbClient, err := m.setClient(dto.Network, dto.Device, false)
		if err != nil {
			m.modbusErrorMsg(err, "ERROR ON set modbus client")
			return nil, err
		}
		if netType == model.TransType.Serial || netType == model.TransType.LoRa {
			if dto.Device.AddressId >= 1 {
				mbClient.RTUClientHandler.SlaveID = byte(dto.Device.AddressId)
			}
		} else if netType == model.TransType.IP {
			url, err := uurl.JoinIpPort(dto.Device.Host, dto.Device.Port)
			if err != nil {
				m.modbusErrorMsg(fmt.Sprintf("failed to validate device IP %s\n", url))
				return nil, err
			}
			mbClient.TCPClientHandler.Address = url
			mbClient.TCPClientHandler.SlaveID = byte(dto.Device.AddressId)
		}
		_, responseValue, err := m.networkRequest(mbClient, dto.Point, false)
		if err != nil {
			return nil, err
		}
		m.modbusDebugMsg("responseValue", responseValue)
		return json.Marshal(responseValue)
	} else if path == wizardTcp {
		var dto wizard
		err := json.Unmarshal(body, &dto)
		if err != nil {
			return nil, err
		}
		n, err := m.wizardTCP(dto)
		if err != nil {
			return nil, err
		}
		return json.Marshal(n)
	} else if path == wizardSerial {
		var dto wizard
		err := json.Unmarshal(body, &dto)
		if err != nil {
			return nil, err
		}
		serial, err := m.wizardSerial(dto)
		if err != nil {
			return nil, err
		}
		return json.Marshal(serial)
	}
	return nil, errors.New("not found")
}

func (m *Module) Put(path, uuid string, body []byte) ([]byte, error) {
	return nil, errors.New("not found")
}

func (m *Module) Patch(path, uuid string, body []byte) ([]byte, error) {
	if path == common.NetworksURL {
		var network *model.Network
		err := json.Unmarshal(body, &network)
		if err != nil {
			return nil, err
		}
		net, err := m.updateNetwork(network)
		if err != nil {
			return nil, err
		}
		return json.Marshal(net)
	} else if path == common.DevicesURL {
		var device *model.Device
		err := json.Unmarshal(body, &device)
		if err != nil {
			return nil, err
		}
		dev, err := m.updateDevice(device)
		if err != nil {
			return nil, err
		}
		return json.Marshal(dev)
	} else if path == common.PointsURL {
		var point *model.Point
		err := json.Unmarshal(body, &point)
		if err != nil {
			return nil, err
		}
		pnt, err := m.updatePoint(point)
		if err != nil {
			return nil, err
		}
		return json.Marshal(pnt)
	} else if path == common.PointsWriteURL {
		var pw *model.PointWriter
		err := json.Unmarshal(body, &pw)
		if err != nil {
			return nil, err
		}
		pnt, err := m.writePoint(uuid, pw)
		if err != nil {
			return nil, err
		}
		return json.Marshal(pnt)
	}
	return nil, errors.New("not found")
}

func (m *Module) Delete(path, uuid string) ([]byte, error) {
	if path == common.NetworksURL {
		_, err := m.deleteNetwork(uuid)
		return nil, err
	} else if path == common.DevicesURL {
		dev, err := m.grpcMarshaller.GetDevice(uuid, argspkg.Args{})
		if err != nil {
			return nil, err
		}
		_, err = m.deleteDevice(dev)
		return nil, err
	} else if path == common.PointsURL {
		pnt, err := m.grpcMarshaller.GetPoint(uuid, argspkg.Args{})
		if err != nil {
			return nil, err
		}
		_, err = m.deletePoint(pnt)
		return nil, err
	}
	return nil, errors.New("not found")
}
