package gemini

import (
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/service/relayconvert"
)

func ApplyImageConfigResponseFormatCompatibility(request *dto.GeminiChatRequest, modelName string) error {
	return relayconvert.ApplyGeminiImageConfigResponseFormatCompatibility(request, modelName)
}
