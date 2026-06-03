package gemini

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestConvertGeminiRequestMirrorsImageConfigToResponseFormat(t *testing.T) {
	request := &dto.GeminiChatRequest{
		Contents: []dto.GeminiChatContent{
			{Parts: []dto.GeminiPart{{Text: "draw a cat"}}},
		},
		GenerationConfig: dto.GeminiChatGenerationConfig{
			ImageConfig: []byte(`{"aspectRatio":"16:9","imageSize":"2K"}`),
		},
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "gemini-3.1-flash-image-preview",
		},
	}

	converted, err := (&Adaptor{}).ConvertGeminiRequest(nil, info, request)
	require.NoError(t, err)

	body, err := common.Marshal(converted)
	require.NoError(t, err)

	var out map[string]interface{}
	require.NoError(t, common.Unmarshal(body, &out))
	generationConfig := requireMap(t, out, "generationConfig")
	imageConfig := requireMap(t, generationConfig, "imageConfig")
	responseFormat := requireMap(t, generationConfig, "responseFormat")
	responseImage := requireMap(t, responseFormat, "image")

	require.Equal(t, "16:9", imageConfig["aspectRatio"])
	require.Equal(t, "2K", imageConfig["imageSize"])
	require.Equal(t, "16:9", responseImage["aspectRatio"])
	require.Equal(t, "2K", responseImage["imageSize"])
}

func TestConvertGeminiRequestNormalizesSnakeCaseImageFields(t *testing.T) {
	request := &dto.GeminiChatRequest{
		Contents: []dto.GeminiChatContent{
			{Parts: []dto.GeminiPart{{Text: "draw a poster"}}},
		},
		GenerationConfig: dto.GeminiChatGenerationConfig{
			ImageConfig: []byte(`{"aspect_ratio":"4:3","image_size":"4K"}`),
		},
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "gemini-3.1-flash-image-preview",
		},
	}

	_, err := (&Adaptor{}).ConvertGeminiRequest(nil, info, request)
	require.NoError(t, err)

	var imageConfig map[string]interface{}
	require.NoError(t, common.Unmarshal(request.GenerationConfig.ImageConfig, &imageConfig))
	require.NotContains(t, imageConfig, "aspect_ratio")
	require.NotContains(t, imageConfig, "image_size")
	require.Equal(t, "4:3", imageConfig["aspectRatio"])
	require.Equal(t, "4K", imageConfig["imageSize"])

	responseImage := responseFormatImage(t, request.GenerationConfig.ResponseFormat)
	require.Equal(t, "4:3", responseImage["aspectRatio"])
	require.Equal(t, "4K", responseImage["imageSize"])
}

func TestImageConfigCompatibilityPreservesExplicitResponseFormat(t *testing.T) {
	request := &dto.GeminiChatRequest{
		GenerationConfig: dto.GeminiChatGenerationConfig{
			ImageConfig:    []byte(`{"aspectRatio":"16:9","imageSize":"2K"}`),
			ResponseFormat: []byte(`{"image":{"aspectRatio":"1:1"}}`),
		},
	}

	require.NoError(t, ApplyImageConfigResponseFormatCompatibility(request, "gemini-3.1-flash-image-preview"))

	responseImage := responseFormatImage(t, request.GenerationConfig.ResponseFormat)
	require.Equal(t, "1:1", responseImage["aspectRatio"])
	require.Equal(t, "2K", responseImage["imageSize"])
}

func TestCovertOpenAI2GeminiMirrorsGoogleImageConfigExtraBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)

	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:       constant.ChannelTypeGemini,
			UpstreamModelName: "gemini-3.1-flash-image-preview",
		},
	}
	request := dto.GeneralOpenAIRequest{
		Model: "gemini-3.1-flash-image-preview",
		Messages: []dto.Message{
			{Role: "user", Content: "draw a landscape"},
		},
		ExtraBody: []byte(`{"google":{"image_config":{"aspect_ratio":"21:9","image_size":"4K"}}}`),
	}

	geminiRequest, err := CovertOpenAI2Gemini(c, request, info)
	require.NoError(t, err)

	responseImage := responseFormatImage(t, geminiRequest.GenerationConfig.ResponseFormat)
	require.Equal(t, "21:9", responseImage["aspectRatio"])
	require.Equal(t, "4K", responseImage["imageSize"])
}

func responseFormatImage(t *testing.T, raw []byte) map[string]interface{} {
	t.Helper()

	var responseFormat map[string]interface{}
	require.NoError(t, common.Unmarshal(raw, &responseFormat))
	return requireMap(t, responseFormat, "image")
}

func requireMap(t *testing.T, data map[string]interface{}, key string) map[string]interface{} {
	t.Helper()

	value, ok := data[key].(map[string]interface{})
	require.Truef(t, ok, "%s is not an object", key)
	return value
}
