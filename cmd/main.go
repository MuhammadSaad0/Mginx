package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"mginx/internals/db"
	"mginx/internals/healthcheck"
	"mginx/internals/loadbalancing"
	"mginx/internals/proxy"
	"mginx/internals/upstreams"
	"mginx/views/layout"
	"net/http"
	"os"

	_ "modernc.org/sqlite"
)

// RETURN FROM HANDLER WHEN ERROR COMPONENT RENDERED
// DONT KEEP FETCHING, RETURN UPSTREAM LIST UPON CHANGE
// HX-SWAP TARGET FOR ERRORS

func ServeHome(writer http.ResponseWriter, request *http.Request) {
	component := layout.BaseLayout()
	component.Render(context.Background(), writer)
}

func main() {
	var err error
	db.ConfigDb, err = sql.Open("sqlite", "config.db:locked.sqlite?cache=shared") // issue 209 golang sqlite interface
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
	}
	db.ConfigDb.SetMaxOpenConns(1)
	upstreamsStatement := "CREATE TABLE IF NOT EXISTS UPSTREAMS (ID INTEGER PRIMARY KEY, URL STRING, ONLINE INTEGER DEFAULT 0 NOT NULL, IS_PRIMARY INTEGER DEFAULT 0 NOT NULL, SHADOW INTEGER DEFAULT 0 NOT NULL);" // initialize config table
	settingsStatement := "CREATE TABLE IF NOT EXISTS SETTINGS (ID INTEGER PRIMARY KEY, SETTING_NAME STRING, SETTING_VALUE INT); INSERT OR IGNORE INTO SETTINGS VALUES (NULL, 'LOAD_BALANCING_STRATEGY', 0);"        // initialize settings table

	db.RwLock.Lock()
	_, err = db.ConfigDb.Exec(upstreamsStatement)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
	}

	_, err = db.ConfigDb.Exec(settingsStatement)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
	}
	db.RwLock.Unlock()

	http.HandleFunc("GET /config/upstreams", upstreams.ReturnUpstreams)
	http.HandleFunc("POST /config/add-upstream", upstreams.AddUpstream)
	http.HandleFunc("POST /config/delete-upstream", upstreams.DeleteUpstream)
	http.HandleFunc("POST /config/set-primary", upstreams.SetPrimary)
	http.HandleFunc("POST /config/toggle-shadow", upstreams.ToggleShadow)
	http.HandleFunc("GET /config/get-load-balancing-strategy", loadbalancing.CurrentLoadBalancingStrat)
	http.HandleFunc("POST /config/update-load-balancing-strategy", loadbalancing.UpdateLoadBalancingStrat)
	http.HandleFunc("GET /config/all-load-balancing-strategies", loadbalancing.LoadBalancingOptions)
	http.HandleFunc("GET /home", ServeHome)
	http.Handle("GET /dist/", http.StripPrefix("/dist/", http.FileServer(http.Dir("./dist"))))
	http.HandleFunc("/proxy/*", proxy.ReverseProxy)
	// cert.GenCerts()
	certFile := flag.String("certfile", "cert.pem", "certificate PEM file")
	keyFile := flag.String("keyfile", "key.pem", "key PEM file")
	go healthcheck.HealthCheck()
	fmt.Println("MGINIX STARTED")
	log.Fatal(http.ListenAndServeTLS("localhost:3690", *certFile, *keyFile, nil))
}
