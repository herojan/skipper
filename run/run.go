package run

import (
	"github.com/zalando/skipper/dispatch"
	"github.com/zalando/skipper/etcd"
	"github.com/zalando/skipper/filters"
	"github.com/zalando/skipper/innkeeper"
	"github.com/zalando/skipper/proxy"
	"github.com/zalando/skipper/settings"
	"github.com/zalando/skipper/skipper"
	"log"
	"net/http"
	"time"
)

// Options to start skipper. Expects address to listen on and one or more urls to find
// the etcd service at. If the flag 'insecure' is true, skipper will accept
// invalid TLS certificates from the backends.
// If a routesFilePath is given, that file will be used _instead_ of etcd.
type Options struct {
	Address              string
	EtcdUrls             []string
	StorageRoot          string
	Insecure             bool
	InnkeeperUrl         string
	InnkeeperPollTimeout time.Duration
	RoutesFilePath       string
	CustomFilters        []skipper.FilterSpec
}

func makeDataClient(o Options) (skipper.DataClient, error) {
	switch {
	case o.RoutesFilePath != "":
		return settings.MakeFileDataClient(o.RoutesFilePath)
	case o.InnkeeperUrl != "":
		return innkeeper.Make(o.InnkeeperUrl, o.InnkeeperPollTimeout), nil
	default:
		return etcd.Make(o.EtcdUrls, o.StorageRoot)
	}
}

// Run skipper.
func Run(o Options) error {
	// create data client
	dataClient, err := makeDataClient(o)
	if err != nil {
		return err
	}

	// create a filter registry with the available filter specs registered,
	// and register the custom filters
	registry := filters.RegisterDefault()
	registry.Add(o.CustomFilters...)

	// create a settings dispatcher instance
	// create a settings source
	// create the proxy instance
	dispatcher := dispatch.Make()
	settingsSource := settings.MakeSource(dataClient, registry, dispatcher)
	proxy := proxy.Make(settingsSource, o.Insecure)

	// subscribe to new settings
	settingsChan := make(chan skipper.Settings)
	dispatcher.Subscribe(settingsChan)

	// start the http server
	log.Printf("listening on %v\n", o.Address)
	return http.ListenAndServe(o.Address, proxy)
}
