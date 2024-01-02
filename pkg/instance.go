package pkg

func (m *Module) setUUID() {
	q, err := m.grpcMarshaller.GetPluginByName(path)
	if err != nil {
		return
	}
	m.pluginUUID = q.UUID
}
