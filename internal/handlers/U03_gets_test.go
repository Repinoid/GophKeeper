package handlers

import (
	pb "gorsovet/cmd/proto"
	_ "net/http/pprof"
)

func (suite *TstHand) Test07List() {
	// Создаем тестовый запрос
	req := &pb.RegisterRequest{Username: "rightUser", Password: "pass", Metadata: "metta"}
	// Вызываем метод
	resp, err := suite.serv.RegisterUser(suite.ctx, req)
	suite.Require().NoError(err)
	suite.Require().True(resp.Success)

}
