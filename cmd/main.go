package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"mginx/internals/db"
	"mginx/internals/proxy"
	"mginx/views/components"
	"mginx/views/layout"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"time"

	_ "modernc.org/sqlite"
)

/* TODO
1) Handle request path/query params MEDIUM ✓
2) METHOD validation in handlers EASY ✓
3) handle https HARD
4) Move the handler functions to internal directory EASY
5) Add active upstream selection EASY
6) Add multiple active upstreams options with some way to direct requests (round robin or load based, something like that) HARD ✓
7) Upstream health check MEDIUM ✓
8) Metrics MEDIUM
*/

func ReturnUpstreams(wrriter http.ResponseWriter, request *http.Request) {
	// returns data for all upstream servers
	var toRet []components.UpstreamsProp
	queryStatement := "SELECT * FROM UPSTREAMS"
	db.RwLock.RLock()
	rows, err := db.ConfigDb.Query(queryStatement)
	db.RwLock.RUnlock()

	if err != nil {
		fmt.Fprintln(wrriter, err.Error())
		return
	}
	defer rows.Close()
	for rows.Next() {
		var upstreamId interface{}
		var upstreamUrl interface{}
		var online interface{}
		var primary interface{}
		err = rows.Scan(&upstreamId, &upstreamUrl, &online, &primary)
		if err != nil {
			fmt.Fprintln(wrriter, err.Error())
			return
		}
		var finalData components.UpstreamsProp
		finalData.Id = upstreamId.(int64)
		finalData.Url = upstreamUrl.(string)
		finalData.Online = online.(int64)
		finalData.Primary = primary.(int64)
		toRet = append(toRet, finalData)
	}
	components.Upstreams(toRet).Render(context.Background(), wrriter)
}

type addUpstream struct {
	Url string `json:"url"`
}

func AddUpstream(writer http.ResponseWriter, request *http.Request) {
	decoder := json.NewDecoder(request.Body)
	defer request.Body.Close()
	var data addUpstream
	var err error
	err = decoder.Decode(&data)
	if err != nil {
		component := components.Message("Error Adding Upstream!")
		component.Render(context.Background(), writer)
		return
	}
	db.RwLock.RLock() // CHECK IF FIRST ADDED, MAKE IT PRIMARY
	rows, err := db.ConfigDb.Query("SELECT COUNT(id) FROM UPSTREAMS;")
	if err != nil {
		component := components.Message("Error Fetching Upstream Count!")
		component.Render(context.Background(), writer)
		return
	}
	rows.Next()
	var count interface{}
	rows.Scan(&count)
	fmt.Println("COUNT", count, " CHECK:", count == 0, reflect.TypeOf(count))
	rows.Close()
	db.RwLock.RUnlock()
	db.RwLock.Lock()
	var insertStatement string
	if count == int64(0) {
		insertStatement = "INSERT INTO UPSTREAMS (URL, IS_PRIMARY) VALUES (?, 1);"
	} else {
		insertStatement = "INSERT INTO UPSTREAMS (URL) VALUES (?);"
	}
	fmt.Println("INSERT STATEMENT", insertStatement)
	_, err = db.ConfigDb.Exec(insertStatement, data.Url)
	db.RwLock.Unlock()
	if err != nil {
		component := components.Message("Error Adding Upstream")
		component.Render(context.Background(), writer)
		return
	}

	ReturnUpstreams(writer, request)
}

type deleteUpstream struct {
	Id string `json:"id"`
}

func DeleteUpstream(writer http.ResponseWriter, request *http.Request) {
	decoder := json.NewDecoder(request.Body)
	defer request.Body.Close()
	var data deleteUpstream
	var err error
	err = decoder.Decode(&data)
	if err != nil {
		component := components.Message("Unable to Delete Upstream!")
		component.Render(context.Background(), writer)
		return
	}
	id, err := strconv.Atoi(data.Id)
	if err != nil {
		component := components.Message("Unable to Delete Upstream!")
		component.Render(context.Background(), writer)
		return
	}
	db.RwLock.Lock()
	_, err = db.ConfigDb.Exec("DELETE FROM UPSTREAMS WHERE ID = ?", id)
	db.RwLock.Unlock()

	if err != nil {
		component := components.Message("Unable to Delete Upstream!")
		component.Render(context.Background(), writer)
		return
	}
	ReturnUpstreams(writer, request)
}

