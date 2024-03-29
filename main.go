package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/alertmanager/api/v2/models"
	"github.com/sensu-community/sensu-plugin-sdk/sensu"
	v2 "github.com/sensu/sensu-go/api/core/v2"
	"github.com/sensu/sensu-go/types"
)

// Config represents the check plugin config.
type Config struct {
	sensu.PluginConfig
	AlertmanagerAPIURL          string
	AgentAPIURL                 string
	AlertmanagerExcludeAlerts   string
	AlertmanagerExternalURL     string
	AlertmanagerLabelEntity     string
	AlertmanagerLabelSelectors  string
	AlertmanagerExcludeLabels   string
	AlertmanagerTargetAlertname string
	SensuProxyEntity            string
	SensuAgentEntity            string
	SensuNamespace              string
	SensuHandler                string
	SensuExtraLabel             string
	SensuExtraAnnotation        string
	RewriteAnnotation           string
	SensuAutoClose              bool
	SensuAutoCloseLabel         string
	APIBackendPass              string
	APIBackendUser              string
	APIBackendKey               string
	APIBackendHost              string
	APIBackendPort              int
	Secure                      bool
	TrustedCAFile               string
	InsecureSkipVerify          bool
	Protocol                    string
	ProxyEntity                 string
	LabelSelector               map[string]string
	ExcludeLabels               map[string]string
}

// Auth represents the authentication info
type Auth struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
}

