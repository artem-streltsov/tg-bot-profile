package handler

import (
	"fmt"
	"log"
	"strings"
	"time"

	"tg-bot-profile/database"
	"tg-bot-profile/models"
	"tg-bot-profile/state"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type userMessages struct {
	promptMessageID  int
	profileMessageID int
	invoiceMessageID int
}

var userMessageIDs = make(map[int64]*userMessages)

func HandleUpdate(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	if update.Message != nil {
		if update.Message.IsCommand() {
			handleCommand(bot, update)
		} else if update.Message.SuccessfulPayment != nil {
			handleSuccessfulPayment(bot, update)
		} else {
			handleMessage(bot, update)
		}
	} else if update.CallbackQuery != nil {
		handleCallbackQuery(bot, update)
	} else if update.PreCheckoutQuery != nil {
		handlePreCheckoutQuery(bot, update)
	}
}

func handleCommand(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	switch update.Message.Command() {
	case "start":
		handleStart(bot, update)
	case "profile":
		userID := update.Message.From.ID
		chatID := update.Message.Chat.ID
		handleProfile(bot, userID, chatID, true)
	default:
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Неизвестная команда.")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending unknown command message: %v", err)
		}
	}
}

func handleStart(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	userID := update.Message.From.ID
	chatID := update.Message.Chat.ID

	removeProfileButtons(bot, userID, chatID)

	existingUser, err := database.GetUser(userID)
	if err == nil && existingUser != nil {
		msg := tgbotapi.NewMessage(chatID, "Вы уже зарегистрированы.")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending already registered message: %v", err)
		}
		return
	}

	user := &models.User{
		ID:        userID,
		FirstName: update.Message.From.FirstName,
		LastName:  update.Message.From.LastName,
		UserName:  update.Message.From.UserName,
	}

	err = database.SaveUser(user)
	if err != nil {
		log.Printf("Error saving user: %v", err)
	}

	msg := tgbotapi.NewMessage(chatID, "Добро пожаловать!")
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending welcome message: %v", err)
	}
}

func handleProfile(bot *tgbotapi.BotAPI, userID int64, chatID int64, sendNew bool) {
	user, err := database.GetUser(userID)
	if err != nil || user == nil {
		msg := tgbotapi.NewMessage(chatID, "Профиль не найден. Пожалуйста, используйте /start для регистрации.")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending message: %v", err)
		}
		return
	}

	removeProfileButtons(bot, userID, chatID)

	premiumStatus := "Нет"
	if user.IsPremium {
		daysLeft := int(user.PremiumExpiry.Sub(time.Now()).Hours() / 24)
		premiumStatus = fmt.Sprintf("Истекает через %d дней", daysLeft)
	}

	firstName := user.FirstName
	if firstName == "" {
		firstName = "неизвестно"
	}

	birthDate := user.BirthDate
	if birthDate == "" {
		birthDate = "неизвестно"
	}

	birthTime := user.BirthTime
	if birthTime == "" {
		birthTime = "неизвестно"
	}

	zodiacSign := user.ZodiacSign
	if zodiacSign == "" {
		zodiacSign = "неизвестно"
	}

	profileText := fmt.Sprintf(
		"Имя: %s\nЗнак зодиака: %s\nДата рождения: %s\nВремя рождения: %s\nПремиум: %s",
		firstName,
		zodiacSign,
		birthDate,
		birthTime,
		premiumStatus,
	)

	userMsg, exists := userMessageIDs[userID]
	if !exists {
		userMsg = &userMessages{}
		userMessageIDs[userID] = userMsg
	}

	if sendNew || userMsg.profileMessageID == 0 {
		msg := tgbotapi.NewMessage(chatID, profileText)
		msg.ReplyMarkup = profileKeyboard()
		sentMsg, err := bot.Send(msg)
		if err != nil {
			log.Printf("Error sending profile message: %v", err)
			return
		}
		userMsg.profileMessageID = sentMsg.MessageID
	} else {
		editMsg := tgbotapi.NewEditMessageText(chatID, userMsg.profileMessageID, profileText)
		editMsg.ReplyMarkup = profileKeyboard()
		_, err := bot.Send(editMsg)
		if err != nil {
			if strings.Contains(err.Error(), "message is not modified") {
			} else {
				log.Printf("Error editing profile message: %v", err)
			}
		}
	}
}

