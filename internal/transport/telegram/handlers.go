package telegram

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/MahmudovMZ/faizapp/internal/models"
	"github.com/MahmudovMZ/faizapp/internal/service"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jackc/pgx/v5"
)

type Handler struct {
	TelegramBot *tgbotapi.BotAPI
	UserService *service.UserService
	BotAdmin    int64

	botState  map[int64]string
	userState map[int64]string
	userData  map[int64]map[string]string

	lock sync.RWMutex
}

type callbackData struct {
	Action   string
	TgID     int64
	ID       int
	Scenario string
}

func NewBotHandler(bot *tgbotapi.BotAPI, user *service.UserService, botAdmin int64) *Handler {
	return &Handler{
		TelegramBot: bot,
		UserService: user,
		BotAdmin:    botAdmin,
		botState:    make(map[int64]string),
		userState:   make(map[int64]string),
		userData:    make(map[int64]map[string]string),
	}
}

func (h *Handler) BotHandler(
	update tgbotapi.Update,
) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if update.CallbackQuery != nil {
		if update.CallbackQuery.From == nil {
			return
		}
		if update.CallbackQuery.From.ID != h.BotAdmin {
			return
		}
		h.handleAdminCallback(ctx, update.CallbackQuery)
		return
	}
	if update.Message == nil || update.Message.From == nil {
		return
	}

	chatID := update.Message.Chat.ID
	text := strings.TrimSpace(update.Message.Text)
	tgID := update.Message.From.ID
	stage := h.getBotState(chatID)

	switch stage {
	case "":
		if text == "/start" {
			h.send(chatID, "Вы обратились в FaizApp.")
			h.send(chatID, "Чем я могу вам помочь?")

			user, err := h.UserService.GetUserByTgID(ctx, tgID)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					h.send(chatID, "Вы ещё не зарегистрированы в FaizApp.")
					h.sendMenuKeyboard(chatID)
					return
				}

				log.Println("[TELEGRAM] failed to get user:", err)
				h.send(chatID, "Произошла ошибка. Попробуйте ещё раз позже.")
				return
			}

			if user == nil {
				h.send(chatID, "Не удалось получить данные пользователя.")
				return
			}

			phone := "не указан"
			if user.Phone != nil {
				phone = *user.Phone
			}

			switch user.Status {
			case "pending":
				h.send(chatID, "Ваша заявка ожидает подтверждения администратора.")
			case "approved":
				if user.Role == nil {
					h.send(
						chatID,
						fmt.Sprintf(
							"ФИО: %s\nТелефон: %s\nРоль: Не назначено",
							user.FullName,
							phone,
						),
					)
					return
				} else {
					h.send(
						chatID,
						fmt.Sprintf(
							"ФИО: %s\nТелефон: %s\nРоль: %s",
							user.FullName,
							phone,
							*user.Role))
					return
				}

			case "rejected":
				h.send(chatID, "Ваша заявка отклонена.")
				return
			}

		}

		if text == "Регистрация" {
			h.initUserData(chatID)
			h.setBotState(chatID, models.STATE_STARTING_REGISTRATION)
			h.setUserState(chatID, models.STATE_WAITING_NAME)
			h.removeKeyboard(chatID, "Начата процедура регистрации.")
			h.send(chatID, "Введите ваше полное имя.")
		}

	case models.STATE_STARTING_REGISTRATION:
		switch h.getUserState(chatID) {
		case models.STATE_WAITING_NAME:
			if text == "" {
				h.send(chatID, "Имя не может быть пустым. Введите полное имя.")
				return
			}

			h.setUserDataValue(chatID, "Name", text)
			h.send(chatID, "Имя сохранено.")

			h.sendPhoneKeyboard(chatID)
			h.setUserState(chatID, models.STATE_WAITING_PHONE)

		case models.STATE_WAITING_PHONE:
			contact := update.Message.Contact

			if contact == nil ||
				strings.TrimSpace(contact.PhoneNumber) == "" {
				h.send(chatID, "Пожалуйста, используйте кнопку для отправки номера телефона.")
				return
			}

			if !isOwnTelegramContact(contact, tgID) {
				h.send(chatID, "Можно отправить только свой Telegram-контакт.")
				return
			}

			h.setUserDataValue(chatID, "Phone", strings.TrimSpace(contact.PhoneNumber))
			h.removeKeyboard(chatID, "Номер телефона сохранён.")

			user := models.User{
				TgID:     tgID,
				FullName: h.getUserDataValue(chatID, "Name"),
				Phone:    stringPointer(h.getUserDataValue(chatID, "Phone")),
			}

			if err := h.UserService.CreateUser(ctx, &user); err != nil {
				log.Println("[TELEGRAM] failed to create user:", err)
				h.send(chatID, "Не удалось завершить регистрацию. Попробуйте ещё раз.")
				return
			}
			h.sendAdminRegistration(h.BotAdmin, user)
			h.send(chatID, "Ваша заявка на регистрацию принята.")
			h.send(chatID, "Ожидайте подтверждения заявки администратором.")
			h.deleteBotState(chatID)
			h.deleteUserData(chatID)
			h.deleteUserState(chatID)
		}
	}
}

