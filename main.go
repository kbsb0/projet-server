package main

import (
	"log"
	"net/http"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"pixel-game/controllers"
	"pixel-game/repository"
	"pixel-game/services"
)

// --- GESTIONNAIRE DE STATISTIQUES ---

type RequestStats struct {
	mu            sync.RWMutex
	RequestCounts map[int64]int // Map[TimestampUnix] -> Nombre de requêtes
}

func NewRequestStats() *RequestStats {
	return &RequestStats{
		RequestCounts: make(map[int64]int),
	}
}

// AddRequest incrémente le compteur pour la seconde actuelle
func (s *RequestStats) AddRequest() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().Unix()
	s.RequestCounts[now]++
}

// GetData retourne les données triées pour les X dernières secondes
func (s *RequestStats) GetData(seconds int) ([]int64, []int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var labels []int64
	var data []int

	now := time.Now().Unix()
	limit := now - int64(seconds)

	// On récupère les clés (timestamps)
	var timestamps []int64
	for t := range s.RequestCounts {
		if t > limit {
			timestamps = append(timestamps, t)
		}
	}
	sort.Slice(timestamps, func(i, j int) bool { return timestamps[i] < timestamps[j] })

	// On remplit les tableaux pour le graph
	// Note: Pour un graph fluide, on peut vouloir remplir les trous (0 req)
	// Ici on fait simple : on envoie ce qu'on a.
	for _, t := range timestamps {
		labels = append(labels, t)
		data = append(data, s.RequestCounts[t])
	}

	return labels, data
}

// CleanOldData nettoie les vieilles données pour éviter de saturer la mémoire
// À lancer dans une goroutine
func (s *RequestStats) CleanOldData() {
	for {
		time.Sleep(1 * time.Minute)
		s.mu.Lock()
		limit := time.Now().Unix() - 3600 // Garde 1 heure d'historique max
		for t := range s.RequestCounts {
			if t < limit {
				delete(s.RequestCounts, t)
			}
		}
		s.mu.Unlock()
	}
}

// MiddlewareStats intercepte toutes les requêtes
func MiddlewareStats(stats *RequestStats) gin.HandlerFunc {
	return func(c *gin.Context) {
		// On compte la requête
		stats.AddRequest()
		c.Next()
	}
}

// --- MAIN ---

func main() {
	// 1. Initialisation BDD
	if _, err := os.Stat("data"); os.IsNotExist(err) {
		os.Mkdir("data", 0755)
	}
	db := repository.NewDatabase("data/history.json")

	// 2. Initialisation Stats
	stats := NewRequestStats()
	go stats.CleanOldData() // Lance le nettoyage en arrière-plan

	// 3. Services & Controllers
	gameService := services.NewGameService(db)
	adminService := services.NewAdminService(gameService)
	ctrl := controllers.NewController(gameService, adminService, db)

	// 4. Configuration Gin
	r := gin.Default()

	// A. AJOUT DU MIDDLEWARE ICI
	r.Use(cors.Default())
	r.Use(MiddlewareStats(stats)) // <-- Le middleware est actif pour TOUTES les routes

	r.LoadHTMLGlob("templates/*")

	// --- Routes Publiques ---
	r.GET("/", ctrl.Dashboard)
	r.GET("/api/state", ctrl.GetState)
	r.POST("/api/submit", ctrl.Submit)

	r.GET("/history", ctrl.HistoryPage)
	r.GET("/api/history", ctrl.GetFullHistory)
	r.GET("/view/:index", ctrl.ViewPage)
	r.GET("/api/history/:index", ctrl.GetHistoryItem)

	// --- NOUVELLES ROUTES STATISTIQUES ---

	// Page HTML pour voir le graph
	r.GET("/stats", func(c *gin.Context) {
		c.HTML(http.StatusOK, "stats.html", nil)
	})

	// API JSON pour alimenter le graph
	r.GET("/api/stats", func(c *gin.Context) {
		// Récupère les 60 dernières secondes
		timestamps, counts := stats.GetData(60)

		// Calcul de la moyenne sur la fenêtre
		total := 0
		for _, v := range counts {
			total += v
		}
		avg := 0.0
		if len(counts) > 0 {
			avg = float64(total) / float64(len(counts))
		}

		c.JSON(http.StatusOK, gin.H{
			"timestamps": timestamps,
			"counts":     counts,
			"average":    avg,
		})
	})

	// --- Routes Admin ---
	admin := r.Group("/admin", gin.BasicAuth(gin.Accounts{
		"admin": "password",
	}))

	admin.GET("/", ctrl.AdminPage)
	admin.POST("/timer", ctrl.SetTimer)
	admin.POST("/model", ctrl.ForceModel)
	admin.POST("/skip", ctrl.SkipModel)
	admin.POST("/auto-switch", ctrl.SetAutoSwitch)

	log.Println("Serveur démarré sur :8080")
	r.Run(":8080")
}
