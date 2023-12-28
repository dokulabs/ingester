package obsPlatform

import (
	"bytes"
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
	grafanaPromUrl        string       // grafanaPrometheusUrl is the URL used to send data to Grafana Prometheus.
	grafanaPromUsername   string       // grafanaPrometheusUsername is the username used to send data to Grafana Prometheus.
	grafanaLokiUrl        string       // grafanaPostUrl is the URL used to send data to Grafana Loki.
	grafanaLokiUsername   string       // grafanaLokiUsername is the username used to send data to Grafana Loki.
	grafanaAccessToken    string       // grafanaAccessToken is the access token used to send data to Grafana.
)

func Init(cfg config.Configuration) error {
	httpClient = &http.Client{Timeout: 5 * time.Second}
	if cfg.ObservabilityPlatform.GrafanaCloud.LokiURL != "" {
		grafanaPromUrl = cfg.ObservabilityPlatform.GrafanaCloud.PromURL
		grafanaPromUsername = cfg.ObservabilityPlatform.GrafanaCloud.PromUsername
		grafanaLokiUrl = cfg.ObservabilityPlatform.GrafanaCloud.LokiURL
		grafanaLokiUsername = cfg.ObservabilityPlatform.GrafanaCloud.LokiUsername
		grafanaAccessToken = cfg.ObservabilityPlatform.GrafanaCloud.AccessToken
	}
	return nil
}

// SendToPlatform sends observability data to the appropriate platform.
func SendToPlatform(data map[string]interface{}) {
	if grafanaLokiUrl != "" {
		if data["endpoint"] == "openai.chat.completions" || data["endpoint"] == "openai.completions" || data["endpoint"] == "anthropic.completions" {
			if data["response"] != nil && data["completionTokens"] != nil {
				metrics := []string{
					fmt.Sprintf(`doku_llm,environment=%v,endpoint=%v,applicationName=%v,source=%v,model=%v completionTokens=%v`, data["environment"], data["endpoint"], data["applicationName"], data["sourceLanguage"], data["model"], data["completionTokens"]),
					fmt.Sprintf(`doku_llm,environment=%v,endpoint=%v,applicationName=%v,source=%v,model=%v promptTokens=%v`, data["environment"], data["endpoint"], data["applicationName"], data["sourceLanguage"], data["model"], data["promptTokens"]),
					fmt.Sprintf(`doku_llm,environment=%v,endpoint=%v,applicationName=%v,source=%v,model=%v totalTokens=%v`, data["environment"], data["endpoint"], data["applicationName"], data["sourceLanguage"], data["model"], data["totalTokens"]),
					fmt.Sprintf(`doku_llm,environment=%v,endpoint=%v,applicationName=%v,source=%v,model=%v requestDuration=%v`, data["environment"], data["endpoint"], data["applicationName"], data["sourceLanguage"], data["model"], data["requestDuration"]),
					fmt.Sprintf(`doku_llm,environment=%v,endpoint=%v,applicationName=%v,source=%v,model=%v usageCost=%v`, data["environment"], data["endpoint"], data["applicationName"], data["sourceLanguage"], data["model"], data["usageCost"]),
				}
				var metricsBody = []byte(strings.Join(metrics, "\n"))
				authHeader := fmt.Sprintf("Bearer %v:%v", grafanaPromUsername, grafanaAccessToken)
				err := sendTelemetry(metricsBody, authHeader, grafanaPromUrl, "POST")
				if err != nil {
					log.Error().Err(err).Msgf("Error sending data to Grafana Cloud Prometheus")
				}

				logs := []byte(fmt.Sprintf("{\"streams\": [{\"stream\": {\"environment\": \"%v\",\"endpoint\": \"%v\", \"applicationName\": \"%v\", \"source\": \"%v\", \"model\": \"%v\", \"prompt\": \"%v\" }, \"values\": [[\"%s\", \"%v\"]]}]}", data["environment"], data["endpoint"], data["applicationName"], data["sourceLanguage"], data["model"], data["prompt"], strconv.FormatInt(time.Now().UnixNano(), 10), data["response"]))
				authHeader = fmt.Sprintf("Bearer %v:%v", grafanaLokiUsername, grafanaAccessToken)
				err = sendTelemetry(logs, authHeader, grafanaLokiUrl, "POST")
				if err != nil {
					log.Error().Err(err).Msgf("Error sending data to Grafana Cloud Loki")
				}
			}
		}
	} else {
		fmt.Println("No Observability Platform configured.")
	}
}

func sendTelemetry(telemetryData []byte, authHeader string, url string, requestType string) error {

	req, err := http.NewRequest(requestType, url, bytes.NewBuffer(telemetryData))
	if err != nil {
		return fmt.Errorf("Error creating request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader)

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
