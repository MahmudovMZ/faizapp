package models

type WorkGroup struct {
	ID   int
	Name string
}

type Territory struct {
	ID           int
	Name         string
	WorkGroupID  int
	SupervisorID *int
}

type SRCode struct {
	ID          int
	Code        string
	TerritoryID int
	CategoryID  int
}
