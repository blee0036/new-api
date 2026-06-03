package relay

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
)

func cleanGeminiSystemInstruction(request *dto.GeminiChatRequest) {
	if request == nil {
		return
	}

	if request.SystemInstructions != nil {
		parts := make([]dto.GeminiPart, 0, len(request.SystemInstructions.Parts))
		for _, part := range request.SystemInstructions.Parts {
			if geminiPartHasData(part) {
				parts = append(parts, part)
			}
		}
		if len(parts) == 0 {
			request.SystemInstructions = nil
		} else {
			request.SystemInstructions.Parts = parts
		}
	}

	for i := range request.Requests {
		cleanGeminiSystemInstruction(&request.Requests[i])
	}
}

func geminiPartHasData(part dto.GeminiPart) bool {
	return part.Text != "" ||
		part.InlineData != nil ||
		part.FunctionCall != nil ||
		part.FunctionResponse != nil ||
		part.FileData != nil ||
		part.ExecutableCode != nil ||
		part.CodeExecutionResult != nil
}

func sanitizeGeminiSystemInstructionJSON(jsonData []byte) ([]byte, bool, error) {
	var data map[string]interface{}
	if err := common.Unmarshal(jsonData, &data); err != nil {
		return nil, false, err
	}

	changed := sanitizeGeminiSystemInstructionMap(data)
	if !changed {
		return jsonData, false, nil
	}

	out, err := common.Marshal(data)
	if err != nil {
		return nil, false, err
	}
	return out, true, nil
}

func sanitizeGeminiSystemInstructionMap(data map[string]interface{}) bool {
	changed := false
	for _, key := range []string{"systemInstruction", "system_instruction"} {
		if sanitizeGeminiSystemInstructionKey(data, key) {
			changed = true
		}
	}

	if requests, ok := data["requests"].([]interface{}); ok {
		for _, item := range requests {
			if requestMap, ok := item.(map[string]interface{}); ok {
				if sanitizeGeminiSystemInstructionMap(requestMap) {
					changed = true
				}
			}
		}
	}

	return changed
}

func sanitizeGeminiSystemInstructionKey(data map[string]interface{}, key string) bool {
	rawInstruction, exists := data[key]
	if !exists {
		return false
	}

	instruction, ok := rawInstruction.(map[string]interface{})
	if !ok {
		delete(data, key)
		return true
	}

	rawParts, ok := instruction["parts"].([]interface{})
	if !ok {
		delete(data, key)
		return true
	}

	parts := make([]interface{}, 0, len(rawParts))
	changed := false
	for _, rawPart := range rawParts {
		part, ok := rawPart.(map[string]interface{})
		if !ok || !rawGeminiPartHasData(part) {
			changed = true
			continue
		}
		parts = append(parts, rawPart)
	}

	if len(parts) == 0 {
		delete(data, key)
		return true
	}
	if changed {
		instruction["parts"] = parts
		data[key] = instruction
	}
	return changed
}

func rawGeminiPartHasData(part map[string]interface{}) bool {
	if text, ok := part["text"].(string); ok && text != "" {
		return true
	}

	for _, key := range []string{
		"inlineData",
		"inline_data",
		"functionCall",
		"function_call",
		"functionResponse",
		"function_response",
		"fileData",
		"file_data",
		"executableCode",
		"executable_code",
		"codeExecutionResult",
		"code_execution_result",
	} {
		if rawGeminiValueHasContent(part[key]) {
			return true
		}
	}

	return false
}

func rawGeminiValueHasContent(value interface{}) bool {
	switch v := value.(type) {
	case nil:
		return false
	case string:
		return v != ""
	case map[string]interface{}:
		return len(v) > 0
	case []interface{}:
		return len(v) > 0
	default:
		return true
	}
}
