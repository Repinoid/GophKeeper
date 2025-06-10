package handlers

import (
	pb "gorsovet/cmd/proto"
	_ "net/http/pprof"
	"os"
	"strings"

	"gorsovet/internal/dbase"
	"gorsovet/internal/models"
)

func (suite *TstHand) Test01CreateBases() {
	db, err := dbase.ConnectToDB(suite.ctx, suite.DBEndPoint)
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

func (suite *TstHand) Test02RegisterUser() {

	// Создаем экземпляр сервера
	//	suite.serv := &GkeeperService{}

	req := &pb.RegisterRequest{Username: "usras1", Password: "pass", Metadata: "metta"}
	_, err := suite.serv.RegisterUser(suite.ctx, req)
	// База ещё не подключена - ОШИБКА
	suite.Require().Error(err)

	// подключаем базу
	models.DBEndPoint = suite.DBEndPoint

	// Создаем тестовый запрос
	req = &pb.RegisterRequest{Username: "rightUser", Password: "pass", Metadata: "metta"}
	// Вызываем метод
	resp, err := suite.serv.RegisterUser(suite.ctx, req)
	suite.Require().NoError(err)
	suite.Require().True(resp.Success)
	// номер юзера - 1. EqualValues чтобы с типами не заморачиваться.  1 - int, resp.UserId - int32
	suite.Require().EqualValues(1, resp.UserId)
	suite.Require().Contains(resp.Reply, "created")
	// дубль
	req = &pb.RegisterRequest{Username: "rightUser", Password: "pass", Metadata: "metta"}
	resp, err = suite.serv.RegisterUser(suite.ctx, req)
	suite.Require().Error(err)
	suite.Require().False(resp.Success)
	suite.Require().Contains(resp.Reply, "already exists")
	// no password
	req = &pb.RegisterRequest{Username: "usras1", Metadata: "metta"}
	resp, err = suite.serv.RegisterUser(suite.ctx, req)
	suite.Require().Error(err)
	suite.Require().False(resp.Success)
	suite.Require().Contains(resp.Reply, "Empty username or password")
	// недопуст символы
	req = &pb.RegisterRequest{Username: "usrasЮ", Password: "pass", Metadata: "metta"}
	resp, err = suite.serv.RegisterUser(suite.ctx, req)
	suite.Require().Error(err)
	suite.Require().False(resp.Success)
	suite.Require().Contains(resp.Reply, "latin symbols & digits")

}
func (suite *TstHand) Test03LoginUser() {
	// нормас
	req := &pb.LoginRequest{Username: "rightUser", Password: "pass", Metadata: "metta"}
	resp, err := suite.serv.LoginUser(suite.ctx, req)
	suite.Require().NoError(err)
	suite.Require().True(resp.Success)
	suite.Require().Contains(resp.Reply, "Auth OK")
	suite.Require().Greater(len(resp.Token), 10)

	db, err := dbase.ConnectToDB(suite.ctx, suite.DBEndPoint)
	suite.Require().NoError(err)
	order := "SELECT token from TOKENA WHERE username = $1 ;"
	row := db.DB.QueryRow(suite.ctx, order, strings.ToUpper("rightUser"))
	err = row.Scan(&suite.token)
	suite.Require().NoError(err)
	db.CloseBase()
	suite.Require().Equal(resp.Token, suite.token)

	// no password
	req = &pb.LoginRequest{Username: "rightUser", Metadata: "metta"}
	resp, err = suite.serv.LoginUser(suite.ctx, req)
	suite.Require().Error(err)
	suite.Require().False(resp.Success)
	suite.Require().Contains(resp.Reply, "Empty username or password")
	// wrong password
	req = &pb.LoginRequest{Username: "rightUser", Password: "passwrong", Metadata: "metta"}
	resp, err = suite.serv.LoginUser(suite.ctx, req)
	suite.Require().Error(err)
	suite.Require().False(resp.Success)
	suite.Require().Contains(resp.Reply, "Wrong username/password")
	// wrong user
	req = &pb.LoginRequest{Username: "leftUser", Password: "pass", Metadata: "metta"}
	resp, err = suite.serv.LoginUser(suite.ctx, req)
	suite.Require().Error(err)
	suite.Require().False(resp.Success)
	suite.Require().Contains(resp.Reply, "Wrong username/password")
}

func (suite *TstHand) Test04GreceiverText() {

	server := suite.serv

	text := "text to send to greceiver"

	firstChunk := &pb.ReceiverChunk{Filename: "fname.test", Token: suite.token, Metadata: "meta test", DataType: "text", ObjectId: 0}
	firstChunk.Content = []byte(text)

	mockStream, err := makeMockStream(suite.ctx, firstChunk)
	suite.Require().NoError(err)

	err = server.Greceiver(mockStream)
	suite.Require().NoError(err)

}
func (suite *TstHand) Test05GreceiverFile() {

	server := suite.serv
	// big file. must exist
	fileBy, err := os.ReadFile("../../cmd/client/client")
	if err != nil {
		return
	}

	firstChunk := &pb.ReceiverChunk{Filename: "client.binary", Token: suite.token, Metadata: "meta client file", DataType: "file", ObjectId: 0}
	firstChunk.Content = fileBy

	mockStream, err := makeMockStream(suite.ctx, firstChunk)
	suite.Require().NoError(err)

	err = server.Greceiver(mockStream)
	suite.Require().NoError(err)

}
func (suite *TstHand) Test06Greceiver_NoBase() {
	// save endpoint
	niceEnd := models.DBEndPoint
	models.DBEndPoint = "postgres://testuser:testpass@localhost:9000/testdb"

	server := suite.serv

	text := "wrong db endpoint"

	firstChunk := &pb.ReceiverChunk{Filename: "client.binary", Token: suite.token, Metadata: "meta client file", DataType: "file", ObjectId: 0}
	firstChunk.Content = []byte(text)

	mockStream, err := makeMockStream(suite.ctx, firstChunk)
	suite.Require().NoError(err)

	err = server.Greceiver(mockStream)
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "can't connect")

	// return endpoint
	models.DBEndPoint = niceEnd

}
func (suite *TstHand) Test07Greceiver_BadToken() {
	// save endpoint

	server := suite.serv

	text := "wrong token"

	firstChunk := &pb.ReceiverChunk{Filename: "client.binary", Token: suite.token + "baddy", Metadata: "meta client file", DataType: "file", ObjectId: 0}
	firstChunk.Content = []byte(text)

	mockStream, err := makeMockStream(suite.ctx, firstChunk)
	suite.Require().NoError(err)

	err = server.Greceiver(mockStream)
	suite.Require().Error(err)
	suite.Require().EqualError(err, "no rows in result set")

}
