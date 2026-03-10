package usecase

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"text/template"

	"github.com/arifth/botthie/model"
)

func extractURL(urlInterface interface{}) string {
	rw := urlInterface.(model.PostmanRequest)
	raw, ok := rw.URL.(map[string]interface{})
	fmt.Println(raw)
	if !ok {
		return ""
	}
	for key, value := range raw {
		if key == "raw" {
			fmt.Println(value)
			vl := value.(string)
			return vl
		}
	}
	return ""
}

// generateDescription generates a description based on field name and value
func generateDescription(fieldName string, value interface{}) string {
	// Convert field name from camelCase/snake_case to readable format
	readable := makeReadable(fieldName)

	// Generate smart description based on type
	valueType := determineType(value)

	switch valueType {
	case "string":
		if strVal, ok := value.(string); ok && strVal != "" {
			return fmt.Sprintf("%s (example: %s)", readable, strVal)
		}
		return readable
	case "integer", "number":
		return fmt.Sprintf("%s value", readable)
	case "boolean":
		return fmt.Sprintf("%s flag", readable)
	case "array":
		return fmt.Sprintf("List of %s", readable)
	case "object":
		return fmt.Sprintf("%s object details", readable)
	default:
		return readable
	}
}

// makeReadable converts field names to readable format
func makeReadable(fieldName string) string {
	// Replace underscores with spaces
	result := strings.ReplaceAll(fieldName, "_", " ")

	// Add space before capital letters (camelCase)
	var readable strings.Builder
	for i, r := range result {
		if i > 0 && r >= 'A' && r <= 'Z' {
			readable.WriteRune(' ')
		}
		if i == 0 {
			readable.WriteRune(r)
		} else {
			readable.WriteRune(r)
		}
	}

	// Capitalize first letter
	finalResult := readable.String()
	if len(finalResult) > 0 {
		finalResult = strings.ToUpper(string(finalResult[0])) + finalResult[1:]
	}

	return finalResult
}

func determineType(value interface{}) string {
	if value == nil {
		return "null"
	}

	switch v := value.(type) {
	case string:
		return "string"
	case int, int8, int16, int32, int64:
		return "integer"
	case float32, float64:
		return "number"
	case bool:
		return "boolean"
	case []interface{}:
		return "array"
	case map[string]interface{}:
		return "object"
	default:
		return fmt.Sprintf("%T", v)
	}
}

// parseJSONBodyFields parses JSON body and extracts field names with their types
func parseJSONBodyFields(rawBody string) []model.BodyField {
	var jsonData map[string]interface{}
	err := json.Unmarshal([]byte(rawBody), &jsonData)
	if err != nil {
		return nil
	}

	var fields []model.BodyField
	index := 1
	for key, value := range jsonData {
		fields = append(fields, model.BodyField{
			Number:      index,
			Field:       key,
			Type:        determineType(value),
			Mandatory:   "No", // Default to No, can be customized
			Description: generateDescription(key, value),
		})
		index++
	}

	return fields
}
func (Usecase) ConvertToHTML(collection model.PostmanCollection, dataTempl string, item model.PostmanItem) string {
	// Extract request data
	reqData := model.RequestData{
		Name:    item.Name,
		Method:  item.Request.Method,
		URL:     extractURL(item.Request),
		Headers: item.Request.Header,
	}

	// Parse body based on mode
	if item.Request.Body != nil {
		reqData.BodyMode = item.Request.Body.Mode

		// Check if body is JSON with fields
		if item.Request.Body.Mode == "raw" && item.Request.Body.Raw != "" {
			// Try to parse as JSON to extract fields
			bodyFields := parseJSONBodyFields(item.Request.Body.Raw)
			if len(bodyFields) > 0 {
				reqData.BodyFields = bodyFields
			} else {
				// If not valid JSON or no fields, just show raw
				reqData.Body = item.Request.Body.Raw
			}
		} else if item.Request.Body.Mode == "formdata" && len(item.Request.Body.FormData) > 0 {
			// Parse form-data fields
			for idx, field := range item.Request.Body.FormData {
				reqData.BodyFields = append(reqData.BodyFields, model.BodyField{
					Number:      idx + 1,
					Field:       field.Key,
					Type:        determineType(field.Value),
					Mandatory:   "No",
					Description: makeReadable(field.Key),
				})
			}
		} else if item.Request.Body.Mode == "urlencoded" && len(item.Request.Body.URLEncoded) > 0 {
			// Parse URL-encoded fields
			for idx, field := range item.Request.Body.URLEncoded {
				reqData.BodyFields = append(reqData.BodyFields, model.BodyField{
					Number:      idx + 1,
					Field:       field.Key,
					Type:        determineType(field.Value),
					Mandatory:   "No",
					Description: makeReadable(field.Key),
				})
			}
		} else if item.Request.Body.Raw != "" {
			reqData.Body = item.Request.Body.Raw
		}
	}
	// Prepare template data
	data := model.TemplateData{
		CollectionName: collection.Info.Name,
		Requests:       reqData,
	}
	// Parse and execute template
	t, err := template.New("postman").Parse(dataTempl)
	if err != nil {
		log.Fatal("error while parsing template", err)
	}

	var buf bytes.Buffer
	err = t.Execute(&buf, data)
	if err != nil {
		return fmt.Sprintf("Template execution error: %v", err)
	}

	return buf.String()
}
