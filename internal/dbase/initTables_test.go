package dbase

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConnectToDB(t *testing.T) {

	dbEndPoint := "postgres://userp:parole@localhost:5432/dbaza"

	db, err := ConnectToDB(context.Background(), dbEndPoint)
	require.NoError(t, err)

	_ = db

}
