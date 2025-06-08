package main

import (
	"encoding/json"
	"fmt"
	"gorsovet/internal/localbase"
	"gorsovet/internal/models"
	"os"
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
	by, err := localbase.GetRecordHead(*localsql, showFlag)
	if err != nil {
		fmt.Printf("no record with number %d\n", showFlag)
		return fmt.Errorf("no record with number %d", showFlag)
	}
	fnam := models.LocalS3Dir + "/" + strings.ToLower(by.GetFilekey()) + "/" + by.GetFileurl()
	fcontent, err := os.ReadFile(fnam)
	if err != nil {
		return
	}

	fmt.Printf("file:\t%s\nmeta:\t%s\ntype:\t%s\ncreated:\t%s\n",
		by.GetFileurl(), by.GetMetadata(), by.GetDataType(), by.GetCreatedAt().AsTime().Format(time.RFC3339))

	if by.GetDataType() == "text" {
		fmt.Println("_____________________________________________________________________________ CONTENT __")
		fmt.Println(string(fcontent))
	}
	if by.GetDataType() == "card" {
		fmt.Println("_____________________________________________________________________________ CARD __")
		card := models.Carda{}
		err := json.Unmarshal(fcontent, &card)
		if err != nil {
			return err
		}
		fmt.Printf("Number     %20d\nExpiration %20s\nCSV        %20s\nHolder     %20s\n", card.Number, card.Expiration, card.CSV, card.Holder)
	}

	return
}

func getFileFlagLocal(getFileFlag int32) (err error) {
	return
}
