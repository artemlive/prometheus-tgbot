package main

import (
	"fmt"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/alertmanager/pkg/parse"
	"github.com/prometheus/alertmanager/client"
	"github.com/prometheus/alertmanager/types"
	"github.com/prometheus/client_golang/api"
	"github.com/prometheus/common/model"
	"time"
	"context"
	"log"
	"math"
)

/**
 * Subdivideb by 1024
 */
const (
	Kb = iota
	Mb
	Gb
	Tb
	Pb
	Eb
	Zb
	Yb
	Information_Size_MAX
)

/**
 * Subdivided by 10000
 */
const (
	K = iota
	M
	G
	T
	P
	E
	Z
	Y
	Scale_Size_MAX
)

type Alerts struct {
	Alerts            []Alert                `json:"alerts"`
	CommonAnnotations map[string]interface{} `json:"commonAnnotations"`
	CommonLabels      map[string]interface{} `json:"commonLabels"`
	ExternalURL       string                 `json:"externalURL"`
	GroupKey          int                    `json:"groupKey"`
	GroupLabels       map[string]interface{} `json:"groupLabels"`
	Receiver          string                 `json:"receiver"`
	Status            string                 `json:"status"`
	Version           int                    `json:"version"`
}

type Alert struct {
	Annotations  map[string]interface{} `json:"annotations"`
	EndsAt       string                 `json:"sendsAt"`
	GeneratorURL string                 `json:"generatorURL"`
	Labels       map[string]interface{} `json:"labels"`
	StartsAt     string                 `json:"startsAt"`
}

// Parse a list of labels, filter by alertname
func parseMatchers(inputLabels []string) ([]labels.Matcher, error) {
	matchers := make([]labels.Matcher, 0)

	for _, v := range inputLabels {
		name, value, matchType, err := parse.Input(v)
		if err != nil {
			return []labels.Matcher{}, err
		}

		matchers = append(matchers, labels.Matcher{
			Type:  labels.MatchType(matchType),
			Name:  name,
			Value: value,
		})
	}

	return matchers, nil
}

// Only valid for when you are going to add a silence
func TypeMatchers(matchers []labels.Matcher) (types.Matchers, error) {
	typeMatchers := types.Matchers{}
	for _, matcher := range matchers {
		typeMatcher, err := TypeMatcher(matcher)
		if err != nil {
			return types.Matchers{}, err
		}
		typeMatchers = append(typeMatchers, &typeMatcher)
	}
	return typeMatchers, nil
}

// Only valid for when you are going to add a silence
// Doesn't allow negative operators
func TypeMatcher(matcher labels.Matcher) (types.Matcher, error) {
	name := model.LabelName(matcher.Name)
	typeMatcher := types.NewMatcher(name, matcher.Value)

	switch matcher.Type {
	case labels.MatchEqual:
		typeMatcher.IsRegex = false
	case labels.MatchRegexp:
		typeMatcher.IsRegex = true
	default:
		return types.Matcher{}, fmt.Errorf("invalid match type for creation operation: %s", matcher.Type)
	}
	return *typeMatcher, nil
}

func silentAlert(silenceTime, alertname, user string) (string, error) {
	var silenceID = ""
	matchers, err := parseMatchers([]string{fmt.Sprintf("alertname=%s", alertname)})
	if err != nil {
		return silenceID, fmt.Errorf("%s", err)
	}
	typeMatchers, err := TypeMatchers(matchers)
	if err != nil {
		return silenceID, fmt.Errorf("%s", err)
	}
	startsAt := time.Now().UTC()
	d, err := model.ParseDuration(silenceTime)
	if err != nil {
		log.Printf("Can't parse duration: %s", err)
	}
	if d == 0 {
		return silenceID, fmt.Errorf("%s", err)
	}

	endsAt := startsAt.Add(time.Duration(d))
	silence := types.Silence{
		Matchers:  typeMatchers,
		StartsAt:  startsAt,
		EndsAt:    endsAt,
		CreatedBy: user,
		Comment:   "disabled via telegram",
	}
	apiClient, err := api.NewClient(api.Config{Address: cfg.AlertManagerAddress})
	if err != nil {
		return silenceID, fmt.Errorf("can't init api client: %s", err)
	}

	silenceAPI := client.NewSilenceAPI(apiClient)
	filterString := fmt.Sprintf("{alertname=%s}", alertname)
	fetchedSilences, err := silenceAPI.List(context.Background(), filterString)
	if err != nil {
		return silenceID, fmt.Errorf("can't get silences list: %s", err)
	}
	displaySilences := []types.Silence{}
	for _, silence := range fetchedSilences {
		if silence.EndsAt.Before(time.Now()) {
			continue
		}
		if silence.EndsAt.After(time.Now()) {
			displaySilences = append(displaySilences, *silence)
		}

	}
	if len(displaySilences) > 0 {
		return silenceID, fmt.Errorf("silence with the same name already active: %s, active to: %s", displaySilences[0].ID, displaySilences[0].EndsAt)
	}
	silenceID, err = silenceAPI.Set(context.Background(), silence)
	if err != nil {
		return silenceID, fmt.Errorf("can't set silence: %s", err)
	}
	return silenceID, nil

}

func RoundPrec(x float64, prec int) float64 {
	if math.IsNaN(x) || math.IsInf(x, 0) {
		return x
	}

	sign := 1.0
	if x < 0 {
		sign = -1
		x *= -1
	}

	var rounder float64
	pow := math.Pow(10, float64(prec))
	intermed := x * pow
	_, frac := math.Modf(intermed)

	if frac >= 0.5 {
		rounder = math.Ceil(intermed)
	} else {
		rounder = math.Floor(intermed)
	}

	return rounder / pow * sign
}

func listAlerts(name string) ([]*client.ExtendedAlert, error) {
	apiClient, err := api.NewClient(api.Config{Address: cfg.AlertManagerAddress})
	if err != nil {
		return nil, fmt.Errorf("can't init api client: %s", err)
	}
	alertsApiClient := client.NewAlertAPI(apiClient)
	//matchers, err := parseMatchers([]string{fmt.Sprintf("alertname=%s", "dsp")})
	list, err := alertsApiClient.List(context.Background(),fmt.Sprintf("{alertname=~.*%s.*}", "dsp"), false, false)
	if err != nil {
		return nil, fmt.Errorf("%s", err)
	}
	return list, nil
}