var (
	tlsConfig tls.Config

	plugin = Config{
		PluginConfig: sensu.PluginConfig{
			Name:     "sensu-alertmanager-events",
			Short:    "Sensu check for alert maanager events",
			Keyspace: "sensu.io/plugins/sensu-alertmanager-events/config",
		},
	}

	options = []*sensu.PluginConfigOption{
		{
			Path:      "alert-manager-api-url",
			Env:       "ALERT_MANAGER_API_URL",
			Argument:  "alert-manager-api-url",
			Shorthand: "a",
			Default:   "http://alertmanager-main.monitoring:9093/api/v2/alerts",
			Usage:     "The URL for the Agent to connect to Alert Manager",
			Value:     &plugin.AlertmanagerAPIURL,
		},
		{
			Path:      "agent-api-url",
			Env:       "AGENT_API_URL",
			Argument:  "agent-api-url",
			Shorthand: "A",
			Default:   "http://127.0.0.1:3031/events",
			Usage:     "The URL for the Agent API used to send events",
			Value:     &plugin.AgentAPIURL,
		},
		{
			Path:      "alert-manager-exclude-alert-list",
			Env:       "ALERT_MANAGER_EXCLUDE_ALERT_LIST",
			Argument:  "alert-manager-exclude-alert-list",
			Shorthand: "x",
			Default:   "Watchdog,",
			Usage:     "Alert Manager alerts to be excluded. split by comma.",
			Value:     &plugin.AlertmanagerExcludeAlerts,
		},
		{
			Path:      "alert-manager-external-url",
			Env:       "ALERT_MANAGER_EXTERNAL_URL",
			Argument:  "alert-manager-external-url",
			Shorthand: "e",
			Default:   "",
			Usage:     "Alert Manager External URL",
			Value:     &plugin.AlertmanagerExternalURL,
		},
		{
			Path:      "alert-manager-cluster-label-entity",
			Env:       "ALERT_MANAGER_CLUSTER_LABEL_ENTITY",
			Argument:  "alert-manager-cluster-label-entity",
			Shorthand: "c",
			Default:   "",
			Usage:     "Alert Manager label that represent a cluster entity inside Sensu",
			Value:     &plugin.AlertmanagerLabelEntity,
		},
		{
			Path:      "alert-manager-label-selectors",
			Env:       "ALERT_MANAGER_LABEL_SELECTORS",
			Argument:  "alert-manager-label-selectors",
			Shorthand: "l",
			Default:   "",
			Usage:     "Query for Alertmanager LabelSelectors (e.g. alertname=TargetDown,environment=dev)",
			Value:     &plugin.AlertmanagerLabelSelectors,
		},
		{
			Path:      "alert-manager-exclude-labels",
			Env:       "ALERT_MANAGER_EXCLUDE_LABELS",
			Argument:  "alert-manager-exclude-labels",
			Shorthand: "L",
			Default:   "",
			Usage:     "Query for Alertmanager Exclude Labels (e.g. alertname=TargetDown,environment=dev)",
			Value:     &plugin.AlertmanagerExcludeLabels,
		},
		{
			Path:      "alert-manager-target-alertname",
			Env:       "ALERT_MANAGER_TARGET_ALERTNAME",
			Argument:  "alert-manager-target-alertname",
			Shorthand: "T",
			Default:   "TargetDown",
			Usage:     "Alert name for Targets in prometheus. It creates a link in label prometheus_targets_url",
			Value:     &plugin.AlertmanagerTargetAlertname,
		},
		{
			Path:      "sensu-proxy-entity",
			Env:       "SENSU_PROXY_ENTITY",
			Argument:  "sensu-proxy-entity",
			Shorthand: "E",
			Default:   "",
			Usage:     "Overwrite Proxy Entity in Sensu",
			Value:     &plugin.SensuProxyEntity,
		},
		{
			Path:      "sensu-agent-entity",
			Env:       "HOSTNAME",
			Argument:  "sensu-agent-entity",
			Shorthand: "",
			Default:   "",
			Usage:     "Overwrite Subscriptions with Agent Entity Hostname when using proxy entity agent",
			Value:     &plugin.SensuAgentEntity,
		},
		{
			Path:      "sensu-namespace",
			Env:       "SENSU_NAMESPACE",
			Argument:  "sensu-namespace",
			Shorthand: "n",
			Default:   "default",
			Usage:     "Configure which Sensu Namespace wll be used by alerts",
			Value:     &plugin.SensuNamespace,
		},
		{
			Path:      "sensu-handler",
			Env:       "SENSU_HANDLER",
			Argument:  "sensu-handler",
			Shorthand: "H",
			Default:   "default,",
			Usage:     "Sensu Handler for alerts. Split by commas",
			Value:     &plugin.SensuHandler,
		},
		{
			Path:      "sensu-extra-label",
			Env:       "SENSU_EXTRA_LABEL",
			Argument:  "sensu-extra-label",
			Shorthand: "",
			Default:   "",
			Usage:     "Add Extra Sensu Check Label in alert send to Sensu Agent API. Format: labelName=labelValue Or for multiple values labelName=labelValue,ExtraLabel=ExtraValue",
			Value:     &plugin.SensuExtraLabel,
		},
		{
			Path:      "sensu-extra-annotation",
			Env:       "SENSU_EXTRA_ANNOTATION",
			Argument:  "sensu-extra-annotation",
			Shorthand: "",
			Default:   "",
			Usage:     "Add Extra Sensu Check Annotation in alert send to Sensu Agent API. Format: annotationName=annotationValue Or for multiples use comma: annotationName=annotationValue,extraTwo=extraValue",
			Value:     &plugin.SensuExtraAnnotation,
		},
		{
			Path:      "rewrite-annotation",
			Env:       "",
			Argument:  "rewrite-annotation",
			Shorthand: "",
			Default:   "",
			Usage:     "Rewrite Annotation from prometheus rules to sensu annotation format to work with sensu plugins. Format: opsgenie_priority=sensu.io/plugins/sensu-opsgenie-handler/config/priority Or for multiples use comma: opsgenie_priority=sensu.io/plugins/sensu-opsgenie-handler/config/priority,extraTwo=extraValue",
			Value:     &plugin.RewriteAnnotation,
		},
		{
			Path:      "auto-close-sensu",
			Env:       "",
			Argument:  "auto-close-sensu",
			Shorthand: "C",
			Default:   false,
			Usage:     "Configure it to Auto Close if event doesn't match any Alerts from Alert Manager. Please configure others api-backend-* options before enable this flag",
			Value:     &plugin.SensuAutoClose,
		},
		{
			Path:      "auto-close-sensu-label",
			Env:       "AUTO_CLOSE_SENSU_LABEL",
			Argument:  "auto-close-sensu-label",
			Shorthand: "",
			Default:   "",
			Usage:     "Configure it to Auto Close if event doesn't match any Alerts from Alert Manager and with these label. e. {\"cluster\":\"k8s-dev\"}",
			Value:     &plugin.SensuAutoCloseLabel,
		},
		{
			Path:      "api-backend-user",
			Env:       "SENSU_API_USER",
			Argument:  "api-backend-user",
			Shorthand: "u",
			Default:   "admin",
			Usage:     "Sensu Go Backend API User",
			Value:     &plugin.APIBackendUser,
		},
		{
			Path:      "api-backend-pass",
			Env:       "SENSU_API_PASSWORD",
			Argument:  "api-backend-pass",
			Shorthand: "P",
			Default:   "P@ssw0rd!",
			Usage:     "Sensu Go Backend API Password",
			Value:     &plugin.APIBackendPass,
		},
		{
			Path:      "api-backend-key",
			Env:       "SENSU_API_KEY",
			Argument:  "api-backend-key",
			Shorthand: "k",
			Default:   "",
			Usage:     "Sensu Go Backend API Key",
			Value:     &plugin.APIBackendKey,
		},
		{
			Path:      "api-backend-host",
			Env:       "",
			Argument:  "api-backend-host",
			Shorthand: "B",
			Default:   "127.0.0.1",
			Usage:     "Sensu Go Backend API Host (e.g. 'sensu-backend.example.com')",
			Value:     &plugin.APIBackendHost,
		},
		{
			Path:      "api-backend-port",
			Env:       "",
			Argument:  "api-backend-port",
			Shorthand: "p",
			Default:   8080,
			Usage:     "Sensu Go Backend API Port (e.g. 4242)",
			Value:     &plugin.APIBackendPort,
		},
		{
			Path:      "secure",
			Env:       "",
			Argument:  "secure",
			Shorthand: "s",
			Default:   false,
			Usage:     "Use TLS connection to API",
			Value:     &plugin.Secure,
		},
		{
			Path:      "insecure-skip-verify",
			Env:       "",
			Argument:  "insecure-skip-verify",
			Shorthand: "i",
			Default:   false,
			Usage:     "skip TLS certificate verification (not recommended!)",
			Value:     &plugin.InsecureSkipVerify,
		},
		{
			Path:      "trusted-ca-file",
			Env:       "",
			Argument:  "trusted-ca-file",
			Shorthand: "t",
			Default:   "",
			Usage:     "TLS CA certificate bundle in PEM format",
			Value:     &plugin.TrustedCAFile,
		},
	}
)

