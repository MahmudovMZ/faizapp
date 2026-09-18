package postgres

import (
	"context"
	"log"

	"github.com/MahmudovMZ/faizapp/internal/models"
)

func (r *FaizAppRepo) GetWorkGroups(ctx context.Context) ([]models.WorkGroup, error) {
	log.Println("[REPOSITORY] getting work groups")
	var workGroups []models.WorkGroup
	query := `SELECT id, name FROM work_groups order by id`
	rows, err := r.Pool.Query(ctx, query)
	if err != nil {
		log.Println("[REPOSITORY] getting work groups: ", err)
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var workGroup models.WorkGroup
		if err := rows.Scan(&workGroup.ID, &workGroup.Name); err != nil {
			log.Println("[REPOSITORY] getting work groups: ", err)
			return nil, err
		}
		workGroups = append(workGroups, workGroup)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return workGroups, nil
}
func (r *FaizAppRepo) GetTerritoriesByGroup(ctx context.Context, workGroupID int) ([]models.Territory, error) {
	log.Println("[REPOSITORY] getting territories by group")
	var territories []models.Territory

	query := `SELECT id, name, work_group_id, supervisor_id FROM territories WHERE work_group_id=$1 ORDER BY id`
	rows, err := r.Pool.Query(ctx, query, workGroupID)
	if err != nil {
		log.Println("[REPOSITORY] getting territories by group: ", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var territory models.Territory
		if err := rows.Scan(&territory.ID,
			&territory.Name,
			&territory.WorkGroupID,
			&territory.SupervisorID); err != nil {
			log.Println("[REPOSITORY] getting territories by group: ", err)
			return nil, err
		}
		territories = append(territories, territory)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return territories, nil
}

func (r *FaizAppRepo) GetAvailableSRCodes(ctx context.Context, territoryID int) ([]models.SRCode, error) {
	log.Println("[REPOSITORY] getting available SR codes")
	var srCodes []models.SRCode
	query := `SELECT s.id, s.code, s.territory_id, s.category_id 
			  FROM sr_codes AS s 
			  LEFT JOIN sr_assignments AS a 
			  ON s.id = a.sr_code_id 
			  WHERE s.territory_id = $1 
			  AND a.sr_code_id IS NULL 
			  ORDER BY s.id`
	rows, err := r.Pool.Query(ctx, query, territoryID)
	if err != nil {
		log.Println("[REPOSITORY] getting available SR codes: ", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var srCode models.SRCode
		if err := rows.Scan(
			&srCode.ID,
			&srCode.Code,
			&srCode.TerritoryID,
			&srCode.CategoryID); err != nil {
			log.Println("[REPOSITORY] getting available SR codes: ", err)
			return nil, err
		}
		srCodes = append(srCodes, srCode)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return srCodes, nil
}
