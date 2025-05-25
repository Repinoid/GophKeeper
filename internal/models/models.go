package models

import "go.uber.org/zap"

var (
	Sugar      zap.SugaredLogger
	Gport      = ":3200"
	DBEndPoint = "postgres://userp:parole@localhost:5432/dbaza"
)
