package shadow

import (
	"crypto/tls"
	"fmt"
	"io"
	"mginx/internals/db"
	"net/http"
	"os"
)

func sendRequest(request *http.Request, retChannel chan *http.Response) {
	defaultClient := http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{},
		},
	}
	response, _ := defaultClient.Do(request)
	retChannel <- response
}

func SendToShadowUrls(writer *os.File, request *http.Request) {
	// send request in parallel to all shadow urls
	// if endpoints added to shadow endpoints make sure only those are shadowed
	rows, err := db.ConfigDb.Query("SELECT ENDPOINT FROM SHADOW_ENDPOINTS;")
	if err != nil {
		fmt.Fprintln(writer, err.Error())
		return
	}
	var shadowChecks []string
	for rows.Next() {
		var shadowEndpoint string
		err = rows.Scan(&shadowEndpoint)
		if err != nil {
			fmt.Fprintln(writer, err.Error())
			return
		}
		shadowChecks = append(shadowChecks, shadowEndpoint)

	}
	urlFound := 0
	for _, url := range shadowChecks {
		if err != nil {
			fmt.Fprintln(writer, "Error Parsing url")
			return
		}
		if url == request.URL.String() {
			urlFound = 1
			break
		}
	}
	if urlFound == 0 {
		return
	}
	var data string
	rows, err = db.ConfigDb.Query("SELECT URL FROM UPSTREAMS WHERE SHADOW=1")
	if err != nil {
		fmt.Fprintln(writer, err.Error())
		return
	}
	channel := make(chan *http.Response)

	for rows.Next() {
		err = rows.Scan(&data)
		if err != nil {
			fmt.Fprintln(writer, err.Error())
			return
		}
		sendRequest(request, channel)
	}
	rows.Close()
	io.Copy(writer, (<-channel).Body)
}