func CurrentLoadBalancingStrat(writer http.ResponseWriter, request *http.Request) {
	db.RwLock.RLock()
	row, err := db.ConfigDb.Query("SELECT * FROM SETTINGS WHERE SETTING_NAME='LOAD_BALANCING_STRATEGY'")
	db.RwLock.RUnlock()
	if err != nil {
		fmt.Println(err.Error())
		component := components.Message("Unable to Fetch Current Load Balancing Strategy!")
		component.Render(context.Background(), writer)
		return
	}
	var id interface{}
	var settingName interface{}
	var settingValue interface{}
	row.Next()
	err = row.Scan(&id, &settingName, &settingValue)
	if err != nil {
		fmt.Println(2, err.Error())
		component := components.Message("Unable to Fetch Current Load Balancing Strategy!")
		component.Render(context.Background(), writer)
		return
	}
	val := settingValue.(int64)
	var name string
	if val == 0 {
		name = "Use Primary"
	} else if val == 1 {
		name = "Round Robin Upstream Selection"
	}
	defer row.Close()
	component := components.LoadBalancingStrat(components.LoadBalancingData{Name: name})
	component.Render(context.Background(), writer)
}

func LoadBalancingOptions(writer http.ResponseWriter, request *http.Request) {
	primary := components.SelectStrat{
		Id:   strconv.Itoa(0),
		Name: "Use Primary",
	}
	RR := components.SelectStrat{
		Id:   strconv.Itoa(1),
		Name: "Round Robin Upstream Selection",
	}
	component := components.SelectLBStrat([]components.SelectStrat{
		primary,
		RR,
	})
	component.Render(context.Background(), writer)
}

type updateLoadBalancing struct {
	Strategy string `json:"strategy"`
}

// 0 -> use primary
// 1 -> round robin upstream selection
func UpdateLoadBalancingStrat(writer http.ResponseWriter, request *http.Request) {
	decoder := json.NewDecoder(request.Body)
	defer request.Body.Close()
	var data updateLoadBalancing
	var err error
	err = decoder.Decode(&data)
	if err != nil {
		component := components.Message("Unable to Update Current Load Balancing Strategy!")
		component.Render(context.Background(), writer)
	}
	strat, err := strconv.Atoi(data.Strategy)
	if err != nil {
		component := components.Message("Unable to Update Current Load Balancing Strategy!")
		component.Render(context.Background(), writer)
	}
	db.RwLock.Lock()
	_, err = db.ConfigDb.Exec("UPDATE SETTINGS SET SETTING_VALUE=? WHERE SETTING_NAME='LOAD_BALANCING_STRATEGY'", strat)
	db.RwLock.Unlock()
	if err != nil {
		fmt.Println(3, err.Error())
		component := components.Message("Unable to Update Current Load Balancing Strategy!")
		component.Render(context.Background(), writer)
	}
	component := components.Message("Load Balancing Strategy Updated!")
	component.Render(context.Background(), writer)
}

type setPrimary struct {
	Id string `json:"id"`
}

func SetPrimary(writer http.ResponseWriter, request *http.Request) {
	decoder := json.NewDecoder(request.Body)
	defer request.Body.Close()
	var data setPrimary
	err := decoder.Decode(&data)
	if err != nil {
		component := components.Message("Unable to Set Primary!")
		component.Render(context.Background(), writer)
	}
	_, err = db.ConfigDb.Exec("UPDATE UPSTREAMS SET IS_PRIMARY = 0 WHERE IS_PRIMARY = 1")
	if err != nil {
		component := components.Message("Error Removing Old Primary!")
		component.Render(context.Background(), writer)
	}
	id, err := strconv.Atoi(data.Id)
	if err != nil {
		component := components.Message("Error Converting id to int!")
		component.Render(context.Background(), writer)
	}
	_, err = db.ConfigDb.Exec("UPDATE UPSTREAMS SET IS_PRIMARY = 1 WHERE ID = ?", id)
	if err != nil {
		component := components.Message("Error Adding New Primary!")
		component.Render(context.Background(), writer)
	}

	component := components.Message("Primary Updated")
	component.Render(context.Background(), writer)
}

