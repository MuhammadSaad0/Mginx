package proxy

import (
	"context"
	"crypto/tls"
	"database/sql"
	"fmt"
	"io"
	"math/rand"
	"mginx/internals/db"
	"net/http"
	"net/url"
	"strings"

	_ "modernc.org/sqlite"
)

func ReverseProxy(writer http.ResponseWriter, request *http.Request) {
	// copy incoming request
	modifiedRequest := request.Clone(context.Background())
	path := request.URL.Path
	query := request.URL.Query()

	var rows *sql.Rows
	var err error
	db.RwLock.RLock()
	rows, err = db.ConfigDb.Query("SELECT SETTING_VALUE FROM SETTINGS WHERE SETTING_NAME='LOAD_BALANCING_STRATEGY';")
	if err != nil {
		db.RwLock.RUnlock()
		fmt.Fprintln(writer, err.Error())
		return
	}
	db.RwLock.RUnlock()
	rows.Next()
	var loadBalancingStrat int
	rows.Scan(&loadBalancingStrat)
	rows.Close()
	var data string
	if loadBalancingStrat == 0 { // use primary
		rows, err = db.ConfigDb.Query("SELECT URL FROM UPSTREAMS WHERE IS_PRIMARY=1")
		if err != nil {
			fmt.Fprintln(writer, err.Error())
			return
		}
		rows.Next()
		err = rows.Scan(&data)
		rows.Close()
		if err != nil {
			fmt.Fprintln(writer, err.Error())
			return
		}

	} else if loadBalancingStrat == 1 { // round robin
		rows, err = db.ConfigDb.Query("SELECT URL FROM UPSTREAMS")
		if err != nil {
			fmt.Fprintln(writer, err.Error())
			return
		}
		upstreams := make([]string, 0, 10)

		for rows.Next() {
			err = rows.Scan(&data)
			if err != nil {
				fmt.Fprintln(writer, err.Error())
				return
			}
			upstreams = append(upstreams, data)
		}
		rows.Close()
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
	modifiedRequest.Host = request.URL.Host
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