func stringPointer(value string) *string {
	return &value
}

func (h *Handler) sendPhoneKeyboard(chatID int64) {
	button := tgbotapi.NewKeyboardButtonContact("Поделиться номером телефона")

	keyboard := tgbotapi.NewReplyKeyboard(
		[]tgbotapi.KeyboardButton{button},
	)

	keyboard.ResizeKeyboard = true
	keyboard.OneTimeKeyboard = true
	keyboard.InputFieldPlaceholder = "Номер телефона"

	msg := tgbotapi.NewMessage(
		chatID,
		"Отправьте номер телефона с помощью кнопки ниже.",
	)
	msg.ReplyMarkup = keyboard

	if _, err := h.TelegramBot.Send(msg); err != nil {
		log.Println("[TELEGRAM] failed to send phone keyboard:", err)
	}
}

func (h *Handler) sendMenuKeyboard(chatID int64) {
	rows := make([][]tgbotapi.KeyboardButton, 0)
	row := make([]tgbotapi.KeyboardButton, 0)

	for i, item := range models.Bot_Menu {
		row = append(row, tgbotapi.NewKeyboardButton(item.Title))

		if (i+1)%2 == 0 {
			rows = append(rows, row)
			row = make([]tgbotapi.KeyboardButton, 0)
		}
	}

	if len(row) > 0 {
		rows = append(rows, row)
	}

	keyboard := tgbotapi.NewReplyKeyboard(rows...)
	msg := tgbotapi.NewMessage(
		chatID,
		"Нажмите кнопку ниже, чтобы зарегистрироваться.",
	)
	msg.ReplyMarkup = keyboard

	if _, err := h.TelegramBot.Send(msg); err != nil {
		log.Println("[TELEGRAM] failed to send registration keyboard:", err)
	}
}

func (h *Handler) removeKeyboard(chatID int64, message string) {
	msg := tgbotapi.NewMessage(chatID, message)
	msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)

	if _, err := h.TelegramBot.Send(msg); err != nil {
		log.Println("[TELEGRAM] failed to remove keyboard:", err)
	}
}
func (h *Handler) send(chatID int64, message string) {
	msg := tgbotapi.NewMessage(chatID, message)

	if _, err := h.TelegramBot.Send(msg); err != nil {
		log.Println("[TELEGRAM] failed to send message:", err)
	}
}

