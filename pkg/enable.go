package pkg

import (
	argspkg "github.com/NubeIO/rubix-os/args"
	"github.com/NubeIO/rubix-os/module/shared/pollqueue"
	"github.com/NubeIO/rubix-os/utils/float"
)

func (m *Module) Enable() error {
	m.enabled = true
	m.fault = false
	m.moduleName = name
	m.setUUID()

	nets, err := m.grpcMarshaller.GetNetworksByPluginName(name, argspkg.Args{})
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
				pollManager := NewPollManager(
					&pollQueueConfig,
					m.grpcMarshaller,
					net.UUID,
					net.Name,
					m.pluginUUID,
					m.moduleName,
					float.NonNil(net.MaxPollRate),
				)
				pollManager.StartPolling()
				m.NetworkPollManagers = append(m.NetworkPollManagers, pollManager)
			}
			m.running = true
			if err := m.ModbusPolling(); err != nil {
				m.fault = true
				m.running = false
				m.modbusErrorMsg("POLLING ERROR on routine: %v\n", err)
			}
		}
	}

	return nil
}

func (m *Module) Disable() error {
	m.modbusPollingMsg("MODBUS Plugin Disable()")
	m.enabled = false
	if m.pollingEnabled {
		m.pollingEnabled = false
		m.pollingCancel()
		m.pollingCancel = nil
		for _, pollMan := range m.NetworkPollManagers {
			pollMan.StopPolling()
		}
		m.NetworkPollManagers = make([]*pollqueue.NetworkPollManager, 0)
	}
	m.running = false
	m.fault = false
	return nil
}
