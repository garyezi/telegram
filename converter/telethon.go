package converter

import (
	"context"
	"database/sql"
	"github.com/gotd/td/crypto"
	"github.com/gotd/td/session"
	_ "github.com/mattn/go-sqlite3"
	"os"
	"strconv"
)

const UseCurrentSession = 1
const CreateNewSession = 2

type telethonSession struct {
	DcId          int    `json:"dc_id"`
	ServerAddress string `json:"server_address"`
	Port          int    `json:"port"`
	AuthKey       []byte `json:"auth_key"`
	TakeoutId     *int   `json:"takeout_id"`
}

func FormTelethon(filePath string, storage session.Storage) error {
	// 打开 SQLite3 数据库文件
	db, err := sql.Open("sqlite3", filePath)
	if err != nil {
		return err
	}
	defer db.Close()
	// 查询数据
	rows, err := db.Query("SELECT * FROM sessions LIMIT 1")
	if err != nil {
		return err
	}
	defer rows.Close()
	tSession := &telethonSession{}
	for rows.Next() {
		err = rows.Scan(&tSession.DcId, &tSession.ServerAddress, &tSession.Port, &tSession.AuthKey, &tSession.TakeoutId)
		if err != nil {
			return err
		}
	}
	loader := session.Loader{
		Storage: storage,
	}
	var authKey crypto.Key
	copy(authKey[:], tSession.AuthKey)
	authKeyId := authKey.ID()
	err = loader.Save(context.Background(), &session.Data{
		Config:    session.Config{},
		DC:        tSession.DcId,
		Addr:      tSession.ServerAddress + ":" + strconv.Itoa(tSession.Port),
		AuthKey:   authKey[:],
		AuthKeyID: authKeyId[:],
	})
	return err
}

func ToTelethon(storage session.Storage, filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	return nil
}
