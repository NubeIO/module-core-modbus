package pkg

import (
	"context"

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

	m.pollingContext, m.pollingCancel = context.WithCancel(context.Background())

	if m.config.EnablePolling {
		for _, pm := range m.NetworkPollManagers {
			pm.StopPolling()
		}

		if m.config.PollQueueLogLevel != "ERROR" && m.config.PollQueueLogLevel != "DEBUG" && m.config.PollQueueLogLevel != "POLLING" {
			m.config.PollQueueLogLevel = "ERROR"
		}
		m.NetworkPollManagers = make([]*pollqueue.NetworkPollManager, 0, len(nets))
		m.mbClients = make(map[string]*smod.ModbusClient, len(nets))

		for _, net := range nets {
			m.initiatePolling(m.pollingContext, net)
		}
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
