package models

type BotMenu struct {
	Id    int
	Title string
}

var Bot_Menu = []BotMenu{
	{Id: 1, Title: "Registration"},
}

type RoleMenu struct {
	Id    int
	Title string
}

var Role_Menu = []RoleMenu{
	{Id: 1, Title: "Sales representatives"},
	{Id: 2, Title: "Supervisor"},
	{Id: 3, Title: "Dispatcher"},
	{Id: 4, Title: "Commercial Director"},
}
