package main

import (
	"github.com/NubeIO/module-core-modbus/pkg"
	"github.com/NubeIO/rubix-os/module/shared"
	"github.com/hashicorp/go-plugin"
)

func ServePlugin() {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: shared.HandshakeConfig,
		Plugins:         plugin.PluginSet{"module-core-modbus": &shared.NubeModule{Impl: &pkg.Module{}}},
		GRPCServer:      plugin.DefaultGRPCServer,
	})
}

func main() {
	ServePlugin()
}
