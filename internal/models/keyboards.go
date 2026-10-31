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

const RoleSalesRep = "Торговый Представитель"
const RoleSupervisor = "Супервайзер"
const RoleDispatcher = "Диспетчер"
const RoleCommercialDirector = "Коммерческий Директор"
const RoleSeniorSupervisor = "Старший Супервайзер"

var Role_Menu = []RoleMenu{
	{Id: 1, Title: RoleSalesRep},
	{Id: 2, Title: RoleSupervisor},
	{Id: 3, Title: RoleDispatcher},
	{Id: 4, Title: RoleCommercialDirector},
	{Id: 5, Title: RoleSeniorSupervisor},
}

type WorkGroupMenu struct {
	Id    int
	Title string
}
