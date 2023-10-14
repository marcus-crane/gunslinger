package db

import (
	"sync"

	"github.com/marcus-crane/gunslinger/models"
)

type mapStore struct {
	m    *sync.Mutex
	data []models.ComboDBMediaItem
}

func newMapStore() *mapStore {
	return &mapStore{
		m:    new(sync.Mutex),
		data: []models.ComboDBMediaItem{},
	}
}

func (ms *mapStore) Store(c models.ComboDBMediaItem) (uint, error) {
	ms.m.Lock()
	defer ms.m.Unlock()
	ms.data = append(ms.data, c)
	return c.ID, nil
}

func (ms *mapStore) RetrieveAll() ([]models.ComboDBMediaItem, error) {
	return ms.data, nil
}
