package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type DLC struct {
	AppID int
	Title string
	Price string
}

type Game struct {
	Title string
	AppID int
	Price string
	Link  string
	DLCs  []DLC
}

type AppDetails struct {
	Name          string `json:"name"`
	IsFree        bool   `json:"is_free"`
	DLC           []int  `json:"dlc"`
	PriceOverview struct {
		FinalFormatted string `json:"final_formatted"`
	} `json:"price_overview"`
}

// 🔍 Основная логика: получить игру + DLC по названию
func GetSteamGameWithDLCs(name string) (Game, error) {
	var result Game

	// Поиск по названию
	game, err := searchGame(name)
	if err != nil {
		return result, err
	}

	// Получение деталей основной игры
	details, err := getAppDetails(game.AppID)
	if err != nil {
		return result, err
	}

	result.Title = game.Title
	result.AppID = game.AppID
	result.Link = game.Link

	// 💥 Проверка доступности игры в регионе
	if !details.IsFree && details.PriceOverview.FinalFormatted == "" {
		return result, fmt.Errorf("игра \"%s\" недоступна в вашем регионе", result.Title)
	}

	// ✅ Определение цены
	if details.IsFree {
		result.Price = "Бесплатно"
	} else {
		result.Price = details.PriceOverview.FinalFormatted
	}

	// 📦 Обработка DLC
	for _, dlcID := range details.DLC {
		time.Sleep(200 * time.Millisecond) // Защита от спама API

		dlcDetails, err := getAppDetails(dlcID)
		if err != nil {
			// Недоступное DLC — тихо пропускаем
			if strings.Contains(err.Error(), "недоступна") {
				log.Printf("⚠️ DLC %d недоступно в регионе, пропущено", dlcID)
				continue
			}
			// Другие ошибки — считаем критическими
			return result, fmt.Errorf("ошибка при получении DLC %d: %v", dlcID, err)
		}

		// Определение цены DLC
		price := "Цена недоступна"
		if dlcDetails.IsFree {
			price = "Бесплатно"
		} else if dlcDetails.PriceOverview.FinalFormatted != "" {
			price = dlcDetails.PriceOverview.FinalFormatted
		}

		result.DLCs = append(result.DLCs, DLC{
			AppID: dlcID,
			Title: dlcDetails.Name,
			Price: price,
		})
	}

	return result, nil
}

// 🔎 Поиск appid по названию
func searchGame(name string) (Game, error) {
	var result Game
	baseURL := "https://store.steampowered.com/api/storesearch"
	u, _ := url.Parse(baseURL)
	q := u.Query()
	q.Set("term", name)
	q.Set("cc", "ru")
	q.Set("l", "russian")
	u.RawQuery = q.Encode()

	resp, err := http.Get(u.String())
	if err != nil {
		return result, err
	}
	defer resp.Body.Close()

	var data struct {
		Items []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return result, err
	}
	if len(data.Items) == 0 {
		return result, fmt.Errorf("игра не найдена")
	}

	result.AppID = data.Items[0].ID
	result.Title = data.Items[0].Name
	result.Link = fmt.Sprintf("https://store.steampowered.com/app/%d", result.AppID)
	return result, nil
}

// 📦 Получение полной информации об игре или DLC по appid
func getAppDetails(appid int) (AppDetails, error) {
	var result AppDetails
	url := fmt.Sprintf("https://store.steampowered.com/api/appdetails?appids=%d&cc=ru&l=russian", appid)

	resp, err := http.Get(url)
	if err != nil {
		return result, err
	}
	defer resp.Body.Close()

	var data map[string]struct {
		Success bool       `json:"success"`
		Data    AppDetails `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return result, err
	}

	entry, ok := data[fmt.Sprintf("%d", appid)]
	if !ok || !entry.Success {
		return result, fmt.Errorf("игра или DLC недоступна в вашем регионе")
	}

	return entry.Data, nil
}
