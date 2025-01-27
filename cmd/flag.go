package cmd

import (
	"github.com/cosmostation/cvms/internal/app/exporter"
	"github.com/spf13/pflag"
)

const (
	// exporter & indexer
	Config          = "config"
	LogLevel        = "log-level"
	LogColorDisable = "log-color-disable"
	Port            = "port"

	// dev
	PackageFilter = "package-filter"
)

func ConfigFlag() *pflag.FlagSet {
	flag := &pflag.FlagSet{}

	flag.String(
		Config,                         // name
		"./config.yaml",                // default value
		"The Path of config yaml file", // usage
	)

	return flag
}

func LogFlags() *pflag.FlagSet {
	flag := &pflag.FlagSet{}

	flag.String(
		LogColorDisable,
		"",
		"The colored log option. default is false for production level if you want to see debug mode, recommend turn it on true",
	)
	flag.String(
		LogLevel,
		"",
		"The level of log for cvms application. default is 4 means INFO level...",
	)

	return flag
}

func PortFlag() *pflag.FlagSet {
	flag := &pflag.FlagSet{}

	flag.String(
		Port,
		"9200",
		"The port is going to listen",
	)

	return flag
}

func FilterFlag() *pflag.FlagSet {
	flag := &pflag.FlagSet{}

	flag.StringVar(
		&exporter.PackageFilter,
		PackageFilter,
		"",
		"default is null\nonly one package running when you want to run specific package",
	)

	return flag
}
