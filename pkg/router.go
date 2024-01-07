package pkg

import (
	"encoding/json"
	"fmt"
	"github.com/NubeIO/lib-module-go/nhttp"
	"github.com/NubeIO/lib-module-go/nmodule"
	"github.com/NubeIO/lib-module-go/router"
	"github.com/NubeIO/module-core-modbus/schema"
	"github.com/NubeIO/nubeio-rubix-lib-helpers-go/pkg/uurl"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/dto"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"
	"net/http"
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

var route *router.Router

func (m *Module) CallModule(
	method nhttp.Method,
	urlString string,
	headers http.Header,
	body []byte,
) ([]byte, error) {
	mod := (nmodule.Module)(m)
	return route.CallHandler(&mod, method, urlString, headers, body)
}

func InitRouter() {
	route = router.NewRouter()

	route.Handle(nhttp.GET, "/api/networks/schema", GetNetworkSchema)
	route.Handle(nhttp.GET, "/api/devices/schema", GetDeviceSchema)
	route.Handle(nhttp.GET, "/api/points/schema", GetPointSchema)

	route.Handle(nhttp.POST, "/api/networks", CreateNetwork)
	route.Handle(nhttp.PATCH, "/api/networks/:uuid", UpdateNetwork)
	route.Handle(nhttp.DELETE, "/api/networks/:uuid", DeleteNetwork)

	route.Handle(nhttp.POST, "/api/devices", CreateDevice)
	route.Handle(nhttp.PATCH, "/api/devices/:uuid", UpdateDevice)
	route.Handle(nhttp.DELETE, "/api/devices/:uuid", DeleteDevice)

	route.Handle(nhttp.POST, "/api/points", CreatePoint)
	route.Handle(nhttp.PATCH, "/api/points/:uuid", UpdatePoint)
	route.Handle(nhttp.DELETE, "/api/points/:uuid", DeletePoint)
	route.Handle(nhttp.PATCH, "/api/points/:uuid/write", PointWrite)

	route.Handle(nhttp.GET, "/api/list/serial", GetListSerial)
	route.Handle(nhttp.GET, "/api/polling/stats/network/name/:name", GetNetworkPollingStats)
	route.Handle(nhttp.POST, "/api/modbus/point/operation", CreatePointOperation)
}

func GetNetworkSchema(m *nmodule.Module, r *router.Request) ([]byte, error) {
	return json.Marshal(schema.GetNetworkSchema())
}

func GetDeviceSchema(m *nmodule.Module, r *router.Request) ([]byte, error) {
	return json.Marshal(schema.GetDeviceSchema())
}

func GetPointSchema(m *nmodule.Module, r *router.Request) ([]byte, error) {
	return json.Marshal(schema.GetPointSchema())
}

func CreateNetwork(m *nmodule.Module, r *router.Request) ([]byte, error) {
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

func UpdateNetwork(m *nmodule.Module, r *router.Request) ([]byte, error) {
	var network *model.Network
	err := json.Unmarshal(r.Body, &network)
	if err != nil {
		return nil, err
	}
	net, err := (*m).(*Module).updateNetwork(r.PathParams["uuid"], network)
	if err != nil {
		return nil, err
	}
	return json.Marshal(net)
}

func DeleteNetwork(m *nmodule.Module, r *router.Request) ([]byte, error) {
	ok, err := (*m).(*Module).deleteNetwork(r.PathParams["uuid"])
	if err != nil {
		return nil, err
	}
	return json.Marshal(ok)
}

func CreateDevice(m *nmodule.Module, r *router.Request) ([]byte, error) {
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

func UpdateDevice(m *nmodule.Module, r *router.Request) ([]byte, error) {
	var device *model.Device
	err := json.Unmarshal(r.Body, &device)
	if err != nil {
		return nil, err
	}
	dev, err := (*m).(*Module).updateDevice(r.PathParams["uuid"], device)
	if err != nil {
		return nil, err
	}
	return json.Marshal(dev)
}

func DeleteDevice(m *nmodule.Module, r *router.Request) ([]byte, error) {
	dev, err := (*m).(*Module).grpcMarshaller.GetDevice(r.PathParams["uuid"], nil)
	if err != nil {
		return nil, err
	}
	ok, err := (*m).(*Module).deleteDevice(dev)
	if err != nil {
		return nil, err
	}
	return json.Marshal(ok)
}

func CreatePoint(m *nmodule.Module, r *router.Request) ([]byte, error) {
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

func UpdatePoint(m *nmodule.Module, r *router.Request) ([]byte, error) {
	var point *model.Point
	err := json.Unmarshal(r.Body, &point)
	if err != nil {
		return nil, err
	}
	pnt, err := (*m).(*Module).updatePoint(r.PathParams["uuid"], point)
	if err != nil {
		return nil, err
	}
	return json.Marshal(pnt)
}

func DeletePoint(m *nmodule.Module, r *router.Request) ([]byte, error) {
	pnt, err := (*m).(*Module).grpcMarshaller.GetPoint(r.PathParams["uuid"], nil)
	if err != nil {
		return nil, err
	}
	ok, err := (*m).(*Module).deletePoint(pnt)
	if err != nil {
		return nil, err
	}
	return json.Marshal(ok)
}

func PointWrite(m *nmodule.Module, r *router.Request) ([]byte, error) {
	var pw *dto.PointWriter
	err := json.Unmarshal(r.Body, &pw)
	if err != nil {
		return nil, err
	}
	pnt, err := (*m).(*Module).writePoint(r.PathParams["uuid"], pw)
	if err != nil {
		return nil, err
	}
	return json.Marshal(pnt)
}

func GetListSerial(m *nmodule.Module, r *router.Request) ([]byte, error) {
	serial, err := (*m).(*Module).listSerialPorts()
	if err != nil {
		return nil, err
	}
	return json.Marshal(serial)
}

func GetNetworkPollingStats(m *nmodule.Module, r *router.Request) ([]byte, error) {
	stats, err := (*m).(*Module).getPollingStats(r.PathParams["name"])
	if err != nil {
		return nil, err
	}
	return json.Marshal(stats)
}

func CreatePointOperation(m *nmodule.Module, r *router.Request) ([]byte, error) {
	var body Body
	err := json.Unmarshal(r.Body, &body)
	if err != nil {
		return nil, err
	}
	netType := body.Network.TransportType
	mbClient, err := (*m).(*Module).setClient(body.Network, body.Device, false)
	if err != nil {
		(*m).(*Module).modbusErrorMsg(err, "ERROR ON set modbus client")
		return nil, err
	}
	if netType == dto.TransType.Serial || netType == dto.TransType.LoRa {
		if body.Device.AddressId >= 1 {
			mbClient.RTUClientHandler.SlaveID = byte(body.Device.AddressId)
		}
	} else if netType == dto.TransType.IP {
		url, err := uurl.JoinIpPort(body.Device.Host, body.Device.Port)
		if err != nil {
			(*m).(*Module).modbusErrorMsg(fmt.Sprintf("failed to validate device IP %s\n", url))
			return nil, err
		}
		mbClient.TCPClientHandler.Address = url
		mbClient.TCPClientHandler.SlaveID = byte(body.Device.AddressId)
	}
	_, responseValue, err := (*m).(*Module).networkRequest(mbClient, body.Point, false)
	if err != nil {
		return nil, err
	}
	(*m).(*Module).modbusDebugMsg("responseValue", responseValue)
	return json.Marshal(responseValue)
}