func main() {
	check := sensu.NewGoCheck(&plugin.PluginConfig, options, checkArgs, executeCheck, false)
	check.Execute()
}

func checkArgs(event *types.Event) (int, error) {
	if plugin.AlertmanagerLabelEntity != "" && plugin.SensuProxyEntity != "" {
		return sensu.CheckStateWarning, fmt.Errorf("Cannot use --alert-manager-cluster-label-entity %s and --sensu-proxy-entity %s together", plugin.AlertmanagerLabelEntity, plugin.SensuProxyEntity)
	}
	// Default proxy entity name is kubernetes resources
	plugin.ProxyEntity = "KubernetesResource"
	// Second way is using one label from alert manager like cluster
	if plugin.AlertmanagerLabelEntity != "" {
		plugin.ProxyEntity = "AlertmanagerLabelEntity"
	}
	// then force to use one proxy entity using a specific proxy entity name
	if plugin.SensuProxyEntity != "" {
		plugin.ProxyEntity = "SensuProxyEntity"
	}
	// LabelsSelectors
	if plugin.AlertmanagerLabelSelectors != "" {
		plugin.LabelSelector = parseLabelArg(plugin.AlertmanagerLabelSelectors)
	}
	// ExcludeLabels
	if plugin.AlertmanagerExcludeLabels != "" {
		plugin.ExcludeLabels = parseLabelArg(plugin.AlertmanagerExcludeLabels)
	}
	// For Sensu Backend Connections
	if plugin.Secure {
		plugin.Protocol = "https"
	} else {
		plugin.Protocol = "http"
	}
	if len(plugin.TrustedCAFile) > 0 {
		caCertPool, err := v2.LoadCACerts(plugin.TrustedCAFile)
		if err != nil {
			return sensu.CheckStateWarning, fmt.Errorf("Error loading specified CA file")
		}
		tlsConfig.RootCAs = caCertPool
	}
	tlsConfig.InsecureSkipVerify = plugin.InsecureSkipVerify

	// tlsConfig.BuildNameToCertificate()
	tlsConfig.CipherSuites = v2.DefaultCipherSuites

	// check if format is correct
	if plugin.SensuExtraLabel != "" {
		if !strings.Contains(plugin.SensuExtraLabel, "=") {
			return sensu.CheckStateWarning, fmt.Errorf("Please use Format: Label=Value. Wrong format --sensu-extra-label %s", plugin.SensuExtraLabel)
		}
	}
	if plugin.SensuExtraAnnotation != "" {
		if !strings.Contains(plugin.SensuExtraAnnotation, "=") {
			return sensu.CheckStateWarning, fmt.Errorf("Please use Format: Annotation=Value. Wrong format --sensu-extra-annotation %s", plugin.SensuExtraAnnotation)
		}
	}

	if plugin.RewriteAnnotation != "" {
		if !strings.Contains(plugin.RewriteAnnotation, "=") {
			return sensu.CheckStateWarning, fmt.Errorf("Please use Format: Annotation=Value. Wrong format --rewrite-annotation %s", plugin.RewriteAnnotation)
		}

	}

	return sensu.CheckStateOK, nil
}

