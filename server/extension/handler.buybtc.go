package extension

// import (
// 	"net/http"

// 	"github.com/sat20-labs/ordx/server/define"
// 	"github.com/gin-gonic/gin"
// )

// func (s *Service) getBuyBtcChannelList(c *gin.Context) {
// 	resp := &BuyBtcChannelListResp{
// 		BaseResp: define.BaseResp{
// 			Code: 0,
// 			Msg:  "ok",
// 		},
// 		Data: []*BuyBtcChannel{
// 			{
// 				Channel: "moonpay",
// 			},
// 			{
// 				Channel: "alchepay",
// 			},
// 		},
// 	}

// 	c.JSON(http.StatusOK, resp)
// }

// func (s *Service) createPaymentUrl(c *gin.Context) {
// 	resp := &BuyBtcCreateResp{
// 		BaseResp: define.BaseResp{
// 			Code: 0,
// 			Msg:  "ok",
// 		},
// 		Data: "", // url
// 	}

// 	var req BuyBtcCreateReq
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		resp.Code = -1
// 		resp.Msg = err.Error()
// 		c.JSON(http.StatusOK, resp)
// 		return
// 	}

// 	c.JSON(http.StatusOK, resp)
// }
