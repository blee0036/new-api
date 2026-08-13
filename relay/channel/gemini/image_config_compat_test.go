package gemini

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/relayconvert"
	"github.com/stretchr/testify/require"
)

func TestConvertGeminiRequestKeepsImageConfigWithoutResponseFormat(t *testing.T) {
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

	require.Equal(t, "16:9", imageConfig["aspectRatio"])
	require.Equal(t, "2K", imageConfig["imageSize"])
	require.NotContains(t, generationConfig, "responseFormat")
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

	require.Empty(t, request.GenerationConfig.ResponseFormat)
}

func TestImageConfigCompatibilityNormalizesExplicitResponseFormatEnums(t *testing.T) {
	request := &dto.GeminiChatRequest{
		GenerationConfig: dto.GeminiChatGenerationConfig{
			ImageConfig:    []byte(`{"aspectRatio":"16:9","imageSize":"2K"}`),
			ResponseFormat: []byte(`{"image":{"aspect_ratio":"1:1","image_size":"1K"}}`),
		},
	}

	require.NoError(t, ApplyImageConfigResponseFormatCompatibility(request, "gemini-3.1-flash-image-preview"))

	responseImage := responseFormatImage(t, request.GenerationConfig.ResponseFormat)
	require.NotContains(t, responseImage, "aspect_ratio")
	require.NotContains(t, responseImage, "image_size")
	require.Equal(t, "ASPECT_RATIO_ONE_BY_ONE", responseImage["aspectRatio"])
	require.Equal(t, "IMAGE_SIZE_ONE_K", responseImage["imageSize"])
}

func TestOpenAIChatRequestToGeminiKeepsGoogleImageConfigInImageConfig(t *testing.T) {
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
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

	geminiRequest, err := relayconvert.OpenAIChatRequestToGeminiGenerateContent(nil, request, info)
	require.NoError(t, err)

	var imageConfig map[string]interface{}
	require.NoError(t, common.Unmarshal(geminiRequest.GenerationConfig.ImageConfig, &imageConfig))
	require.Equal(t, "21:9", imageConfig["aspectRatio"])
	require.Equal(t, "4K", imageConfig["imageSize"])
	require.Empty(t, geminiRequest.GenerationConfig.ResponseFormat)
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