func executeCheck(event *types.Event) (int, error) {
	// log.Printf("executing check with %s, %s, %s", plugin.AlertmanagerAPIURL, plugin.AgentAPIURL, plugin.AlertmanagerLabelEntity)
	alerts, err := getAlertManagerEvents()
	if err != nil {
		return sensu.CheckStateCritical, err
	}
	var AlertmanagerExcludeAlertList []string
	if strings.Contains(plugin.AlertmanagerExcludeAlerts, ",") {
		AlertmanagerExcludeAlertList = strings.Split(plugin.AlertmanagerExcludeAlerts, ",")
	}
	numAlerts := len(alerts)
	log.Printf("Number of Alerts found: %d", numAlerts)
	// create an event into sensu
	var countErrors, countErrorsClosing int
	// parallel
	results := make(chan error, 2)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		if numAlerts != 0 {
			countErrors = processAlertsToSensuAgent(alerts, AlertmanagerExcludeAlertList)
		}
		results <- nil
	}()
	go func() {
		defer wg.Done()
		// Compare sensu events with alerts and resolved it
		if plugin.SensuAutoClose {
			var autherr error
			auth := Auth{}
			if len(plugin.APIBackendKey) == 0 {
				auth, autherr = authenticate()

				if autherr != nil {
					// return sensu.CheckStateUnknown, autherr
					results <- autherr
				}
			}
			events, err := getEvents(auth, plugin.SensuNamespace)
			if err != nil {
				// return sensu.CheckStateCritical, err
				results <- err
			}
			numEvents := len(events)
			log.Printf("Number of Events found: %d\n", numEvents)
			if numEvents != 0 {
				countErrorsClosing = processSensuEventsToClose(events, alerts)
			}
		}
		results <- nil
	}()
	wg.Wait()
	close(results)
	for err := range results {
		if err != nil {
			return sensu.CheckStateCritical, err
		}
	}

	if countErrors != 0 {
		return sensu.CheckStateCritical, fmt.Errorf("cannot create all events in sensu")
	}
	if countErrorsClosing != 0 {
		return sensu.CheckStateWarning, fmt.Errorf("cannot close all events in sensu backend")
	}
	return sensu.CheckStateOK, nil
}

func processAlertsToSensuAgent(alerts []models.GettableAlert, AlertmanagerExcludeAlertList []string) int {
	count := 0
	results := make(chan int, len(alerts))
	var wg sync.WaitGroup
	for _, a := range alerts {
		wg.Add(1)
		go func(a models.GettableAlert) {
			defer wg.Done()
			for k, v := range a.Labels {
				if k == "alertname" && !stringInSlice(v, AlertmanagerExcludeAlertList) {

					alertName, sensuAlertName, clusterName, kubernetesResource, labels, annotations := alertDetails(a)
					output := printAlert(a, alertName)
					var proxyEntityName string
					switch plugin.ProxyEntity {
					case "KubernetesResource":
						proxyEntityName = kubernetesResource

					case "AlertmanagerLabelEntity":
						proxyEntityName = clusterName

					case "SensuProxyEntity":
						proxyEntityName = plugin.SensuProxyEntity

					default:
						proxyEntityName = kubernetesResource
						if kubernetesResource == "" {
							proxyEntityName = removeSpecialCharacters(alertName)
						}
					}
					if *a.Status.State != "active" {
						// if not active, don't post it to sensu
						log.Printf("Not Sending Alert %s", a.Labels["alertname"])
						continue
					}
					if plugin.SensuExtraLabel != "" {
						extraLabels := parseLabelArg(plugin.SensuExtraLabel)
						// log.Println(extraLabels)
						labels = mergeStringMaps(labels, extraLabels)
					}
					if plugin.SensuExtraAnnotation != "" {
						extraAnnotations := parseLabelArg(plugin.SensuExtraAnnotation)
						// log.Println(extraAnnotations)
						annotations = mergeStringMaps(annotations, extraAnnotations)
					}
					log.Printf("Sending Alert %s to %s", sensuAlertName, proxyEntityName)
					err := sendAlertsToSensu(alertName, sensuAlertName, proxyEntityName, output, labels, annotations, 2)
					if err != nil {
						log.Printf("Error sending Alert %s to %s", sensuAlertName, proxyEntityName)
						results <- 1
						// count++
					}

				}
			}
		}(a)
	}
	wg.Wait()
	close(results)
	for r := range results {
		count += r
	}
	return count
}

