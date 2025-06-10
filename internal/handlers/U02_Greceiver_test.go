package handlers

import (
	pb "gorsovet/cmd/proto"
	_ "net/http/pprof"
	"os"

	"gorsovet/internal/models"
)

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
