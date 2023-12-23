package cost

import (
	"encoding/json"
	"fmt"
	"os"
)

var Pricing PricingModel

// PricingModel is the pricing information for the different models and features.
type PricingModel struct {
    Embeddings map[string]float64                            `json:"embeddings"`
    Images     map[string]map[string]map[string]float64      `json:"images"`
    Chat       map[string]struct {
        PromptPrice     float64 `json:"promptPrice"`
        CompletionPrice float64 `json:"completionPrice"`
    } `json:"chat"`
}

// validatePricingData validates the pricing data for the different models and features.
func validatePricingData(pricingModel PricingModel) error {
    // Example validation for Embeddings pricing
    if len(pricingModel.Embeddings) == 0 {
        return fmt.Errorf("Embeddings pricing data is not defined")
    }
    
    // Validate the Images pricing, which has nested maps
    for model, qualityMap := range pricingModel.Images {
        if len(qualityMap) == 0 {
            return fmt.Errorf("image pricing data for model '%s' is not defined in the JSON File", model)
        }
        for quality, sizeMap := range qualityMap {
            if len(sizeMap) == 0 {
                return fmt.Errorf("image pricing data for model '%s', quality '%s' is not defined in the JSON File", model, quality)
            }
        }
    }

    // Validate the Chat pricing 
    for model, chatPricing := range pricingModel.Chat {
        if chatPricing.PromptPrice == 0 {
            return fmt.Errorf("Prompt Tokens pricing data for model '%s' is not defined in the JSON File", model)
        } else if chatPricing.CompletionPrice ==0 {
			return fmt.Errorf("Completion Tokens pricing data for model '%s' is not defined in the JSON File", model)
		}
    }

    return nil
}

// LoadPricing loads the pricing information from the given file.
func LoadPricing(filename string) error {
    bytes, err := os.ReadFile(filename)
    if err != nil {
        return fmt.Errorf("Failed to read costing file: %w", err)
    }

    if err = json.Unmarshal(bytes, &Pricing); err != nil {
        return fmt.Errorf("Failed to unmarshal costing JSON: %w", err)
    }

    // Perform validation after the PricingModel has been populated.
    if err = validatePricingData(Pricing); err != nil {
        return err
    }

    return nil
}

// calculateEmbeddingsCost calculates the cost for embeddings based on the model and prompt tokens.
func CalculateEmbeddingsCost(promptTokens float64, model string) (float64, error) {
    price, _ := Pricing.Embeddings[model]
    return (promptTokens / 1000) * price, nil
}

// calculateImageCost calculates the cost for images based on the model, image size, and quality.
func CalculateImageCost(model, imageSize, quality string) (float64, error) {
    models, _ := Pricing.Images[model]
    qualities, _ := models[quality]
    price, _ := qualities[imageSize]
    return price, nil
}

// calculateChatCost calculates the cost for chat based on the model, prompt tokens, and completion tokens.
func CalculateChatCost(promptTokens, completionTokens float64, model string) (float64, error) {
    chatModel, _ := Pricing.Chat[model]
    cost := ((promptTokens/1000) * chatModel.PromptPrice) + ((completionTokens/1000) * chatModel.CompletionPrice)
    return cost, nil
}
