package handlers

import (
	_ "net/http/pprof"
	"os"

	"gorsovet/internal/models"

	"google.golang.org/grpc/metadata"
)

func (suite *TstHand) Test04GreceiverText() {

	server := suite.serv
	text := []byte("text to send to greceiver")
	md := metadata.Pairs(
		"Token", suite.token,
		"MetaData", "metta lurg",
		"DataType", "text",
		"ObjectId", "0",
		"FileName", "client.txt",
	)
	ctx := metadata.NewOutgoingContext(suite.ctx, md)

	mockStream, err := makeMockStream(ctx, text)
	suite.Require().NoError(err)

	//mockStream

	err = server.Greceiver(mockStream)
	suite.Require().NoError(err)
	// успешный засыл текстовой записи
}
func (suite *TstHand) Test05GreceiverFile() {

	server := suite.serv
	// big file. must exist
	fileBy, err := os.ReadFile("../../cmd/client/client")
	if err != nil {
		return
	}
	md := metadata.Pairs(
		"Token", suite.token,
		"MetaData", "metta lurg",
		"DataType", "file",
		"ObjectId", "0",
		"FileName", "client.binary",
	)
	suite.ctx = metadata.NewOutgoingContext(suite.ctx, md)
	mockStream, err := makeMockStream(suite.ctx, fileBy)
	suite.Require().NoError(err)

	err = server.Greceiver(mockStream)
	suite.Require().NoError(err)
	// успешный засыл записи - файла. теперь 2 записи в базах и Minio
}
func (suite *TstHand) Test06Greceiver_NoBase() {
	// save endpoint
	niceEnd := models.DBEndPoint
	models.DBEndPoint = "postgres://testuser:testpass@localhost:9000/testdb"

	server := suite.serv

	text := []byte("wrong db endpoint")

	md := metadata.Pairs(
		"Token", suite.token,
		"MetaData", "metta lurg",
		"DataType", "text",
		"ObjectId", "0",
		"FileName", "doesnot.matter",
	)
	suite.ctx = metadata.NewOutgoingContext(suite.ctx, md)

	mockStream, err := makeMockStream(suite.ctx, text)
	suite.Require().NoError(err)

	err = server.Greceiver(mockStream)
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "can't connect")

	// return endpoint
	models.DBEndPoint = niceEnd
	// ничего не изменилось в базе

}
func (suite *TstHand) Test07Greceiver_BadToken() {
	// save endpoint

	// server := suite.serv
	// text := "wrong token"

	server := suite.serv
	text := []byte("wrong token")
	md := metadata.Pairs(
		"Token", suite.token+"baddy",
		"MetaData", "metta lurg",
		"DataType", "text",
		"ObjectId", "0",
		"FileName", "ugly.bad",
	)
	suite.ctx = metadata.NewOutgoingContext(suite.ctx, md)
	mockStream, err := makeMockStream(suite.ctx, text)
	suite.Require().NoError(err)

	err = server.Greceiver(mockStream)
	suite.Require().Error(err)

	suite.Require().EqualError(err, "no rows in result set")
}
