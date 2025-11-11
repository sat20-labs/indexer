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

/*
ord0.9 --rpc-url 127.0.0.1:8332 --data-dir /Users/wenjiechen/ord/0.9 --bitcoin-rpc-user jacky --bitcoin-rpc-pass _RZekaGRgKQJSIOYi6vq0_CkJtjoCootamy81J2cDn0 --first-inscription-height 767430 --height-limit 824545 -e server --http
ord0.23.3  --bitcoin-rpc-url 127.0.0.1:8332 --data-dir /Users/wenjiechen/ord/0.23.3 --bitcoin-rpc-username jacky --bitcoin-rpc-password _RZekaGRgKQJSIOYi6vq0_CkJtjoCootamy81J2cDn0  server --http --http-port 81
ord0.14.0 --rpc-url 127.0.0.1:8332 --data-dir /Users/wenjiechen/ord/0.14 --bitcoin-rpc-user jacky --bitcoin-rpc-pass _RZekaGRgKQJSIOYi6vq0_CkJtjoCootamy81J2cDn0 --first-inscription-height 767430 server --http --http-port 82
*/

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
	P      string `json:"p"`
	Op     string `json:"op"`
	Ticker string `json:"tick"`
}

type Store struct {
	BrcTick       string
	InscriptionId string
	Content       string
}

var (
	err_parse_brc20       = fmt.Errorf("parse brc20 error")
	cur_inscriptin_number = 348020 // first brc inscriptin_number is 348020(height 779832), end inscriptin number = 110020899(height 923108)
	brc20CurseFile        *os.File
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
		var deployContent *common.BRC20DeployContent
		var mintContent *common.BRC20MintContent
		var transferContent *common.BRC20TransferContent
		deployContent = common.ParseBrc20DeployContent(string(body))
		if deployContent != nil {
			data.P = deployContent.P
			data.Op = deployContent.Op
			data.Ticker = deployContent.Ticker
		} else {
			mintContent = common.ParseBrc20MintContent(string(body))
			if mintContent != nil {
				data.P = mintContent.P
				data.Op = mintContent.Op
				data.Ticker = mintContent.Ticker
			} else {
				transferContent = common.ParseBrc20TransferContent(string(body))
				if transferContent != nil {
					data.P = transferContent.P
					data.Op = transferContent.Op
					data.Ticker = transferContent.Ticker
				}
			}
		}

		// err = json.Unmarshal(body, &data)
		if deployContent == nil && mintContent == nil && transferContent == nil {
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

	const (
		url_0_9    = "127.0.0.1:80"
		url_0_14   = "127.0.0.1:82"
		url_0_23_0 = "127.0.0.1:81"

		height_limit_1 = 824544
		height_limit_2 = 923108
		curse_out_dir  = "./cmd/brc20_curse.txt"
	)

	var err error
	brc20CurseFile, err = os.OpenFile((curse_out_dir), os.O_APPEND|os.O_WRONLY|os.O_CREATE /*|os.O_TRUNC*/, 0644)
	if err != nil {
		common.Log.Fatal(err)
	}
	defer brc20CurseFile.Close()

	// 760000(767430)-816000，使用ord0.9版本的数据，诅咒铭文无效 (最新版本与ord0.9对比)
	// 816001-824544，使用ord0.9版本的数据，诅咒铭文无效  (最新版本与ord0.9对比)
	err = CheckInscriptionId(url_0_23_0, url_0_9, height_limit_1)
	if err != nil {
		common.Log.Fatal(err)
	}

	// 824545-latest，Jubilee 生效，有效铭文定义采用ord0.14版本，理论上，其定义应该和最新版本一致  (最新版本与ord0.14对比)
	err = CheckInscriptionId(url_0_23_0, url_0_14, height_limit_2)
	if err != nil {
		common.Log.Fatal(err)
	}
}

func CheckInscriptionId(lastOrdUrl, oldOrdUrl string, height_limit int) error {
	for {
		lastOrdVersionServiceInscription, err := getInscription(lastOrdUrl, strconv.FormatInt(int64(cur_inscriptin_number), 10))
		if err != nil {
			common.Log.Info(err)
			return err
		}

		if lastOrdVersionServiceInscription.Height == 0 {
			break
		}

		if lastOrdVersionServiceInscription.Height == height_limit+1 {
			cur_inscriptin_number--
			break
		}

		_, index, err := common.ParseOrdInscriptionID(lastOrdVersionServiceInscription.ID)
		if err != nil {
			return err
		}
		if index != 0 {
			cur_inscriptin_number++
			continue
		}
		switch lastOrdVersionServiceInscription.ContentType {
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
		case "model/gltf+json":
			fallthrough
		case "model/gltf-binary":
			fallthrough
		case "application/pdf":
			common.Log.Infof("skip %s: %s", lastOrdVersionServiceInscription.ContentType, lastOrdVersionServiceInscription.ID)
			cur_inscriptin_number++
			continue
		case "text/plain;charset=utf-8":
		default:
			common.Log.Infof("default skip %s: %s", lastOrdVersionServiceInscription.ContentType, lastOrdVersionServiceInscription.ID)
		}

		_, brc, err := getBrcContent(lastOrdUrl, lastOrdVersionServiceInscription.ID)
		if err == err_parse_brc20 {
			cur_inscriptin_number++
			continue
		} else if err != nil {
			return err
		}

		oldVersionOrdServiceInscription, err := getInscription(oldOrdUrl, lastOrdVersionServiceInscription.ID)
		if err != nil {
			format := "not find in 09, id:%s，num:%d, tick:%s, op:%s"
			printStr := fmt.Sprintf(format, lastOrdVersionServiceInscription.ID, lastOrdVersionServiceInscription.Number, brc.Ticker, brc.Op)
			_, err = brc20CurseFile.Write([]byte(printStr + "\n"))
			if err != nil {
				return err
			}
			cur_inscriptin_number++
			common.Log.Infof(printStr)
			continue
		}

		format := "id:%s，num:%d, 09Num:%d, brc:%s"
		printStr := fmt.Sprintf(format, lastOrdVersionServiceInscription.ID, lastOrdVersionServiceInscription.Number, oldVersionOrdServiceInscription.Number, brc.Ticker)
		common.Log.Infof(printStr)

		if oldVersionOrdServiceInscription.Number < 0 {
			format := "curse, id:%s，num:%d, address:%s, 09Num:%d, tick:%s, op:%s"
			printStr := fmt.Sprintf(format, lastOrdVersionServiceInscription.ID, lastOrdVersionServiceInscription.Number, lastOrdVersionServiceInscription.Address, oldVersionOrdServiceInscription.Number, brc.Ticker, brc.Op)
			_, err = brc20CurseFile.Write([]byte(printStr + "\n"))
			if err != nil {
				return err
			}
			cur_inscriptin_number++
			continue
		}
		cur_inscriptin_number++

	}

	return nil
}
