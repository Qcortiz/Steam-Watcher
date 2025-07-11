package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var userWatches = map[int64]map[int]bool{} // chatID → appIDs
var lastNotified = map[int64]map[int]int{} // chatID → appID → last %

func main() {
	config := LoadConfig()

	bot, err := tgbotapi.NewBotAPI(config.TelegramToken)
	if err != nil {
		log.Panic(err)
	}
	bot.Debug = false
	log.Printf("✅ Бот запущен как %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	go startDiscountWatcher(bot)

	for update := range updates {
		if update.CallbackQuery != nil {
			handleCallback(bot, update.CallbackQuery)
			continue
		}

		if update.Message == nil {
			continue
		}

		text := strings.TrimSpace(update.Message.Text)

		if text == "/start" {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, `👋 Добро пожаловать в Steam Price Bot!
Отправь название игры, и я покажу цену и DLC. Можно подписаться на скидку!`)
			bot.Send(msg)
			continue
		}

		log.Printf("🔍 Запрос: %s", text)

		game, err := GetSteamGameWithDLCs(text)
		if err != nil {
			sendError(bot, update.Message.Chat.ID, err)
			continue
		}

		msgText := fmt.Sprintf("🎮 %s\n💸 Цена: %s\n🔗 %s", game.Title, game.Price, game.Link)
		if len(game.DLCs) > 0 {
			msgText += "\n\n📦 DLC:"
			for _, dlc := range game.DLCs {
				msgText += fmt.Sprintf("\n- %s — %s", dlc.Title, dlc.Price)
			}
		}

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, msgText)

		// Показываем кнопку только если игра не бесплатна
		if game.Price != "Бесплатно" {
			btn := tgbotapi.NewInlineKeyboardButtonData("🔔 Напоминать о скидке", fmt.Sprintf("watch_%d", game.AppID))
			msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(btn))
		}

		bot.Send(msg)
	}
}

func handleCallback(bot *tgbotapi.BotAPI, query *tgbotapi.CallbackQuery) {
	if strings.HasPrefix(query.Data, "watch_") {
		appIDStr := strings.TrimPrefix(query.Data, "watch_")
		var appID int
		_, err := fmt.Sscanf(appIDStr, "%d", &appID)
		if err != nil {
			log.Printf("❌ Невозможно распарсить appID: %v", err)
			bot.Send(tgbotapi.NewMessage(query.Message.Chat.ID, "❌ Ошибка добавления в отслеживание"))
			return
		}

		chatID := query.Message.Chat.ID
		if userWatches[chatID] == nil {
			userWatches[chatID] = make(map[int]bool)
		}
		if lastNotified[chatID] == nil {
			lastNotified[chatID] = make(map[int]int)
		}

		userWatches[chatID][appID] = true
		lastNotified[chatID][appID] = 0

		cb := tgbotapi.NewCallback(query.ID, "🎮 Игра добавлена в отслеживание!")
		bot.Request(cb)

		bot.Send(tgbotapi.NewMessage(chatID, "🔔 Я сообщу, когда появится скидка на эту игру!"))
	}
}

func sendError(bot *tgbotapi.BotAPI, chatID int64, err error) {
	msg := "❌ Ошибка: " + err.Error()
	if strings.Contains(err.Error(), "недоступна в вашем регионе") {
		msg = "⛔ " + err.Error()
	}
	bot.Send(tgbotapi.NewMessage(chatID, msg))
}

func startDiscountWatcher(bot *tgbotapi.BotAPI) {
	for {
		time.Sleep(30 * time.Minute)

		for chatID, appSet := range userWatches {
			for appID := range appSet {
				details, err := getAppDetails(appID)
				if err != nil || details.IsFree || details.PriceOverview.FinalFormatted == "" {
					continue
				}

				discount := extractDiscount(details)
				last := lastNotified[chatID][appID]

				if discount > 0 && discount != last {
					msg := fmt.Sprintf("🎉 Скидка на %s: -%d%%\n💸 Сейчас: %s\n🔗 https://store.steampowered.com/app/%d",
						details.Name, discount, details.PriceOverview.FinalFormatted, appID)

					bot.Send(tgbotapi.NewMessage(chatID, msg))
					lastNotified[chatID][appID] = discount
				}
			}
		}
	}
}

// Пока возвращаем 0 — можно расширить при наличии старой цены
func extractDiscount(details AppDetails) int {
	return 0
}