func processSensuEventsToClose(events []*v2.Event, alerts []models.GettableAlert) int {
	count := 0
	results := make(chan int, len(events))
	var wg sync.WaitGroup
	for _, e := range events {
		wg.Add(1)
		go func(e *v2.Event) {
			defer wg.Done()
			for k, v := range e.Check.Labels {
				if k == "fingerprint" {
					if !checkFingerprint(alerts, v) {
						log.Printf("Closing %s \n", e.Check.Name)
						output := fmt.Sprintf("Resolved Automatically \n %s", e.Check.Output)
						err := sendAlertsToSensu(e.Check.Labels["alertname"], e.Check.Name, e.Check.ProxyEntityName, output, e.Check.Labels, e.Check.Annotations, 0)
						if err != nil {
							log.Printf("Error closing %s \n", e.Check.Name)
							results <- 1
							// count++
						}
					}
				}
			}
		}(e)
	}
	wg.Wait()
	close(results)
	for r := range results {
		count += r
	}
	return count
}

// get alerts from AM
func getAlertManagerEvents() ([]models.GettableAlert, error) {
	body, err := getAlerts()
	alerts := []models.GettableAlert{}
	if err != nil {
		return alerts, fmt.Errorf("Failed to get alert manager alerts: %v", err)
	}

	_ = json.Unmarshal(body, &alerts)

	result := filterAlerts(alerts)

	return result, nil
}

// send alerts to Sensu Agent API
func sendAlertsToSensu(alertName, sensuAlertName, proxyEntity, output string, labels, annotations map[string]string, sensuStatus uint32) error {
	var SensuHandlers []string
	if strings.Contains(plugin.SensuHandler, ",") {
		SensuHandlers = strings.Split(plugin.SensuHandler, ",")
	}
	agentEntity := fmt.Sprintf("entity:%s", plugin.SensuAgentEntity)
	payload := &v2.Event{
		Check: &v2.Check{
			Output:          output,
			Command:         removeSpecialCharacters(alertName),
			Status:          sensuStatus,
			ProxyEntityName: proxyEntity,
			Subscriptions:   []string{agentEntity},
			Handlers:        SensuHandlers,
			ObjectMeta: v2.ObjectMeta{
				Name:        removeSpecialCharacters(sensuAlertName),
				Namespace:   plugin.SensuNamespace,
				Labels:      labels,
				Annotations: annotations,
				CreatedBy:   plugin.Name,
			},
		},
	}
	err := submitEventAgentAPI(payload)
	if err != nil {
		return fmt.Errorf("[ERROR] postOrGet %s", err)
	}
	return nil

}

// Print check output
func printAlert(alert models.GettableAlert, alertName string) (value string) {
	var valueLabels, valueAnnotations, status string
	for k, v := range alert.Labels {
		valueLabels += fmt.Sprintf(" - %s: %s \n", k, v)
	}
	for k, v := range alert.Annotations {
		valueAnnotations += fmt.Sprintf(" - %s: %s \n", k, v)
	}
	status = *alert.Status.State
	value = "Labels: \n"
	value += valueLabels
	value += "Annotations: \n"
	value += valueAnnotations

	value += "Alert Manager: \n"
	value += fmt.Sprintf(" - status: %s \n", status)
	if plugin.AlertmanagerExternalURL != "" {
		value += fmt.Sprintf(" - source: %s", printAlertManagerURL(alertName))
	}
	value += fmt.Sprintf("Prometheus:\n - source: %s \n", alert.GeneratorURL)
	return value
}

