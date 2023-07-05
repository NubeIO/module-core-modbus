package pkg

import argspkg "github.com/NubeIO/rubix-os/args"

func (m *Module) setUUID() {
	q, err := m.grpcMarshaller.GetPluginByPath(path, argspkg.Args{})
	if err != nil {
		return
	}
	m.pluginUUID = q.UUID
}
