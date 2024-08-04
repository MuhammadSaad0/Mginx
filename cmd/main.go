package main

import (
	"context"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
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
	rows, err := configDb.Query("SELECT URL FROM UPSTREAMS")
	if err != nil {
		fmt.Fprintln(writer, err.Error())
		return
	}
	rows.Next() // use the first upstream url for now
	var data string
	err = rows.Scan(&data)
	if err != nil {
		fmt.Fprintln(writer, err.Error())
		return
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
			TLSClientConfig: &tls.Config{
			},
		},	
	}
	response, err := defaultClient.Do(modifiedRequest) // perform http request
	if err != nil {
		fmt.Fprintf(writer, "Error sending request %s", err.Error())
		return
	}
	defer response.Body.Close() // close body to make sure socket isnt left hanging
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
	for rows.Next(){
		var upstreamId interface {}
		var upstreamUrl interface {}
		err = rows.Scan(&upstreamId, &upstreamUrl)
		if err != nil {
			rwLock.RUnlock()
			fmt.Fprintln(wrriter, err.Error())
			return
		}
		var finalData map[string] interface {} = make(map[string]interface{})
		finalData["id"] = upstreamId
		finalData["url"] = upstreamUrl
		toRet = append(toRet, finalData)
	}
	rwLock.RUnlock()
	response := make(map[string][]interface{})
	if len(toRet) > 0 {
		response["proxies"] = toRet
	}else{
		response["proxies"] = make([]interface{}, 0)
	}
	encoder := json.NewEncoder(wrriter)
	encoder.Encode(response)
}

type addUpstream struct {
	Url string `json:"url"`
}

func AddUpstream(writer http.ResponseWriter, request *http.Request){
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

	_, err = configDb.Query("DELETE FROM UPSTREAMS WHERE ID = ?", data.Id)

	if err != nil {
		fmt.Fprintln(writer, err.Error())
		return
	}

	fmt.Fprintln(writer, "Upstream deleted")

}

func main() {
	var err error
	configDb, err = sql.Open("sqlite", "config.db")
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
	}
	configStatement := "CREATE TABLE IF NOT EXISTS UPSTREAMS (ID INTEGER PRIMARY KEY, URL STRING);" // initialize config table
	_, err = configDb.Exec(configStatement)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
	}

	http.HandleFunc("GET /config/upstreams", ReturnUpstreams)
	http.HandleFunc("POST /config/add-upstream", AddUpstream)
	http.HandleFunc("DELETE /config/delete-upstream", DeleteUpstream)
	// http.Handle("GET /", http.FileServer(http.Dir("./")))
	http.HandleFunc("/proxy/*", ReverseProxy)
	log.Fatal(http.ListenAndServe(":3690", nil))
}