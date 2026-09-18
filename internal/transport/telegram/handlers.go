package telegram

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/MahmudovMZ/faizapp/internal/models"
	"github.com/MahmudovMZ/faizapp/internal/service"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jackc/pgx/v5"
)

var botState = make(map[int64]string)
var userState = make(map[int64]string)
var userData = make(map[int64]map[string]string)
var bot *tgbotapi.BotAPI

func BotHandler(
	bot2 *tgbotapi.BotAPI,
	update tgbotapi.Update,
	userService *service.UserService,
	botAdmin int64,
) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	bot = bot2

	if update.CallbackQuery != nil {
		if update.CallbackQuery.From == nil {
			return
		}
		if update.CallbackQuery.From.ID != botAdmin {
			return
		}
		handleAdminCallback(ctx, update.CallbackQuery, userService)

		return
	}
	if update.Message == nil || update.Message.From == nil {
		return
	}

	chatID := update.Message.Chat.ID
	text := strings.TrimSpace(update.Message.Text)
	tgID := update.Message.From.ID
	stage := botState[chatID]

	switch stage {
	case "":
		if text == "/start" {
			send(chatID, "Вы обратились в FaizApp.")
			send(chatID, "Чем я могу вам помочь?")

			user, err := userService.GetUserByTgID(ctx, tgID)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					send(chatID, "Вы ещё не зарегистрированы в FaizApp.")
					sendMenuKeyboard(chatID)
					return
				}

				log.Println("[TELEGRAM] failed to get user:", err)
				send(chatID, "Произошла ошибка. Попробуйте ещё раз позже.")
				return
			}

			if user == nil {
				send(chatID, "Не удалось получить данные пользователя.")
				return
			}

			phone := "не указан"
			if user.Phone != nil {
				phone = *user.Phone
			}

			switch user.Status {
			case "pending":
				send(chatID, "Ваша заявка ожидает подтверждения администратора.")
			case "approved":
				if user.Role == nil {
					send(
						chatID,
						fmt.Sprintf(
							"ФИО: %s\nТелефон: %s\nРоль: Не назначено",
							user.FullName,
							phone,
						),
					)
					return
				} else {
					send(
						chatID,
						fmt.Sprintf(
							"ФИО: %s\nТелефон: %s\nРоль: %s",
							user.FullName,
							phone,
							*user.Role))
					return
				}

			case "rejected":
				send(chatID, "Ваша заявка отклонена.")
				return
			}

		}

		if text == "Регистрация" {
			userData[chatID] = make(map[string]string)
			botState[chatID] = models.STATE_STARTING_REGISTRATION
			userState[chatID] = models.STATE_WAITING_NAME
			removeKeyboard(chatID, "Начата процедура регистрации.")
			send(chatID, "Введите ваше полное имя.")
		}

	case models.STATE_STARTING_REGISTRATION:
		switch userState[chatID] {
		case models.STATE_WAITING_NAME:
			if text == "" {
				send(chatID, "Имя не может быть пустым. Введите полное имя.")
				return
			}

			userData[chatID]["Name"] = text
			send(chatID, "Имя сохранено.")

			sendPhoneKeyboard(chatID)
			userState[chatID] = models.STATE_WAITING_PHONE

		case models.STATE_WAITING_PHONE:
			if update.Message.Contact == nil ||
				strings.TrimSpace(update.Message.Contact.PhoneNumber) == "" {
				send(chatID, "Пожалуйста, используйте кнопку для отправки номера телефона.")
				return
			}

			userData[chatID]["Phone"] =
				strings.TrimSpace(update.Message.Contact.PhoneNumber)
			removeKeyboard(chatID, "Номер телефона сохранён.")

			user := models.User{
				TgID:     tgID,
				FullName: userData[chatID]["Name"],
				Phone:    stringPointer(userData[chatID]["Phone"]),
			}

			if err := userService.CreateUser(ctx, &user); err != nil {
				log.Println("[TELEGRAM] failed to create user:", err)
				send(chatID, "Не удалось завершить регистрацию. Попробуйте ещё раз.")
				return
			}
			sendAdminRegistration(botAdmin, user)
			send(chatID, "Ваша заявка на регистрацию принята.")
			send(chatID, "Ожидайте подтверждения заявки администратором.")
			delete(userData, chatID)
			delete(userState, chatID)
			delete(botState, chatID)
		}
	}
}

func stringPointer(value string) *string {
	return &value
}

func sendPhoneKeyboard(chatID int64) {
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

	if _, err := bot.Send(msg); err != nil {
		log.Println("[TELEGRAM] failed to send phone keyboard:", err)
	}
}

func sendMenuKeyboard(chatID int64) {
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

	if _, err := bot.Send(msg); err != nil {
		log.Println("[TELEGRAM] failed to send registration keyboard:", err)
	}
}

