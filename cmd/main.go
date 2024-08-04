package main

import (
	"context"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"

	_ "modernc.org/sqlite"
)

/* TODO
1) Handle request path/query params MEDIUM ✓
2) METHOD validation in handlers EASY ✓
3) handle https HARD
4) Move the handler functions to internal directory EASY
5) Add active upstream selection EASY
6) Add multiple active upstreams options with some way to direct requests (round robin or load based, something like that) HARD
7) Upstream health check MEDIUM
8) Metrics MEDIUM
*/

var configDb *sql.DB
var rwLock = sync.RWMutex{}

func ReverseProxy(writer http.ResponseWriter, request *http.Request) {
	// copy incoming request
	modifiedRequest := request.Clone(context.Background())
	path := request.URL.Path
	query := request.URL.Query()

	var rows *sql.Rows
	var err error

	rows, err = configDb.Query("SELECT SETTING_VALUE FROM SETTINGS WHERE SETTING_NAME='LOAD_BALANCING_STRATEGY';")
	if err != nil {
		fmt.Fprintln(writer, err.Error())
		return
	}

	rows.Next()
	var loadBalancingStrat int
	rows.Scan(&loadBalancingStrat)

	rows, err = configDb.Query("SELECT URL FROM UPSTREAMS")
	if err != nil {
		fmt.Fprintln(writer, err.Error())
		return
	}
	var data string
	if loadBalancingStrat == 0 { // just use first url
		rows.Next()
		err = rows.Scan(&data)
		if err != nil {
			fmt.Fprintln(writer, err.Error())
			return
		}

	} else if loadBalancingStrat == 1 { // round robin
		upstreams := make([]string, 0, 10)

		for rows.Next() {
			err = rows.Scan(&data)
			if err != nil {
				fmt.Fprintln(writer, err.Error())
				return
			}
			upstreams = append(upstreams, data)
		}

		roundRobinNum := rand.Intn(len(upstreams))
		data = upstreams[roundRobinNum]
	}

	url, err := url.Parse(data)
	if err != nil {
		fmt.Fprintln(writer, "Error Parsing url")
		return
	}
	modifiedRequest.RequestURI = "" // httpRequestUri cant be set in client request
	modifiedRequest.URL = url
	modifiedRequest.URL.Path = "/" + strings.Join((strings.Split(path, "/"))[2:], "/")
	modifiedRequest.URL.RawQuery = query.Encode()

	defaultClient := http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{},
		},
	}
	response, err := defaultClient.Do(modifiedRequest) // perform http request
	if err != nil {
		fmt.Fprintf(writer, "Error sending request %s", err.Error())
		return
	}
	defer response.Body.Close()                // close body to make sure socket isnt left hanging
	for key, values := range response.Header { // write response headers to writer
		for _, value := range values {
			writer.Header().Add(key, value)
		}
	}
	writer.WriteHeader(response.StatusCode) // write response status code to writer

	io.Copy(writer, response.Body) // copy response body to writer

}

func ReturnUpstreams(wrriter http.ResponseWriter, request *http.Request) {
	// returns data for all upstream servers
	var toRet []interface{}
	queryStatement := "SELECT * FROM UPSTREAMS"
	rows, err := configDb.Query(queryStatement)

	if err != nil {
		fmt.Fprintln(wrriter, err.Error())
		return
	}
	rwLock.RLock() // acquire read lock to make sure if write lock has been acquired this read is blocked
	for rows.Next() {
		var upstreamId interface{}
		var upstreamUrl interface{}
		err = rows.Scan(&upstreamId, &upstreamUrl)
		if err != nil {
			rwLock.RUnlock()
			fmt.Fprintln(wrriter, err.Error())
			return
		}
		var finalData map[string]interface{} = make(map[string]interface{})
		finalData["id"] = upstreamId
		finalData["url"] = upstreamUrl
		toRet = append(toRet, finalData)
	}
	rwLock.RUnlock()
	response := make(map[string][]interface{})
	if len(toRet) > 0 {
		response["proxies"] = toRet
	} else {
		response["proxies"] = make([]interface{}, 0)
	}
	encoder := json.NewEncoder(wrriter)
	encoder.Encode(response)
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
		fmt.Fprintln(writer, err.Error())
		return
	}
	rwLock.Lock()
	_, err = configDb.Query("INSERT INTO UPSTREAMS VALUES (NULL, ?)", data.Url)
	rwLock.Unlock()
	if err != nil {
		fmt.Fprintln(writer, err.Error())
		return
	}

	fmt.Fprintf(writer, "UPSTREAM %s added to configuration", data.Url)
}

type deleteUpstream struct {
	Id int `json:"id"`
}

func DeleteUpstream(writer http.ResponseWriter, request *http.Request) {
	decoder := json.NewDecoder(request.Body)
	defer request.Body.Close()
	var data deleteUpstream
	var err error
	err = decoder.Decode(&data)
	if err != nil {
		fmt.Fprintln(writer, err.Error())
		return
	}

	rwLock.Lock()
	_, err = configDb.Query("DELETE FROM UPSTREAMS WHERE ID = ?", data.Id)
	rwLock.Unlock()

	if err != nil {
		fmt.Fprintln(writer, err.Error())
		return
	}

	fmt.Fprintln(writer, "Upstream deleted")

}

type updateLoadBalancing struct {
	Strategy int `json:"strategy"`
}

// 0 -> use first upstream url
// 1 -> round robin upstream selection
func UpdateLoadBalancingStrat(writer http.ResponseWriter, request *http.Request) {
	decoder := json.NewDecoder(request.Body)
	defer request.Body.Close()
	var data updateLoadBalancing
	var err error
	err = decoder.Decode(&data)
	if err != nil {
		fmt.Fprintln(writer, err.Error())
		return
	}

	rwLock.Lock()
	_, err = configDb.Query("UPDATE SETTINGS SET SETTING_VALUE=? WHERE SETTING_NAME='LOAD_BALANCING_STRATEGY'", data.Strategy)
	rwLock.Unlock()
	if err != nil {
		fmt.Fprintln(writer, err.Error())
		return
	}
	fmt.Fprintln(writer, "Load balancing strategy updated")
}

func main() {
	var err error
	configDb, err = sql.Open("sqlite", "config.db")
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
	}
	upstreamsStatement := "CREATE TABLE IF NOT EXISTS UPSTREAMS (ID INTEGER PRIMARY KEY, URL STRING);"                                                                                                       // initialize config table
	settingsStatement := "CREATE TABLE IF NOT EXISTS SETTINGS (ID INTEGER PRIMARY KEY, SETTING_NAME STRING, SETTING_VALUE INT); INSERT OR IGNORE INTO SETTINGS VALUES (NULL, 'LOAD_BALANCING_STRATEGY', 0);" // initialize settings table

	_, err = configDb.Exec(upstreamsStatement)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
	}

	_, err = configDb.Exec(settingsStatement)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
	}

	http.HandleFunc("GET /config/upstreams", ReturnUpstreams)
	http.HandleFunc("POST /config/add-upstream", AddUpstream)
	http.HandleFunc("DELETE /config/delete-upstream", DeleteUpstream)
	http.HandleFunc("PATCH /config/update-load-balancing-strategy", UpdateLoadBalancingStrat)
	// http.Handle("GET /", http.FileServer(http.Dir("./")))
	http.HandleFunc("/proxy/*", ReverseProxy)
	log.Fatal(http.ListenAndServe(":3690", nil))
}
