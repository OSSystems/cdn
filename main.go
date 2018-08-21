package main

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"plugin"

	coap "github.com/OSSystems/go-coap"
	"github.com/labstack/echo"
	"github.com/spf13/cobra"
)

var logger Logger

var rootCmd *cobra.Command

func init() {
	rootCmd = &cobra.Command{
		Use: "cdn",
		Run: execute,
	}
}

func main() {
	rootCmd.PersistentFlags().StringP("backend", "", "", "Backend HTTP server URL")
	rootCmd.PersistentFlags().StringP("logger", "", "", "Logger plugin")
	rootCmd.PersistentFlags().StringP("http", "", "0.0.0.0:8080", "HTTP listen address")
	rootCmd.PersistentFlags().StringP("coap", "", "0.0.0.0:5000", "CoAP listen address")
	rootCmd.MarkPersistentFlagRequired("backend")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func execute(cmd *cobra.Command, args []string) {
	_, err := url.ParseRequestURI(rootCmd.Flag("backend").Value.String())
	if err != nil {
		panic(err)
	}

	loggerPlugin := cmd.Flag("logger").Value.String()
	if loggerPlugin != "" {
		plug, err := plugin.Open(loggerPlugin)
		if err != nil {
			panic(err)
		}

		sym, err := plug.Lookup("Logger")
		if err != nil {
			panic(err)
		}

		var ok bool
		logger, ok = sym.(Logger)
		if !ok {
			panic("unexpected type from module symbol")
		}

		logger.Init()
	}

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

		e.GET("*", handleHTTP)

		err = e.Start(cmd.Flag("http").Value.String())
		if err != nil {
			panic(err)
		}
	}()

	select {}
}
