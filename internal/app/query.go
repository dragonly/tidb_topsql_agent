/*
Copyright © 2021 Li Yilong <liyilongko@gmail.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package app

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	influxdb "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
)

// TODO: join with sql/plan meta
func queryInfluxDB(queryAPI api.QueryAPI, startTs, endTs, instance_id int) {
	query := fmt.Sprintf(`from(bucket:"test")
	|> range(start:%d, stop:%d)
	|> filter(fn: (r) =>
		r._measurement == "cpu_time" and
		r.instance_id == "%d"
	)
	// |> window(every: 1m)
	// |> sum(column: "_value")
	// |> duplicate(column: "_stop", as: "_time")
	|> drop(columns: ["_start", "_stop", "_measurement"])
	|> group(columns: ["_time"])
	|> sort(columns: ["_value"], desc: true)
	|> limit(n:5)
	//|> window(every: inf)
	`, startTs, endTs, instance_id)
	result, err := queryAPI.Query(context.TODO(), query)
	if err != nil {
		log.Printf("failed to execute query, %v\n%s\n", err, query)
		return
	}
	for result.Next() {
		if result.TableChanged() {
			log.Printf("table: %s\n", result.TableMetadata().String())
		}
		log.Printf("value: %v\n", result.Record().Values())
	}
	if result.Err() != nil {
		log.Printf("query parsing error: %v\n", result.Err().Error())
	}
}

func QueryInfluxDB() {
	url := "http://localhost:2333"
	token := "cUDigADLBUvQHTabhzjBjL_YM1MVofBUUSZx_-uwKy8mR4S_Eqjt6myugvj3ryOfRUBHOGnlyCbTkKbNGVt1rQ=="
	org := "pingcap"
	client := influxdb.NewClient(url, token)
	queryAPI := client.QueryAPI(org)
	queryInfluxDB(queryAPI, 0, 60*60, 1)
}

func QueryTiDB(dsn string, workers, queryNum int, randomQuery bool) {
	if randomQuery {
		rand.Seed(time.Now().Unix())
	}
	log.Printf("workers: %d, queryNum: %d, randomQuery: %t\n", workers, queryNum, randomQuery)

	var wg sync.WaitGroup
	workCount := int32(queryNum)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go queryTiDBWorker(&wg, &workCount, dsn, randomQuery, 0, 3600*24, 300, 0, 5)
	}
	wg.Wait()
}

// the worker could do more work than workCount, because of the unprotected check-and-set behavior
func queryTiDBWorker(wg *sync.WaitGroup, workCount *int32, dsn string, randomQuery bool, startTs, endTs, interval, instance_id, top_n int) {
	log.Printf("[%d] start worker\n", getGoroutineID())
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("failed to open db: %v\n", err)
	}
	db.Exec("set @@session.tidb_isolation_read_engines = \"tidb, tiflash\"")
	defer db.Close()
	for {
		if atomic.LoadInt32(workCount) <= 0 {
			break
		}
		atomic.AddInt32(workCount, -1)
		if randomQuery {
			duration := (endTs - startTs) * int(0.5+0.5*rand.Float32())
			endTs = startTs + duration
			interval *= int(0.5 + rand.Float32())
			instance_id = int(rand.Float32() * instanceNum)
			top_n *= int(0.5 + rand.Float32())
		}
		log.Printf("[%d] start query\n", getGoroutineID())
		now := time.Now()
		queryTiDB(db, startTs, endTs, interval, instance_id, top_n)
		log.Printf("[%d] finish query in %dms\n", getGoroutineID(), time.Since(now).Milliseconds())
	}
	wg.Done()
	log.Printf("[%d] stop worker\n", getGoroutineID())
}

// TODO: join with sql/plan meta
func queryTiDB(db *sql.DB, startTs, endTs, interval, instance_id, top_n int) {
	sql := `
	SELECT *
	FROM (SELECT *, RANK() OVER (PARTITION BY topsql.time_window ORDER BY topsql.cpu_time_sum DESC ) AS rk
		FROM (SELECT instance_id, sql_digest, floor(timestamp / %d) AS time_window, sum(cpu_time_ms) AS cpu_time_sum
			FROM cpu_time
			WHERE timestamp >= %d
				AND timestamp < %d
				AND instance_id = %d
			GROUP BY time_window, sql_digest) topsql) sql_ranked
	WHERE sql_ranked.rk <= %d
	`
	sql = fmt.Sprintf(sql, interval, startTs, endTs, instance_id, top_n)
	rows, err := db.Query(sql)
	if err != nil {
		log.Fatalf("failed to query TiDB, %v", err)
	}
	defer rows.Close()
	// log.Println("cpu_time: cpu_time_count, minute")
	for rows.Next() {
		var instanceID, timeWindow, cpuTimeMsSum, rank int
		var sqlDigest string
		if err := rows.Scan(&instanceID, &sqlDigest, &timeWindow, &cpuTimeMsSum, &rank); err != nil {
			log.Fatalf("failed to iterate rows, %v", err)
		}
		// log.Printf("row: %d, %d\n", cpu_time_count, minute)
	}
	log.Println("query success")
}

func getGoroutineID() int {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	idField := strings.Fields(strings.TrimPrefix(string(buf[:n]), "goroutine "))[0]
	id, err := strconv.Atoi(idField)
	if err != nil {
		panic(fmt.Sprintf("cannot get goroutine id: %v", err))
	}
	return id
}
