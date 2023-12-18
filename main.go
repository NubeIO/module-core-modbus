package main

import (
	"github.com/NubeIO/lib-module-go/module"
	"github.com/NubeIO/module-core-modbus/pkg"
	"github.com/hashicorp/go-plugin"
)

func ServePlugin() {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: module.HandshakeConfig,
		Plugins:         plugin.PluginSet{"module-core-modbus": &module.NubeModule{Impl: &pkg.Module{}}},
		GRPCServer:      plugin.DefaultGRPCServer,
	})
}

func main() {
	ServePlugin()
}
