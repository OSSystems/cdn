package main

import (
	"net"
	"net/http"
	"net/url"
	"plugin"

	"github.com/OSSystems/cdn/cluster"
	"github.com/OSSystems/cdn/journal"
	"github.com/OSSystems/cdn/objstore"
	"github.com/OSSystems/cdn/pkg/monitors"
	"github.com/OSSystems/cdn/storage"
	coap "github.com/OSSystems/go-coap"
	"github.com/boltdb/bolt"
	"github.com/labstack/echo"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type App struct {
	cmd *cobra.Command

	objstore *objstore.ObjStore
	journal  *journal.Journal
	storage  *storage.Storage
	cluster  *cluster.Cluster
	node     string

	monitor monitors.Monitor
	cache   string
}

func main() {
	app := &App{
		cmd: &cobra.Command{
			Use: "cdn",
		},
	}

	app.cmd.PersistentFlags().StringP("cache", "", "", "Path will be cached")
	app.cmd.PersistentFlags().StringP("backend", "", "", "Backend HTTP server URL")
	app.cmd.PersistentFlags().StringP("monitor", "", "", "Monitor plugin")
	app.cmd.PersistentFlags().StringP("db", "", "state.db", "Database file")
	app.cmd.PersistentFlags().StringP("storage", "", "./", "Storage dir")
	app.cmd.PersistentFlags().IntP("size", "", -1, "Max storage size in bytes (-1 for unlimited)")
	app.cmd.PersistentFlags().StringP("http", "", "0.0.0.0:8080", "HTTP listen address")
	app.cmd.PersistentFlags().StringP("coap", "", "0.0.0.0:5683", "CoAP listen address")
	app.cmd.PersistentFlags().StringP("nodes", "", "", "Nodes to join")
	app.cmd.PersistentFlags().StringP("cluster", "", "0.0.0.0:1313", "Cluster listen address")
	app.cmd.PersistentFlags().StringP("log", "", "info", "Log level (debug, info, warn, error, fatal, panic)")
	app.cmd.MarkPersistentFlagRequired("backend")

	app.cmd.Run = func(cmd *cobra.Command, args []string) { app.execute(cmd, args) }

	if err := app.cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func (app *App) execute(cmd *cobra.Command, args []string) {
	level, err := log.ParseLevel(cmd.Flag("log").Value.String())
	if err != nil {
		log.Fatal(err)
	}

	log.SetLevel(level)

	backend, err := url.ParseRequestURI(cmd.Flag("backend").Value.String())
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Fatal("Failed to parse backend arg")
	}

	db, err := bolt.Open(cmd.Flag("db").Value.String(), 0600, nil)
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Fatal("Failed to open database file")
	}

	size, err := cmd.Flags().GetInt("size")
	if err != nil {
		log.Fatal(err)
	}

	cache := cmd.Flag("cache").Value.String()

	app.storage = storage.NewStorage(cmd.Flag("storage").Value.String())
	app.journal = journal.NewJournal(db, int64(size))
	app.objstore = objstore.NewObjStore(backend.String(), app.journal, app.storage)
	app.cache = cache

	monitorPlugin := cmd.Flag("monitor").Value.String()
	if monitorPlugin != "" {
		plug, err := plugin.Open(monitorPlugin)
		if err != nil {
			log.WithFields(log.Fields{"plugin": monitorPlugin, "err": err}).Fatal("Failed to open monitor plugin file")
		}

		sym, err := plug.Lookup("Monitor")
		if err != nil {
			log.WithFields(log.Fields{"plugin": monitorPlugin, "err": err}).Fatal("Failed to open monitor plugin file")
		}

		var ok bool
		app.monitor, ok = sym.(monitors.Monitor)
		if !ok {
			log.WithFields(log.Fields{"plugin": monitorPlugin, "err": err}).Fatal("Unexpected type from module symbol")
		}

		log.WithFields(log.Fields{"plugin": monitorPlugin}).Info("Monitor plugin loaded")
	} else {
		app.monitor = &monitors.DummyMonitor{}
	}

	app.monitor.Init()

	udpAddr, err := net.ResolveUDPAddr("udp", cmd.Flag("coap").Value.String())
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Fatal("Failed to parse coap address")
	}

	udpListener, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Fatal("Failed to listen for coap")
	}

	app.cluster = cluster.NewCluster()
	log.WithFields(log.Fields{
		"node": app.cluster.NodeID(),
	}).Info("Starting cdn")

	cl, err := app.cluster.ListenAndServe(cmd.Flag("cluster").Value.String())
	if err != nil {
		log.Fatal(err)
	}

	if err = app.cluster.Join(cmd.Flag("nodes").Value.String()); err != nil {
		log.Fatal(err)
	}

	go func() {
		err = coap.Serve(udpListener, app)
		if err != nil {
			log.Fatal(err)
		}
	}()

	go func() {
		e := echo.New()
		e.HideBanner = true
		e.HidePort = true

		e.GET("*", app.handleHTTP)
		e.POST("*", app.handleHTTP)
		e.PUT("*", app.handleHTTP)
		e.HEAD("*", app.handleHTTP)
		e.OPTIONS("*", app.handleHTTP)
		e.DELETE("*", app.handleHTTP)
		e.TRACE("*", app.handleHTTP)

		go func() {
			e := echo.New()
			e.GET("*", app.internalHandler)
			err := http.Serve(cl, e)
			log.Fatal(err)
		}()

		err = e.Start(cmd.Flag("http").Value.String())
		if err != nil {
			log.WithFields(log.Fields{"err": err}).Fatal("Failed to listen for http")
		}
	}()

	select {}
}

func nodeName() string {
	return uuid.Must(uuid.NewV4()).String()
}
