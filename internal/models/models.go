package models

import "go.uber.org/zap"

var (
	Sugar      zap.SugaredLogger
	Gport      = ":3200"
	DBEndPoint = "postgres://username:parole@localhost:5433/baza"
)
