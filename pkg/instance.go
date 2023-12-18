package pkg

import (
	"github.com/NubeIO/nubeio-rubix-lib-models-go/nargs"
)

func (m *Module) setUUID() {
	q, err := m.grpcMarshaller.GetPluginByName(path, nargs.Args{})
	if err != nil {
		return
	}
	m.pluginUUID = q.UUID
}
