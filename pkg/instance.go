package pkg

func (m *Module) setUUID() {
	q, err := m.grpcMarshaller.GetPluginByName(path, nil)
	if err != nil {
		return
	}
	m.pluginUUID = q.UUID
}
