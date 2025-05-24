package db_package

import (
	"context"
)

func (suite *TstBase) Test00InitDB() {
	tests := []struct {
		name       string
		ctx        context.Context
		dbEndPoint string
		wantErr    bool
	}{
		{
			name:       "InitDB Bad BASE",
			ctx:        context.Background(),
			dbEndPoint: suite.DBEndPoint + "a",
			wantErr:    true,
		},
		// {
		// 	name:       "Bad PASSWORD",
		// 	ctx:        context.Background(),
		// 	dbEndPoint: "postgres://testuser:testpassbad@localhost:5432/testdb",
		// 	wantErr:    true,
		// },
		{
			name:       "InitDB Grace manner", // last - RIGHT base params. чтобы база была открыта для дальнейших тестов
			ctx:        context.Background(),
			dbEndPoint: suite.DBEndPoint,
			wantErr:    false,
		},
	}
	for _, tt := range tests {
		suite.Run(tt.name, func() {
			db, err := ConnectToDB(tt.ctx, tt.dbEndPoint)
			if err != nil {
				Sugar.Debugln(err)
			}
			suite.dataBase = db
			suite.Require().Equal(err != nil, tt.wantErr) //
		})
	}
}

func (suite *TstBase) Test01AddUser() {

	err := suite.dataBase.AddUser(suite.ctx, "userName", "password", "metaData")
	suite.Require().NoError(err)
	yes := suite.dataBase.CheckUserPassword(suite.ctx, "userName", "password")
	suite.Require().True(yes)
	yes = suite.dataBase.CheckUserPassword(suite.ctx, "userName", "passwordas")
	suite.Require().False(yes)
	yes = suite.dataBase.IfUserExists(suite.ctx, "userName")
	suite.Require().True(yes)
	yes = suite.dataBase.IfUserExists(suite.ctx, "userName1")
	suite.Require().False(yes)

}