func printAlertManagerURL(alertName string) string {
	sourceURL := url.QueryEscape(fmt.Sprintf("{alertname=\"%s\"}", alertName))
	return fmt.Sprintf("%s/#/alerts?silenced=false&inhibited=false&active=true&filter=%s \n", plugin.AlertmanagerExternalURL, sourceURL)
}

// Parse alert data
func alertDetails(alert models.GettableAlert) (alertName, sensuAlertName, cluster, kubernetesResource string, label, annotation map[string]string) {
	labels := make(map[string]string)
	annotations := make(map[string]string)
	var withNamespace bool
	var withExtraName bool
	for k, v := range alert.Labels {
		if k == plugin.AlertmanagerLabelEntity {
			cluster = v
		}
		if k == "alertname" {
			alertName = v
		}
		if k == "namespace" {
			withNamespace = true
		}
		if k == "job_name" || k == "statefulset" || k == "daemonset" || k == "deployment" || k == "service" || k == "pod" {
			withExtraName = true
		}
		if k == "node" {
			withNamespace = false
		}
		key := k
		// if plugin.RewriteAnnotation != "" {
		// 	rule := makeRewriteAnnotation(plugin.RewriteAnnotation)
		// 	tmp, err := rewriteAnnotation(key, rule)
		// 	if err == nil && tmp != "" {
		// 		key = tmp
		// 	}
		// }
		labels[key] = v
	}
	// extra label
	labels[plugin.Name] = "owner"
	for k, v := range alert.Annotations {
		key := k
		if plugin.RewriteAnnotation != "" {
			rule := makeRewriteAnnotation(plugin.RewriteAnnotation)
			tmp, err := rewriteAnnotation(key, rule)
			if err == nil && tmp != "" {
				key = tmp
			}
		}
		annotations[key] = v
	}
	labels["fingerprint"] = *alert.Fingerprint
	// add extra annotation
	if checkURL(string(alert.GeneratorURL)) {
		annotations["prometheus_url"] = string(alert.GeneratorURL)
	}
	if plugin.AlertmanagerExternalURL != "" && checkURL(plugin.AlertmanagerExternalURL) {
		annotations["alertmanager_url"] = printAlertManagerURL(alertName)
	}
	// if alertname TargetDown, we want to include a prometheus targets page to make easy to debug
	if alertName == plugin.AlertmanagerTargetAlertname && checkURL(string(alert.GeneratorURL)) {
		promTargets, err := url.Parse(string(alert.GeneratorURL))
		if err == nil {
			promTargets.RawQuery = ""
			promTargets.Path = "targets"
			annotations["prometheus_targets_url"] = fmt.Sprintln(promTargets)
		}
	}
	sensuAlertName = removeSpecialCharacters(alertName)
	if withNamespace {
		sensuAlertName = fmt.Sprintf("%s-%s", alertName, labels["namespace"])
		onlyPod := true
		if withExtraName {
			for k, v := range labels {
				if k == "job_name" {
					sensuAlertName = fmt.Sprintf("%s-%s-%s", alertName, labels["namespace"], v)
					onlyPod = false
					kubernetesResource = v
				}
				if k == "daemonset" {
					sensuAlertName = fmt.Sprintf("%s-%s-%s", alertName, labels["namespace"], v)
					onlyPod = false
					kubernetesResource = v
				}
				if k == "statefulset" {
					sensuAlertName = fmt.Sprintf("%s-%s-%s", alertName, labels["namespace"], v)
					onlyPod = false
					kubernetesResource = v
				}
				if k == "deployment" {
					sensuAlertName = fmt.Sprintf("%s-%s-%s", alertName, labels["namespace"], v)
					onlyPod = false
					kubernetesResource = v
				}
				if k == "service" {
					sensuAlertName = fmt.Sprintf("%s-%s-%s", alertName, labels["namespace"], v)
					onlyPod = false
					kubernetesResource = v
				}
				if k == "node" {
					onlyPod = false
					kubernetesResource = v
				}
			}
			if onlyPod {
				sensuAlertName = fmt.Sprintf("%s-%s-%s", alertName, labels["namespace"], labels["pod"])
				kubernetesResource = labels["pod"]
			}
		}
	} else {
		if labels["node"] != "" {
			sensuAlertName = fmt.Sprintf("%s-%s", alertName, labels["node"])
		}
	}
	return alertName, sensuAlertName, cluster, kubernetesResource, labels, annotations
}

