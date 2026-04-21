package db

import (
	"miraclevpn/internal/models"

	"gorm.io/gorm"
)

type previewSeed struct {
	Code    string
	Name    string
	FlagURL string
	Lat     float64
	Lng     float64
}

var previewSeeds = []previewSeed{
	{"RU", "Россия", "", 55.75, 37.62},
	{"US", "США", "", 38.89, -77.04},
	{"DE", "Германия", "", 52.52, 13.40},
	{"NL", "Нидерланды", "", 52.37, 4.90},
	{"FR", "Франция", "", 48.86, 2.35},
	{"GB", "Великобритания", "", 51.51, -0.13},
	{"CH", "Швейцария", "", 47.37, 8.54},
	{"JP", "Япония", "", 35.68, 139.69},
	{"SG", "Сингапур", "", 1.35, 103.82},
	{"CA", "Канада", "", 45.42, -75.70},
	{"AU", "Австралия", "", -33.87, 151.21},
	{"SE", "Швеция", "", 59.33, 18.07},
	{"FI", "Финляндия", "", 60.17, 24.94},
	{"NO", "Норвегия", "", 59.91, 10.75},
	{"DK", "Дания", "", 55.68, 12.57},
	{"PL", "Польша", "", 52.23, 21.01},
	{"CZ", "Чехия", "", 50.08, 14.43},
	{"AT", "Австрия", "", 48.21, 16.37},
	{"IT", "Италия", "", 41.90, 12.50},
	{"ES", "Испания", "", 40.42, -3.70},
	{"PT", "Португалия", "", 38.72, -9.14},
	{"BE", "Бельгия", "", 50.85, 4.35},
	{"IE", "Ирландия", "", 53.33, -6.25},
	{"TR", "Турция", "", 39.92, 32.85},
	{"UA", "Украина", "", 50.45, 30.52},
	{"RO", "Румыния", "", 44.43, 26.10},
	{"HU", "Венгрия", "", 47.50, 19.04},
	{"SK", "Словакия", "", 48.15, 17.11},
	{"BG", "Болгария", "", 42.70, 23.32},
	{"RS", "Сербия", "", 44.80, 20.47},
	{"LT", "Литва", "", 54.69, 25.28},
	{"LV", "Латвия", "", 56.95, 24.11},
	{"EE", "Эстония", "", 59.44, 24.75},
	{"BR", "Бразилия", "", -15.78, -47.93},
	{"AR", "Аргентина", "", -34.60, -58.38},
	{"MX", "Мексика", "", 19.43, -99.13},
	{"IN", "Индия", "", 28.61, 77.21},
	{"KR", "Южная Корея", "", 37.57, 126.98},
	{"HK", "Гонконг", "", 22.32, 114.17},
	{"TW", "Тайвань", "", 25.05, 121.53},
	{"IL", "Израиль", "", 31.78, 35.22},
	{"AE", "ОАЭ", "", 24.45, 54.38},
	{"ZA", "ЮАР", "", -25.75, 28.19},
	{"NZ", "Новая Зеландия", "", -36.87, 174.77},
	{"IS", "Исландия", "", 64.13, -21.82},
	{"LU", "Люксембург", "", 49.61, 6.13},
	{"MD", "Молдова", "", 47.01, 28.86},
	{"HR", "Хорватия", "", 45.81, 15.98},
	{"GR", "Греция", "", 37.98, 23.73},
	{"CN", "Китай", "", 39.90, 116.41},
}

func seedPreviewServers(db *gorm.DB) error {
	for _, s := range previewSeeds {
		host := "preview-" + s.Code + ".seed.internal"
		server := models.Server{
			Host:       host,
			Name:       s.Code + "-preview",
			Type:       models.ServerTypeOVPN,
			Region:     s.Code,
			RegionName: s.Name,
			Lat:        s.Lat,
			Lng:        s.Lng,
			Preview:    true,
			Active:     true,
		}
		if err := db.Where(models.Server{Host: host}).
			Attrs(server).
			FirstOrCreate(&server).Error; err != nil {
			return err
		}
	}
	return nil
}

func updateRealServerCoords(db *gorm.DB) error {
	return db.Model(&models.Server{}).
		Where("region = ? AND preview = ? AND lat = ? AND lng = ?", "CH", false, 0.0, 0.0).
		Updates(map[string]any{"lat": 47.37, "lng": 8.54}).Error
}

var reviewSeeds = []models.Review{
	{Name: "@morozz_ea", Text: "Все работает как надо, ничего не тупит и не вылетает. Рекомендую", PhotoURL: "/img/reviews/6.jpg", URL: "https://vk.com/morozz_ea", Active: true, SortOrder: 0},
	{Name: "Наталья Комкова (Лаврова)", Text: "Отлично работает\nВсе приложения открываются быстро\nНе вылетает", PhotoURL: "/img/reviews/5.jpg", URL: "https://vk.com/id167108613", Active: true, SortOrder: 1},
	{Name: "@alexvitov", Text: "Пользуюсь сервисом уже несколько месяцев и могу уверенно порекомендовать его.", PhotoURL: "/img/reviews/1.jpg", URL: "https://t.me/alexvitov", Active: true, SortOrder: 2},
	{Name: "@EeeelllAl", Text: "Зашибись. Мощный быстрый. Ютуб грузит", PhotoURL: "/img/reviews/0.jpg", URL: "https://t.me/EeeelllAl", Active: true, SortOrder: 3},
	{Name: "@steshaworks", Text: "Ура работает\nВсе супер", PhotoURL: "/img/reviews/4.jpg", URL: "https://t.me/steshaworks", Active: true, SortOrder: 4},
	{Name: "@Gleb656", Text: "бомба", PhotoURL: "/img/reviews/3.jpg", URL: "https://t.me/Gleb656", Active: true, SortOrder: 5},
}

func seedReviews(db *gorm.DB) error {
	for i := range reviewSeeds {
		r := reviewSeeds[i]
		if err := db.Where(models.Review{Name: r.Name}).FirstOrCreate(&r).Error; err != nil {
			return err
		}
	}
	return nil
}

func runSeed(db *gorm.DB) error {
	if err := seedPreviewServers(db); err != nil {
		return err
	}
	if err := seedReviews(db); err != nil {
		return err
	}
	return updateRealServerCoords(db)
}
