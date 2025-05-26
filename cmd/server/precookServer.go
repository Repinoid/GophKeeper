package main

import (
	"context"
	"gorsovet/internal/dbase"
	"gorsovet/internal/minio"
	"gorsovet/internal/models"
	"log"

	"go.uber.org/zap"
)

func initServer(ctx context.Context) (err error) {

	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal("cannot initialize zap")
	}
	defer logger.Sync()
	models.Sugar = *logger.Sugar()

	// S3 Create one client and reuse it (it's thread-safe)
	// дескриптор S3, один на всех
	models.MinioClient, err = minio.ConnectToS3()
	if err != nil {
		models.Sugar.Fatalf("No connection with S3. %w", err)
	}

	db, err := dbase.ConnectToDB(ctx, models.DBEndPoint)
	if err != nil {
		models.Sugar.Debugln(err)
		return
	}
	defer db.CloseBase()

	if err = db.UsersTableCreation(ctx); err != nil {
		return
	}
	if err = db.TokensTableCreation(ctx); err != nil {
		return
	}
	if err = db.DataTableCreation(ctx); err != nil {
		return
	}
	// ....
	return
}
