package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/prometheus/client_golang/prometheus"
	cnf "github.com/MSJantana/zabbix-exporter-3000/config"
	zbx "github.com/MSJantana/zabbix-exporter-3000/zabbix"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var reg = regexp.MustCompile("[^a-zA-Z0-9]+")

func cleanUpName(name string) string {
	return reg.ReplaceAllString(strings.ToLower(name), "")
}

func uniqueSlice(intSlice []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range intSlice {
		if !keys[entry] {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

func registerMetric(metric *prometheus.GaugeVec) error {
	if cnf.StrictRegister {
		return prometheus.Register(metric)
	}
	// Use MustRegister, que pode panic, só se for StrictRegister=true
	prometheus.MustRegister(metric)
	return nil
}

type ZabbixResponse struct {
	Result []map[string]interface{} `json:"result"`
}

func queryZabbix() ([]map[string]interface{}, error) {
	items, err := zbx.Session.Do(zbx.Query)
	if err != nil {
		return nil, err
	}

	var resp ZabbixResponse
	err = json.Unmarshal(items.Body, &resp)
	if err != nil {
		return nil, err
	}

	if len(resp.Result) == 0 {
		return nil, errors.New("empty response from Zabbix; check ZE3000_ZABBIX_QUERY")
	}

	return resp.Result, nil
}

func parseLabels(rawLabels []string) (promLabels, complexLabels, avgLabels []string) {
	for _, vl := range rawLabels {
		if strings.Contains(vl, ">") {
			promLabels = append(promLabels, strings.ReplaceAll(vl, ">", "_"))
			complexLabels = append(complexLabels, vl)
		} else {
			promLabels = append(promLabels, vl)
			avgLabels = append(avgLabels, vl)
		}
	}
	return
}

func extractMetricNamesAndDesc(results []map[string]interface{}) (names, desc []string) {
	for k, result := range results {
		if nameRaw, ok := result[cnf.MetricNameField]; ok {
			name, ok2 := nameRaw.(string)
			if !ok2 {
				name = "invalid_name"
			}
			cleanName := cleanUpName(name)
			names = append(names, cleanName)
		}

		if helpRaw, ok := result[cnf.MetricHelpField]; ok {
			help, ok2 := helpRaw.(string)
			if ok2 && help != "" {
				desc = append(desc, help)
			} else {
				desc = append(desc, "NA_"+strconv.Itoa(k))
			}
		} else {
			desc = append(desc, "NA_"+strconv.Itoa(k))
		}
	}
	return
}

func createMetrics(names, desc, promLabels []string) (map[string]*prometheus.GaugeVec, *prometheus.GaugeVec, error) {
	metricsMap := make(map[string]*prometheus.GaugeVec)
	var singleMetric *prometheus.GaugeVec

	uniqNames := uniqueSlice(names)
	uniqDesc := uniqueSlice(desc)

	if len(uniqNames) != len(uniqDesc) {
		log.Printf("WARNING: Number of metrics and description not equal")

		if len(uniqNames) < len(uniqDesc) {
			return nil, nil, errors.New("insufficient unique metrics; use more unique ZE3000_METRIC_NAME_FIELD or ZE3000_SINGLE_METRIC_NAME=true")
		}

		// "Heal" by padding descriptions
		for i := len(uniqDesc); i < len(uniqNames); i++ {
			uniqDesc = append(uniqDesc, "NA_"+strconv.Itoa(i))
		}
	}

	if cnf.SingleMetric {
		fullName := cnf.MetricNamePrefix
		singleMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: cnf.MetricNamespace,
			Subsystem: cnf.MetricSubsystem,
			Name:      fullName,
			Help:      cnf.SingleMetricHelp,
		}, promLabels)

		err := registerMetric(singleMetric)
		if err != nil {
			return nil, nil, err
		}

	} else {
		for i, name := range uniqNames {
			fullName := cnf.MetricNamePrefix
			if cnf.MetricNameField != "" {
				fullName = fullName + "_" + name
			}

			gauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{
				Namespace: cnf.MetricNamespace,
				Subsystem: cnf.MetricSubsystem,
				Name:      fullName,
				Help:      uniqDesc[i],
			}, promLabels)

			metricsMap[name] = gauge
		}

		for _, metric := range metricsMap {
			err := registerMetric(metric)
			if err != nil {
				return nil, nil, err
			}
		}
	}

	return metricsMap, singleMetric, nil
}

// RecordMetricsWithContext roda a coleta de métricas, pode ser cancelado pelo context
func RecordMetricsWithContext(ctx context.Context) error {
	rawLabels := strings.Split(cnf.MetricLabels, ",")
	promLabels, complexLabels, avgLabels := parseLabels(rawLabels)

	log.Print("Labels that will be produced      :", promLabels)
	log.Print("Complex labels that will be parsed:", complexLabels)
	log.Print("Plain labels that will be parsed  :", avgLabels)

	results, err := queryZabbix()
	if err != nil {
		return err
	}

	names, desc := extractMetricNamesAndDesc(results)

	metricsMap, singleMetric, err := createMetrics(names, desc, promLabels)
	if err != nil {
		return err
	}

	refreshSec, err := strconv.Atoi(cnf.SourceRefresh)
	if err != nil {
		refreshSec = 60
	}

	go func() {
		ticker := time.NewTicker(time.Duration(refreshSec) * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Print("Metrics collection stopped by context cancellation")
				return
			case <-ticker.C:
				results, err := queryZabbix()
				if err != nil {
					log.Printf("Error querying Zabbix: %v", err)
					continue
				}

				for _, result := range results {
					labelsWithValues := make(map[string]string)

					for _, vAvg := range avgLabels {
						if val, ok := result[vAvg]; ok {
							if sval, ok2 := val.(string); ok2 {
								labelsWithValues[vAvg] = sval
							} else {
								labelsWithValues[vAvg] = "NA"
							}
						} else {
							labelsWithValues[vAvg] = "NA"
						}
					}

					for _, vCplx := range complexLabels {
						promLabel := strings.ReplaceAll(vCplx, ">", "_")
						path := strings.Split(vCplx, ">")

						if val, ok := result[path[0]]; ok {
							if slice, ok := val.([]interface{}); ok && len(slice) > 0 {
								for _, cplx := range slice {
									if subMap, ok := cplx.(map[string]interface{}); ok {
										if subVal, ok := subMap[path[1]]; ok {
											if s, ok := subVal.(string); ok {
												labelsWithValues[promLabel] = s
											} else {
												labelsWithValues[promLabel] = "NA"
											}
										} else {
											labelsWithValues[promLabel] = "NA"
										}
									}
								}
							} else {
								labelsWithValues[promLabel] = "NA"
							}
						} else {
							labelsWithValues[promLabel] = "NA"
						}
					}

					var f float64
					if val, ok := result[cnf.MetricValue]; ok {
						if sval, ok2 := val.(string); ok2 {
							f, _ = strconv.ParseFloat(sval, 64)
						}
					}

					if cnf.SingleMetric {
						singleMetric.With(labelsWithValues).Set(f)
					} else {
						if nameRaw, ok := result[cnf.MetricNameField]; ok {
							if nameStr, ok2 := nameRaw.(string); ok2 {
								cleanName := cleanUpName(nameStr)
								if metricVec, ok := metricsMap[cleanName]; ok {
									metricVec.With(labelsWithValues).Set(f)
								}
							}
						}
					}
				}
			}
		}
	}()

	return nil
}
