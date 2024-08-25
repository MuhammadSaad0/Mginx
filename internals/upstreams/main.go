package upstreams

import (
	"context"
	"encoding/json"
	"fmt"
	"mginx/internals/db"
	"mginx/views/components"
	"net/http"
	"reflect"
	"strconv"

	_ "modernc.org/sqlite"
)

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
