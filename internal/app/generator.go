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
	"fmt"
	"math/rand"
	"time"

	influxdb "github.com/influxdata/influxdb-client-go/v2"
	"github.com/pingcap/tipb/go-tipb"
)

const (
	sqlNum      = 200
	instanceNum = 100
	startTs     = 0
	endTs       = 100
)

func GenerateCPUTimeRecords(recordChan chan *tipb.CPUTimeRecord) {
	for ts := startTs; ts < endTs; ts++ {
		for i := 0; i < instanceNum; i++ {
			for j := 0; j < sqlNum; j++ {
				recordChan <- &tipb.CPUTimeRecord{
					SqlDigest:     []byte(fmt.Sprintf("i%d_sql%d", i, j)),
					PlanDigest:    []byte(fmt.Sprintf("i%d_plan%d", i, j)),
					TimestampList: []uint64{uint64(ts)},
					CpuTimeMsList: []uint32{uint32(rand.Uint32() % 1000)},
				}
			}
		}
	}
}

func GenerateSQLMeta(sqlMetaChan chan *tipb.SQLMeta) {
	for i := 0; i < instanceNum; i++ {
		for j := 0; j < sqlNum; j++ {
			sqlDigest := []byte(fmt.Sprintf("i%d_sql%d", i, j))
			sql := make([]byte, rand.Uint32()%1024*1000+1024*100)
			sql = append(sql, sqlDigest...)
			for i := len(sqlDigest); i < len(sql); i++ {
				sql[i] = 'o'
			}
			sqlMetaChan <- &tipb.SQLMeta{
				SqlDigest:     sqlDigest,
				NormalizedSql: string(sql),
			}
		}
	}
}

func GeneratePlanMeta(planMetaChan chan *tipb.PlanMeta) {
	for i := 0; i < instanceNum; i++ {
		for j := 0; j < sqlNum; j++ {
			planDigest := []byte(fmt.Sprintf("i%d_plan%d", i, j))
			plan := make([]byte, rand.Uint32()%1024*1000+1024*100)
			plan = append(plan, planDigest...)
			for i := len(planDigest); i < len(plan); i++ {
				plan[i] = 'o'
			}
			planMetaChan <- &tipb.PlanMeta{
				PlanDigest:     planDigest,
				NormalizedPlan: string(plan),
			}
		}
	}
}

func WriteInfluxDB() {
	recordChan := make(chan *tipb.CPUTimeRecord, 1)
	sqlMetaChan := make(chan *tipb.SQLMeta, 1)
	planMetaChan := make(chan *tipb.PlanMeta, 1)
	go GenerateCPUTimeRecords(recordChan)
	go GenerateSQLMeta(sqlMetaChan)
	go GeneratePlanMeta(planMetaChan)

	org := "pingcap"
	bucket := "test"
	token := "cUDigADLBUvQHTabhzjBjL_YM1MVofBUUSZx_-uwKy8mR4S_Eqjt6myugvj3ryOfRUBHOGnlyCbTkKbNGVt1rQ=="
	url := "http://localhost:2333"
	client := influxdb.NewClient(url, token)
	writeAPI := client.WriteAPIBlocking(org, bucket)
	p := influxdb.NewPoint("stat",
		map[string]string{"uint": "temperatur"},
		map[string]interface{}{"avg": 24.5, "max": 45},
		time.Now(),
	)
	writeAPI.WritePoint(context.TODO(), p)
	client.Close()
}