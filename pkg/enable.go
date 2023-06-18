package pkg

import (
	"github.com/NubeIO/rubix-os/services/pollqueue"
	"github.com/NubeIO/rubix-os/utils/float"
)

func (m *Module) Enable() error {
	m.enabled = true
	m.fault = false

	nets, err := m.grpcMarshaller.GetNetworksByPluginName(name, "")
	if err != nil {
		m.networks = nil
	} else if nets != nil {
		m.networks = nets
	}

	if m.config.EnablePolling {
		if !m.pollingEnabled {
			var arg polling
			m.pollingEnabled = true
			arg.enable = true
			m.NetworkPollManagers = make([]*pollqueue.NetworkPollManager, 0)
			for _, net := range nets {
				conf := m.GetConfig().(*Config)
				if conf.PollQueueLogLevel != "ERROR" && conf.PollQueueLogLevel != "DEBUG" && conf.PollQueueLogLevel != "POLLING" {
					conf.PollQueueLogLevel = "ERROR"
				}
				pollQueueConfig := pollqueue.Config{EnablePolling: conf.EnablePolling, LogLevel: conf.PollQueueLogLevel}
				pollManager := NewPollManager( // TODO: Check this
					&pollQueueConfig,
					&m.dbHelper,
					net.UUID,
					net.Name,
					m.pluginUUID,
					m.moduleName,
					float.NonNil(net.MaxPollRate),
				)
				pollManager.StartPolling() // TODO: Check this
				m.NetworkPollManagers = append(m.NetworkPollManagers, pollManager)
			}
		}
	}

	return nil
}

func (m *Module) Disable() error {
	// TODO implement me
	panic("implement me")
}
