package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/config"
)

type Inscription struct {
	Address       string `json:"address"`
	ContentLength int    `json:"content_length"`
	ContentType   string `json:"content_type"`
	Height        int    `json:"height"`
	ID            string `json:"id"`
	Number        int    `json:"number"`
	InscriptionID string `json:"inscription_id"`
	GenesisHeight int    `json:"genesis_height"`
}

type Brc struct {
	P    string `json:"p"`
	Op   string `json:"op"`
	Tick string `json:"tick"`
}

type Store struct {
	BrcTick       string
	InscriptionId string
	Content       string
}

const (
	url_0_23_0 = "127.0.0.1:81"
	url_0_9    = "127.0.0.1:80"

	start_inscriptin_number = 27544158 //348020
	// end_inscriptin_number   = 66796146
	end_height = 837090
)

var (
	err_parse_brc20 = fmt.Errorf("parse brc20 error")
)

func getInscription(url string, id2number string) (ret *Inscription, err error) {
	for {
		req, _ := http.NewRequest("GET", fmt.Sprintf("http://%s/inscription/%s", url, id2number), nil)
		req.Header.Set("Accept", "application/json")
		client := &http.Client{}
		resp, cerr := client.Do(req)
		if cerr != nil {

			common.Log.Info(cerr)
			continue
		}
		defer resp.Body.Close()

		var body []byte
		body, err = io.ReadAll(resp.Body)
		if err != nil {
			common.Log.Info(err)
			break
		}

		var data Inscription
		err = json.Unmarshal(body, &data)
		if err == nil {
			ret = &data
		}
		break
	}

	return
}

func getBrcContent(url string, inscriptionId string) (ret string, brc *Brc, err error) {
	for {
		req, _ := http.NewRequest("GET", fmt.Sprintf("http://%s/content/%s", url, inscriptionId), nil)
		client := &http.Client{}
		resp, cerr := client.Do(req)
		if cerr != nil {
			continue
		}
		defer resp.Body.Close()

		var body []byte
		body, err = io.ReadAll(resp.Body)
		if err != nil {
			break
		}

		var data Brc
		err = json.Unmarshal(body, &data)
		if err != nil {
			err = err_parse_brc20
		}
		if data.P != "brc-20" {
			err = err_parse_brc20
		}
		ret = string(body)
		brc = &data
		break
	}
	return
}

func main() {
	yamlcfg := config.InitConfig("../testnet.env")
	config.InitLog(yamlcfg)
	err := CheckInscriptionId()
	if err != nil {
		common.Log.Info(err)
	}

}

func CheckInscriptionId() error {
	file, err := os.OpenFile("file.txt", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	inscriptionNum := start_inscriptin_number
	for {
		inscription_0230, err := getInscription(url_0_23_0, strconv.FormatInt(int64(inscriptionNum), 10))
		if err != nil {
			common.Log.Info(err)
			return err
		}

		if inscription_0230.Height == end_height+1 {
			break
		}

		switch inscription_0230.ContentType {
		case "application/json":
		case "text/plain":
		case "text/html;charset=utf-8":
			fallthrough
		case "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
			fallthrough
		case "image/webp":
			fallthrough
		case "image/svg+xml":
			fallthrough
		case "image/jpeg":
			fallthrough
		case "image/png":
			fallthrough
		case "video/webm":
			fallthrough
		case "video/mp4":
			fallthrough
		case "image/gif":
			fallthrough
		case "audio/mpeg":
			fallthrough
		case "audio/wav":
			fallthrough
		case "text/markdown;charset=utf-8":
			fallthrough
		case "text/javascript":
			fallthrough
		case "image/avif":
			fallthrough
		case "model/gltf-binary":
			fallthrough
		case "application/pdf":
			common.Log.Infof("skip %s: %s", inscription_0230.ContentType, inscription_0230.ID)
			inscriptionNum++
			continue
		case "text/plain;charset=utf-8":
		default:
			common.Log.Infof("default skip %s: %s", inscription_0230.ContentType, inscription_0230.ID)
		}

		content_0230, brc, err := getBrcContent(url_0_23_0, inscription_0230.ID)
		if err == err_parse_brc20 {
			inscriptionNum++
			continue
		} else if err != nil {
			return err
		}

		inscription_09, err := getInscription(url_0_9, inscription_0230.ID)
		if err != nil {
			return err
		}
		common.Log.Infof("num:%+v, 0230id:%+v，09id:%+v, brc:%+v", inscriptionNum, inscription_0230.ID, inscription_09.InscriptionID, brc.Tick)

		content_09, _, err := getBrcContent(url_0_9, inscription_0230.ID)
		if err == err_parse_brc20 {
			_, err = file.Write([]byte(inscription_0230.ID + "\n"))
			if err != nil {
				return err
			}
			inscriptionNum++
			continue
		} else if err != nil {
			return err
		}

		if content_0230 != content_09 {
			_, err = file.Write([]byte(inscription_0230.ID + "\n"))
			if err != nil {
				return err
			}
			inscriptionNum++
			continue
		}
		inscriptionNum++

	}

	return nil
}
