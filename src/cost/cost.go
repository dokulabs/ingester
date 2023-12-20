package cost

// calculateEmbeddingsCost calculates the cost for embeddings based on the model and prompt tokens.
func CalculateEmbeddingsCost(promptTokens float64, model string) float64 {
	// Define the pricing information for different models
	prices := map[string]float64{
		"text-embedding-ada-002": 0.0001,
		"ada":                    0.0001,
		"text-ada-001":           0.0001,
	}

	promptPrice := prices[model]
	cost := (promptTokens / 1000) * promptPrice
	return cost
}

// calculateImageCost calculates the cost for image generations based on the model, imageSize, and quality.
func CalculateImageCost(model string, imageSize string, quality string) float64 {
	prices := map[string]map[string]map[string]float64{
		"dalle-3": {
			"standard": {"1024x1024": 0.040, "1024x1792": 0.020, "1792x1024": 0.080},
			"hd":       {"1024x1024": 0.080, "1024x1792": 0.120, "1792x1024": 0.120},
		},
		"dalle-2": {
			"standard": {"1024x1024": 0.020, "512x512": 0.018, "256x256": 0.016},
		},
	}

	modelPrices, _ := prices[model]
	cost, _ := modelPrices[quality][imageSize]
	return cost
}

// calculateChatCost calculates the cost for chat completions based on the model, prompt tokens, and completion tokens.
func CalculateChatCost(promptTokens, completionTokens float64, model string) float64 {
	// Define the pricing information for different models
	prices := map[string]struct{ promptPrice, completionPrice float64 }{
		"ada":                       {0.0004, 0.0004},
		"text-ada-001":              {0.0004, 0.0004},
		"babbage":                   {0.0005, 0.0005},
		"babbage-002":               {0.0004, 0.0004},
		"text-babbage-001":          {0.0005, 0.0005},
		"curie":                     {0.0020, 0.0020},
		"text-curie-001":            {0.0020, 0.0020},
		"davinci":                   {0.0200, 0.0200},
		"davinci-002":               {0.0020, 0.0020},
		"text-davinci-001":          {0.0200, 0.0200},
		"text-davinci-002":          {0.0200, 0.0200},
		"text-davinci-003":          {0.0200, 0.0200},
		"gpt-3.5-turbo":             {0.0010, 0.0020},
		"gpt-3.5-turbo-instruct":    {0.0015, 0.0020},
		"gpt-4":                     {0.03, 0.06},
		"gpt-4-32k":                 {0.06, 0.12},
		"gpt-4-1106-preview":        {0.01, 0.03},
		"gpt-4-1106-vision-preview": {0.01, 0.03},
		"claude-instant-1":          {0.00163, 0.00551},
		"claude-2":                  {0.01102, 0.03268},
		"command":                   {0.00150, 0.00200},
		"command-nightly":           {0.00150, 0.00200},
		"command-light":             {0.00150, 0.00200},
		"command-light-nightly":     {0.00150, 0.00200},
	}

	price, ok := prices[model]
	if ok {
		cost := (promptTokens/1000)*price.promptPrice + (completionTokens/1000)*price.completionPrice
		return cost
	}
	return 0
}
