package relay

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/stretchr/testify/require"
)

func TestCleanGeminiSystemInstructionRemovesEmptyParts(t *testing.T) {
	request := &dto.GeminiChatRequest{
		SystemInstructions: &dto.GeminiChatContent{
			Parts: []dto.GeminiPart{
				{},
				{Text: ""},
				{Text: "keep this"},
			},
		},
	}

	cleanGeminiSystemInstruction(request)

	require.NotNil(t, request.SystemInstructions)
	require.Len(t, request.SystemInstructions.Parts, 1)
	require.Equal(t, "keep this", request.SystemInstructions.Parts[0].Text)
}

func TestCleanGeminiSystemInstructionDropsInstructionWithoutData(t *testing.T) {
	request := &dto.GeminiChatRequest{
		SystemInstructions: &dto.GeminiChatContent{
			Parts: []dto.GeminiPart{
				{},
				{Thought: true},
			},
		},
	}

	cleanGeminiSystemInstruction(request)

	require.Nil(t, request.SystemInstructions)
}

func TestSanitizeGeminiSystemInstructionJSONRemovesEmptyParts(t *testing.T) {
	input := []byte(`{
		"system_instruction":{
			"parts":[{}, {"text":""}, {"thought":true}, {"text":"keep"}]
		},
		"requests":[
			{"systemInstruction":{"parts":[{}]}}
		]
	}`)

	out, changed, err := sanitizeGeminiSystemInstructionJSON(input)
	require.NoError(t, err)
	require.True(t, changed)

	var data map[string]interface{}
	require.NoError(t, common.Unmarshal(out, &data))
	systemInstruction := data["system_instruction"].(map[string]interface{})
	parts := systemInstruction["parts"].([]interface{})
	require.Len(t, parts, 1)
	require.Equal(t, "keep", parts[0].(map[string]interface{})["text"])

	requests := data["requests"].([]interface{})
	nested := requests[0].(map[string]interface{})
	require.NotContains(t, nested, "systemInstruction")
}
