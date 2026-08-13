package gemini

import (
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/relayconvert"
)

func ApplyImageConfigResponseFormatCompatibility(request *dto.GeminiChatRequest, modelName string) error {
	return relayconvert.ApplyGeminiImageConfigResponseFormatCompatibility(request, modelName)
}