// get http alerts from AM
func getAlerts() (result []byte, err error) {
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	req, err := http.NewRequest(http.MethodGet, plugin.AlertmanagerAPIURL, nil)
	if err != nil {
		log.Printf("[ERROR]  GET %s", err)
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[ERROR] client %s", err)
		return nil, err
	}
	result, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[ERROR] ReadAll %s", err)
		return nil, err
	}
	defer resp.Body.Close()
	return result, nil
}

// check if fingerprint matches
func checkFingerprint(alerts []models.GettableAlert, f string) bool {
	for _, a := range alerts {
		if *a.Fingerprint == f {
			return true
		}
	}
	return false
}

// post http content to Sensu agent API
func submitEventAgentAPI(event *v2.Event) error {

	encoded, _ := json.Marshal(event)
	resp, err := http.Post(plugin.AgentAPIURL, "application/json", bytes.NewBuffer(encoded))
	if err != nil {
		return fmt.Errorf("Failed to post event to %s failed: %v", plugin.AgentAPIURL, err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("POST of event to %s failed with status %v\nevent: %s", plugin.AgentAPIURL, resp.Status, string(encoded))
	}

	return nil
}

// authenticate funcion to work with api-backend-* flags
func authenticate() (Auth, error) {
	var auth Auth
	client := http.DefaultClient
	client.Transport = http.DefaultTransport

	if plugin.Secure {
		client.Transport.(*http.Transport).TLSClientConfig = &tlsConfig
	}

	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s://%s:%d/auth", plugin.Protocol, plugin.APIBackendHost, plugin.APIBackendPort),
		nil,
	)
	if err != nil {
		return auth, fmt.Errorf("error generating auth request: %v", err)
	}

	req.SetBasicAuth(plugin.APIBackendUser, plugin.APIBackendPass)

	resp, err := client.Do(req)
	if err != nil {
		return auth, fmt.Errorf("error executing auth request: %v", err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return auth, fmt.Errorf("error reading auth response: %v", err)
	}

	if strings.HasPrefix(string(body), "Unauthorized") {
		return auth, fmt.Errorf("authorization failed for user %s", plugin.APIBackendUser)
	}

	err = json.NewDecoder(bytes.NewReader(body)).Decode(&auth)

	if err != nil {
		trim := 64
		return auth, fmt.Errorf("error decoding auth response: %v\nFirst %d bytes of response: %s", err, trim, trimBody(body, trim))
	}

	return auth, err
}

// get events from sensu-backend-api
func getEvents(auth Auth, namespace string) ([]*types.Event, error) {
	client := http.DefaultClient
	client.Transport = http.DefaultTransport

	url := fmt.Sprintf("%s://%s:%d/api/core/v2/namespaces/%s/events", plugin.Protocol, plugin.APIBackendHost, plugin.APIBackendPort, namespace)
	events := []*types.Event{}

	if plugin.Secure {
		client.Transport.(*http.Transport).TLSClientConfig = &tlsConfig
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return events, fmt.Errorf("error creating GET request for %s: %v", url, err)
	}

	if len(plugin.APIBackendKey) == 0 {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", auth.AccessToken))
	} else {
		req.Header.Set("Authorization", fmt.Sprintf("Key %s", plugin.APIBackendKey))
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return events, fmt.Errorf("error executing GET request for %s: %v", url, err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return events, fmt.Errorf("error reading response body during getEvents: %v", err)
	}

	err = json.Unmarshal(body, &events)
	if err != nil {
		trim := 64
		return events, fmt.Errorf("error unmarshalling response during getEvents: %v\nFirst %d bytes of response: %s", err, trim, trimBody(body, trim))
	}
	result := filterEvents(events)

	return result, err
}

// filter events from sensu-backend-api to look only events created by this plugin
func filterEvents(events []*types.Event) []*types.Event {
	var result, partialResult []*types.Event
	for _, event := range events {
		if event.Check.ObjectMeta.Labels[plugin.Name] == "owner" && event.Check.Status != 0 {
			result = append(result, event)
		}
	}
	onlyTheseLabels := make(map[string]string)
	if plugin.SensuAutoCloseLabel != "" {
		// if there are specific label to be used as entity name
		err := json.Unmarshal([]byte(plugin.SensuAutoCloseLabel), &onlyTheseLabels)
		// fmt.Println(onlyTheseLabels)
		if err != nil {
			log.Println("fail in SensuAutoCloseLabel Unmarshal")
			return partialResult
		}
		for _, event := range result {
			if searchLabels(event, onlyTheseLabels) {
				partialResult = append(partialResult, event)
			}
		}
		return partialResult
	}
	return result
}

// parse selector labels to filter then in Alert Manager alerts endpoint
func parseLabelArg(labelArg string) map[string]string {
	labels := map[string]string{}

	pairs := strings.Split(labelArg, ",")

	for _, pair := range pairs {
		parts := strings.Split(pair, "=")
		if len(parts) == 2 {
			labels[parts[0]] = parts[1]
		}
	}

	return labels
}

// filter alerts using map[string]string from plugin.LabelSelector
func filterAlerts(alerts []models.GettableAlert) (result []models.GettableAlert) {

	for _, alert := range alerts {
		selected := true
		// check if it find labels selector
		for key, value := range plugin.LabelSelector {
			if alert.Labels[key] != value {
				selected = false
				break
			}
		}
		// exclude alerts based on labels
		// if found, remove from alert manager list
		for key, value := range plugin.ExcludeLabels {
			if alert.Labels[key] == value {
				selected = false
				break
			}
		}
		if selected {
			result = append(result, alert)
		}

	}

	return result
}

// Use to exclude some alerts from alert manager before sending it to sensu agent api
func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

// used to clean errors output
func trimBody(body []byte, maxlen int) string {
	if len(string(body)) < maxlen {
		maxlen = len(string(body))
	}

	return string(body)[0:maxlen]
}

func searchLabels(event *types.Event, labels map[string]string) bool {
	if len(labels) == 0 {
		return false
	}
	count := 0
	for key, value := range labels {
		if event.Labels != nil {
			for k, v := range event.Labels {
				if k == key && v == value {
					count++
				}
			}
		}
		if event.Entity.Labels != nil {
			for k, v := range event.Entity.Labels {
				if k == key && v == value {
					count++
				}
			}
		}
		if event.Check.Labels != nil {
			for k, v := range event.Check.Labels {
				if k == key && v == value {
					count++
				}
			}
		}
		if count >= len(labels) {
			return true
		}
	}
	return false
}

func removeSpecialCharacters(s string) string {
	// regex to remove all nonalphanumeric characters and keep -
	re := regexp.MustCompile(`[^A-Za-z0-9.-.]+`)
	value := re.ReplaceAllString(s, "-")
	// remove all - in the check prefix
	value = strings.TrimPrefix(value, "-")
	// remove all - in the check suffix
	value = strings.TrimSuffix(value, "-")
	// fmt.Println(value)
	return value
}

func mergeStringMaps(left, right map[string]string) map[string]string {
	for k, v := range right {
		// fmt.Println(left[k])
		if left[k] == "" {
			left[k] = v
		}
	}
	return left
}

func rewriteAnnotation(s string, rule map[string]string) (string, error) {
	if rule[s] != "" {
		return rule[s], nil
	}
	return "", fmt.Errorf("not found")
}

func makeRewriteAnnotation(s string) map[string]string {
	rewrite := make(map[string]string)
	if strings.Contains(s, ",") {
		splited := strings.Split(s, ",")
		for _, v := range splited {
			a, b := splitString(v, "=")
			if a != "" && b != "" {
				rewrite[a] = b
			}
		}
	} else {
		a, b := splitString(s, "=")
		if a != "" && b != "" {
			rewrite[a] = b
		}
	}
	return rewrite
}

func splitString(s, div string) (string, string) {
	if div != "" {
		splited := strings.Split(s, div)
		if len(splited) == 2 {
			return splited[0], splited[1]
		}
	}
	return "", ""
}

func checkURL(str string) bool {
	u, err := url.Parse(str)
	return err == nil && u.Scheme != "" && u.Host != ""
}
