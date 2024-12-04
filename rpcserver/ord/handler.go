package ord

import (
	"embed"
	"fmt"
	"net/http"
	"text/template"

	"github.com/gin-gonic/gin"
	"github.com/sat20-labs/indexer/share/base_indexer"
)

//go:embed templates/*
var templatesRes embed.FS

// @Summary get ordinal status
// @Description get ordinal status
// @Tags ordx.ord
// @Produce json
// @Security Bearer
// @Success 200 {object} []byte "Successful response"
// @Failure 401 "Invalid API Key"
// @Router /ord/status [get]
// func (s *Service) getOrdinalStatus(c *gin.Context) {
// 	ordinalStatusResp, err := ord_rpc.ShareOrdinalsRpc.GetStatus()
// 	if err != nil {
// 		c.JSON(http.StatusOK, err.Error())
// 		return
// 	}
// 	c.Data(http.StatusOK, CONTENT_TYPE_JSON, ordinalStatusResp)
// }

// @Summary get ordinal content 一个字符串，判断这个字符串每个字符
// @Description get ordinal content
// @Tags ordx.ord
// @Produce image/*
// @Param inscriptionid path string true "inscription ID" example:"5ea0f691bc818afd978e26d308f56ac2bdadfe6c403f53f73872aaa0cef55fd1i0"
// @Security Bearer
// @Success 200 {object} []byte "Successful response"
// @Failure 401 "Invalid API Key"
// @Router /ord/content/{inscriptionid} [get]
func (s *Service) getInscriptionContent(c *gin.Context) {
	inscriptionId := c.Param("inscriptionid")
	err := checkInscriptionId(inscriptionId)
	if err != nil {
		c.Data(http.StatusBadRequest, CONTEXT_TYPE_TEXT, []byte(err.Error()))
		return
	}
	nft := base_indexer.ShareBaseIndexer.GetNftInfoWithInscriptionId(inscriptionId)
	if nft == nil {
		c.Data(http.StatusNotFound, CONTEXT_TYPE_TEXT, []byte(fmt.Sprintf(`%s not found`, inscriptionId)))
		return
	}
	if nft.Base.Delegate != "" {
		nft = base_indexer.ShareBaseIndexer.GetNftInfoWithInscriptionId(nft.Base.Delegate)
	}
	if nft == nil {
		c.Data(http.StatusNotFound, CONTEXT_TYPE_TEXT, []byte(fmt.Sprintf(`delegate %s not found`, inscriptionId)))
		return
	}

	contentType, err := genContentHeader(c, nft)
	if err != nil {
		c.Data(http.StatusInternalServerError, CONTEXT_TYPE_TEXT, []byte(err.Error()))
		return
	}
	c.Data(http.StatusOK, contentType, nft.Base.Content)
}

