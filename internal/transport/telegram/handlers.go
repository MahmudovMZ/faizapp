package telegram

import (
	"context"
	"errors"
	"fmt"
	"log"
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
) {
	if update.Message == nil || update.Message.From == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	bot = bot2

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

			send(
				chatID,
				fmt.Sprintf(
					"ФИО: %s\nТелефон: %s\nРоль: %s",
					user.FullName,
					phone,
					user.Role,
				),
			)

			return
		}

		if text == "Регистрация" {
			userData[chatID] = make(map[string]string)
			botState[chatID] = models.STATE_STARTING_REGISTRATION
			userState[chatID] = models.STATE_WAITING_NAME

			removeKeyboard(chatID)
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

			send(chatID, "Номер телефона сохранён.")
			sendRoleKeyboard(chatID)

			userState[chatID] = models.STATE_WAITING_ROLE

		case models.STATE_WAITING_ROLE:
			if !isValidRole(text) {
				send(chatID, "Пожалуйста, выберите роль с помощью кнопок.")
				return
			}

			userData[chatID]["Role"] = text

			user := models.User{
				TgID:     tgID,
				FullName: userData[chatID]["Name"],
				Phone:    stringPointer(userData[chatID]["Phone"]),
				Role:     userData[chatID]["Role"],
			}

			if err := userService.CreateUser(ctx, &user); err != nil {
				log.Println("[TELEGRAM] failed to create user:", err)
				send(chatID, "Не удалось завершить регистрацию. Попробуйте ещё раз.")
				return
			}

			removeKeyboard(chatID)
			send(chatID, "Регистрация успешно завершена.")

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

func sendRoleKeyboard(chatID int64) {
	rows := make([][]tgbotapi.KeyboardButton, 0)
	row := make([]tgbotapi.KeyboardButton, 0)

	for i, item := range models.Role_Menu {
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
	msg := tgbotapi.NewMessage(chatID, "Выберите вашу роль.")
	msg.ReplyMarkup = keyboard

	if _, err := bot.Send(msg); err != nil {
		log.Println("[TELEGRAM] failed to send role keyboard:", err)
	}
}

func isValidRole(role string) bool {
	role = strings.TrimSpace(role)

	for _, item := range models.Role_Menu {
		if item.Title == role {
			return true
		}
	}

	return false
}

func removeKeyboard(chatID int64) {
	msg := tgbotapi.NewMessage(chatID, "Клавиатура убрана.")
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
