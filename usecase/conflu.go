package usecase

import (
	"arifthalhah/waBot/config"
)

func PostToConfluence() {

	clt := config.NewClient()
	clt.Post("")
}
