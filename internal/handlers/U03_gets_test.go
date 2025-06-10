package handlers

import (
	pb "gorsovet/cmd/proto"
	_ "net/http/pprof"
)

func (suite *TstHand) Test07List() {
	// Создаем тестовый запрос
	req := &pb.ListObjectsRequest{Token: suite.token}
	resp, err := suite.serv.ListObjects(suite.ctx, req)
	suite.Require().NoError(err)
	suite.Require().True(resp.Success)

	list := resp.Listing
	// 2 успешных закачки до этого, текст и файл
	suite.Require().EqualValues(2, len(list))

	// bad token
	req = &pb.ListObjectsRequest{Token: suite.token + "badd"}
	resp, err = suite.serv.ListObjects(suite.ctx, req)
	suite.Require().Error(err)
	suite.Require().False(resp.Success)
}
func (suite *TstHand) Test08_Remove() {
	// запрос на удаление объекта номер 1
	req := &pb.RemoveObjectsRequest{Token: suite.token, ObjectId: 1}
	resp, err := suite.serv.RemoveObjects(suite.ctx, req)
	suite.Require().NoError(err)
	suite.Require().True(resp.Success)
	suite.Require().Equal("OK remove object", resp.Reply)

	// запрос на удаление несуществующего объекта номер 7
	req = &pb.RemoveObjectsRequest{Token: suite.token, ObjectId: 7}
	resp, err = suite.serv.RemoveObjects(suite.ctx, req)
	suite.Require().Error(err)
	suite.Require().False(resp.Success)
	suite.Require().Equal("bad RemoveObjects", resp.Reply)

	reqL := &pb.ListObjectsRequest{Token: suite.token}
	respL, err := suite.serv.ListObjects(suite.ctx, reqL)
	suite.Require().NoError(err)
	suite.Require().True(respL.Success)

	list := respL.Listing
	// 2 успешных закачки до этого, текст и файл, удаление первого номера, 1 in the rest should be
	suite.Require().EqualValues(1, len(list))
}