func profileKeyboard() *tgbotapi.InlineKeyboardMarkup {
	buttons := [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData("Изменить имя", "edit_name"),
			tgbotapi.NewInlineKeyboardButtonData("Изменить знак зодиака", "edit_zodiac"),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("Изменить дату рождения", "edit_birthdate"),
			tgbotapi.NewInlineKeyboardButtonData("Изменить время рождения", "edit_birthtime"),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("Купить премиум", "buy_premium"),
		},
	}
	markup := tgbotapi.NewInlineKeyboardMarkup(buttons...)
	return &markup
}

func handleCallbackQuery(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	data := update.CallbackQuery.Data
	chatID := update.CallbackQuery.Message.Chat.ID
	messageID := update.CallbackQuery.Message.MessageID
	userID := update.CallbackQuery.From.ID

	userMsg, exists := userMessageIDs[userID]
	if !exists {
		userMsg = &userMessages{}
		userMessageIDs[userID] = userMsg
	}

	switch data {
	case "edit_name":
		state.SetState(userID, state.StateEditingName)
		editMsg := tgbotapi.NewEditMessageText(chatID, messageID, "Пожалуйста, введите Ваше имя:")
		replyMarkup := cancelKeyboard()
		editMsg.ReplyMarkup = replyMarkup
		if _, err := bot.Send(editMsg); err != nil {
			log.Printf("Error editing message for edit_name: %v", err)
		}
		userMsg.promptMessageID = messageID
	case "edit_zodiac":
		state.SetState(userID, state.StateEditingZodiac)
		editMsg := tgbotapi.NewEditMessageText(chatID, messageID, "Пожалуйста, выберите Ваш знак зодиака:")
		replyMarkup := zodiacKeyboard()
		editMsg.ReplyMarkup = replyMarkup
		if _, err := bot.Send(editMsg); err != nil {
			log.Printf("Error editing message for edit_zodiac: %v", err)
		}
		userMsg.promptMessageID = messageID
	case "edit_birthdate":
		state.SetState(userID, state.StateEditingBirthDate)
		editMsg := tgbotapi.NewEditMessageText(chatID, messageID, "Пожалуйста, введите Вашу дату рождения (дд/мм/гггг):")
		replyMarkup := cancelKeyboard()
		editMsg.ReplyMarkup = replyMarkup
		if _, err := bot.Send(editMsg); err != nil {
			log.Printf("Error editing message for edit_birthdate: %v", err)
		}
		userMsg.promptMessageID = messageID
	case "edit_birthtime":
		state.SetState(userID, state.StateEditingBirthTime)
		editMsg := tgbotapi.NewEditMessageText(chatID, messageID, "Пожалуйста, введите Ваше время рождения (чч:мм):")
		replyMarkup := cancelKeyboard()
		editMsg.ReplyMarkup = replyMarkup
		if _, err := bot.Send(editMsg); err != nil {
			log.Printf("Error editing message for edit_birthtime: %v", err)
		}
		userMsg.promptMessageID = messageID
	case "buy_premium":
		sendInvoice(bot, chatID, update.CallbackQuery.From)
	case "cancel":
		state.ClearState(userID)
		handleProfile(bot, userID, chatID, false)
	default:
		if strings.HasPrefix(data, "zodiac_") {
			zodiacCode := strings.TrimPrefix(data, "zodiac_")
			zodiacMap := map[string]string{
				"aries":       "♈️ Овен",
				"taurus":      "♉️ Телец",
				"gemini":      "♊️ Близнецы",
				"cancer":      "♋️ Рак",
				"leo":         "♌️ Лев",
				"virgo":       "♍️ Дева",
				"libra":       "♎️ Весы",
				"scorpio":     "♏️ Скорпион",
				"sagittarius": "♐️ Стрелец",
				"capricorn":   "♑️ Козерог",
				"aquarius":    "♒️ Водолей",
				"pisces":      "♓️ Рыбы",
			}

			zodiac, exists := zodiacMap[zodiacCode]
			if !exists {
				log.Printf("Unknown zodiac code: %v", zodiacCode)
				return
			}

			user, err := database.GetUser(userID)
			if err != nil {
				log.Printf("Error fetching user: %v", err)
				return
			}
			user.ZodiacSign = zodiac
			err = database.SaveUser(user)
			if err != nil {
				log.Printf("Error updating user zodiac sign: %v", err)
			}
			state.ClearState(userID)

			handleProfile(bot, userID, chatID, false)
		} else {
			log.Printf("Unknown callback data: %v", data)
		}
	}

	ack := tgbotapi.NewCallback(update.CallbackQuery.ID, "")
	if _, err := bot.Request(ack); err != nil {
		log.Printf("Error acknowledging callback query: %v", err)
	}
}

