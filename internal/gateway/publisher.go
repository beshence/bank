package gateway

import (
	"bank/internal/settings"
	"log"
	"time"

	"gorm.io/gorm"
)

func StartPublisher(db *gorm.DB) {
	interval := 15 * time.Minute // TODO: from env

	go func() {
		err := publish(db)
		if err != nil {
			log.Fatal("[gateway] could not publish urls:", err)
		}

		log.Println("[gateway] published urls successfully")

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			err := publish(db)
			if err != nil {
				log.Fatal("could not publish urls:", err)
			}
			log.Println("[gateway] published urls successfully")
		}
	}()
}

func publish(db *gorm.DB) error {
	gatewayURL := "https://gateway.beshence.com/api"
	bankID := settings.GetBankID(db)

	token, err := GetGatewayToken(db)

	if err != nil {
		return err
	}

	_, err = post(
		gatewayURL+"/bank/"+bankID+"/urls",
		map[string][]string{
			"api_urls": settings.GetAPIUrls(),
		},
		token,
	)

	return err
}