func removeKeyboard(chatID int64, message string) {
	msg := tgbotapi.NewMessage(chatID, message)
	msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)

	if _, err := bot.Send(msg); err != nil {
		log.Println("[TELEGRAM] failed to remove keyboard:", err)
	}
}
func send(chatID int64, message string) {
	msg := tgbotapi.NewMessage(chatID, message)

	if _, err := bot.Send(msg); err != nil {
		log.Println("[TELEGRAM] failed to send message:", err)
	}
}

func sendAdminRegistration(adminID int64, user models.User) {
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

	if _, err := bot.Send(msg); err != nil {
		log.Println("[TELEGRAM] failed to send admin registration:", err)
	}
}

func handleAdminCallback(
	ctx context.Context,
	callback *tgbotapi.CallbackQuery,
	userService *service.UserService,
) {
	if callback == nil {
		return
	}

	answer := func(text string) {
		if _, err := bot.Request(tgbotapi.NewCallback(callback.ID, text)); err != nil {
			log.Println("[TELEGRAM] failed to answer callback:", err)
		}
	}

	if callback.Data == "" {
		answer("Некорректный запрос")
		return
	}

	parts := strings.SplitN(callback.Data, ":", 4)
	if len(parts) < 2 {
		answer("Некорректные данные запроса")
		return
	}

	tgID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		answer("Некорректный Telegram ID")
		return
	}

	scenario := ""

	callbackText := "Неизвестное действие"
	switch parts[0] {
	case "approve":
		if callback.Message == nil {
			callbackText = "Сообщение заявки недоступно"
			break
		}

		if err := sendRoleSelectionKeyboard(
			callback.Message.Chat.ID,
			tgID,
		); err != nil {
			log.Println("[TELEGRAM] failed to show role keyboard:", err)
			callbackText = "Не удалось показать роли"
			break
		}

		removeInlineKeyboard(callback)
		callbackText = "Выберите роль пользователя"

	case "role":
		if len(parts) != 3 {
			callbackText = "Роль не указана"
			break
		}
		if callback.Message == nil {
			callbackText = "Сообщение заявки недоступно"
			break
		}

		roleID, err := strconv.Atoi(parts[2])
		if err != nil {
			callbackText = "Некорректный ID роли"
			break
		}

		role, ok := roleTitleByID(roleID)
		if !ok {
			callbackText = "Неизвестная роль"
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
		if err := sendWorkGroupKeyboard(
			ctx,
			callback.Message.Chat.ID,
			tgID,
			userService,
			scenario,
		); err != nil {
			log.Println("[TELEGRAM] failed to show work groups:", err)
			callbackText = "Не удалось показать рабочие группы"
			break
		}

		removeInlineKeyboard(callback)
		callbackText = "Выберите рабочую группу"

	case "group":
		if len(parts) != 4 {
			callbackText = "Рабочая группа не указана"
			break
		}
		scenario = parts[3]
		if callback.Message == nil {
			callbackText = "Сообщение с выбором группы недоступно"
			break
		}

		workGroupID, err := strconv.Atoi(parts[2])
		if err != nil {
			callbackText = "Некорректный ID рабочей группы"
			break
		}

		if err := sendTerritoryKeyboard(
			ctx,
			callback.Message.Chat.ID,
			tgID,
			userService,
			workGroupID,
			scenario,
		); err != nil {
			log.Println("[TELEGRAM] failed to show territories:", err)
			callbackText = "Не удалось показать территории"
			break
		}

		removeInlineKeyboard(callback)
		callbackText = "Выберите территорию"

	case "territory":
		if len(parts) != 4 {
			callbackText = "Территория не указана"
			break
		}
		scenario = parts[3]
		if callback.Message == nil {
			callbackText = "Сообщение с выбором кода ТП недоступна."
			break
		}
		territoryID, err := strconv.Atoi(parts[2])
		if err != nil {
			callbackText = "Некорректный ID территории."
			break
		}

		if scenario == "sr" {
			if err := sendSRCodeKeyboard(
				ctx,
				callback.Message.Chat.ID,
				tgID,
				userService,
				territoryID,
			); err != nil {
				log.Println("[TELEGRAM] failed to show sr-code keyboard:", err)
				callbackText = "Не удалось показать коды ТП"
				break
			}

			removeInlineKeyboard(callback)
			callbackText = "Выберите код ТП"
			break
		}
		if scenario != "sv" {
			callbackText = "Неизвестный сценарий"
			break
		}

		user, err := userService.GetUserByTgID(ctx, tgID)
		if err != nil {
			log.Println("[TELEGRAM] failed to get user:", err)
			callbackText = "Не удалось найти пользователя"
			break
		}

		if user == nil {
			callbackText = "Пользователь не найден"
			break
		}

		if err := userService.AssignSVTerritoryAndUpdate(ctx, user.ID, territoryID, tgID); err != nil {
			log.Println("[TELEGRAM] failed to assign sv:", err)
			callbackText = "Не удалось назначит Супервайзера на территорию"
			break
		}
		removeInlineKeyboard(callback)

		send(
			tgID,
			"Ваша заявка одобрена.\nВы назначены супервайзером территории.",
		)

		callbackText = "Супервайзер успешно назначен"
		break

	case "code":
		if len(parts) != 3 {
			callbackText = "Код ТП не указан"
			break
		}
		srCodeID, err := strconv.Atoi(parts[2])
		if err != nil {
			callbackText = "Некорректный ID кода ТП"
			break
		}
		user, err := userService.GetUserByTgID(ctx, tgID)
		if err != nil {
			log.Println("[TELEGRAM] failed to get user:", err)
			callbackText = "Не удалось найти Пользователя"
			break
		}
		if user == nil {
			callbackText = "Пользователь не найден"
			break
		}
		if err := userService.AssignSRCodeAndApprove(
			ctx,
			user.ID,
			srCodeID,
			tgID,
		); err != nil {
			log.Println("[TELEGRAM] failed to assign sr-code:", err)
			callbackText = "Не удалось назначить код ТП"
			break
		}

		removeInlineKeyboard(callback)
		send(
			tgID,
			"Ваша заявка одобрена.\nКод ТП успешно назначен.",
		)
		callbackText = "Пользователь успешно сохранен"

	case "reject":
		if err := userService.RejectUser(ctx, tgID); err != nil {
			log.Println("[TELEGRAM] failed to reject user:", err)
			callbackText = "Не удалось отклонить заявку"
			break
		}

		removeInlineKeyboard(callback)
		send(tgID, "Ваша заявка отклонена администратором.")
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

func sendRoleSelectionKeyboard(adminID, tgID int64) error {
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

	_, err := bot.Send(msg)
	return err
}

func sendWorkGroupKeyboard(
	ctx context.Context,
	adminID int64,
	tgID int64,
	userService *service.UserService,
	scenario string,
) error {
	workGroups, err := userService.GetWorkGroups(ctx)
	if err != nil {
		send(adminID, "Не удалось загрузить рабочие группы.")
		return err
	}
	if len(workGroups) == 0 {
		send(adminID, "Рабочие группы не найдены.")
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

	_, err = bot.Send(msg)
	return err
}

func sendTerritoryKeyboard(
	ctx context.Context,
	adminID int64,
	tgID int64,
	userService *service.UserService,
	workGroupID int,
	scenario string,
) error {
	territories, err := userService.GetTerritoriesByGroup(ctx, workGroupID)
	if err != nil {
		send(adminID, "Не удалось загрузить территории.")
		return err
	}
	if len(territories) == 0 {
		send(adminID, "В этой рабочей группе территории не найдены.")
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

	_, err = bot.Send(msg)
	return err
}

func sendSRCodeKeyboard(ctx context.Context,
	adminID int64,
	tgID int64,
	userService *service.UserService,
	territoryID int) error {
	srCode, err := userService.GetAvailableSRCodes(ctx, territoryID)
	if err != nil {
		send(adminID, "Не удалось загрузить коды ТП.")
		return err
	}
	if len(srCode) == 0 {
		send(adminID, "На этой территории свободных кодов для ТП нет.")
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

	_, err = bot.Send(msg)
	return err
}

func removeInlineKeyboard(callback *tgbotapi.CallbackQuery) {
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

	if _, err := bot.Send(edit); err != nil {
		log.Println(
			"[TELEGRAM] failed to remove inline keyboard:",
			err,
		)
	}
}

func parseCallbackData(data string) (
	action string,
	tgID int64,
	role string,
	err error,
) {
	parts := strings.SplitN(data, ":", 3)

	if len(parts) < 2 {
		return "", 0, "", fmt.Errorf("invalid callback data")
	}

	action = parts[0]

	tgID, err = strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return "", 0, "", fmt.Errorf("invalid telegram id: %w", err)
	}

	if action == "role" {
		if len(parts) != 3 || strings.TrimSpace(parts[2]) == "" {
			return "", 0, "", fmt.Errorf("role is required")
		}

		role = strings.TrimSpace(parts[2])
	}

	if action != "approve" &&
		action != "reject" &&
		action != "role" &&
		action != "group" &&
		action != "territory" &&
		action != "code" {
		return "", 0, "", fmt.Errorf("unknown action: %s", action)
	}

	return action, tgID, role, nil
}
