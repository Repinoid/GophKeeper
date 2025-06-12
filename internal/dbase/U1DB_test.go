package dbase

import (
	"context"
	"gorsovet/internal/minios3"
	"gorsovet/internal/models"
	"strings"
)

func (suite *TstBase) Test00InitDB() {
	tests := []struct {
		name    string
		ctx     context.Context
		dbe     string
		wantErr bool
	}{
		{
			name:    "InitDB Bad BASE",
			ctx:     context.Background(),
			dbe:     suite.DBEndPoint + "a",
			wantErr: true,
		},
		{
			name:    "InitDB Grace manner", // last - RIGHT base params. чтобы база была открыта для дальнейших тестов
			ctx:     context.Background(),
			dbe:     suite.DBEndPoint,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		suite.Run(tt.name, func() {
			db, err := ConnectToDB(tt.ctx, tt.dbe)
			if err != nil {
				models.Sugar.Debugln(err)
			} else {
				db.CloseBase()
			}
			suite.Require().Equal(err != nil, tt.wantErr) //
		})
	}
}

func (suite *TstBase) Test01CreateBases() {
	db, err := ConnectToDB(suite.ctx, suite.DBEndPoint)
	suite.Require().NoError(err)
	// create tables USERA TOKENA DATAS
	err = db.UsersTableCreation(suite.ctx)
	suite.Require().NoError(err)
	err = db.TokensTableCreation(suite.ctx)
	suite.Require().NoError(err)
	err = db.DataTableCreation(suite.ctx)
	suite.Require().NoError(err)
	db.CloseBase()
}

func (suite *TstBase) Test02AddCheckUser() {
	db, err := ConnectToDB(suite.ctx, suite.DBEndPoint)
	suite.Require().NoError(err)

	err = db.AddUser(suite.ctx, "userName", "password", "metaData")
	suite.Require().NoError(err)
	err = db.CheckUserPassword(suite.ctx, "userName", "password")
	suite.Require().NoError(err)
	// wrong user
	err = db.CheckUserPassword(suite.ctx, "userNames", "password")
	suite.Require().Error(err)
	// wrong password
	err = db.CheckUserPassword(suite.ctx, "userName", "passwordas")
	suite.Require().Error(err)
	_, err = db.IfUserExists(suite.ctx, "userName")
	suite.Require().NoError(err)
	// this user does not exist
	_, err = db.IfUserExists(suite.ctx, "userNames")
	suite.Require().Error(err)

	db.CloseBase()
}

func (suite *TstBase) Test03Tokens() {
	db, err := ConnectToDB(suite.ctx, suite.DBEndPoint)
	suite.Require().NoError(err)

	userName := "userName"
	err = db.PutToken(suite.ctx, userName, "tokenstring", "metadata string")
	suite.Require().NoError(err)

	token, err := db.GetUserNameByToken(suite.ctx, "tokenstring")
	suite.Require().NoError(err)
	// имена пользователей переводятся при регистрации в UPPERCASE, названия бакетов - изначально те же, но LOWERCASE
	suite.Require().Equal(strings.ToUpper(userName), token)
	// несуществующий токен
	_, err = db.GetUserNameByToken(suite.ctx, "tokenstringer")
	suite.Require().Error(err)

	_, bucketname, err := db.GetBucketKeyByUserName(suite.ctx, "userName")
	suite.Require().NoError(err)
	suite.Require().Equal(strings.ToLower(userName), bucketname)
	// несуществующий юзер
	_, _, err = db.GetBucketKeyByUserName(suite.ctx, "userNamer")
	suite.Require().Error(err)

	db.CloseBase()
}

func (suite *TstBase) Test04PutFileParams() {
	db, err := ConnectToDB(suite.ctx, suite.DBEndPoint)
	suite.Require().NoError(err)

	// номер записи 0 - новая запись, присвоится 1й номер
	err = db.PutFileParams(suite.ctx, 0, "username", "fileURL", "dataType", "fileKey", "metaData", 1111)
	suite.Require().NoError(err)

	err = minios3.CreateBucket(suite.ctx, suite.minioClient, "username")
	suite.Require().NoError(err)
	// "username" here - bucket name
	info, err := minios3.S3PutFile(suite.ctx, suite.minioClient, "username", "objectName", "cert.bat", suite.sse)
	suite.Require().NoError(err)

	// должна быть ошибка т.к. записи 7 нет
	err = db.PutFileParams(suite.ctx, 7, "username", "fileURL", "dataType", "fileKey", "metaData", 1111)
	suite.Require().Error(err)
	err = db.PutFileParams(suite.ctx, 1, "username", "fileURL", "dataType", "fileKey", "metaData", int32(info.Size))
	suite.Require().NoError(err)

	_, err = db.RemoveObjects(suite.ctx, "username", 1)
	suite.Require().NoError(err)
	// remove removed - error
	_, err = db.RemoveObjects(suite.ctx, "username", 1)
	suite.Require().Error(err)

	db.CloseBase()
}
