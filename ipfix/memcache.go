//: ----------------------------------------------------------------------------
//: Copyright (C) 2017 Verizon.  All Rights Reserved.
//: All Rights Reserved
//:
//: file:    memcache.go
//: details: TODO
//: author:  Mehrdad Arshad Rad
//: date:    02/01/2017
//:
//: Licensed under the Apache License, Version 2.0 (the "License");
//: you may not use this file except in compliance with the License.
//: You may obtain a copy of the License at
//:
//:     http://www.apache.org/licenses/LICENSE-2.0
//:
//: Unless required by applicable law or agreed to in writing, software
//: distributed under the License is distributed on an "AS IS" BASIS,
//: WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//: See the License for the specific language governing permissions and
//: limitations under the License.
//: ----------------------------------------------------------------------------
package ipfix

import (
	"encoding/binary"
	"encoding/json"
	"hash/fnv"
	"io/ioutil"
	"net"
	"sync"
	"time"
)

var ShardNo = 32

type MemCache []*TemplatesShard

type Data struct {
	Template  TemplateRecords
	Timestamp int64
}

type TemplatesShard struct {
	Templates map[uint32]Data
	sync.RWMutex
}
type MemCacheDisk struct {
	Cache   MemCache
	ShardNo int
}

func GetCache() MemCache {
	var (
		mem MemCacheDisk
		err error
	)

	b, err := ioutil.ReadFile("/tmp/vflow.templates")
	if err == nil {
		err = json.Unmarshal(b, &mem)
		if err == nil && mem.ShardNo == ShardNo {
			return mem.Cache
		}
	}

	m := make(MemCache, ShardNo)
	for i := 0; i < ShardNo; i++ {
		m[i] = &TemplatesShard{Templates: make(map[uint32]Data)}
	}

	return m
}

func (m MemCache) getShard(id uint16, addr net.IP) (*TemplatesShard, uint32) {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, id)
	key := append(addr, b...)

	hash := fnv.New32()
	hash.Write(key)
	hSum32 := hash.Sum32()

	return m[uint(hSum32)%uint(ShardNo)], hSum32
}

func (m *MemCache) insert(id uint16, addr net.IP, tr TemplateRecords) {
	shard, key := m.getShard(id, addr)
	shard.Lock()
	defer shard.Unlock()
	shard.Templates[key] = Data{tr, time.Now().Unix()}
}

func (m *MemCache) retrieve(id uint16, addr net.IP) (TemplateRecords, bool) {
	shard, key := m.getShard(id, addr)
	shard.RLock()
	defer shard.RUnlock()
	v, ok := shard.Templates[key]

	return v.Template, ok
}

func (m MemCache) Dump() error {
	// TODO
	b, err := json.Marshal(
		MemCacheDisk{
			m,
			ShardNo,
		},
	)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile("/tmp/vflow.templates", b, 0644)
	if err != nil {
		return err
	}

	return nil
}

/*
func memKey(ip []byte, id uint16) string {
	hash := fnv.New32()
	hash.Write(ip)
	//fmt.Printf("HASH:%#v\n", hash.Sum32())
	return fmt.Sprintf("%d:%d", hash.Sum32(), id)
}

func memKey1(ip []byte, id uint16) uint32 {
	b := make([]byte, 2)
	b[0] = byte(id >> 8)
	b[1] = byte(id)

	ip = append(ip, b...)
	hash := fnv.New32()
	hash.Write(ip)

	return hash.Sum32()
}
*/