// @Summary get ordinal preview
// @Description get ordinal preview
// @Tags ordx.ord
// @Produce image/*
// @Param inscriptionid path string true "inscription ID" example:"5ea0f691bc818afd978e26d308f56ac2bdadfe6c403f53f73872aaa0cef55fd1i0"
// @Security Bearer
// @Success 200 {object} []byte "Successful response"
// @Failure 401 "Invalid API Key"
// @Router /ord/preview/{inscriptionid} [get]
func (s *Service) getInscriptionPreview(c *gin.Context) {
	inscriptionId := c.Param("inscriptionid")
	err := checkInscriptionId(inscriptionId)
	if err != nil {
		c.Data(http.StatusBadRequest, CONTEXT_TYPE_TEXT, []byte(err.Error()))
		return
	}
	nft := base_indexer.ShareBaseIndexer.GetNftInfoWithInscriptionId(inscriptionId)
	if nft == nil {
		c.Data(http.StatusNotFound, CONTEXT_TYPE_TEXT, []byte(fmt.Sprintf(`%s not found`, inscriptionId)))
		return
	}
	if nft.Base.Delegate != "" {
		nft = base_indexer.ShareBaseIndexer.GetNftInfoWithInscriptionId(nft.Base.Delegate)
	}
	if nft == nil {
		c.Data(http.StatusNotFound, CONTEXT_TYPE_TEXT, []byte(fmt.Sprintf(`delegate %s not found`, inscriptionId)))
		return
	}

	type TemplateData struct {
		InscriptionID string
	}
	var data any

	mediaProperty := ""
	if nft.Base.ContentType != nil {
		media := MediaList[string(nft.Base.ContentType)]
		if media != nil {
			mediaProperty = media.Property
		}
	}

	templateFile := ""
	mediaType := getMediaType(nft)
	switch mediaType {
	case Audio:
		templateFile = "templates/preview-audio.html"
	case Code:
		c.Writer.Header().Set(CONTENT_SECURITY_POLICY, "script-src-elem 'self' https://cdn.jsdelivr.net")
		templateFile = "templates/preview-code.html"
		type CodeTemplateData struct {
			TemplateData
			Language string
		}
		data = CodeTemplateData{
			TemplateData: TemplateData{
				InscriptionID: nft.Base.InscriptionId,
			},
			Language: mediaProperty,
		}
	case Font:
		templateFile = "templates/preview-font.html"
		c.Writer.Header().Set(CONTENT_SECURITY_POLICY, "script-src-elem 'self'; style-src 'self' 'unsafe-inline';")
		data = TemplateData{
			InscriptionID: nft.Base.InscriptionId,
		}
	case Iframe:

		contentType, err := genContentHeader(c, nft)
		if err != nil {
			c.Data(http.StatusInternalServerError, CONTEXT_TYPE_TEXT, []byte(err.Error()))
			return
		}
		c.Data(http.StatusOK, contentType, nft.Base.Content)
		return
	case Image:
		templateFile = "templates/preview-image.html"
		c.Writer.Header().Set(CONTENT_SECURITY_POLICY, "default-src 'self' 'unsafe-inline'")
		type ImageTemplateData struct {
			TemplateData
			ImageRendering string
		}
		data = ImageTemplateData{
			TemplateData: TemplateData{
				InscriptionID: nft.Base.InscriptionId,
			},
			ImageRendering: mediaProperty,
		}
	case Markdown:
		templateFile = "templates/preview-markdown.html"
		c.Writer.Header().Set(CONTENT_SECURITY_POLICY, "script-src-elem 'self' https://cdn.jsdelivr.net")
		data = TemplateData{
			InscriptionID: nft.Base.InscriptionId,
		}
	case Model:
		templateFile = "templates/preview-model.html"
		c.Writer.Header().Set(CONTENT_SECURITY_POLICY, "script-src-elem 'self' https://ajax.googleapis.com")
		data = TemplateData{
			InscriptionID: nft.Base.InscriptionId,
		}
	case Pdf:
		templateFile = "templates/preview-pdf.html"
		c.Writer.Header().Set(CONTENT_SECURITY_POLICY, "script-src-elem 'self' https://cdn.jsdelivr.net")
		data = TemplateData{
			InscriptionID: nft.Base.InscriptionId,
		}
	case Text:
		templateFile = "templates/preview-text.html"
		data = TemplateData{
			InscriptionID: nft.Base.InscriptionId,
		}
	case Unknown:
		templateFile = "templates/preview-unknown.html"
		data = TemplateData{
			InscriptionID: nft.Base.InscriptionId,
		}
	case Video:
		templateFile = "templates/preview-video.html"
		data = TemplateData{
			InscriptionID: nft.Base.InscriptionId,
		}
	}

	t, err := template.ParseFS(templatesRes, templateFile)
	if err != nil {
		c.Data(http.StatusInternalServerError, CONTEXT_TYPE_TEXT, []byte(err.Error()))
		return
	}

	err = t.Execute(c.Writer, data)
	if err != nil {
		c.Data(http.StatusInternalServerError, CONTEXT_TYPE_TEXT, []byte(err.Error()))
		return
	}
}

// @Summary ordinal recursive endpoint for get block hash
// @Description ordinal recursive endpoint for get block hash
// @Tags ordx.ord.r
// @Produce json
// @Param height path uint64 true "height"
// @Security Bearer
// @Success 200 {object} []byte "Successful response"
// @Failure 401 "Invalid API Key"
// @Router /ord/r/blockhash/{height} [get]
// func (s *Service) getRBlockHash(c *gin.Context) {
// 	height := c.Param("height")
// 	blockHash, err := ord_rpc.ShareOrdinalsRpc.GetRBlockHash(height)
// 	if err != nil {
// 		c.JSON(http.StatusOK, err.Error())
// 		return
// 	}
// 	c.Data(http.StatusOK, CONTENT_TYPE_JSON, blockHash)
// }

