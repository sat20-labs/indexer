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
ord0.14.0 --rpc-url 127.0.0.1:8332 --data-dir /Users/wenjiechen/ord/0.14 --bitcoin-rpc-user jacky --bitcoin-rpc-pass _RZekaGRgKQJSIOYi6vq0_CkJtjoCootamy81J2cDn0 --first-inscription-height 767430 server -j --http --http-port 82
*/

type Inscription struct {
	Address           string `json:"address"`            // for all
	ContentLength     int    `json:"content_length"`     // for all
	ContentType       string `json:"content_type"`       // for all
	Height            int    `json:"height"`             // for ord0.23.0
	ID                string `json:"id"`                 // for ord0.23.0
	Number            int    `json:"number"`             // for ord0.23.0 ord0.9.0
	InscriptionNumber int    `json:"inscription_number"` // for ord0.14.0
	InscriptionID     string `json:"inscription_id"`     // for ord0.9.0 ord0.14.0
	GenesisHeight     int    `json:"genesis_height"`     // for ord0.9.0 ord0.14.0
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

type Tx struct {
	Chain            string `json:"chain"`
	InscriptionCount int    `json:"inscription_count"`
}

var (
	err_parse_brc20 = fmt.Errorf("parse brc20 error")
	// first brc20 inscriptin number 348020(height 779832)
	// first curse inscriptin number 53393277(height 824544)
	// first brc20 curse inscriptin number 53396631 (height 824548 )
	cur_inscriptin_number = 53393277
	brc20CurseFile        *os.File
	curseFile             *os.File
)

const (
	ord_internal_err = "Internal Server Error"
)

func GetTx(url string, tx string) (ret *Tx, err error) {
	for {
		req, _ := http.NewRequest("GET", fmt.Sprintf("http://%s/tx/%s", url, tx), nil)
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

		if string(body) == ord_internal_err {
			common.Log.Info("getTx:", ord_internal_err)
			continue
		}
		var data Tx
		err = json.Unmarshal(body, &data)
		if err == nil {
			ret = &data
		}
		break
	}

	return
}

func GetBrcContent(url string, inscriptionId string) (brc *Brc, err error) {
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
		if string(body) == ord_internal_err {
			common.Log.Info("getBrcContent:", ord_internal_err)
			continue
		}
		brc = &data
		break
	}
	return
}

