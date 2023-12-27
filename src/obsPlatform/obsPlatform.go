package obsPlatform

import (
	"bytes"
	"encoding/json"
	"fmt"
	"ingester/config"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

var (
	ObservabilityPlatform string       // ObservabilityPlatform contains the information on the platform in use.
	httpClient            *http.Client // httpClient is the HTTP client used to send data to the Observability Platform.
	grafanaLokiPostUrl        string       // grafanaPostUrl is the URL used to send data to Grafana Loki.
)

func Init(cfg config.Configuration) error {
	httpClient = &http.Client{Timeout: 5 * time.Second}
	if cfg.ObservabilityPlatform.GrafanaCloud.LogsURL != "" {

		grafanaLogsUsername := cfg.ObservabilityPlatform.GrafanaCloud.LogsUsername
		grafanaAccessToken := cfg.ObservabilityPlatform.GrafanaCloud.CloudAccessToken
		grafanaLokiUrl := cfg.ObservabilityPlatform.GrafanaCloud.LogsURL
		// Use strings.TrimPrefix to remove the schemes "http://" and "https://"
		grafanaLokiUrl = strings.TrimPrefix(grafanaLokiUrl, "http://")
		grafanaLokiUrl = strings.TrimPrefix(grafanaLokiUrl, "https://")
		grafanaLokiPostUrl = fmt.Sprintf(
			"https://%s:%s@%s",
			grafanaLogsUsername,
			grafanaAccessToken,
			grafanaLokiUrl,
		)
	}
	return nil
}

// SendToPlatform sends observability data to the appropriate platform.
func SendToPlatform(data map[string]interface{}) {
	switch ObservabilityPlatform {
	case "GRAFANA":
		if data["endpoint"] == "openai.chat.completions" {
			if data["response"] != nil {
				metrics := []string{
					fmt.Sprintf(`doku_llm,environment=%v,applicationName=%v,source=%v,model=%v completionTokens=%v`, data["environment"], data["applicationName"],data["sourceLanguage"], data["model"], data["completionTokens"]),
					fmt.Sprintf(`doku_llm,environment=%v,applicationName=%v,source=%v,model=%v promptTokens=%v`, data["environment"], data["applicationName"], data["sourceLanguage"], data["model"], data["promptTokens"]),
					fmt.Sprintf(`doku_llm,environment=%v,applicationName=%v,source=%v,model=%v totalTokens=%v`, data["environment"], data["applicationName"], data["sourceLanguage"], data["model"], data["totalTokens"]),
					fmt.Sprintf(`doku_llm,environment=%v,applicationName=%v,source=%v,model=%v requestDuration=%v`, data["environment"], data["applicationName"], data["sourceLanguage"], data["model"], data["requestDuration"]),
					fmt.Sprintf(`doku_llm,environment=%v,applicationName=%v,source=%v,model=%v usageCost=%v`, data["environment"], data["applicationName"], data["sourceLanguage"], data["model"], data["usageCost"]),
				}
				metricsBody := strings.Join(metrics, "\n")

				req, err := http.NewRequest("POST", URL, bytes(metricsBody))
				req.Header.Set("Content-Type", "text/plain")
				req.Header.Set("Authorization", bearer)
				client := &http.Client{}
				resp, err := client.Do(req)

				if err != nil {
					panic(err)
				}
				defer resp.Body.Close()
				
				logs := []byte(fmt.Sprintf("{\"streams\": [{\"stream\": {\"environment\": \"%v\", \"applicationName\": \"%v\", \"source\": \"%v\", \"model\": \"%v\", \"prompt\": \"%v\" }, \"values\": [[\"%s\", \"%v\"]]}]}",data["environment"], data["applicationName"],data["sourceLanguage"], data["model"], data["prompt"], strconv.FormatInt(time.Now().UnixNano(), 10), data["response"]))
				http.Post(grafanaLokiPostUrl, "application/json", bytes.NewBuffer(logs))
		}
	}

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
