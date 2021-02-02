package cli

import (
	"encoding/json"
	"os/user"
	"path/filepath"
	"reflect"
	"sort"
	"time"

	"github.com/qjpcpu/common.v2/assert"
	"github.com/xujiajun/nutsdb"
	"github.com/xujiajun/nutsdb/ds/zset"
)

const (
	defaultDir = ".disk-cache"
)

var (
	MaxItemHistoryBucket = 100
)

type FileDB struct {
	db         *nutsdb.DB
	bucketSize map[string]int
}

func MustNewHomeFileDB(ns ...string) *FileDB {
	f, err := NewHomeFileDB(ns...)
	assert.ShouldBeNil(err, "Can't create file db")
	return f
}

func NewHomeFileDB(ns ...string) (*FileDB, error) {
	usr, err := user.Current()
	if err != nil {
		return nil, err
	}
	ps := append([]string{usr.HomeDir, defaultDir}, ns...)
	return NewFileDB(filepath.Join(ps...))
}

func NewFileDB(dbdir string) (*FileDB, error) {
	opt := nutsdb.DefaultOptions
	opt.Dir = dbdir
	if db, err := nutsdb.Open(opt); err != nil {
		return nil, err
	} else {
		return &FileDB{db: db, bucketSize: make(map[string]int)}, nil
	}
}

func (fdb *FileDB) Close() {
	fdb.db.Close()
}

func (fdb *FileDB) GetItemHistoryBucket(bucket string, size int) *ItemHistoryBucket {
	return &ItemHistoryBucket{
		DB:     fdb,
		bucket: bucket,
		size:   size,
	}
}

func (fdb *FileDB) zsetBucketExist(bucket string) bool {
	if fdb.db.SortedSetIdx == nil {
		return false
	} else if _, ok := fdb.db.SortedSetIdx[bucket]; !ok {
		return false
	}
	return true
}

type Keyer interface {
	GetKey() string
}

type ItemHistoryBucket struct {
	DB     *FileDB
	size   int
	bucket string
}

func (ih *ItemHistoryBucket) InsertItem(v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	now := time.Now().UnixNano()
	return ih.getNutsDB().Update(
		func(tx *nutsdb.Tx) error {
			if ih.size > 0 && ih.DB.zsetBucketExist(ih.bucket) {
				if size, _ := tx.ZCard(ih.bucket); size >= ih.size {
					for i := 0; i < size-ih.size+1; i++ {
						tx.ZPopMin(ih.bucket)
					}
				}
			}
			key := data
			if keyer, ok := v.(Keyer); ok {
				key = []byte(keyer.GetKey())
			}
			return tx.ZAdd(ih.bucket, key, float64(now), data)
		})
}

func (ih *ItemHistoryBucket) ListItem(retSlicePtr interface{}) error {
	if !ih.DB.zsetBucketExist(ih.bucket) {
		return nil
	}
	var list []*zset.SortedSetNode
	if err := ih.getNutsDB().View(
		func(tx *nutsdb.Tx) error {
			if nodes, err := tx.ZMembers(ih.bucket); err != nil {
				return err
			} else {
				for _, node := range nodes {
					list = append(list, node)
				}
			}
			return nil
		}); err != nil {
		return err
	}
	/* sort by timestamp desc */
	sort.SliceStable(list, func(i, j int) bool {
		return int64(list[i].Score()) > int64(list[j].Score())
	})

	/* fill retSlicePtr */
	tp := reflect.TypeOf(retSlicePtr)
	v := reflect.ValueOf(retSlicePtr)
	vElem := v.Elem()
	elemType := tp.Elem().Elem()
	var elemIsPtr bool
	if elemIsPtr = elemType.Kind() == reflect.Ptr; elemIsPtr {
		elemType = elemType.Elem()
	}
	for _, node := range list {
		nv := reflect.New(elemType)
		if err := json.Unmarshal(node.Value, nv.Interface()); err != nil {
			continue
		}
		if !elemIsPtr {
			nv = nv.Elem()
		}
		vElem = reflect.Append(vElem, nv)
	}
	v.Elem().Set(vElem)
	return nil
}

func (ih *ItemHistoryBucket) getNutsDB() *nutsdb.DB {
	return ih.DB.db
}

type BucketKV struct {
	DB     *FileDB
	bucket string
}

func (fdb *FileDB) GetBucketKV(bucket string) *BucketKV {
	return &BucketKV{
		DB:     fdb,
		bucket: bucket,
	}
}

func (kv *BucketKV) Put(key string, val interface{}) error {
	return kv.PutWithTTL(key, val, 0)
}

func (kv *BucketKV) PutWithTTL(key string, val interface{}, ttlSec uint32) error {
	data, err := json.Marshal(val)
	if err != nil {
		return err
	}
	return kv.DB.db.Update(
		func(tx *nutsdb.Tx) error {
			if err := tx.Put(kv.bucket, []byte(key), data, ttlSec); err != nil {
				return err
			}
			return nil
		})
}

func (kv *BucketKV) Get(key string, valPtr interface{}) error {
	return kv.DB.db.View(
		func(tx *nutsdb.Tx) error {
			if e, err := tx.Get(kv.bucket, []byte(key)); err != nil {
				return err
			} else if err = json.Unmarshal(e.Value, valPtr); err != nil {
				return err
			}
			return nil
		})
}

func (kv *BucketKV) Delete(key string) error {
	return kv.DB.db.Update(
		func(tx *nutsdb.Tx) error {
			if err := tx.Delete(kv.bucket, []byte(key)); err != nil {
				return err
			}
			return nil
		})
}

func (kv *BucketKV) GetBytes(key string) []byte {
	var ret []byte
	kv.Get(key, &ret)
	return ret
}

func (kv *BucketKV) GetString(key string) string {
	return string(kv.GetBytes(key))
}