func (h *Handler) sendAdminRegistration(adminID int64, user models.User) {
	phone := "не указан"
	if user.Phone != nil {
		phone = *user.Phone
	}

	messageText := fmt.Sprintf(
		"Новая заявка на регистрацию\n\n"+
			"ФИО: %s\n"+
			"Телефон: %s\n"+
			"Telegram ID: %d",
		user.FullName,
		phone,
		user.TgID,
	)

	approveButton := tgbotapi.NewInlineKeyboardButtonData(
		"Принять",
		fmt.Sprintf("approve:%d", user.TgID),
	)

	rejectButton := tgbotapi.NewInlineKeyboardButtonData(
		"Отклонить",
		fmt.Sprintf("reject:%d", user.TgID),
	)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			approveButton,
			rejectButton,
		),
	)

	msg := tgbotapi.NewMessage(adminID, messageText)
	msg.ReplyMarkup = keyboard

	if _, err := h.TelegramBot.Send(msg); err != nil {
		log.Println("[TELEGRAM] failed to send admin registration:", err)
	}
}

func (h *Handler) handleAdminCallback(
	ctx context.Context,
	callback *tgbotapi.CallbackQuery,
) {
	if callback == nil {
		return
	}

	answer := func(text string) {
		if _, err := h.TelegramBot.Request(tgbotapi.NewCallback(callback.ID, text)); err != nil {
			log.Println("[TELEGRAM] failed to answer callback:", err)

		}
	}

	if callback.Data == "" {
		answer("Некорректный запрос")
		return
	}

	parsed, err := parseCallbackData(callback.Data)
	if err != nil {
		log.Println("[TELEGRAM] failed to parse callback data:", err)
		answer("Некорректные данные запроса")
		return
	}

	scenario := ""

	callbackText := "Неизвестное действие"
	switch parsed.Action {
	case "approve":
		if callback.Message == nil {
			callbackText = "Сообщение заявки недоступно"
			break
		}

		if err := h.sendRoleSelectionKeyboard(
			callback.Message.Chat.ID,
			parsed.TgID,
		); err != nil {
			log.Println("[TELEGRAM] failed to show role keyboard:", err)
			callbackText = "Не удалось показать роли"
			break
		}

		h.removeInlineKeyboard(callback)
		callbackText = "Выберите роль пользователя"

	case "role":

		if callback.Message == nil {
			callbackText = "Сообщение заявки недоступно"
			break
		}

		role, ok := roleTitleByID(parsed.ID)
		if !ok {
			callbackText = "Неизвестная роль"
			break
		}

		if role == "Диспетчер" || role == "Коммерческий Директор" || role == "Старший Супервайзер" {
			if err := h.UserService.ApproveUser(ctx, parsed.TgID, role); err != nil {
				log.Println("[TELEGRAM] failed to approve user:", err)
				callbackText = "Не удалось назначить роль сотрудника"
				break
			}
			h.removeInlineKeyboard(callback)
			h.send(parsed.TgID, "Ваша заявка одобрена. Вам назначена роль: "+role)
			callbackText = "Роль сотрудника назначена"
			break
		}

		if role != "Торговый Представитель" && role != "Супервайзер" {
			callbackText = "Сценарий для этой роли ещё не настроен"
			break
		}
		if role == "Торговый Представитель" {
			scenario = "sr"
		}
		if role == "Супервайзер" {
			scenario = "sv"
		}
		if err := h.sendWorkGroupKeyboard(
			ctx,
			callback.Message.Chat.ID,
			parsed.TgID,
			h.UserService,
			scenario,
		); err != nil {
			log.Println("[TELEGRAM] failed to show work groups:", err)
			callbackText = "Не удалось показать рабочие группы"
			break
		}

		h.removeInlineKeyboard(callback)
		callbackText = "Выберите рабочую группу"

	case "group":
		scenario = parsed.Scenario

		if callback.Message == nil {
			callbackText = "Сообщение с выбором группы недоступно"
			break
		}

		// Для супервайзера рабочая группа является конечным выбором.
		if scenario == "sv" {
			user, err := h.UserService.GetUserByTgID(ctx, parsed.TgID)
			if err != nil {
				log.Println("[TELEGRAM] failed to get user:", err)
				callbackText = "Не удалось найти пользователя"
				break
			}

			if user == nil {
				callbackText = "Пользователь не найден"
				break
			}

			if err := h.UserService.AssignSVWorkGroupAndUpdate(
				ctx,
				user.ID,
				parsed.ID,
				parsed.TgID,
			); err != nil {
				log.Println("[TELEGRAM] failed to assign sv to work group:", err)
				callbackText = "Не удалось назначить Супервайзера на рабочую группу"
				break
			}

			h.removeInlineKeyboard(callback)

			h.send(
				parsed.TgID,
				"Ваша заявка одобрена.\nВы назначены супервайзером рабочей группы.",
			)

			callbackText = "Супервайзер успешно назначен"
			break
		}

		// Для торгового представителя после группы выбирается территория.
		if scenario != "sr" {
			callbackText = "Неизвестный сценарий"
			break
		}

		if err := h.sendTerritoryKeyboard(
			ctx,
			callback.Message.Chat.ID,
			parsed.TgID,
			h.UserService,
			parsed.ID,
			scenario,
		); err != nil {
			log.Println("[TELEGRAM] failed to show territories:", err)
			callbackText = "Не удалось показать территории"
			break
		}

		h.removeInlineKeyboard(callback)
		callbackText = "Выберите территорию"
	case "territory":
		scenario = parsed.Scenario
		if callback.Message == nil {
			callbackText = "Сообщение с выбором кода ТП недоступна."
			break
		}

		if scenario == "sr" {
			if err := h.sendSRCodeKeyboard(
				ctx,
				callback.Message.Chat.ID,
				parsed.TgID,
				h.UserService,
				parsed.ID,
			); err != nil {
				log.Println("[TELEGRAM] failed to show sr-code keyboard:", err)
				callbackText = "Не удалось показать коды ТП"
				break
			}

			h.removeInlineKeyboard(callback)
			callbackText = "Выберите код ТП"
			break
		}
		if scenario != "sv" {
			callbackText = "Неизвестный сценарий"
			break
		}

		user, err := h.UserService.GetUserByTgID(ctx, parsed.TgID)
		if err != nil {
			log.Println("[TELEGRAM] failed to get user:", err)
			callbackText = "Не удалось найти пользователя"
			break
		}

		if user == nil {
			callbackText = "Пользователь не найден"
			break
		}

		if err := h.UserService.AssignSVTerritoryAndUpdate(ctx, user.ID, parsed.ID, parsed.TgID); err != nil {
			log.Println("[TELEGRAM] failed to assign sv:", err)
			callbackText = "Не удалось назначит Супервайзера на территорию"
			break
		}
		h.removeInlineKeyboard(callback)

		h.send(
			parsed.TgID,
			"Ваша заявка одобрена.\nВы назначены супервайзером территории.",
		)

		callbackText = "Супервайзер успешно назначен"
		break

	case "code":
		user, err := h.UserService.GetUserByTgID(ctx, parsed.TgID)
		if err != nil {
			log.Println("[TELEGRAM] failed to get user:", err)
			callbackText = "Не удалось найти Пользователя"
			break
		}
		if user == nil {
			callbackText = "Пользователь не найден"
			break
		}
		if err := h.UserService.AssignSRCodeAndApprove(
			ctx,
			user.ID,
			parsed.ID,
			parsed.TgID,
		); err != nil {
			log.Println("[TELEGRAM] failed to assign sr-code:", err)
			callbackText = "Не удалось назначить код ТП"
			break
		}

		h.removeInlineKeyboard(callback)
		h.send(
			parsed.TgID,
			"Ваша заявка одобрена.\nКод ТП успешно назначен.",
		)
		callbackText = "Пользователь успешно сохранен"

	case "reject":
		if err := h.UserService.RejectUser(ctx, parsed.TgID); err != nil {
			log.Println("[TELEGRAM] failed to reject user:", err)
			callbackText = "Не удалось отклонить заявку"
			break
		}

		h.removeInlineKeyboard(callback)
		h.send(parsed.TgID, "Ваша заявка отклонена администратором.")
		callbackText = "Заявка отклонена"
	}

	answer(callbackText)
}

