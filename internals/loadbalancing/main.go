package loadbalancing

import (
	"context"
	"encoding/json"
	"fmt"
	"mginx/internals/db"
	"mginx/views/components"
	"net/http"
	"strconv"

	_ "modernc.org/sqlite"
)

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
