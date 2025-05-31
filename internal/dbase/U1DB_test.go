package dbase

import (
	"context"
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
		// {
		// 	name:       "Bad PASSWORD",
		// 	ctx:        context.Background(),
		// 	dbEndPoint: "postgres://testuser:testpassbad@localhost:5432/testdb",
		// 	wantErr:    true,
		// },
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
				Sugar.Debugln(err)
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
	yes := db.CheckUserPassword(suite.ctx, "userName", "password")
	suite.Require().True(yes)
	// wrong user
	yes = db.CheckUserPassword(suite.ctx, "userNames", "password")
	suite.Require().False(yes)
	// wrong password
	yes = db.CheckUserPassword(suite.ctx, "userName", "passwordas")
	suite.Require().False(yes)
	yes, _ = db.IfUserExists(suite.ctx, "userName")
	suite.Require().True(yes)
	// this user does not exist
	yes, _ = db.IfUserExists(suite.ctx, "userNames")
	suite.Require().False(yes)

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

func (suite *TstBase) Test04Tokens() {
	db, err := ConnectToDB(suite.ctx, suite.DBEndPoint)
	suite.Require().NoError(err)
	
	db.CloseBase()
}