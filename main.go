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
	app.cmd.PersistentFlags().StringP("http", "", "0.0.0.0:8080", "HTTP listen address")
	app.cmd.PersistentFlags().StringP("coap", "", "0.0.0.0:5683", "CoAP listen address")
	app.cmd.MarkPersistentFlagRequired("backend")

	if err := app.cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func execute(cmd *cobra.Command, args []string) {
	backend, err := url.ParseRequestURI(cmd.Flag("backend").Value.String())
	if err != nil {
		panic(err)
	}

	db, err := bolt.Open(cmd.Flag("db").Value.String(), 0600, nil)
	if err != nil {
		panic(err)
	}

	app.storage = storage.NewStorage(cmd.Flag("storage").Value.String())
	app.journal = journal.NewJournal(db, 9999999)
	app.objstore = objstore.NewObjStore(backend.String(), app.journal, app.storage)

	monitorPlugin := cmd.Flag("monitor").Value.String()
	if monitorPlugin != "" {
		plug, err := plugin.Open(monitorPlugin)
		if err != nil {
			panic(err)
		}

		sym, err := plug.Lookup("Monitor")
		if err != nil {
			panic(err)
		}

		var ok bool
		app.monitor, ok = sym.(Monitor)
		if !ok {
			panic("unexpected type from module symbol")
		}

	} else {
		app.monitor = &dummyMonitor{}
	}

	app.monitor.Init()

	udpAddr, err := net.ResolveUDPAddr("udp", cmd.Flag("coap").Value.String())
	if err != nil {
		panic(err)
	}

	udpListener, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		panic(err)
	}

	go func() {
		err = coap.Serve(udpListener, &coapHandler{})
		if err != nil {
			panic(err)
		}
	}()

	go func() {
		e := echo.New()
		e.HideBanner = true

		e.Use(middleware.BodyDump(logHTTPRequest))

		e.GET("*", handleHTTP)

		err = e.Start(cmd.Flag("http").Value.String())
		if err != nil {
			panic(err)
		}
	}()

	select {}
}
