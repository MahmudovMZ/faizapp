package service

import (
	"context"

	"github.com/MahmudovMZ/faizapp/internal/models"
)

func (u *UserService) GetWorkGroups(ctx context.Context) ([]models.WorkGroup, error) {
	workGroups, err := u.repo.GetWorkGroups(ctx)
	if err != nil {
		return nil, err
	}
	if len(workGroups) == 0 {
		return []models.WorkGroup{}, nil
	}
	return workGroups, nil
}
func (u *UserService) GetTerritoriesByGroup(ctx context.Context, workGroupID int) ([]models.Territory, error) {
	territories, err := u.repo.GetTerritoriesByGroup(ctx, workGroupID)
	if err != nil {
		return nil, err
	}
	if len(territories) == 0 {
		return []models.Territory{}, nil
	}
	return territories, nil
}

func (u *UserService) GetAvailableSRCodes(ctx context.Context, territoryID int) ([]models.SRCode, error) {
	codes, err := u.repo.GetAvailableSRCodes(ctx, territoryID)
	if err != nil {
		return nil, err
	}
	if len(codes) == 0 {
		return []models.SRCode{}, nil
	}
	return codes, nil
}
