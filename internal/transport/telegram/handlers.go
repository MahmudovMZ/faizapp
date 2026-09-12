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

	callbackText := "Некорректный запрос"

	if callback.Data == "" {
		answer := tgbotapi.NewCallback(
			callback.ID,
			callbackText,
		)

		if _, err := bot.Request(answer); err != nil {
			log.Println("[TELEGRAM] failed to answer callback:", err)
		}

		return
	}

	parts := strings.SplitN(callback.Data, ":", 3)
	if len(parts) < 2 {
		callbackText = "Некорректные данные запроса"

		answer := tgbotapi.NewCallback(
			callback.ID,
			callbackText,
		)

		if _, err := bot.Request(answer); err != nil {
			log.Println("[TELEGRAM] failed to answer callback:", err)
		}

		return
	}

	action := parts[0]

	tgID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		callbackText = "Некорректный Telegram ID"

		answer := tgbotapi.NewCallback(
			callback.ID,
			callbackText,
		)

		if _, err := bot.Request(answer); err != nil {
			log.Println("[TELEGRAM] failed to answer callback:", err)
		}

		return
	}

	switch action {
	case "approve":
		if callback.Message == nil {
			callbackText = "Сообщение заявки недоступно"
			break
		}

		sendRoleSelectionKeyboard(
			callback.Message.Chat.ID,
			tgID,
		)

		removeInlineKeyboard(callback)
		callbackText = "Выберите роль пользователя"

	case "role":
		if len(parts) != 3 {
			callbackText = "Роль не указана"
			break
		}

		role := strings.TrimSpace(parts[2])
		if role == "" {
			callbackText = "Роль не может быть пустой"
			break
		}

		if err := userService.ApproveUser(
			ctx,
			tgID,
			role,
		); err != nil {
			log.Println(
				"[TELEGRAM] failed to approve user:",
				err,
			)
			callbackText = "Не удалось одобрить заявку"
			break
		}

		removeInlineKeyboard(callback)

		send(
			tgID,
			fmt.Sprintf(
				"Ваша заявка одобрена.\nНазначенная роль: %s",
				role,
			),
		)

		callbackText = "Заявка одобрена"

	case "reject":
		if err := userService.RejectUser(ctx, tgID); err != nil {
			log.Println(
				"[TELEGRAM] failed to reject user:",
				err,
			)
			callbackText = "Не удалось отклонить заявку"
			break
		}

		removeInlineKeyboard(callback)

		send(
			tgID,
			"Ваша заявка отклонена администратором.",
		)

		callbackText = "Заявка отклонена"

	default:
		callbackText = "Неизвестное действие"
	}

	callbackAnswer := tgbotapi.NewCallback(
		callback.ID,
		callbackText,
	)

	if _, err := bot.Request(callbackAnswer); err != nil {
		log.Println(
			"[TELEGRAM] failed to answer callback:",
			err,
		)
	}
}
func sendRoleSelectionKeyboard(adminID int64, tgID int64) {
	rows := make([][]tgbotapi.InlineKeyboardButton, 0)
	row := make([]tgbotapi.InlineKeyboardButton, 0)

	for i, item := range models.Role_Menu {
		button := tgbotapi.NewInlineKeyboardButtonData(
			item.Title,
			fmt.Sprintf("role:%d:%s", tgID, item.Title),
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

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)

	msg := tgbotapi.NewMessage(
		adminID,
		"Выберите роль для пользователя:",
	)
	msg.ReplyMarkup = keyboard

	if _, err := bot.Send(msg); err != nil {
		log.Println(
			"[TELEGRAM] failed to send role keyboard:",
			err,
		)
	}
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
		action != "role" {
		return "", 0, "", fmt.Errorf("unknown action: %s", action)
	}

	return action, tgID, role, nil
}
