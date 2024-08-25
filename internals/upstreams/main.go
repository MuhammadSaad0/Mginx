package upstreams

import (
	"context"
	"encoding/json"
	"fmt"
	"mginx/internals/db"
	"mginx/internals/types"
	"mginx/views/components"
	"net/http"
	"strconv"

	_ "modernc.org/sqlite"
)

func ReturnUpstreams(wrriter http.ResponseWriter, request *http.Request) {
	// returns data for all upstream servers
	var toRet []types.UpstreamRow
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
		var upstream types.UpstreamRow
		err = rows.Scan(&upstream.UpstreamId, &upstream.UpstreamUrl, &upstream.Online, &upstream.Primary, &upstream.Shadow)
		if err != nil {
			fmt.Fprintln(wrriter, err.Error())
			return
		}
		toRet = append(toRet, upstream)
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
	rows.Close()
	db.RwLock.RUnlock()
	db.RwLock.Lock()
	var insertStatement string
	if count == int64(0) {
		insertStatement = "INSERT INTO UPSTREAMS (URL, IS_PRIMARY, SHADOW) VALUES (?, 1, 0);"
	} else {
		insertStatement = "INSERT INTO UPSTREAMS (URL) VALUES (?);"
	}
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

type toggleShadow struct {
	Id string `json:"id"`
}

func ToggleShadow(writer http.ResponseWriter, request *http.Request) {
	decoder := json.NewDecoder(request.Body)
	defer request.Body.Close()
	var data toggleShadow
	err := decoder.Decode(&data)
	if err != nil {
		component := components.Message("Unable to Set as Shadow!")
		component.Render(context.Background(), writer)
	}
	id, err := strconv.Atoi(data.Id)
	if err != nil {
		component := components.Message("Unable to Set as Shadow!")
		component.Render(context.Background(), writer)
	}
	rows, err := db.ConfigDb.Query("SELECT SHADOW FROM UPSTREAMS WHERE ID = ?", id)
	if err != nil {
		component := components.Message("Unable to Set as Shadow!")
		component.Render(context.Background(), writer)
	}
	var currentShadow int64
	rows.Next()
	err = rows.Scan(&currentShadow)
	rows.Close()
	if err != nil {
		component := components.Message("Unable to Set as Shadow!")
		component.Render(context.Background(), writer)
	}
	var newShadow int64
	if currentShadow == 0 {
		newShadow = 1
	} else {
		newShadow = 0
	}
	_, err = db.ConfigDb.Exec("UPDATE UPSTREAMS SET SHADOW = ? WHERE ID = ?", newShadow, id)
	if err != nil {
		component := components.Message("Unable to Set as Shadow!")
		component.Render(context.Background(), writer)
	}
	ReturnUpstreams(writer, request)
}
