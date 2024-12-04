package extension

import (
	"net/http"

	"github.com/gin-gonic/gin"
	rpcwire "github.com/sat20-labs/indexer/rpcserver/wire"
)

func (s *Service) version_detail(c *gin.Context) {
	resp := &VersionDetailResp{
		BaseResp: rpcwire.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Data: &VersionDetailData{
			Version:    "?",
			Title:      "A new version v? is available",
			Changelogs: []interface{}{},
		},
	}

	c.JSON(http.StatusOK, resp)
}
