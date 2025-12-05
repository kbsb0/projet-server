package main

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// --- CONFIGURATION ---
var (
	// MODIF: TimerDuration est maintenant une variable pour être modifiée par l'admin
	TimerDuration = 1 * time.Minute
	ServerPort    = ":8080"
)

// --- MODELES DE DESSINS ---
var gridInvader = [][]int{
	{0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
}

var gridHeart = [][]int{
	{0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
}

// --- STRUCTURES DE DONNÉES ---

type HistoryEntry struct {
	Name      string     `json:"name"`
	Grid      [][]string `json:"grid"`
	Timestamp time.Time  `json:"timestamp"`
	ModelID   int        `json:"modelId"`
}

type CheckRequest struct {
	UserGrid [][]string `json:"grid"`
	Name     string     `json:"name"`
}

// Structures pour l'Admin
type AdminTimerReq struct {
	DurationSec int `json:"duration"`
}

type AdminModelReq struct {
	ModelIndex int `json:"modelIndex"`
}

// --- VARIABLES GLOBALES ---

var (
	mutex        sync.Mutex
	models       = [][][]int{gridInvader, gridHeart}
	currentModel = 0

	solvedGrid [][]string
	solvedBy   string
	isSolved   bool
	nextSwitch time.Time

	history         []HistoryEntry
	submissionCount int
)

func main() {
	r := gin.Default()
	r.Use(cors.Default())
	r.LoadHTMLGlob("templates/*")

	// Initialisation du premier timer
	nextSwitch = time.Now().Add(TimerDuration)
	go gameLoop()

	// ---------------------------------------------------------
	// ROUTES PUBLIQUES
	// ---------------------------------------------------------

	// 1. Dashboard Principal
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "dashboard.html", nil)
	})

	// 2. API Status (Dashboard)
	r.GET("/api/state", func(c *gin.Context) {
		mutex.Lock()
		defer mutex.Unlock()

		timeLeft := time.Until(nextSwitch).Seconds()
		if timeLeft < 0 {
			timeLeft = 0
		}

		// Récupérer les 10 derniers pour la sidebar
		lastIdx := len(history)
		startIdx := lastIdx - 10
		if startIdx < 0 {
			startIdx = 0
		}
		recentHistory := make([]HistoryEntry, 0)
		for i := lastIdx - 1; i >= startIdx; i-- {
			recentHistory = append(recentHistory, history[i])
		}

		c.JSON(http.StatusOK, gin.H{
			"timeLeft":        timeLeft,
			"isSolved":        isSolved,
			"solvedBy":        solvedBy,
			"targetGrid":      models[currentModel],
			"solvedGrid":      solvedGrid,
			"currentModel":    currentModel,
			"totalModels":     len(models), // Nouveau: utile pour l'admin
			"submissionCount": submissionCount,
			"recentHistory":   recentHistory,
		})
	})

	// 3. API Soumission (Jeu)
	r.POST("/api/submit", func(c *gin.Context) {
		var req CheckRequest
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Bad request"})
			return
		}

		mutex.Lock()
		defer mutex.Unlock()

		target := models[currentModel]
		correct := true
		for rIndex, row := range target {
			for cIndex, val := range row {
				if rIndex >= len(req.UserGrid) || cIndex >= len(req.UserGrid[rIndex]) {
					correct = false
					break
				}
				userColor := req.UserGrid[rIndex][cIndex]
				// Gestion des "vides" (transparent, vide ou blanc)
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
			// Normaliser les couleurs
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

			solvedGrid = normalizedGrid
			isSolved = true
			solvedBy = req.Name
			submissionCount++

			entry := HistoryEntry{
				Name:      req.Name,
				Grid:      normalizedGrid,
				Timestamp: time.Now(),
				ModelID:   currentModel,
			}
			history = append(history, entry)

			c.JSON(http.StatusOK, gin.H{"success": true, "message": "BRAVO " + req.Name + " ! Validé !"})
		} else {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "Incorrect. Vérifie le modèle."})
		}
	})

	// 4. Page Historique (Galerie)
	r.GET("/history", func(c *gin.Context) {
		c.HTML(http.StatusOK, "history.html", nil)
	})

	// 5. API Historique Complet
	r.GET("/api/history", func(c *gin.Context) {
		mutex.Lock()
		defer mutex.Unlock()
		c.JSON(http.StatusOK, history)
	})

	// 6. Page Détail (Zoom)
	r.GET("/view/:index", func(c *gin.Context) {
		c.HTML(http.StatusOK, "view.html", nil)
	})

	// 7. API Détail unitaire
	r.GET("/api/history/:index", func(c *gin.Context) {
		mutex.Lock()
		defer mutex.Unlock()

		idxStr := c.Param("index")
		var idx int
		_, err := fmt.Sscanf(idxStr, "%d", &idx)

		if err != nil || idx < 0 || idx >= len(history) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Dessin introuvable"})
			return
		}
		c.JSON(http.StatusOK, history[idx])
	})

	// ---------------------------------------------------------
	// ROUTES ADMIN (PROTÉGÉES)
	// ---------------------------------------------------------

	// Login: admin / Password: password
	admin := r.Group("/admin", gin.BasicAuth(gin.Accounts{
		"admin": "password",
	}))

	// 8. Page Admin (HTML)
	admin.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "admin.html", nil)
	})

	// 9. API Admin: Changer le Timer
	admin.POST("/timer", func(c *gin.Context) {
		var req AdminTimerReq
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}

		mutex.Lock()
		defer mutex.Unlock()

		if req.DurationSec < 10 {
			req.DurationSec = 10 // Minimum de sécurité
		}

		TimerDuration = time.Duration(req.DurationSec) * time.Second
		// Reset immédiat du timer
		nextSwitch = time.Now().Add(TimerDuration)

		fmt.Printf("ADMIN: Timer modifié à %d secondes\n", req.DurationSec)
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "Timer mis à jour et réinitialisé."})
	})

	// 10. API Admin: Forcer un Modèle
	admin.POST("/model", func(c *gin.Context) {
		var req AdminModelReq
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}

		mutex.Lock()
		defer mutex.Unlock()

		if req.ModelIndex < 0 || req.ModelIndex >= len(models) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Index modèle invalide"})
			return
		}

		currentModel = req.ModelIndex

		// Reset du jeu pour le nouveau dessin
		isSolved = false
		solvedGrid = nil
		solvedBy = ""
		nextSwitch = time.Now().Add(TimerDuration)

		fmt.Printf("ADMIN: Modèle forcé à l'index %d\n", req.ModelIndex)
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "Modèle changé avec succès."})
	})

	// 11. API Admin: Skip (Suivant)
	admin.POST("/skip", func(c *gin.Context) {
		mutex.Lock()
		defer mutex.Unlock()

		currentModel = (currentModel + 1) % len(models)

		isSolved = false
		solvedGrid = nil
		solvedBy = ""
		nextSwitch = time.Now().Add(TimerDuration)

		fmt.Println("ADMIN: Skip demandé -> Dessin suivant.")
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "Passé au dessin suivant."})
	})

	// Démarrage du serveur
	r.Run(ServerPort)
}

// Boucle de jeu (Timer automatique)
func gameLoop() {
	ticker := time.NewTicker(1 * time.Second)
	for range ticker.C {
		mutex.Lock()
		if time.Now().After(nextSwitch) {
			// Changement automatique
			currentModel = (currentModel + 1) % len(models)

			isSolved = false
			solvedGrid = nil
			solvedBy = ""

			// On utilise la variable globale TimerDuration (qui peut avoir été changée par l'admin)
			nextSwitch = time.Now().Add(TimerDuration)
			fmt.Println("AUTO: Nouveau dessin (Timer écoulé).")
		}
		mutex.Unlock()
	}
}
