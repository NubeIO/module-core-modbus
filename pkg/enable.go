package pkg

import (
	"github.com/NubeIO/module-core-modbus/pollqueue"
	"github.com/NubeIO/module-core-modbus/smod"
	log "github.com/sirupsen/logrus"
)

func (m *Module) Enable() error {
	log.Info("plugin Enable()")
	m.setUUID()

	nets, err := m.grpcMarshaller.GetNetworksByPlugin(m.pluginUUID)
	if err != nil {
		return err
	}

	if m.config.EnablePolling {
		var arg polling
		arg.enable = true
		for _, pm := range m.NetworkPollManagers {
			pm.StopPolling()
		}
		m.NetworkPollManagers = make([]*pollqueue.NetworkPollManager, len(nets))
		m.mbClients = make(map[string]*smod.ModbusClient, len(nets))
		for i, net := range nets {
			if m.config.PollQueueLogLevel != "ERROR" && m.config.PollQueueLogLevel != "DEBUG" && m.config.PollQueueLogLevel != "POLLING" {
				m.config.PollQueueLogLevel = "ERROR"
			}
			pollQueueConfig := pollqueue.Config{EnablePolling: m.config.EnablePolling, LogLevel: m.config.PollQueueLogLevel}
			pollManager := pollqueue.NewPollManager(&pollQueueConfig, m.grpcMarshaller, net.UUID, net.Name, m.moduleName)
			pollManager.StartPolling()
			m.NetworkPollManagers[i] = pollManager
		}
		m.initiatePolling()
	}
	return nil
}

func (m *Module) Disable() error {
	m.modbusPollingMsg("MODBUS Plugin Disable()")
	m.pollingCancel()
	m.pollingCancel = nil
	for _, pollMan := range m.NetworkPollManagers {
		pollMan.StopPolling()
	}
	m.NetworkPollManagers = nil
	m.mbClients = nil
	return nil
}
