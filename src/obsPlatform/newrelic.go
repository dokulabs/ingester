package obsPlatform

import (
	"bytes"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

func configureNewRelicData(data map[string]interface{}) {
	// The current time for the timestamp field.
	currentTime := strconv.FormatInt(time.Now().Unix(), 10)

	if data["endpoint"] == "openai.chat.completions" || data["endpoint"] == "openai.completions" || data["endpoint"] == "cohere.generate" || data["endpoint"] == "cohere.chat" || data["endpoint"] == "cohere.summarize" || data["endpoint"] == "anthropic.completions" {
		if data["finishReason"] == nil {
			data["finishReason"] = "null"
		}

		jsonMetrics := []string{
			fmt.Sprintf(`{
			"name": "doku.LLM.Completion.Tokens",
			"type": "gauge",
			"value": %f,
			"timestamp": %s,
			"attributes": {"environment": "%v", "endpoint": "%v", "applicationName": "%v", "source": "%v", "model": "%v", "finishReason": "%v"}
		}`, float64(data["completionTokens"].(int)), currentTime, data["environment"], data["endpoint"], data["applicationName"], data["sourceLanguage"], data["model"], data["finishReason"]),
			fmt.Sprintf(`{
			"name": "doku.LLM.Prompt.Tokens",
			"type": "gauge",
			"value": %f,
			"timestamp": %s,
			"attributes": {"environment": "%v", "endpoint": "%v", "applicationName": "%v", "source": "%v", "model": "%v", "finishReason": "%v"}
		}`, float64(data["promptTokens"].(int)), currentTime, data["environment"], data["endpoint"], data["applicationName"], data["sourceLanguage"], data["model"], data["finishReason"]),
			fmt.Sprintf(`{
			"name": "doku.LLM.Total.Tokens",
			"type": "gauge",
			"value": %f,
			"timestamp": %s,
			"attributes": {"environment": "%v", "endpoint": "%v", "applicationName": "%v", "source": "%v", "model": "%v", "finishReason": "%v"}
		}`, float64(data["totalTokens"].(int)), currentTime, data["environment"], data["endpoint"], data["applicationName"], data["sourceLanguage"], data["model"], data["finishReason"]),
			fmt.Sprintf(`{
			"name": "doku.LLM.Request.Duration",
			"type": "gauge",
			"value": %v,
			"timestamp": %s,
			"attributes": {"environment": "%v", "endpoint": "%v", "applicationName": "%v", "source": "%v", "model": "%v", "finishReason": "%v"}
		}`, data["requestDuration"], currentTime, data["environment"], data["endpoint"], data["applicationName"], data["sourceLanguage"], data["model"], data["finishReason"]),
			fmt.Sprintf(`{
			"name": "doku.LLM.Usage.Cost",
			"type": "gauge",
			"value": %v,
			"timestamp": %s,
			"attributes": {"environment": "%v", "endpoint": "%v", "applicationName": "%v", "source": "%v", "model": "%v", "finishReason": "%v"}
		}`, data["usageCost"], currentTime, data["environment"], data["endpoint"], data["applicationName"], data["sourceLanguage"], data["model"], data["finishReason"]),
		}

		// Join the individual metric strings into a comma-separated string and enclose in a JSON array.
		jsonData := fmt.Sprintf(`[{"metrics": [%s]}]`, strings.Join(jsonMetrics, ","))

		sendTelemetryNewRelic(jsonData, newRelicLicenseKey, "Api-Key", newRelicMetricsUrl, "POST")

	} else if data["endpoint"] == "openai.embeddings" || data["endpoint"] == "cohere.embed" {
		if data["endpoint"] == "openai.embeddings" {
			jsonMetrics := []string{
				fmt.Sprintf(`{
					"name": "doku.LLM.Prompt.Tokens",
					"type": "gauge",
					"value": %v,
					"timestamp": %s,
					"attributes": { "environment": "%v", "endpoint": "%v", "applicationName": "%v", "source": "%v", "model": "%v"}
				}`, float64(data["promptTokens"].(int)), currentTime, data["environment"], data["endpoint"], data["applicationName"], data["sourceLanguage"], data["model"]),
				fmt.Sprintf(`{
					"name": "doku.LLM.Total.Tokens",
					"type": "gauge",
					"value": %v,
					"timestamp": %s,
					"attributes": {"environment": "%v", "endpoint": "%v", "applicationName": "%v", "source": "%v", "model": "%v"}
				}`, float64(data["totalTokens"].(int)), currentTime, data["environment"], data["endpoint"], data["applicationName"], data["sourceLanguage"], data["model"]),
				fmt.Sprintf(`{
					"name": "doku.LLM.Request.Duration",
					"type": "gauge",
					"value": %v,
					"timestamp": %s,
					"attributes": { "environment": "%v", "endpoint": "%v", "applicationName": "%v", "source": "%v", "model": "%v"}
				}`, data["requestDuration"], currentTime, data["environment"], data["endpoint"], data["applicationName"], data["sourceLanguage"], data["model"]),
				fmt.Sprintf(`{
					"name": "doku.LLM.Usage.Cost",
					"type": "gauge",
					"value": %v,
					"timestamp": %s,
					"attributes": { "environment": "%v", "endpoint": "%v", "applicationName": "%v", "source": "%v", "model": "%v"}
				}`, data["usageCost"], currentTime, data["environment"], data["endpoint"], data["applicationName"], data["sourceLanguage"], data["model"]),
			}

			// Join the individual metric strings into a comma-separated string and enclose in a JSON array.
			jsonData := fmt.Sprintf(`[{"metrics": [%s]}]`, strings.Join(jsonMetrics, ","))

			sendTelemetryNewRelic(jsonData, newRelicLicenseKey, "Api-Key", newRelicMetricsUrl, "POST")
		} else {
			jsonMetrics := []string{
				fmt.Sprintf(`{
					"name": "doku.LLM.Prompt.Tokens",
					"type": "gauge",
					"value": %v,
					"timestamp": %s,
					"attributes": {"environment": "%v", "endpoint": "%v", "applicationName": "%v", "source": "%v", "model": "%v"}
				}`, float64(data["promptTokens"].(int)), currentTime, data["environment"], data["endpoint"], data["applicationName"], data["sourceLanguage"], data["model"]),
				fmt.Sprintf(`{
					"name": "doku.LLM.Request.Duration",
					"type": "gauge",
					"value": %v,
					"timestamp": %s,
					"attributes": {"environment": "%v", "endpoint": "%v", "applicationName": "%v", "source": "%v", "model": "%v"}
				}`, data["requestDuration"], currentTime, data["environment"], data["endpoint"], data["applicationName"], data["sourceLanguage"], data["model"]),
				fmt.Sprintf(`{
					"name": "doku.LLM.Usage.Cost",
					"type": "gauge",
					"value": %v,
					"timestamp": %s,
					"attributes": {"environment": "%v", "endpoint": "%v", "applicationName": "%v", "source": "%v", "model": "%v"}
				}`, data["usageCost"], currentTime, data["environment"], data["endpoint"], data["applicationName"], data["sourceLanguage"], data["model"]),
			}

			// Join the individual metric strings into a comma-separated string and enclose in a JSON array.
			jsonData := fmt.Sprintf(`[{"metrics": [%s]}]`, strings.Join(jsonMetrics, ","))

			sendTelemetryNewRelic(jsonData, newRelicLicenseKey, "Api-Key", newRelicMetricsUrl, "POST")
		}
	} else if data["endpoint"] == "openai.fine_tuning" {
		jsonMetrics := []string{
			fmt.Sprintf(`{
					"name": "doku.LLM.Request.Duration",
					"type": "gauge",
					"value": %v,
					"timestamp": %s,
					"attributes": {"environment": "%v", "endpoint": "%v", "applicationName": "%v", "source": "%v", "model": "%v", "finetuneJobId": "%v"}
				}`, data["requestDuration"], currentTime, data["environment"], data["endpoint"], data["applicationName"], data["sourceLanguage"], data["model"], data["finetuneJobId"]),
		}
		// Join the individual metric strings into a comma-separated string and enclose in a JSON array.
		jsonData := fmt.Sprintf(`[{"metrics": [%s]}]`, strings.Join(jsonMetrics, ","))

		sendTelemetryNewRelic(jsonData, newRelicLicenseKey, "Api-Key", newRelicMetricsUrl, "POST")
	} else if data["endpoint"] == "openai.images.create" || data["endpoint"] == "openai.images.create.variations" {
		jsonMetrics := []string{
			fmt.Sprintf(`{
					"name": "doku_llm.RequestDuration",
					"type": "gauge",
					"value": %v,
					"timestamp": %s,
					"attributes": {
						"environment": "%v",
						"endpoint": "%v",
						"applicationName": "%v",
						"source": "%v",
						"model": "%v",
						"imageSize": "%v",
						"imageQuality": "%v"
					}
				}`, data["requestDuration"], currentTime, data["environment"], data["endpoint"], data["applicationName"], data["sourceLanguage"], data["model"], data["imageSize"], data["imageQuality"]),
			fmt.Sprintf(`{
					"name": "doku_llm.UsageCost",
					"type": "gauge",
					"value": %v,
					"timestamp": %s,
					"attributes": {
						"environment": "%v",
						"endpoint": "%v",
						"applicationName": "%v",
						"source": "%v",
						"model": "%v",
						"imageSize": "%v",
						"imageQuality": "%v"
					}
				}`, data["usageCost"], currentTime, data["environment"], data["endpoint"], data["applicationName"], data["sourceLanguage"], data["model"], data["imageSize"], data["imageQuality"]),
		}

		// Join the individual metric strings into a comma-separated string and enclose in a JSON array.
		jsonData := fmt.Sprintf(`[{"metrics": [%s]}]`, strings.Join(jsonMetrics, ","))

		sendTelemetryNewRelic(jsonData, newRelicLicenseKey, "Api-Key", newRelicMetricsUrl, "POST")
	} else if data["endpoint"] == "openai.audio.speech.create" {
		jsonMetrics := []string{
			fmt.Sprintf(`{
					"name": "doku_llm.RequestDuration",
					"type": "gauge",
					"value": %v,
					"timestamp": %s,
					"attributes": {"environment": "%v", "endpoint": "%v", "applicationName": "%v", "source": "%v", "model": "%v", "audioVoice": "%v"}
				}`, data["requestDuration"], currentTime, data["environment"], data["endpoint"], data["applicationName"], data["sourceLanguage"], data["model"], data["audioVoice"]),
			fmt.Sprintf(`{
					"name": "doku_llm.UsageCost",
					"type": "gauge",
					"value": %v,
					"timestamp": %s,
					"attributes": {"environment": "%v", "endpoint": "%v", "applicationName": "%v", "source": "%v", "model": "%v", "audioVoice": "%v"}
				}`, data["usageCost"], currentTime, data["environment"], data["endpoint"], data["applicationName"], data["sourceLanguage"], data["model"], data["audioVoice"]),
		}

		// Join the individual metric strings into a comma-separated string and enclose in a JSON array.
		jsonData := fmt.Sprintf(`[{"metrics": [%s]}]`, strings.Join(jsonMetrics, ","))

		sendTelemetryNewRelic(jsonData, newRelicLicenseKey, "Api-Key", newRelicMetricsUrl, "POST")
	}
}

func sendTelemetryNewRelic(telemetryData, authHeader string, headerKey string, url string, requestType string) error {

	req, err := http.NewRequest(requestType, url, bytes.NewBufferString(telemetryData))
	if err != nil {
		return fmt.Errorf("Error creating request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(fmt.Sprintf("%s", headerKey), authHeader)

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("Error sending request to %v", url)
	} else if resp.StatusCode == 404 {
		return fmt.Errorf("Provided URL %v is not valid", url)
	} else if resp.StatusCode == 401 {
		return fmt.Errorf("Provided credentials are not valid")
	}

	defer resp.Body.Close()

	log.Info().Msgf("Successfully exported data to %v", url)
	return nil
}
