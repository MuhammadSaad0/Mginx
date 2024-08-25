package healthcheck

import (
	"fmt"
	"mginx/internals/db"
	"net/http"
	"os"
	"time"
)

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
