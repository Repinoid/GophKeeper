package handlers

import (
	"context"
	pb "gorsovet/cmd/proto"
	"io"
	_ "net/http/pprof"
	"strings"

	"gorsovet/internal/dbase"
	"gorsovet/internal/models"

	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/metadata"
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

func (suite *TstHand) Test04Greceiver() {
	//	tlsCreds, err := privacy.LoadClientTLSCredentials("../../cmd/tls/public.crt")
	//	suite.Require().NoError(err)
	//	conn, err := grpc.NewClient(models.Gport, grpc.WithTransportCredentials(tlsCreds))
	//	suite.Require().NoError(err)

	text := "text to send to greceiver"
	reader := strings.NewReader(text)
	buffer := make([]byte, 64*1024) // 64KB chunks
	// Send first chunk with filename
	firstChunk := &pb.ReceiverChunk{Filename: "fname.test", Token: suite.token, Metadata: "meta test", DataType: "text", ObjectId: 0}
	n, err := reader.Read(buffer)
	if err != nil && err != io.EOF {
		return
	}
	firstChunk.Content = buffer[:n]

	msgs := make([]*pb.ReceiverChunk, 1)
	msgs[0] = firstChunk

	// Создаем mock stream
	mockStream := &MockClientStream{
		Ctx:      context.Background(),
		recvMsgs: msgs,
		//	RecvMsg:  &pb.ReceiverResponse{Success: true},
	//	HeaderMD: metadata.New(map[string]string{"header-key": "value"}),
	}

	server := suite.serv
	err = server.Greceiver(mockStream)
	suite.Require().NoError(err)
}

// MockClientStream реализует grpc.ClientStream для тестирования
type MockClientStream struct {
	//HeaderMD  metadata.MD
	//TrailerMD metadata.MD

	mock.Mock
	Ctx         context.Context
	recvMsgs    []*pb.ReceiverChunk
	currentRecv int
}

func (m *MockClientStream) Context() context.Context {
	return m.Ctx
}

func (m *MockClientStream) SendMsg(msg interface{}) error {
	return nil
}

func (m *MockClientStream) RecvMsg(msg interface{}) error {
	return nil
}

func (m *MockClientStream) SendAndClose(a *pb.ReceiverResponse) error {
	return nil
}

func (m *MockClientStream) SendHeader(metadata.MD) error {
	return nil
}

func (m *MockClientStream) SetHeader(metadata.MD) error {
	return nil
}

func (m *MockClientStream) SetTrailer(metadata.MD) {
	//return //m.TrailerMD
}
func (m *MockClientStream) Recv() (a *pb.ReceiverChunk, err error) {
	if m.currentRecv >= len(m.recvMsgs) {
		return nil, io.EOF
	}
	msg := m.recvMsgs[m.currentRecv]
	m.currentRecv++
	return msg, nil
}
