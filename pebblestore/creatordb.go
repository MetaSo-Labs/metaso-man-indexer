package pebblestore

import (
	"manindexer/pin"
	"sync"

	"github.com/cockroachdb/pebble"
)

func (idx *Database) BatchInsertCreator(data map[string]string, allMap *sync.Map) error {
	batch := idx.CreatorDb.NewBatch()
	localMap := make(map[string]string)
	for k, v := range data {
		batch.Set([]byte(k), []byte(v), nil)
		localMap[k] = v
	}
	if err := batch.Commit(nil); err != nil {
		batch.Close()
		return err
	}
	batch.Close()
	for k, v := range localMap {
		allMap.Store(k, v)
	}
	return nil
}
func GetAllCreator(db *pebble.DB, allMap *sync.Map) (err error) {
	it, err := db.NewIter(nil)
	if err != nil {
		return
	}
	defer it.Close()
	localMap := make(map[string]string)
	for it.First(); it.Valid(); it.Next() {
		localMap[string(it.Key())] = string(it.Value())
	}
	for k, v := range localMap {
		allMap.Store(k, v)
	}
	return
}
func (idx *Database) BatchDeleteCreator(txId []string) error {
	batch := idx.CreatorDb.NewBatch()
	for _, v := range txId {
		batch.Delete([]byte(v), nil)
		pin.AllCreatorAddress.Delete(v)
	}
	if err := batch.Commit(nil); err != nil {
		batch.Close()
		return err
	}
	batch.Close()

	return nil
}
func (idx *Database) InsertMrcData(pinId string) error {
	return idx.MrcDb.Set([]byte(pinId), nil, pebble.Sync)
}
func (idx *Database) CheckMrcData(pinId string) error {
	_, closer, err := idx.MrcDb.Get([]byte(pinId))
	if err != nil {
		return err
	}
	closer.Close()
	return nil
}
func GetAllMrc(db *pebble.DB, allMap *sync.Map) (err error) {
	it, err := db.NewIter(nil)
	if err != nil {
		return
	}
	defer it.Close()
	localMap := make(map[string]string)
	for it.First(); it.Valid(); it.Next() {
		localMap[string(it.Key())] = string(it.Value())
	}
	for k, v := range localMap {
		allMap.Store(k, v)
	}
	return
}
