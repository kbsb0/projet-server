package controllers

import (
	"net/http"
	"pixel-game/models"
	"pixel-game/repository"
	"pixel-game/services"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Controller struct {
	GameService  *services.GameService
	AdminService *services.AdminService
	Repo         *repository.Database
}

func NewController(gs *services.GameService, as *services.AdminService, r *repository.Database) *Controller {
	return &Controller{
		GameService:  gs,
		AdminService: as,
		Repo:         r,
	}
}

// --- Pages HTML ---
func (ctrl *Controller) Dashboard(c *gin.Context) {
	c.HTML(http.StatusOK, "dashboard.html", nil)
}

func (ctrl *Controller) HistoryPage(c *gin.Context) {
	c.HTML(http.StatusOK, "history.html", nil)
}

func (ctrl *Controller) ViewPage(c *gin.Context) {
	c.HTML(http.StatusOK, "view.html", nil)
}

func (ctrl *Controller) AdminPage(c *gin.Context) {
	c.HTML(http.StatusOK, "admin.html", nil)
}

// --- API Publique ---

func (ctrl *Controller) GetState(c *gin.Context) {
	state := ctrl.GameService.GetState()
	c.JSON(http.StatusOK, state)
}

func (ctrl *Controller) Submit(c *gin.Context) {
	var req models.CheckRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Bad request"})
		return
	}

	success, msg := ctrl.GameService.ProcessSubmission(req)
	c.JSON(http.StatusOK, gin.H{"success": success, "message": msg})
}

func (ctrl *Controller) GetFullHistory(c *gin.Context) {
	hist := ctrl.Repo.GetFullHistory()
	c.JSON(http.StatusOK, hist)
}

func (ctrl *Controller) GetHistoryItem(c *gin.Context) {
	idxStr := c.Param("index")
	idx, err := strconv.Atoi(idxStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Index invalide"})
		return
	}

	entry, found := ctrl.Repo.GetHistoryEntry(idx)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Dessin introuvable"})
		return
	}
	c.JSON(http.StatusOK, entry)
}

// --- API Admin ---

func (ctrl *Controller) SetTimer(c *gin.Context) {
	var req models.AdminTimerReq
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Utilisation de AdminService
	ctrl.AdminService.SetTimer(req.DurationSec)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Timer mis à jour"})
}

func (ctrl *Controller) ForceModel(c *gin.Context) {
	var req models.AdminModelReq
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Utilisation de AdminService
	err := ctrl.AdminService.ForceModel(req.ModelIndex)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Modèle changé"})
}

func (ctrl *Controller) SkipModel(c *gin.Context) {
	// Utilisation de AdminService
	ctrl.AdminService.SkipModel()
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Dessin suivant"})
}

func (ctrl *Controller) SetAutoSwitch(c *gin.Context) {
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Appel au service
	ctrl.AdminService.SetAutoSwitch(req.Enabled)

	state := "activé"
	if !req.Enabled {
		state = "désactivé"
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Mode Auto-Switch " + state})
}
