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

import "github.com/pingcap/tipb/go-tipb"

type Store interface {
	WriteCPUTimeRecord(*tipb.CPUTimeRecord)
	WriteSQLMeta(*tipb.SQLMeta)
	WritePlanMeta(*tipb.PlanMeta)
}

// MemStore is for testing purpose
type MemStore struct {
	cpuTimeRecordList []*tipb.CPUTimeRecord
	sqlMetaList       []*tipb.SQLMeta
	planMetaList      []*tipb.PlanMeta
}

func NewMemStore() *MemStore {
	return &MemStore{}
}

func (s *MemStore) WriteCPUTimeRecord(records *tipb.CPUTimeRecord) {
	s.cpuTimeRecordList = append(s.cpuTimeRecordList, records)
}

func (s *MemStore) WriteSQLMeta(meta *tipb.SQLMeta) {
	s.sqlMetaList = append(s.sqlMetaList, meta)
}

func (s *MemStore) WritePlanMeta(meta *tipb.PlanMeta) {
	s.planMetaList = append(s.planMetaList, meta)
}
