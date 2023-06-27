package pkg

func (m *Module) setUUID() {
	q, err := m.grpcMarshaller.GetPluginByPath(path, "")
	if err != nil {
		return
	}
	m.pluginUUID = q.UUID
}
