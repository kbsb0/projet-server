package services

import (
	"fmt"
	"time"
)

type AdminService struct {
	game *GameService
}

func NewAdminService(g *GameService) *AdminService {
	return &AdminService{
		game: g,
	}
}

func (as *AdminService) SetTimer(durationSec int) {
	as.game.SetTimerDuration(time.Duration(durationSec) * time.Second)
}

// NOUVEAU : Méthode appelée par le Controller
func (as *AdminService) SetAutoSwitch(enabled bool) {
	fmt.Printf("ADMIN: Auto-Switch set to %v\n", enabled)
	as.game.SetAutoSwitchMode(enabled)
}

func (as *AdminService) ForceModel(index int) error {
	fmt.Printf("ADMIN: Force model index %d\n", index)
	return as.game.ForceModelIndex(index)
}

func (as *AdminService) SkipModel() {
	fmt.Println("ADMIN: Skip model requested")
	as.game.ForceNextRound()
}
