package pkg

import (
	"encoding/json"
	"fmt"
	"github.com/NubeIO/lib-module-go/http"
	"github.com/NubeIO/lib-module-go/module"
	"github.com/NubeIO/lib-module-go/router"
	"github.com/NubeIO/module-core-modbus/schema/modbus"
	"github.com/NubeIO/nubeio-rubix-lib-helpers-go/pkg/uurl"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/nargs"
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

var route *router.Router

func (m *Module) CallModule(
	method http.Method,
	api string,
	args nargs.Args,
	body []byte,
) ([]byte, error) {
	mod := (module.Module)(m)
	return route.CallHandler(&mod, method, api, args, body)
}

func InitRouter() {
	route = router.NewRouter()

	route.Handle(http.GET, "/api/network/schema", GetNetworkSchema)
	route.Handle(http.GET, "/api/devices/schema", GetDeviceSchema)
	route.Handle(http.GET, "/api/points/schema", GetPointSchema)

	route.Handle(http.POST, "/api/networks", CreateNetwork)
	route.Handle(http.PATCH, "/api/networks/:uuid", UpdateNetwork)
	route.Handle(http.DELETE, "/api/networks/:uuid", DeleteNetwork)

	route.Handle(http.POST, "/api/devices", CreateDevice)
	route.Handle(http.PATCH, "/api/devices/:uuid", UpdateDevice)
	route.Handle(http.DELETE, "/api/devices/:uuid", DeleteDevice)

	route.Handle(http.POST, "/api/points", CreatePoint)
	route.Handle(http.PATCH, "/api/points/:uuid", UpdatePoint)
	route.Handle(http.DELETE, "/api/points/:uuid", DeletePoint)
	route.Handle(http.PATCH, "/api/points/:uuid/write", PointWrite)

	route.Handle(http.GET, "/api/list/serial", GetListSerial)
	route.Handle(http.GET, "/api/polling/stats/network/name/:name", GetNetworkPollingStats)
	route.Handle(http.POST, "/api/modbus/point/operation", CreatePointOperation)
	route.Handle(http.POST, "/api/modbus/wizard/tcp", CreateWizardTcp)
	route.Handle(http.POST, "/api/modbus/wizard/serial", CreateWizardSerial)
}

func GetNetworkSchema(m *module.Module, r *router.Request) ([]byte, error) {
	return json.Marshal(modbus.GetNetworkSchema())
}

func GetDeviceSchema(m *module.Module, r *router.Request) ([]byte, error) {
	return json.Marshal(modbus.GetDeviceSchema())
}

func GetPointSchema(m *module.Module, r *router.Request) ([]byte, error) {
	return json.Marshal(modbus.GetPointSchema())
}

func CreateNetwork(m *module.Module, r *router.Request) ([]byte, error) {
	var network *model.Network
	err := json.Unmarshal(r.Body, &network)
	if err != nil {
		return nil, err
	}
	net, err := (*m).(*Module).addNetwork(network)
	if err != nil {
		return nil, err
	}
	return json.Marshal(net)
}

func UpdateNetwork(m *module.Module, r *router.Request) ([]byte, error) {
	var network *model.Network
	err := json.Unmarshal(r.Body, &network)
	if err != nil {
		return nil, err
	}
	net, err := (*m).(*Module).updateNetwork(r.Params["uuid"], network)
	if err != nil {
		return nil, err
	}
	return json.Marshal(net)
}

func DeleteNetwork(m *module.Module, r *router.Request) ([]byte, error) {
	ok, err := (*m).(*Module).deleteNetwork(r.Params["uuid"])
	if err != nil {
		return nil, err
	}
	return json.Marshal(ok)
}

func CreateDevice(m *module.Module, r *router.Request) ([]byte, error) {
	var device *model.Device
	err := json.Unmarshal(r.Body, &device)
	if err != nil {
		return nil, err
	}
	dev, err := (*m).(*Module).addDevice(device)
	if err != nil {
		return nil, err
	}
	return json.Marshal(dev)
}

func UpdateDevice(m *module.Module, r *router.Request) ([]byte, error) {
	var device *model.Device
	err := json.Unmarshal(r.Body, &device)
	if err != nil {
		return nil, err
	}
	dev, err := (*m).(*Module).updateDevice(r.Params["uuid"], device)
	if err != nil {
		return nil, err
	}
	return json.Marshal(dev)
}

func DeleteDevice(m *module.Module, r *router.Request) ([]byte, error) {
	dev, err := (*m).(*Module).grpcMarshaller.GetDevice(r.Params["uuid"], nargs.Args{})
	if err != nil {
		return nil, err
	}
	ok, err := (*m).(*Module).deleteDevice(dev)
	if err != nil {
		return nil, err
	}
	return json.Marshal(ok)
}

func CreatePoint(m *module.Module, r *router.Request) ([]byte, error) {
	var point *model.Point
	err := json.Unmarshal(r.Body, &point)
	if err != nil {
		return nil, err
	}
	pnt, err := (*m).(*Module).addPoint(point)
	if err != nil {
		return nil, err
	}
	return json.Marshal(pnt)
}

func UpdatePoint(m *module.Module, r *router.Request) ([]byte, error) {
	var point *model.Point
	err := json.Unmarshal(r.Body, &point)
	if err != nil {
		return nil, err
	}
	pnt, err := (*m).(*Module).updatePoint(r.Params["uuid"], point)
	if err != nil {
		return nil, err
	}
	return json.Marshal(pnt)
}

func DeletePoint(m *module.Module, r *router.Request) ([]byte, error) {
	pnt, err := (*m).(*Module).grpcMarshaller.GetPoint(r.Params["uuid"], nargs.Args{})
	if err != nil {
		return nil, err
	}
	ok, err := (*m).(*Module).deletePoint(pnt)
	if err != nil {
		return nil, err
	}
	return json.Marshal(ok)
}

func PointWrite(m *module.Module, r *router.Request) ([]byte, error) {
	var pw *model.PointWriter
	err := json.Unmarshal(r.Body, &pw)
	if err != nil {
		return nil, err
	}
	pnt, err := (*m).(*Module).writePoint(r.Params["uuid"], pw)
	if err != nil {
		return nil, err
	}
	return json.Marshal(pnt)
}

func GetListSerial(m *module.Module, r *router.Request) ([]byte, error) {
	serial, err := (*m).(*Module).listSerialPorts()
	if err != nil {
		return nil, err
	}
	return json.Marshal(serial)
}

func GetNetworkPollingStats(m *module.Module, r *router.Request) ([]byte, error) {
	stats, err := (*m).(*Module).getPollingStats(r.Params["name"])
	if err != nil {
		return nil, err
	}
	return json.Marshal(stats)
}

func CreatePointOperation(m *module.Module, r *router.Request) ([]byte, error) {
	var dto Body
	err := json.Unmarshal(r.Body, &dto)
	if err != nil {
		return nil, err
	}
	netType := dto.Network.TransportType
	mbClient, err := (*m).(*Module).setClient(dto.Network, dto.Device, false)
	if err != nil {
		(*m).(*Module).modbusErrorMsg(err, "ERROR ON set modbus client")
		return nil, err
	}
	if netType == model.TransType.Serial || netType == model.TransType.LoRa {
		if dto.Device.AddressId >= 1 {
			mbClient.RTUClientHandler.SlaveID = byte(dto.Device.AddressId)
		}
	} else if netType == model.TransType.IP {
		url, err := uurl.JoinIpPort(dto.Device.Host, dto.Device.Port)
		if err != nil {
			(*m).(*Module).modbusErrorMsg(fmt.Sprintf("failed to validate device IP %s\n", url))
			return nil, err
		}
		mbClient.TCPClientHandler.Address = url
		mbClient.TCPClientHandler.SlaveID = byte(dto.Device.AddressId)
	}
	_, responseValue, err := (*m).(*Module).networkRequest(mbClient, dto.Point, false)
	if err != nil {
		return nil, err
	}
	(*m).(*Module).modbusDebugMsg("responseValue", responseValue)
	return json.Marshal(responseValue)
}

func CreateWizardTcp(m *module.Module, r *router.Request) ([]byte, error) {
	var dto wizard
	err := json.Unmarshal(r.Body, &dto)
	if err != nil {
		return nil, err
	}
	n, err := (*m).(*Module).wizardTCP(dto)
	if err != nil {
		return nil, err
	}
	return json.Marshal(n)
}

func CreateWizardSerial(m *module.Module, r *router.Request) ([]byte, error) {
	var dto wizard
	err := json.Unmarshal(r.Body, &dto)
	if err != nil {
		return nil, err
	}
	serial, err := (*m).(*Module).wizardSerial(dto)
	if err != nil {
		return nil, err
	}
	return json.Marshal(serial)
}
