package main

import (
	"log"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"pixel-game/controllers"
	"pixel-game/repository"
	"pixel-game/services"
)

func main() {
	// 1. Initialisation de la BDD (fichier JSON)
	// Assure-toi que le dossier data existe ou gère-le
	if _, err := os.Stat("data"); os.IsNotExist(err) {
		os.Mkdir("data", 0755)
	}
	db := repository.NewDatabase("data/history.json")

	// 2. Initialisation des Services
	// On crée d'abord le GameService (logique de jeu)
	gameService := services.NewGameService(db)

	// Ensuite on crée l'AdminService en lui injectant le GameService
	// C'est ici que se fait le lien entre les commandes admin et le jeu
	adminService := services.NewAdminService(gameService)

	// 3. Initialisation du Controller (Routes)
	// Le contrôleur reçoit maintenant les deux services + la BDD
	ctrl := controllers.NewController(gameService, adminService, db)

	// 4. Configuration Gin
	r := gin.Default()
	r.Use(cors.Default())
	r.LoadHTMLGlob("templates/*")

	// --- Routes Publiques ---
	r.GET("/", ctrl.Dashboard)
	r.GET("/api/state", ctrl.GetState)
	r.POST("/api/submit", ctrl.Submit)

	r.GET("/history", ctrl.HistoryPage)
	r.GET("/api/history", ctrl.GetFullHistory)
	r.GET("/view/:index", ctrl.ViewPage)
	r.GET("/api/history/:index", ctrl.GetHistoryItem)

	// --- Routes Admin ---
	admin := r.Group("/admin", gin.BasicAuth(gin.Accounts{
		"admin": "password",
	}))

	admin.GET("/", ctrl.AdminPage)
	admin.POST("/timer", ctrl.SetTimer)
	admin.POST("/model", ctrl.ForceModel)
	admin.POST("/skip", ctrl.SkipModel)
	admin.POST("/auto-switch", ctrl.SetAutoSwitch)

	// 5. Lancement
	log.Println("Serveur démarré sur :8080")
	r.Run(":8080")
}