// @Summary ordinal recursive endpoint for get lastest block hash
// @Description ordinal recursive endpoint for get lastest block hash
// @Tags ordx.ord.r
// @Produce json
// @Security Bearer
// @Success 200 {object} string "Successful response"
// @Failure 401 "Invalid API Key"
// @Router /ord/r/blockhash [get]
// func (s *Service) getRLastestBlockHash(c *gin.Context) {
// 	blockHash, err := ord_rpc.ShareOrdinalsRpc.GetRLastestBlockHash()
// 	if err != nil {
// 		c.JSON(http.StatusOK, err.Error())
// 		return
// 	}
// 	c.Data(http.StatusOK, CONTENT_TYPE_JSON, blockHash)
// }

// @Summary ordinal recursive endpoint for get lastest block height
// @Description ordinal recursive endpoint for get lastest block height
// @Tags ordx.ord.r
// @Produce json
// @Security Bearer
// @Success 200 {object} []byte "Successful response"
// @Failure 401 "Invalid API Key"
// @Router /ord/r/blockheight [get]
// func (s *Service) getRLastestBlockHeight(c *gin.Context) {
// 	blockHeight, err := ord_rpc.ShareOrdinalsRpc.GetRLastestBlockHeight()
// 	if err != nil {
// 		c.JSON(http.StatusOK, err.Error())
// 		return
// 	}
// 	c.Data(http.StatusOK, CONTENT_TYPE_JSON, blockHeight)
// }

// @Summary ordinal recursive endpoint for get block info
// @Description ordinal recursive endpoint for get block info
// @Tags ordx.ord.r
// @Produce json
// @Param query path string true "block height or block hash"
// @Security Bearer
// @Success 200 {object} []byte "Successful response"
// @Failure 401 "Invalid API Key"
// @Router /ord/r/blockinfo/{query} [get]
// func (s *Service) getRBlockInfo(c *gin.Context) {
// 	blockInfo, err := ord_rpc.ShareOrdinalsRpc.GetRBlockInfo(c.Param("query"))
// 	if err != nil {
// 		c.JSON(http.StatusOK, err.Error())
// 		return
// 	}
// 	c.Data(http.StatusOK, CONTENT_TYPE_JSON, blockInfo)
// }

// @Summary ordinal recursive endpoint for get UNIX time stamp of latest block
// @Description ordinal recursive endpoint for get UNIX time stamp of latest block
// @Tags ordx.ord.r
// @Produce json
// @Param query path string true "block height or block hash"
// @Security Bearer
// @Success 200 {object} []byte "Successful response"
// @Failure 401 "Invalid API Key"
// @Router /ord/r/blocktime [get]
// func (s *Service) getRLatestBlockTimestamp(c *gin.Context) {
// 	blockInfo, err := ord_rpc.ShareOrdinalsRpc.GetRLatestBlockTimestamp()
// 	if err != nil {
// 		c.JSON(http.StatusOK, err.Error())
// 		return
// 	}
// 	c.Data(http.StatusOK, CONTENT_TYPE_JSON, blockInfo)
// }

// @Summary ordinal recursive endpoint for get the first 100 children ids
// @Description ordinal recursive endpoint for get the first 100 children ids
// @Tags ordx.ord.r
// @Produce json
// @Param inscriptionid path string true "inscription ID example: 79b0e9dbfaf11e664abafbd8fec7d734bfa2d59013f25c50aaac1264f700832di0"
// @Param page path string false "page example: 0"
// @Security Bearer
// @Success 200 {object} []byte "Successful response"
// @Failure 401 "Invalid API Key"
// @Router /ord/r/children/{inscriptionid}/{page} [get]
// func (s *Service) getRChildrenInscriptionIdList(c *gin.Context) {
// 	inscriptionIdList, err := ord_rpc.ShareOrdinalsRpc.GetRChildrenInscriptionIdList(c.Param("inscriptionid"), c.Param("page"))
// 	if err != nil {
// 		c.JSON(http.StatusOK, err.Error())
// 		return
// 	}
// 	c.Data(http.StatusOK, CONTENT_TYPE_JSON, inscriptionIdList)
// }