func handleMessage(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	text := update.Message.Text
	userID := update.Message.From.ID
	chatID := update.Message.Chat.ID

	currentState := state.GetState(userID)

	userMsg, exists := userMessageIDs[userID]
	if !exists {
		userMsg = &userMessages{}
		userMessageIDs[userID] = userMsg
	}

	switch currentState {
	case state.StateEditingName:
		user, err := database.GetUser(userID)
		if err != nil {
			log.Printf("Error fetching user: %v", err)
			return
		}
		user.FirstName = text
		err = database.SaveUser(user)
		if err != nil {
			log.Printf("Error updating user name: %v", err)
		}

		delUserMsg := tgbotapi.NewDeleteMessage(chatID, update.Message.MessageID)
		if _, err := bot.Request(delUserMsg); err != nil {
			log.Printf("Error deleting user's message: %v", err)
		}

		state.ClearState(userID)
		handleProfile(bot, userID, chatID, false)

	case state.StateEditingBirthDate:
		if !isValidDate(text) {
			msg := tgbotapi.NewMessage(chatID, "Некорректная дата. Пожалуйста, введите дату в формате дд/мм/гггг:")
			msg.ReplyMarkup = cancelKeyboard()
			if _, err := bot.Send(msg); err != nil {
				log.Printf("Error sending invalid date message: %v", err)
			}
			return
		}
		user, err := database.GetUser(userID)
		if err != nil {
			log.Printf("Error fetching user: %v", err)
			return
		}
		user.BirthDate = text
		err = database.SaveUser(user)
		if err != nil {
			log.Printf("Error updating user birth date: %v", err)
		}

		delUserMsg := tgbotapi.NewDeleteMessage(chatID, update.Message.MessageID)
		if _, err := bot.Request(delUserMsg); err != nil {
			log.Printf("Error deleting user's message: %v", err)
		}

		state.ClearState(userID)
		handleProfile(bot, userID, chatID, false)

	case state.StateEditingBirthTime:
		if !isValidTime(text) {
			msg := tgbotapi.NewMessage(chatID, "Некорректное время. Пожалуйста, введите время в формате чч:мм:")
			msg.ReplyMarkup = cancelKeyboard()
			if _, err := bot.Send(msg); err != nil {
				log.Printf("Error sending invalid time message: %v", err)
			}
			return
		}
		user, err := database.GetUser(userID)
		if err != nil {
			log.Printf("Error fetching user: %v", err)
			return
		}
		user.BirthTime = text
		err = database.SaveUser(user)
		if err != nil {
			log.Printf("Error updating user birth time: %v", err)
		}

		delUserMsg := tgbotapi.NewDeleteMessage(chatID, update.Message.MessageID)
		if _, err := bot.Request(delUserMsg); err != nil {
			log.Printf("Error deleting user's message: %v", err)
		}

		state.ClearState(userID)
		handleProfile(bot, userID, chatID, false)

	default:
		msg := tgbotapi.NewMessage(chatID, "Я не понимаю это сообщение. Пожалуйста, используйте команды или кнопки для взаимодействия со мной.")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending default message: %v", err)
		}

		removeProfileButtons(bot, userID, chatID)
	}
}

func isValidDate(dateStr string) bool {
	_, err := time.Parse("02/01/2006", dateStr)
	return err == nil
}

func isValidTime(timeStr string) bool {
	_, err := time.Parse("15:04", timeStr)
	return err == nil
}

