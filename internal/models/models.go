package models

import "go.uber.org/zap"

var (
	Sugar zap.SugaredLogger
	Gport = ":3200"
	// password minimum 8 symbols !!!!
	DBEndPoint = "postgres://userw:myparole@localhost:5432/baza"
)
