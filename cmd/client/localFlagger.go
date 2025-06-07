package main

import (
	"fmt"
	"gorsovet/internal/localbase"
	"gorsovet/internal/models"
	"strings"
	"time"
)

func loginFlagLocal(loginFlag string) (err error) {
	return
}
func listFlagLocal() (err error) {
	list, err := localbase.GetList(*localsql, strings.ToUpper(currentUser))
	if err != nil {
		models.Sugar.Debugf("GetList %v", err)
		return err
	}
	fmt.Printf("%10s\t%20s\t%10s\t%15s\t%20s\t%s\n", "ID", "File URL", "Data type", "file size", "created", "metadata")

	for _, v := range list {
		fmt.Printf("%10d\t%20s\t%10s\t%15d\t%20s\t%s\n", v.GetId(), v.GetFileurl(), v.GetDataType(), v.GetSize(),
			(v.GetCreatedAt()).AsTime().Format(time.RFC3339), v.GetMetadata())
	}
	return
}

func showFlagLocal(showFlag int32) (err error) {
	return
}

func getFileFlagLocal(getFileFlag int32) (err error) {
	return
}
