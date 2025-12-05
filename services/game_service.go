package services

import (
	"encoding/json"
	"fmt"
	"os"
	"pixel-game/models"
	"pixel-game/repository"
	"sync"
	"time"
)

// ---------------------
// Structures et types
// ---------------------

type GameService struct {
	repo *repository.Database
	mu   sync.Mutex

	// Config
	TimerDuration  time.Duration
	Models         [][][]int
	AutoSwitchMode bool // <--- NOUVEAU : État du mode Auto-Switch

	// État courant
	CurrentModelIdx int
	SolvedGrid      [][]string
	SolvedBy        string
	IsSolved        bool
	NextSwitch      time.Time
	SubmissionCount int
}

// ---------------------
// Chargement des modèles
// ---------------------

func loadModelsFromFile(path string) ([][][]int, error) {
	type modelEntry struct {
		Name string  `json:"name"`
		Grid [][]int `json:"grid"`
	}

	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var data struct {
		Models []modelEntry `json:"models"`
	}

	if err := json.Unmarshal(file, &data); err != nil {
		return nil, err
	}

	modelsList := make([][][]int, len(data.Models))
	for i, m := range data.Models {
		modelsList[i] = m.Grid
	}

	return modelsList, nil
}

// ---------------------
// Constructeur
// ---------------------

func NewGameService(repo *repository.Database) *GameService {
	modelsList, err := loadModelsFromFile("data/pixel_model.json")
	if err != nil {
		panic("Impossible de charger les modèles: " + err.Error())
	}

	gs := &GameService{
		repo:            repo,
		TimerDuration:   1 * time.Minute,
		Models:          modelsList,
		CurrentModelIdx: 0,
		NextSwitch:      time.Now().Add(1 * time.Minute),
		AutoSwitchMode:  false, // Par défaut désactivé
	}

	go gs.gameLoop()
	return gs
}

// ---------------------
// Boucle de jeu
// ---------------------

func (s *GameService) gameLoop() {
	ticker := time.NewTicker(1 * time.Second)
	for range ticker.C {
		s.mu.Lock()
		// On ne change automatiquement par le timer que si on n'est PAS en mode "Solved"
		// Ou selon tes règles. Ici je laisse la logique timer classique.
		if time.Now().After(s.NextSwitch) {
			s.nextModelInternal()
			fmt.Println("AUTO: Nouveau dessin (Timer écoulé).")
		}
		s.mu.Unlock()
	}
}

// Fonction interne pour passer au suivant (ne pas appeler sans Lock)
func (s *GameService) nextModelInternal() {
	s.CurrentModelIdx = (s.CurrentModelIdx + 1) % len(s.Models)
	s.IsSolved = false
	s.SolvedGrid = nil
	s.SolvedBy = ""
	s.NextSwitch = time.Now().Add(s.TimerDuration)
}

// ---------------------
// Méthodes publiques (Lecture / Jeu)
// ---------------------

func (s *GameService) GetState() models.GameStateResponse {
	s.mu.Lock()
	defer s.mu.Unlock()

	timeLeft := time.Until(s.NextSwitch).Seconds()
	if timeLeft < 0 {
		timeLeft = 0
	}

	fullHistory := s.repo.GetFullHistory()
	lastIdx := len(fullHistory)
	startIdx := lastIdx - 10
	if startIdx < 0 {
		startIdx = 0
	}

	recent := make([]models.HistoryEntry, 0)
	for i := lastIdx - 1; i >= startIdx; i-- {
		recent = append(recent, fullHistory[i])
	}

	return models.GameStateResponse{
		TimeLeft:        timeLeft,
		IsSolved:        s.IsSolved,
		SolvedBy:        s.SolvedBy,
		TargetGrid:      s.Models[s.CurrentModelIdx],
		SolvedGrid:      s.SolvedGrid,
		CurrentModel:    s.CurrentModelIdx,
		TotalModels:     len(s.Models),
		SubmissionCount: s.SubmissionCount,
		RecentHistory:   recent,
		AutoSwitch:      s.AutoSwitchMode, // <--- NOUVEAU : Envoyer l'état au front
	}
}

func (s *GameService) ProcessSubmission(req models.CheckRequest) (bool, string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	target := s.Models[s.CurrentModelIdx]
	correct := true

	for rIndex, row := range target {
		for cIndex, val := range row {
			if rIndex >= len(req.UserGrid) || cIndex >= len(req.UserGrid[rIndex]) {
				correct = false
				break
			}

			userColor := req.UserGrid[rIndex][cIndex]
			isEmpty := userColor == "" || userColor == "#ffffff" || userColor == "transparent"

			if val == 1 && isEmpty {
				correct = false
			}
			if val == 0 && !isEmpty {
				correct = false
			}
		}
	}

	if correct {
		normalizedGrid := make([][]string, len(req.UserGrid))
		for r := range req.UserGrid {
			normalizedGrid[r] = make([]string, len(req.UserGrid[r]))
			for c := range req.UserGrid[r] {
				val := req.UserGrid[r][c]
				if val == "" || val == "transparent" {
					normalizedGrid[r][c] = "#ffffff"
				} else {
					normalizedGrid[r][c] = val
				}
			}
		}

		// 1. Enregistrer la victoire dans l'historique
		s.SolvedGrid = normalizedGrid
		s.IsSolved = true
		s.SolvedBy = req.Name
		s.SubmissionCount++

		entry := models.HistoryEntry{
			Name:      req.Name,
			Grid:      normalizedGrid,
			Timestamp: time.Now(),
			ModelID:   s.CurrentModelIdx,
		}
		s.repo.AddHistoryEntry(entry)

		// 2. LOGIQUE AUTO-SWITCH
		// Si le mode est activé, on passe immédiatement au suivant
		if s.AutoSwitchMode {
			s.nextModelInternal() // Passe au suivant et remet IsSolved à false
			return true, "BRAVO " + req.Name + " ! (Passage automatique au suivant)"
		}

		return true, "BRAVO " + req.Name + " ! Validé !"
	}

	return false, "Incorrect. Vérifie le modèle."
}

// ---------------------
// Méthodes de mutation d'état (utilisées par AdminService)
// ---------------------

func (s *GameService) SetTimerDuration(d time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.TimerDuration = d
	s.NextSwitch = time.Now().Add(s.TimerDuration)
}

// NOUVEAU : Activer/Désactiver le mode Auto-Switch
func (s *GameService) SetAutoSwitchMode(enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.AutoSwitchMode = enabled
}

func (s *GameService) ForceModelIndex(index int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if index < 0 || index >= len(s.Models) {
		return fmt.Errorf("index invalide")
	}
	s.CurrentModelIdx = index
	s.IsSolved = false
	s.SolvedGrid = nil
	s.SolvedBy = ""
	s.NextSwitch = time.Now().Add(s.TimerDuration)
	return nil
}

func (s *GameService) ForceNextRound() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextModelInternal()
}