func zodiacKeyboard() *tgbotapi.InlineKeyboardMarkup {
	zodiacSigns := []struct {
		Code  string
		Label string
	}{
		{"aries", "♈️ Овен"},
		{"taurus", "♉️ Телец"},
		{"gemini", "♊️ Близнецы"},
		{"cancer", "♋️ Рак"},
		{"leo", "♌️ Лев"},
		{"virgo", "♍️ Дева"},
		{"libra", "♎️ Весы"},
		{"scorpio", "♏️ Скорпион"},
		{"sagittarius", "♐️ Стрелец"},
		{"capricorn", "♑️ Козерог"},
		{"aquarius", "♒️ Водолей"},
		{"pisces", "♓️ Рыбы"},
	}
	var buttons [][]tgbotapi.InlineKeyboardButton
	for i := 0; i < len(zodiacSigns); i += 2 {
		row := []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(zodiacSigns[i].Label, "zodiac_"+zodiacSigns[i].Code),
		}
		if i+1 < len(zodiacSigns) {
			row = append(row, tgbotapi.NewInlineKeyboardButtonData(zodiacSigns[i+1].Label, "zodiac_"+zodiacSigns[i+1].Code))
		}
		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(row...))
	}
	buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("Отменить", "cancel"),
	))
	markup := tgbotapi.NewInlineKeyboardMarkup(buttons...)
	return &markup
}

func cancelKeyboard() *tgbotapi.InlineKeyboardMarkup {
	buttons := [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData("Отменить", "cancel"),
		},
	}
	markup := tgbotapi.NewInlineKeyboardMarkup(buttons...)
	return &markup
}

func sendInvoice(bot *tgbotapi.BotAPI, chatID int64, user *tgbotapi.User) {
	invoice := tgbotapi.InvoiceConfig{
		BaseChat:            tgbotapi.BaseChat{ChatID: chatID},
		Title:               "Премиум подписка",
		Description:         "Получите доступ к премиум функциям",
		Payload:             "payload_premium_subscription",
		ProviderToken:       "",
		Currency:            "XTR",
		Prices:              []tgbotapi.LabeledPrice{{Label: "Премиум подписка на 1 месяц", Amount: 1}},
		SuggestedTipAmounts: []int{},
	}

	sentMsg, err := bot.Send(invoice)
	if err != nil {
		log.Printf("Error sending invoice: %v", err)
		return
	}

	userMsg, exists := userMessageIDs[user.ID]
	if !exists {
		userMsg = &userMessages{}
		userMessageIDs[user.ID] = userMsg
	}
	userMsg.invoiceMessageID = sentMsg.MessageID
}

func handlePreCheckoutQuery(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	preCheckoutConfig := tgbotapi.PreCheckoutConfig{
		PreCheckoutQueryID: update.PreCheckoutQuery.ID,
		OK:                 true,
	}
	if _, err := bot.Request(preCheckoutConfig); err != nil {
		log.Printf("Error in pre-checkout: %v", err)
	}
}

func handleSuccessfulPayment(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	userID := update.Message.From.ID
	chatID := update.Message.Chat.ID

	delPaymentMsg := tgbotapi.NewDeleteMessage(chatID, update.Message.MessageID)
	if _, err := bot.Request(delPaymentMsg); err != nil {
		log.Printf("Error deleting payment confirmation message: %v", err)
	}

	user, err := database.GetUser(userID)
	if err != nil {
		log.Printf("Error fetching user: %v", err)
		return
	}

	user.IsPremium = true
	user.PremiumExpiry = time.Now().AddDate(0, 1, 0)
	err = database.SaveUser(user)
	if err != nil {
		log.Printf("Error updating user: %v", err)
	}

	userMsg, exists := userMessageIDs[userID]
	if exists {
		if userMsg.invoiceMessageID != 0 {
			delInvoiceMsg := tgbotapi.NewDeleteMessage(chatID, userMsg.invoiceMessageID)
			if _, err := bot.Request(delInvoiceMsg); err != nil {
				log.Printf("Error deleting invoice message: %v", err)
			}
			userMsg.invoiceMessageID = 0
		}

		handleProfile(bot, userID, chatID, false)
	}
}

func removeProfileButtons(bot *tgbotapi.BotAPI, userID int64, chatID int64) {
	userMsg, exists := userMessageIDs[userID]
	if !exists || userMsg.profileMessageID == 0 {
		return
	}

	editMarkup := tgbotapi.NewEditMessageReplyMarkup(chatID, userMsg.profileMessageID, tgbotapi.InlineKeyboardMarkup{InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{}})
	_, err := bot.Request(editMarkup)
	if err != nil {
		if strings.Contains(err.Error(), "message is not modified") {
			// Ignore this error
		} else {
			log.Printf("Error removing buttons from profile message: %v", err)
		}
	}
}