func ServeHome(writer http.ResponseWriter, request *http.Request) {
	component := layout.BaseLayout()
	component.Render(context.Background(), writer)
}

func HealthCheck() {
	ticker := time.NewTicker(time.Second * 20) // health check period needs to be from settings
	for range ticker.C {
		db.RwLock.RLock()
		rows, err := db.ConfigDb.Query("SELECT URL FROM UPSTREAMS;")
		if err != nil {
			db.RwLock.RUnlock()
			fmt.Println("HealthCheck error: ", err.Error())
			break
		}
		db.RwLock.RUnlock()
		upstreamUpdate := make(map[string]int)
		for rows.Next() { // since max connections is set to one, cant perform updates inside rows Next0! Took really long to debug why server kept hanging.
			var data string
			err = rows.Scan(&data)
			if err != nil {
				fmt.Println("HealthCheck error1: ", err.Error())
				break
			}
			_, err = http.Get(data)
			if err != nil || os.IsTimeout(err) {
				upstreamUpdate[data] = 0

			} else {
				upstreamUpdate[data] = 1
			}
		}
		rows.Close()
		for upstream, value := range upstreamUpdate {
			db.RwLock.Lock()
			db.ConfigDb.Exec("UPDATE UPSTREAMS SET ONLINE = ? WHERE URL = ?", value, upstream)
			db.RwLock.Unlock()
		}
	}
}

func main() {
	var err error
	db.ConfigDb, err = sql.Open("sqlite", "config.db:locked.sqlite?cache=shared") // issue 209 golang sqlite interface
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
	}
	db.ConfigDb.SetMaxOpenConns(1)
	upstreamsStatement := "CREATE TABLE IF NOT EXISTS UPSTREAMS (ID INTEGER PRIMARY KEY, URL STRING, ONLINE INTEGER DEFAULT 0 NOT NULL, IS_PRIMARY INTEGER DEFAULT 0 NOT NULL);"                             // initialize config table
	settingsStatement := "CREATE TABLE IF NOT EXISTS SETTINGS (ID INTEGER PRIMARY KEY, SETTING_NAME STRING, SETTING_VALUE INT); INSERT OR IGNORE INTO SETTINGS VALUES (NULL, 'LOAD_BALANCING_STRATEGY', 0);" // initialize settings table

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

	http.HandleFunc("GET /config/upstreams", ReturnUpstreams)
	http.HandleFunc("POST /config/add-upstream", AddUpstream)
	http.HandleFunc("POST /config/delete-upstream", DeleteUpstream)
	http.HandleFunc("POST /config/set-primary", SetPrimary)
	http.HandleFunc("GET /config/get-load-balancing-strategy", CurrentLoadBalancingStrat)
	http.HandleFunc("POST /config/update-load-balancing-strategy", UpdateLoadBalancingStrat)
	http.HandleFunc("GET /config/all-load-balancing-strategies", LoadBalancingOptions)
	http.HandleFunc("GET /home", ServeHome)
	http.Handle("GET /dist/", http.StripPrefix("/dist/", http.FileServer(http.Dir("./dist"))))
	http.HandleFunc("/proxy/*", proxy.ReverseProxy)
	// cert.GenCerts()
	certFile := flag.String("certfile", "cert.pem", "certificate PEM file")
	keyFile := flag.String("keyfile", "key.pem", "key PEM file")
	go HealthCheck()
	fmt.Println("MGINIX STARTED")
	log.Fatal(http.ListenAndServeTLS("localhost:3690", *certFile, *keyFile, nil))
}