// @Summary ordinal recursive endpoint for get inscription info
// @Description ordinal recursive endpoint for get inscription info
// @Tags ordx.ord.r
// @Produce json
// @Param inscriptionid path string true "inscription ID example: 79b0e9dbfaf11e664abafbd8fec7d734bfa2d59013f25c50aaac1264f700832di0"
// @Security Bearer
// @Success 200 {object} []byte "Successful response"
// @Failure 401 "Invalid API Key"
// @Router /ord/r/inscription/{inscriptionid} [get]
// func (s *Service) getRInscriptionInfo(c *gin.Context) {
// 	inscription, err := ord_rpc.ShareOrdinalsRpc.GetRInscriptionInfo(c.Param("inscriptionid"))
// 	if err != nil {
// 		c.JSON(http.StatusOK, err.Error())
// 		return
// 	}
// 	c.Data(http.StatusOK, CONTENT_TYPE_JSON, inscription)
// }

// @Summary ordinal recursive endpoint for get hex-encoded CBOR metadata of an inscription
// @Description ordinal recursive endpoint for get hex-encoded CBOR metadata of an inscription
// @Tags ordx.ord.r
// @Produce json
// @Param inscriptionid path string true "inscription ID example: a4b6fccd00222e79ec0307d52fe9f8bfa3713cd0c170f95065f5d859e0c6a0f5i0"
// @Security Bearer
// @Success 200 {object} []byte "Successful response"
// @Failure 401 "Invalid API Key"
// @Router /ord/r/metadata/{inscriptionid} [get]
func (s *Service) getRMetadata(c *gin.Context) {
	inscriptionId := c.Param("inscriptionid")
	err := checkInscriptionId(inscriptionId)
	if err != nil {
		c.Data(http.StatusBadRequest, CONTEXT_TYPE_TEXT, []byte(err.Error()))
		return
	}

	nft := base_indexer.ShareBaseIndexer.GetNftInfoWithInscriptionId(inscriptionId)
	if nft == nil {
		c.Data(http.StatusNotFound, CONTEXT_TYPE_TEXT, []byte(fmt.Sprintf(`inscription %s not found`, inscriptionId)))
		return
	}
	if nft.Base.MetaData == nil {
		c.Data(http.StatusNotFound, CONTEXT_TYPE_TEXT, []byte(fmt.Sprintf(`inscription %s metadata not found`, inscriptionId)))
		return
	}
	cborData := fmt.Sprintf(`"%x"`, nft.Base.MetaData)
	c.Data(http.StatusOK, CONTENT_TYPE_JSON, []byte(cborData))
	c.Writer.Flush()
}

// @Summary ordinal recursive endpoint for get the first 100 inscription ids on a sat
// @Description ordinal recursive endpoint for get the first 100 inscription ids on a sat
// @Tags ordx.ord.r
// @Produce json
// @Param satnumber path string true "sat number example: 1165647477496168"
// @Param page path string false "page example: 0"
// @Security Bearer
// @Success 200 {object} []byte "Successful response"
// @Failure 401 "Invalid API Key"
// @Router /ord/r/sat/{satnumber}/{page} [get]
// func (s *Service) getRSatInscriptionIdList(c *gin.Context) {
// 	inscriptionIdList, err := ord_rpc.ShareOrdinalsRpc.GetRSatInscriptionIdList(c.Param("satnumber"), c.Param("page"))
// 	if err != nil {
// 		c.JSON(http.StatusOK, err.Error())
// 		return
// 	}
// 	c.Data(http.StatusOK, CONTENT_TYPE_JSON, inscriptionIdList)
// }

// @Summary ordinal recursive endpoint for get the inscription id at <INDEX> of all inscriptions on a sat
// @Description ordinal recursive endpoint for get the inscription id at <INDEX> of all inscriptions on a sat
// @Tags ordx.ord.r
// @Produce json
// @Param satnumber path string true "sat number example: 1165647477496168"
// @Param index path string false "page example: -1"
// @Security Bearer
// @Success 200 {object} []byte "Successful response"
// @Failure 401 "Invalid API Key"
// @Router /ord/r/sat/{satnumber}/at/{index} [get]
// func (s *Service) getRSatInscriptionId(c *gin.Context) {
// 	inscriptionID, err := ord_rpc.ShareOrdinalsRpc.GetRSatInscriptionId(c.Param("satnumber"), c.Param("index"))
// 	if err != nil {
// 		c.JSON(http.StatusOK, err.Error())
// 		return
// 	}
// 	c.Data(http.StatusOK, CONTENT_TYPE_JSON, inscriptionID)
// }
