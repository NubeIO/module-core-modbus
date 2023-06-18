package pkg

import (
	"encoding/json"
	"errors"
	"github.com/NubeIO/lib-schema/modbuschema"
	"github.com/NubeIO/rubix-os/module/common"
)

const (
	listSerial        = "/list/serial"
	schemaNetwork     = "/schema/network"
	schemaDevice      = "/schema/device"
	schemaPoint       = "/schema/point"
	jsonSchemaNetwork = "/schema/json/network"
	jsonSchemaDevice  = "/schema/json/device"
	jsonSchemaPoint   = "/schema/json/point"
)

func (m *Module) Get(path string) ([]byte, error) {
	if path == jsonSchemaNetwork {
		fns, err := m.grpcMarshaller.GetFlowNetworks("")
		if err != nil {
			return nil, err
		}
		networkSchema := modbuschema.GetNetworkSchema()
		networkSchema.AutoMappingFlowNetworkName.Options = common.GetFlowNetworkNames(fns)
		return json.Marshal(networkSchema)
	} else if path == jsonSchemaDevice {
		return json.Marshal(modbuschema.GetDeviceSchema())
	} else if path == jsonSchemaPoint {
		return json.Marshal(modbuschema.GetPointSchema())
	}
	return nil, errors.New("not found")
}

func (m *Module) Post(path string, body []byte) ([]byte, error) {
	// TODO implement me
	panic("implement me")
}

func (m *Module) Put(path, uuid string, body []byte) ([]byte, error) {
	// TODO implement me
	panic("implement me")
}

func (m *Module) Patch(path, uuid string, body []byte) ([]byte, error) {
	// TODO implement me
	panic("implement me")
}

func (m *Module) Delete(path, uuid string) ([]byte, error) {
	// TODO implement me
	panic("implement me")
}
