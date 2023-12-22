package obsPlatform

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	ObservabilityPlatform string       // ObservabilityPlatform contains the information on the platform in use.
	httpClient            *http.Client // httpClient is the HTTP client used to send data to the Observability Platform.
	grafanaPostUrl        string       // grafanaPostUrl is the URL used to send data to Grafana Loki.
)

func Init() {
	httpClient = &http.Client{Timeout: 5 * time.Second}
	ObservabilityPlatform = os.Getenv("OBSERVABILITY_PLATFORM")
	if ObservabilityPlatform == "GRAFANA" {
		grafanaLogsUsername := os.Getenv("GRAFANA_LOGS_USER")
		grafanaAccessToken := os.Getenv("GRAFANA_ACCESS_TOKEN")
		grafanaLokiUrl := os.Getenv("GRAFANA_LOKI_URL")
		// Use strings.TrimPrefix to remove the schemes "http://" and "https://"
		grafanaLokiUrl = strings.TrimPrefix(grafanaLokiUrl, "http://")
		grafanaLokiUrl = strings.TrimPrefix(grafanaLokiUrl, "https://")
		grafanaPostUrl = fmt.Sprintf(
			"https://%s:%s@%s",
			grafanaLogsUsername,
			grafanaAccessToken,
			grafanaLokiUrl,
		)
	}
}

// SendToPlatform sends observability data to the appropriate platform.
func SendToPlatform(data map[string]interface{}) {
	switch ObservabilityPlatform {
	case "GRAFANA":
		streamData := make(map[string]interface{})
		fields := []string{
			"environment",
			"sourceLanguage",
			"applicationName",
			"completionTokens",
			"promptTokens",
			"totalTokens",
			"finishReason",
			"requestDuration",
			"usageCost",
			"model",
			"prompt",
			"imageSize",
			"image",
			"revisedPrompt",
			"audioVoice",
			"finetuneJobId",
			"finetuneJobStatus",
		}

		// Populate the streamData only with non-nil fields
		for _, field := range fields {
			if value, ok := data[field]; ok && value != nil {
				streamData[field] = value
			}
		}

		// Creating the values array
		timestamp := strconv.FormatInt(time.Now().UnixNano(), 10)
		response := data["response"]
		if response == nil {
			response = "Non Text Generation data collected by Doku"
		}
		values := [][]interface{}{{timestamp, response}}

		// Construct the log stream object
		stream := map[string]interface{}{
			"stream": streamData,
			"values": values,
		}

		// Construct the final logStream structure with the streams array
		logStream := map[string]interface{}{
			"streams": []interface{}{stream},
		}

		// Marshal the logStream into JSON
		logs, _ := json.MarshalIndent(logStream, "", "  ") // Using Indent for pretty print

		_, err := httpClient.Post(grafanaPostUrl, "application/json", bytes.NewBuffer(logs))
		if err != nil {
			log.Error().Err(err).Msg("Error sending data to Grafana Loki")
		}
		log.Info().Msg("Data sent to Grafana Loki")

	case "Datadog":
		fmt.Println("DataDog Observability Platform")
	case "New Relic":
		fmt.Println("New Relic Observability Platform")
	case "Dynatrace":
		fmt.Println("Dynatrace Observability Platform")
	default:
		fmt.Println("Unknown Observability Platform")
	}
}
