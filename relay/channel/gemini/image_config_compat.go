package gemini

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/setting/model_setting"
)

// ApplyImageConfigResponseFormatCompatibility mirrors Gemini imageConfig into
// responseFormat.image for Gemini image models. Recent Gemini image REST
// examples use responseFormat.image, while older clients and docs still send
// imageConfig.
func ApplyImageConfigResponseFormatCompatibility(request *dto.GeminiChatRequest, modelName string) error {
	if request == nil || !isGeminiImageConfigModel(modelName) {
		return nil
	}

	if err := normalizeGenerationConfigImageFields(&request.GenerationConfig); err != nil {
		return err
	}

	for i := range request.Requests {
		if err := ApplyImageConfigResponseFormatCompatibility(&request.Requests[i], modelName); err != nil {
			return err
		}
	}

	return nil
}

func isGeminiImageConfigModel(modelName string) bool {
	modelName = strings.TrimPrefix(modelName, "models/")
	return model_setting.IsGeminiModelSupportImagine(modelName) ||
		strings.Contains(modelName, "image") ||
		strings.Contains(modelName, "nano-banana")
}

func normalizeGenerationConfigImageFields(config *dto.GeminiChatGenerationConfig) error {
	imageConfig, hasImageConfig, err := rawJSONObject(config.ImageConfig)
	if err != nil {
		return err
	}

	responseFormat, hasResponseFormat, err := rawJSONObject(config.ResponseFormat)
	if err != nil {
		return err
	}

	changedImageConfig := false
	if hasImageConfig {
		changedImageConfig = normalizeGeminiImageFields(imageConfig)
	}

	changedResponseFormat := false
	if hasResponseFormat {
		if image, ok := responseFormat["image"].(map[string]interface{}); ok {
			changedResponseFormat = normalizeGeminiImageFields(image)
		}
	}

	if hasImageConfig && hasGeminiImageOutputFields(imageConfig) {
		if responseFormat == nil {
			responseFormat = make(map[string]interface{})
		}

		image, ok := responseFormat["image"].(map[string]interface{})
		if !ok {
			if responseFormat["image"] != nil {
				return marshalNormalizedImageConfig(config, imageConfig, changedImageConfig, responseFormat, changedResponseFormat)
			}
			image = make(map[string]interface{})
			responseFormat["image"] = image
			changedResponseFormat = true
		}

		if copyMissingImageField(image, imageConfig, "aspectRatio") {
			changedResponseFormat = true
		}
		if copyMissingImageField(image, imageConfig, "imageSize") {
			changedResponseFormat = true
		}
	}

	return marshalNormalizedImageConfig(config, imageConfig, changedImageConfig, responseFormat, changedResponseFormat)
}

func marshalNormalizedImageConfig(
	config *dto.GeminiChatGenerationConfig,
	imageConfig map[string]interface{},
	changedImageConfig bool,
	responseFormat map[string]interface{},
	changedResponseFormat bool,
) error {
	if changedImageConfig {
		data, err := common.Marshal(imageConfig)
		if err != nil {
			return err
		}
		config.ImageConfig = data
	}

	if changedResponseFormat {
		data, err := common.Marshal(responseFormat)
		if err != nil {
			return err
		}
		config.ResponseFormat = data
	}

	return nil
}

func rawJSONObject(raw []byte) (map[string]interface{}, bool, error) {
	if len(raw) == 0 || common.GetJsonType(raw) != "object" {
		return nil, false, nil
	}

	var out map[string]interface{}
	if err := common.Unmarshal(raw, &out); err != nil {
		return nil, false, err
	}
	return out, true, nil
}

func normalizeGeminiImageFields(image map[string]interface{}) bool {
	changed := false
	if value, ok := image["aspect_ratio"]; ok {
		if _, exists := image["aspectRatio"]; !exists {
			image["aspectRatio"] = value
		}
		delete(image, "aspect_ratio")
		changed = true
	}
	if value, ok := image["image_size"]; ok {
		if _, exists := image["imageSize"]; !exists {
			image["imageSize"] = value
		}
		delete(image, "image_size")
		changed = true
	}
	return changed
}

func hasGeminiImageOutputFields(image map[string]interface{}) bool {
	_, hasAspectRatio := image["aspectRatio"]
	_, hasImageSize := image["imageSize"]
	return hasAspectRatio || hasImageSize
}

func copyMissingImageField(target, source map[string]interface{}, camelKey string, sourceKeys ...string) bool {
	if _, exists := target[camelKey]; exists {
		return false
	}

	keys := append([]string{camelKey}, sourceKeys...)
	for _, key := range keys {
		value, ok := source[key]
		if !ok {
			continue
		}
		target[camelKey] = value
		return true
	}
	return false
}