func GetInscription(url string, id2number string) (ret *Inscription, err error) {
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

		if string(body) == ord_internal_err {
			common.Log.Info("getInscription:", ord_internal_err)
			continue
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

const (
	url_0_9    = "127.0.0.1:80"
	url_0_23_0 = "127.0.0.1:81"

	height_limit        = 925101
	brc20_curse_out_dir = "./cmd/brc20_curse.txt"
	curse_out_dir       = "./cmd/curse.txt"
)

func main() {
	yamlcfg := config.InitConfig("../testnet.env")
	config.InitLog(yamlcfg)

	var err error
	brc20CurseFile, err = os.OpenFile((brc20_curse_out_dir), os.O_APPEND|os.O_WRONLY|os.O_CREATE /*|os.O_TRUNC*/, 0644)
	if err != nil {
		common.Log.Fatal(err)
	}
	defer brc20CurseFile.Close()

	curseFile, err = os.OpenFile((curse_out_dir), os.O_APPEND|os.O_WRONLY|os.O_CREATE /*|os.O_TRUNC*/, 0644)
	if err != nil {
		common.Log.Fatal(err)
	}
	defer curseFile.Close()

	cur_inscriptin_number = 111639775
	err = CheckBrc20CursedInscription2(url_0_23_0, url_0_9, height_limit)
	if err != nil {
		common.Log.Fatal(err)
	}
}

func CheckBrc20CursedInscription1(lastOrdUrl, oldOrdUrl string, height_limit int) error {
	for {
		lastOrdVersionServiceInscription, err := GetInscription(lastOrdUrl, strconv.FormatInt(int64(cur_inscriptin_number), 10))
		if err != nil {
			return err
		}

		if lastOrdVersionServiceInscription.Height == 0 {
			break
		}

		if lastOrdVersionServiceInscription.Height == height_limit+1 {
			cur_inscriptin_number--
			break
		}

		txStr, index, err := common.ParseOrdInscriptionID(lastOrdVersionServiceInscription.ID)
		if err != nil {
			return err
		}
		if index != 0 {
			tx, err := GetTx(url_0_23_0, txStr)
			if err != nil {
				return err
			}
			cur_inscriptin_number = cur_inscriptin_number + tx.InscriptionCount - 1
			continue
		}
		switch lastOrdVersionServiceInscription.ContentType {
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
		case "application/json":
		case "text/plain":
		case "text/plain;charset=utf-8":
		default:
			common.Log.Infof("default skip %s: %s", lastOrdVersionServiceInscription.ContentType, lastOrdVersionServiceInscription.ID)
		}

		brc, err := GetBrcContent(lastOrdUrl, lastOrdVersionServiceInscription.ID)
		if err == err_parse_brc20 {
			cur_inscriptin_number++
			continue
		} else if err != nil {
			return err
		}

		oldVersionOrdServiceInscription, err := GetInscription(oldOrdUrl, lastOrdVersionServiceInscription.ID)
		if err != nil {
			format := "not find in old ord, id:%s，num:%d, tick:%s, op:%s"
			printStr := fmt.Sprintf(format, lastOrdVersionServiceInscription.ID, lastOrdVersionServiceInscription.Number, brc.Ticker, brc.Op)
			_, err = brc20CurseFile.Write([]byte(printStr + "\n"))
			if err != nil {
				return err
			}
			cur_inscriptin_number++
			common.Log.Infof(printStr)
			continue
		}

		format := "height:%d, id:%s, num:%d, 09Num:%d, brc:%s"
		printStr := fmt.Sprintf(format, lastOrdVersionServiceInscription.Height, lastOrdVersionServiceInscription.ID,
			lastOrdVersionServiceInscription.Number, oldVersionOrdServiceInscription.Number, brc.Ticker)
		common.Log.Infof(printStr)

		if oldVersionOrdServiceInscription.Number < 0 {
			format := "curse, height:%d, id:%s，num:%d, address:%s, 09Num:%d, tick:%s, op:%s"
			printStr := fmt.Sprintf(format, lastOrdVersionServiceInscription.Height, lastOrdVersionServiceInscription.ID, lastOrdVersionServiceInscription.Number,
				lastOrdVersionServiceInscription.Address, oldVersionOrdServiceInscription.Number, brc.Ticker, brc.Op)
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

func CheckBrc20CursedInscription2(lastOrdUrl, oldOrdUrl string, height_limit int) error {
	for {
		newOrdInscription, err := GetInscription(lastOrdUrl, strconv.FormatInt(int64(cur_inscriptin_number), 10))
		if err != nil {
			return err
		}

		if newOrdInscription.Height == 0 {
			break
		}

		if newOrdInscription.Height == height_limit+1 {
			cur_inscriptin_number--
			break
		}

		_, index, err := common.ParseOrdInscriptionID(newOrdInscription.ID)
		if err != nil {
			return err
		}

		oldOrdInscription, oldOrdInscriptionErr := GetInscription(oldOrdUrl, newOrdInscription.ID)
		if oldOrdInscriptionErr != nil {
			format := "not find in old ord, height:%d, id:%s，num:%d"
			lineStr := fmt.Sprintf(format, newOrdInscription.Height, newOrdInscription.ID, newOrdInscription.Number)
			_, err = curseFile.Write([]byte(lineStr + "\n"))
			if err != nil {
				return err
			}
			common.Log.Infof(lineStr)
			if index != 0 {
				cur_inscriptin_number++
				continue
			}
		} else if oldOrdInscription.Number < 0 {
			format := "curse, height:%d, id:%s，num:%d, address:%s, 09Num:%d"
			lineStr := fmt.Sprintf(format, newOrdInscription.Height, newOrdInscription.ID, newOrdInscription.Number,
				newOrdInscription.Address, oldOrdInscription.Number)
			_, err = curseFile.Write([]byte(lineStr + "\n"))
			if err != nil {
				return err
			}
			common.Log.Infof(lineStr)
			if index != 0 {
				cur_inscriptin_number++
				continue
			}
		}

		if oldOrdInscriptionErr == nil && oldOrdInscription.Number >= 0 {
			format := "height:%d, id:%s, num:%d, 09Num:%d"
			printStr := fmt.Sprintf(format, newOrdInscription.Height, newOrdInscription.ID,
				newOrdInscription.Number, oldOrdInscription.Number)
			common.Log.Infof(printStr)
			cur_inscriptin_number++
			continue
		}

		switch newOrdInscription.ContentType {
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
			common.Log.Infof("skip %s: %s", newOrdInscription.ContentType, newOrdInscription.ID)
			cur_inscriptin_number++
			continue
		case "application/json":
		case "text/plain":
		case "text/plain;charset=utf-8":
		default:
			common.Log.Infof("default skip %s: %s", newOrdInscription.ContentType, newOrdInscription.ID)
		}

		brc, err := GetBrcContent(lastOrdUrl, newOrdInscription.ID)
		if err == err_parse_brc20 {
			cur_inscriptin_number++
			continue
		} else if err != nil {
			return err
		}

		if oldOrdInscriptionErr != nil {
			format := "not find in old ord, height:%d, id:%s，num:%d, tick:%s, op:%s"
			lineStr := fmt.Sprintf(format, newOrdInscription.Height, newOrdInscription.ID, newOrdInscription.Number, brc.Ticker, brc.Op)
			_, err = brc20CurseFile.Write([]byte(lineStr + "\n"))
			if err != nil {
				return err
			}
			cur_inscriptin_number++
			common.Log.Infof(lineStr)
			continue
		}

		format := "height:%d, id:%s, num:%d, 09Num:%d, brc:%s"
		printStr := fmt.Sprintf(format, newOrdInscription.Height, newOrdInscription.ID,
			newOrdInscription.Number, oldOrdInscription.Number, brc.Ticker)
		common.Log.Infof(printStr)

		if oldOrdInscription.Number < 0 {
			format := "curse, height:%d, id:%s，num:%d, address:%s, 09Num:%d, tick:%s, op:%s"
			lineStr := fmt.Sprintf(format, newOrdInscription.Height, newOrdInscription.ID, newOrdInscription.Number,
				newOrdInscription.Address, oldOrdInscription.Number, brc.Ticker, brc.Op)
			_, err = brc20CurseFile.Write([]byte(lineStr + "\n"))
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
