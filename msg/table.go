package msg

import (
	"time"
)

type MqTable struct {
	TableId    string
	Owner      string
	Created    time.Time
	LastAccess time.Time
	Expiry     time.Duration
	Items      map[string]interface{}
	Indexes    map[string]string
}

func NewTable(tableid string, owner string) *MqTable {
	ret := new(MqTable)
	ret.TableId = tableid
	ret.Owner = owner
	ret.Created = time.Now()
	ret.Items = make(map[string]interface{})
	ret.Indexes = make(map[string][]string)
	return ret
}

func (t *MqTable) RunIndex(indexname string, indexFunction func(d) string) error {
	indexes := make(map[string]string)
	for k, v := range t.Items {
		indexKey := indexFunction(v)
		_, e := indexes[k]
		if !e {
			indexes[indexKey] = make([]string, 0)
		}
		indexes[indexKey] = append(indexes[indexKey], k)
	}
	t.Indexes[indexname] = indexes
	return nil
}

func (t *MqTable) DropIndex(indexname string) {
	remove(t.Indexes[indexname])
}