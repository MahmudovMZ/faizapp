package models

type BotMenu struct {
	Id    int
	Title string
}

var Bot_Menu = []BotMenu{
	{Id: 1, Title: "Регистрация"},
}

type RoleMenu struct {
	Id    int
	Title string
}

var Role_Menu = []RoleMenu{
	{Id: 1, Title: "Торговый Представитель"},
	{Id: 2, Title: "Супервайзер"},
	{Id: 3, Title: "Диспетчер"},
	{Id: 4, Title: "Коммерческий Директор"},
}