func roleTitleByID(roleID int) (string, bool) {
	for _, role := range models.Role_Menu {
		if role.Id == roleID {
			return role.Title, true
		}
	}

	return "", false
}

func (h *Handler) sendRoleSelectionKeyboard(adminID, tgID int64) error {
	rows := make([][]tgbotapi.InlineKeyboardButton, 0)
	row := make([]tgbotapi.InlineKeyboardButton, 0)

	for i, item := range models.Role_Menu {
		button := tgbotapi.NewInlineKeyboardButtonData(
			item.Title,
			fmt.Sprintf("role:%d:%d", tgID, item.Id),
		)
		row = append(row, button)

		if (i+1)%2 == 0 {
			rows = append(rows, row)
			row = make([]tgbotapi.InlineKeyboardButton, 0)
		}
	}

	if len(row) > 0 {
		rows = append(rows, row)
	}

	msg := tgbotapi.NewMessage(adminID, "Выберите роль для пользователя:")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)

	_, err := h.TelegramBot.Send(msg)
	return err
}

func (h *Handler) sendWorkGroupKeyboard(
	ctx context.Context,
	adminID int64,
	tgID int64,
	userService *service.UserService,
	scenario string,
) error {
	workGroups, err := userService.GetWorkGroups(ctx)
	if err != nil {
		h.send(adminID, "Не удалось загрузить рабочие группы.")
		return err
	}
	if len(workGroups) == 0 {
		h.send(adminID, "Рабочие группы не найдены.")
		return fmt.Errorf("no work groups found")
	}

	rows := make([][]tgbotapi.InlineKeyboardButton, 0)
	row := make([]tgbotapi.InlineKeyboardButton, 0)

	for i, item := range workGroups {
		button := tgbotapi.NewInlineKeyboardButtonData(
			item.Name,
			fmt.Sprintf("group:%d:%d:%s", tgID, item.ID, scenario),
		)
		row = append(row, button)

		if (i+1)%2 == 0 {
			rows = append(rows, row)
			row = make([]tgbotapi.InlineKeyboardButton, 0)
		}
	}

	if len(row) > 0 {
		rows = append(rows, row)
	}

	msg := tgbotapi.NewMessage(
		adminID,
		"Выберите рабочую группу для торгового представителя:",
	)
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)

	_, err = h.TelegramBot.Send(msg)
	return err
}

