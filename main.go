package main

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"plugin"

	coap "github.com/OSSystems/go-coap"
	"github.com/boltdb/bolt"
	"github.com/gustavosbarreto/cdn/journal"
	"github.com/gustavosbarreto/cdn/objstore"
	"github.com/gustavosbarreto/cdn/storage"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type App struct {
	cmd *cobra.Command

	objstore *objstore.ObjStore
	journal  *journal.Journal
	storage  *storage.Storage

	monitor Monitor
}

var app *App

func init() {
	app = &App{
		cmd: &cobra.Command{
			Use: "cdn",
			Run: execute,
		},
	}
}

func main() {
	app.cmd.PersistentFlags().StringP("backend", "", "", "Backend HTTP server URL")
	app.cmd.PersistentFlags().StringP("monitor", "", "", "Monitor plugin")
	app.cmd.PersistentFlags().StringP("db", "", "state.db", "Database file")
	app.cmd.PersistentFlags().StringP("storage", "", "./", "Storage dir")
	app.cmd.PersistentFlags().IntP("size", "", -1, "Max storage size in bytes (-1 for unlimited)")
	app.cmd.PersistentFlags().StringP("http", "", "0.0.0.0:8080", "HTTP listen address")
	app.cmd.PersistentFlags().StringP("coap", "", "0.0.0.0:5683", "CoAP listen address")
	app.cmd.PersistentFlags().StringP("log", "", "info", "Log level (debug, info, warn, error, fatal, panic)")
	app.cmd.MarkPersistentFlagRequired("backend")

	if err := app.cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func execute(cmd *cobra.Command, args []string) {
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

	app.storage = storage.NewStorage(cmd.Flag("storage").Value.String())
	app.journal = journal.NewJournal(db, int64(size))
	app.objstore = objstore.NewObjStore(backend.String(), app.journal, app.storage)

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
		app.monitor, ok = sym.(Monitor)
		if !ok {
			log.WithFields(log.Fields{"plugin": monitorPlugin, "err": err}).Fatal("Unexpected type from module symbol")
		}

		log.WithFields(log.Fields{"plugin": monitorPlugin}).Info("Monitor plugin loaded")
	} else {
		app.monitor = &dummyMonitor{}
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

	go func() {
		err = coap.Serve(udpListener, &coapHandler{})
		if err != nil {
			log.Fatal(err)
		}
	}()

	go func() {
		e := echo.New()
		e.HideBanner = true
		e.HidePort = true

		e.Use(middleware.BodyDump(logHTTPRequest))

		e.GET("*", handleHTTP)

		err = e.Start(cmd.Flag("http").Value.String())
		if err != nil {
			log.WithFields(log.Fields{"err": err}).Fatal("Failed to listen for http")
		}
	}()

	select {}
}
