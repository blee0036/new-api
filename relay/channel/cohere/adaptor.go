package cohere

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"one-api/dto"
	"one-api/relay/channel"
	relaycommon "one-api/relay/common"
	"one-api/relay/constant"
)

type Adaptor struct {
}

func (a *Adaptor) ConvertAudioRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.AudioRequest) (io.Reader, error) {
	//TODO implement me
	return nil, errors.New("not implemented")
}

func (a *Adaptor) ConvertImageRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.ImageRequest) (any, error) {
	//TODO implement me
	return nil, errors.New("not implemented")
}

func (a *Adaptor) Init(info *relaycommon.RelayInfo) {
}

func (a *Adaptor) GetRequestURL(info *relaycommon.RelayInfo) (string, error) {
	if info.RelayMode == constant.RelayModeRerank {
		return fmt.Sprintf("%s/v1/rerank", info.BaseUrl), nil
	} else {
		return fmt.Sprintf("%s/v1/chat", info.BaseUrl), nil
	}
}

func (a *Adaptor) SetupRequestHeader(c *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error {
	channel.SetupApiRequestHeader(info, c, req)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", info.ApiKey))
	req.Header.Set("Accept", "*/*")
	req.Header.Set("User-Agent", " sd/JS 4.54.0")
	req.Header.Set("X-Middleware-Subrequest", "app/api/chat/openai/route")
	req.Header.Set("X-Stainless-Arch", "other:edge-runtime")
	req.Header.Set("X-Stainless-Lang", "js")
	req.Header.Set("X-Stainless-Os", "Unknown")
	req.Header.Set("X-Stainless-Package-Version", "4.54.0")
	req.Header.Set("X-Stainless-Runtime", "edge")
	req.Header.Set("Accept-Language", "*")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	return nil
}

func (a *Adaptor) ConvertRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeneralOpenAIRequest) (any, error) {
	return requestOpenAI2Cohere(*request), nil
}

func (a *Adaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoApiRequest(a, c, info, requestBody)
}

func (a *Adaptor) ConvertRerankRequest(c *gin.Context, relayMode int, request dto.RerankRequest) (any, error) {
	return requestConvertRerank2Cohere(request), nil
}

func (a *Adaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (usage *dto.Usage, err *dto.OpenAIErrorWithStatusCode) {
	if info.RelayMode == constant.RelayModeRerank {
		err, usage = cohereRerankHandler(c, resp, info)
	} else {
		if info.IsStream {
			err, usage = cohereStreamHandler(c, resp, info)
		} else {
			err, usage = cohereHandler(c, resp, info.UpstreamModelName, info.PromptTokens)
		}
	}
	return
}

func (a *Adaptor) GetModelList() []string {
	return ModelList
}

func (a *Adaptor) GetChannelName() string {
	return ChannelName
}
