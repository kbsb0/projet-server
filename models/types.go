package models

import "time"

// Structures de données stockées en BDD (JSON) ou échangées
type HistoryEntry struct {
	Name      string     `json:"name"`
	Grid      [][]string `json:"grid"`
	Timestamp time.Time  `json:"timestamp"`
	ModelID   int        `json:"modelId"`
}

// Requêtes HTTP
type CheckRequest struct {
	UserGrid [][]string `json:"grid"`
	Name     string     `json:"name"`
}

type AdminTimerReq struct {
	DurationSec int `json:"duration"`
}

type AdminModelReq struct {
	ModelIndex int `json:"modelIndex"`
}

// Réponse pour l'état du jeu
type GameStateResponse struct {
	TimeLeft        float64        `json:"timeLeft"`
	IsSolved        bool           `json:"isSolved"`
	SolvedBy        string         `json:"solvedBy"`
	TargetGrid      [][]int        `json:"targetGrid"`
	SolvedGrid      [][]string     `json:"solvedGrid"`
	CurrentModel    int            `json:"currentModel"`
	TotalModels     int            `json:"totalModels"`
	SubmissionCount int            `json:"submissionCount"`
	RecentHistory   []HistoryEntry `json:"recentHistory"`
	AutoSwitch      bool           `json:"autoSwitch"`
}
