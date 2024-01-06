package pkg

import (
	"fmt"
	"github.com/NubeIO/module-core-modbus/pollqueue"
	"github.com/NubeIO/module-core-modbus/smod"
	"github.com/NubeIO/nubeio-rubix-lib-helpers-go/pkg/nils"
	"github.com/NubeIO/nubeio-rubix-lib-helpers-go/pkg/uurl"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/dto"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"
	"github.com/grid-x/modbus"
	"time"
)

type Client struct {
	Host       string        `json:"ip"`
	Port       string        `json:"port"`
	SerialPort string        `json:"serial_port"`
	BaudRate   uint          `json:"baud_rate"` // 38400
	Parity     string        `json:"parity"`    // none, odd, even DEFAULT IS PARITY_NONE
	DataBits   uint          `json:"data_bits"` // 7 or 8
	StopBits   uint          `json:"stop_bits"` // 1 or 2
	Timeout    time.Duration `json:"device_timeout_in_ms"`
}

func (m *Module) createMbClient(netPollMan *pollqueue.NetworkPollManager, net *model.Network, dev *model.Device) (*smod.ModbusClient, error) {
	mbClient, err := m.setClient(net, dev, true)
	if err != nil {
		if mbClient.PortUnavailable {
			netPollMan.PortUnavailable()
			unpauseFunc := func() {
				netPollMan.PortAvailable()
			}
			netPollMan.PortUnavailableTimeout = time.AfterFunc(10*time.Second, unpauseFunc)
		}
		m.updateNetworkMessage(net, "", err, m.pollCounter)
		return nil, err
	}
	m.mbClients[net.UUID] = mbClient
	return mbClient, nil
}

func (m *Module) setClient(network *model.Network, device *model.Device, cacheClient bool) (mbClient *smod.ModbusClient, err error) {
	mbClient = &smod.ModbusClient{}
	if network.TransportType == dto.TransType.Serial || network.TransportType == dto.TransType.LoRa {
		serialPort := "/dev/ttyUSB0"
		baudRate := 38400
		stopBits := 1
		dataBits := 8
		parity := "N"
		timeout := 2 * time.Second
		if network.SerialPort != nil && *network.SerialPort != "" {
			serialPort = nils.StringIsNil(network.SerialPort)
		}
		if network.SerialBaudRate != nil {
			baudRate = int(nils.UnitIsNil(network.SerialBaudRate))
		}
		if network.SerialDataBits != nil {
			dataBits = int(nils.UnitIsNil(network.SerialDataBits))
		}
		if network.SerialStopBits != nil {
			stopBits = int(nils.UnitIsNil(network.SerialStopBits))
		}
		if network.SerialParity != nil {
			parity = nils.StringIsNil(network.SerialParity)
		}
		if network.SerialTimeout != nil {
			timeoutSecs := int64(nils.IntIsNil(network.SerialTimeout))
			if timeoutSecs > 0 {
				timeout = time.Duration(timeoutSecs) * time.Second
			}
		}
		handler := modbus.NewRTUClientHandler(serialPort)
		handler.BaudRate = baudRate
		handler.DataBits = dataBits
		handler.Parity = setParity(parity)
		handler.StopBits = stopBits
		handler.Timeout = timeout

		err := handler.Connect()
		defer handler.Close()
		if err != nil {
			m.modbusErrorMsg(fmt.Sprintf("setClient:  %v. port:%s", err, serialPort))
			return nil, err
		}
		mc := modbus.NewClient(handler)
		mbClient.RTUClientHandler = handler
		mbClient.Client = mc
		return mbClient, nil

	} else {
		url, err := uurl.JoinIpPort(device.Host, device.Port)
		if err != nil {
			m.modbusErrorMsg(fmt.Sprintf("modbus: failed to validate device IP %s\n", url))
			return nil, err
		}
		handler := modbus.NewTCPClientHandler(url)
		err = handler.Connect()
		defer handler.Close()
		if err != nil {
			m.modbusErrorMsg(fmt.Sprintf("setClient:  %v. port:%s", err, url))
			return nil, err
		}
		mc := modbus.NewClient(handler)
		mbClient.TCPClientHandler = handler
		mbClient.Client = mc
		return mbClient, nil
	}
}

func setParity(in string) string {
	if in == dto.SerialParity.None {
		return "N"
	} else if in == dto.SerialParity.Odd {
		return "O"
	} else if in == dto.SerialParity.Even {
		return "E"
	} else {
		return "N"
	}
}