func (h *Handler) sendTerritoryKeyboard(
	ctx context.Context,
	adminID int64,
	tgID int64,
	userService *service.UserService,
	workGroupID int,
	scenario string,
) error {
	territories, err := userService.GetTerritoriesByGroup(ctx, workGroupID)
	if err != nil {
		h.send(adminID, "Не удалось загрузить территории.")
		return err
	}
	if len(territories) == 0 {
		h.send(adminID, "В этой рабочей группе территории не найдены.")
		return fmt.Errorf("no territories found for work group %d", workGroupID)
	}

	rows := make([][]tgbotapi.InlineKeyboardButton, 0)
	row := make([]tgbotapi.InlineKeyboardButton, 0)

	for i, item := range territories {
		button := tgbotapi.NewInlineKeyboardButtonData(
			item.Name,
			fmt.Sprintf("territory:%d:%d:%s", tgID, item.ID, scenario),
		)
		row = append(row, button)

		if (i+1)%2 == 0 {
			rows = append(rows, row)
			row = make([]tgbotapi.InlineKeyboardButton, 0)
		}
	}

	if len(row) > 0 {
		rows = append(rows, row)
	}

	msg := tgbotapi.NewMessage(
		adminID,
		"Выберите территорию для торгового представителя:",
	)
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)

	_, err = h.TelegramBot.Send(msg)
	return err
}

