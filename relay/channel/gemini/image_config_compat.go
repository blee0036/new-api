package gemini

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/setting/model_setting"
)

// ApplyImageConfigResponseFormatCompatibility normalizes Gemini image config
// fields. generationConfig.imageConfig accepts string values like "1:1" and
// "1K"; generationConfig.responseFormat.image uses proto enum names for the
// same concepts, so it must only be normalized when explicitly present.
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
		changedImageConfig = normalizeGeminiImageConfigFields(imageConfig)
	}

	changedResponseFormat := false
	if hasResponseFormat {
		if image, ok := responseFormat["image"].(map[string]interface{}); ok {
			changedResponseFormat = normalizeGeminiImageResponseFormatFields(image)
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

func normalizeGeminiImageConfigFields(image map[string]interface{}) bool {
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

func normalizeGeminiImageResponseFormatFields(image map[string]interface{}) bool {
	changed := normalizeGeminiImageConfigFields(image)
	if normalizeImageResponseEnumField(image, "aspectRatio", geminiImageResponseAspectRatioEnums) {
		changed = true
	}
	if normalizeImageResponseEnumField(image, "imageSize", geminiImageResponseSizeEnums) {
		changed = true
	}
	return changed
}

func normalizeImageResponseEnumField(image map[string]interface{}, key string, enumMap map[string]string) bool {
	value, ok := image[key].(string)
	if !ok {
		return false
	}
	enumValue, ok := enumMap[strings.ToUpper(strings.TrimSpace(value))]
	if !ok || enumValue == value {
		return false
	}
	image[key] = enumValue
	return true
}

var geminiImageResponseAspectRatioEnums = map[string]string{
	"1:1":                             "ASPECT_RATIO_ONE_BY_ONE",
	"2:3":                             "ASPECT_RATIO_TWO_BY_THREE",
	"3:2":                             "ASPECT_RATIO_THREE_BY_TWO",
	"3:4":                             "ASPECT_RATIO_THREE_BY_FOUR",
	"4:3":                             "ASPECT_RATIO_FOUR_BY_THREE",
	"4:5":                             "ASPECT_RATIO_FOUR_BY_FIVE",
	"5:4":                             "ASPECT_RATIO_FIVE_BY_FOUR",
	"9:16":                            "ASPECT_RATIO_NINE_BY_SIXTEEN",
	"16:9":                            "ASPECT_RATIO_SIXTEEN_BY_NINE",
	"21:9":                            "ASPECT_RATIO_TWENTY_ONE_BY_NINE",
	"1:8":                             "ASPECT_RATIO_ONE_BY_EIGHT",
	"8:1":                             "ASPECT_RATIO_EIGHT_BY_ONE",
	"1:4":                             "ASPECT_RATIO_ONE_BY_FOUR",
	"4:1":                             "ASPECT_RATIO_FOUR_BY_ONE",
	"ASPECT_RATIO_ONE_BY_ONE":         "ASPECT_RATIO_ONE_BY_ONE",
	"ASPECT_RATIO_TWO_BY_THREE":       "ASPECT_RATIO_TWO_BY_THREE",
	"ASPECT_RATIO_THREE_BY_TWO":       "ASPECT_RATIO_THREE_BY_TWO",
	"ASPECT_RATIO_THREE_BY_FOUR":      "ASPECT_RATIO_THREE_BY_FOUR",
	"ASPECT_RATIO_FOUR_BY_THREE":      "ASPECT_RATIO_FOUR_BY_THREE",
	"ASPECT_RATIO_FOUR_BY_FIVE":       "ASPECT_RATIO_FOUR_BY_FIVE",
	"ASPECT_RATIO_FIVE_BY_FOUR":       "ASPECT_RATIO_FIVE_BY_FOUR",
	"ASPECT_RATIO_NINE_BY_SIXTEEN":    "ASPECT_RATIO_NINE_BY_SIXTEEN",
	"ASPECT_RATIO_SIXTEEN_BY_NINE":    "ASPECT_RATIO_SIXTEEN_BY_NINE",
	"ASPECT_RATIO_TWENTY_ONE_BY_NINE": "ASPECT_RATIO_TWENTY_ONE_BY_NINE",
	"ASPECT_RATIO_ONE_BY_EIGHT":       "ASPECT_RATIO_ONE_BY_EIGHT",
	"ASPECT_RATIO_EIGHT_BY_ONE":       "ASPECT_RATIO_EIGHT_BY_ONE",
	"ASPECT_RATIO_ONE_BY_FOUR":        "ASPECT_RATIO_ONE_BY_FOUR",
	"ASPECT_RATIO_FOUR_BY_ONE":        "ASPECT_RATIO_FOUR_BY_ONE",
}

var geminiImageResponseSizeEnums = map[string]string{
	"512":                    "IMAGE_SIZE_FIVE_TWELVE",
	"1K":                     "IMAGE_SIZE_ONE_K",
	"2K":                     "IMAGE_SIZE_TWO_K",
	"4K":                     "IMAGE_SIZE_FOUR_K",
	"IMAGE_SIZE_FIVE_TWELVE": "IMAGE_SIZE_FIVE_TWELVE",
	"IMAGE_SIZE_ONE_K":       "IMAGE_SIZE_ONE_K",
	"IMAGE_SIZE_TWO_K":       "IMAGE_SIZE_TWO_K",
	"IMAGE_SIZE_FOUR_K":      "IMAGE_SIZE_FOUR_K",
	"IMAGE_SIZE_UNSPECIFIED": "IMAGE_SIZE_UNSPECIFIED",
}
