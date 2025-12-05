package repository

import (
	"encoding/json"
	"os"
	"pixel-game/models" // Remplace par le nom de ton module
	"sync"
)

type Database struct {
	mu          sync.Mutex
	HistoryFile string

	// Cache en mémoire pour éviter de relire le fichier tout le temps
	historyCache []models.HistoryEntry
}

func NewDatabase(historyFile string) *Database {
	db := &Database{
		HistoryFile:  historyFile,
		historyCache: []models.HistoryEntry{},
	}
	db.loadHistory()
	return db
}

func (db *Database) loadHistory() {
	file, err := os.ReadFile(db.HistoryFile)
	if err == nil {
		json.Unmarshal(file, &db.historyCache)
	}
}

func (db *Database) AddHistoryEntry(entry models.HistoryEntry) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.historyCache = append(db.historyCache, entry)

	// Sauvegarde dans le fichier JSON
	data, err := json.MarshalIndent(db.historyCache, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(db.HistoryFile, data, 0644)
}

func (db *Database) GetFullHistory() []models.HistoryEntry {
	db.mu.Lock()
	defer db.mu.Unlock()
	return db.historyCache
}

func (db *Database) GetHistoryEntry(index int) (models.HistoryEntry, bool) {
	db.mu.Lock()
	defer db.mu.Unlock()
	if index < 0 || index >= len(db.historyCache) {
		return models.HistoryEntry{}, false
	}
	return db.historyCache[index], true
}