func (h *Handler) sendSRCodeKeyboard(ctx context.Context,
	adminID int64,
	tgID int64,
	userService *service.UserService,
	territoryID int) error {
	srCode, err := userService.GetAvailableSRCodes(ctx, territoryID)
	if err != nil {
		h.send(adminID, "Не удалось загрузить коды ТП.")
		return err
	}
	if len(srCode) == 0 {
		h.send(adminID, "На этой территории свободных кодов для ТП нет.")
		return fmt.Errorf("no available SR codes found for territory %d", territoryID)
	}
	rows := make([][]tgbotapi.InlineKeyboardButton, 0)
	row := make([]tgbotapi.InlineKeyboardButton, 0)

	for i, item := range srCode {
		button := tgbotapi.NewInlineKeyboardButtonData(
			item.Code,
			fmt.Sprintf("code:%d:%d", tgID, item.ID),
		)
		row = append(row, button)

		if (i+1)%2 == 0 {
			rows = append(rows, row)
			row = make([]tgbotapi.InlineKeyboardButton, 0)
		}
	}

	if len(row) > 0 {
		rows = append(rows, row)
	}

	msg := tgbotapi.NewMessage(
		adminID,
		"Выберите код ТП для торгового представителя:",
	)
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)

	_, err = h.TelegramBot.Send(msg)
	return err
}

func (h *Handler) removeInlineKeyboard(callback *tgbotapi.CallbackQuery) {
	if callback == nil || callback.Message == nil {
		return
	}

	emptyKeyboard := tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: make(
			[][]tgbotapi.InlineKeyboardButton,
			0,
		),
	}

	edit := tgbotapi.NewEditMessageReplyMarkup(
		callback.Message.Chat.ID,
		callback.Message.MessageID,
		emptyKeyboard,
	)

	if _, err := h.TelegramBot.Send(edit); err != nil {
		log.Println(
			"[TELEGRAM] failed to remove inline keyboard:",
			err,
		)
	}
}

func parseCallbackData(data string) (
	callback *callbackData,
	err error,
) {
	parts := strings.SplitN(data, ":", 4)

	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid callback data")
	}

	tgID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid telegram id: %w", err)
	}

	result := &callbackData{
		Action: parts[0],
		TgID:   tgID}

	switch result.Action {
	case "approve", "reject":
		if len(parts) != 2 {
			return nil, fmt.Errorf("%s does not accept extra parameters", result.Action)
		}

	case "role", "code":
		if len(parts) != 3 {
			return nil, fmt.Errorf("%s requires an id", result.Action)
		}

		result.ID, err = strconv.Atoi(parts[2])
		if err != nil {
			return nil, fmt.Errorf("invalid %s id: %w", result.Action, err)
		}

	case "group", "territory":
		if len(parts) != 4 {
			return nil, fmt.Errorf("%s requires id and scenario", result.Action)
		}

		result.ID, err = strconv.Atoi(parts[2])
		if err != nil {
			return nil, fmt.Errorf("invalid %s id: %w", result.Action, err)
		}

		result.Scenario = strings.TrimSpace(parts[3])
		if result.Scenario != "sr" && result.Scenario != "sv" {
			return nil, fmt.Errorf("invalid scenario: %s", result.Scenario)
		}
	default:
		return nil, fmt.Errorf("unknown action: %s", result.Action)
	}

	return result, nil
}

func isOwnTelegramContact(contact *tgbotapi.Contact, tgID int64) bool {
	if contact == nil {
		return false
	}

	if strings.TrimSpace(contact.PhoneNumber) == "" {
		return false
	}

	return contact.UserID == tgID
}
