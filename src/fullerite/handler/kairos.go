package handler

import (
	"fullerite/metric"

	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	l "github.com/Sirupsen/logrus"
)

// Kairos handler
type Kairos struct {
	BaseHandler
	server string
	port   string
}

// KairosMetric structure
type KairosMetric struct {
	Name       string            `json:"name"`
	Timestamp  int64             `json:"timestamp"`
	MetricType string            `json:"type"`
	Value      float64           `json:"value"`
	Tags       map[string]string `json:"tags"`
}

// NewKairos returns a new Kairos handler
func NewKairos(
	channel chan metric.Metric,
	initialInterval int,
	initialBufferSize int,
	initialTimeout time.Duration,
	log *l.Entry) *Kairos {

	inst := new(Kairos)
	inst.name = "Kairos"

	inst.interval = initialInterval
	inst.maxBufferSize = initialBufferSize
	inst.timeout = initialTimeout
	inst.log = log
	inst.channel = channel

	return inst
}

// Configure the Kairos handler
func (k *Kairos) Configure(configMap map[string]interface{}) {
	if server, exists := configMap["server"]; exists {
		k.server = server.(string)
	} else {
		k.log.Error("There was no server specified for the Kairos Handler, there won't be any emissions")
	}

	if port, exists := configMap["port"]; exists {
		k.port = fmt.Sprint(port)
	} else {
		k.log.Error("There was no port specified for the Kairos Handler, there won't be any emissions")
	}
	k.configureCommonParams(configMap)
}

// Server returns the Kairos server's hostname or IP address
func (k *Kairos) Server() string {
	return k.server
}

// Port returns the Kairos server's port number
func (k *Kairos) Port() string {
	return k.port
}

// Run runs the handler main loop
func (k *Kairos) Run() {
	k.run(k.emitMetrics)
}

func (k *Kairos) convertToKairos(incomingMetric metric.Metric) (datapoint KairosMetric) {
	km := new(KairosMetric)
	km.Name = k.Prefix() + incomingMetric.Name
	km.Value = incomingMetric.Value
	km.MetricType = "double"
	km.Timestamp = time.Now().Unix() * 1000 // Kairos require timestamps to be milliseconds
	km.Tags = incomingMetric.GetDimensions(k.DefaultDimensions())
	return *km
}

func (k *Kairos) emitMetrics(metrics []metric.Metric) bool {
	k.log.Info("Starting to emit ", len(metrics), " metrics")

	if len(metrics) == 0 {
		k.log.Warn("Skipping send because of an empty payload")
		return false
	}

	series := make([]KairosMetric, 0, len(metrics))
	for _, m := range metrics {
		series = append(series, k.convertToKairos(m))
	}

	payload, err := json.Marshal(series)
	if err != nil {
		k.log.Error("Failed marshaling datapoints to Kairos format")
		k.log.Error("Dropping Kairos datapoints ", series)
		return false
	}

	apiURL := fmt.Sprintf("http://%s:%s/api/v1/datapoints", k.server, k.port)
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(payload))
	if err != nil {
		k.log.Error("Failed to create a request to API url ", apiURL)
		return false
	}
	req.Header.Set("Content-Type", "application/json")

	transport := http.Transport{
		Dial: k.dialTimeout,
	}
	client := &http.Client{
		Transport: &transport,
	}
	rsp, err := client.Do(req)
	if err != nil {
		k.log.Error("Failed to complete POST ", err)
		return false
	}

	defer rsp.Body.Close()
	if rsp.StatusCode == http.StatusNoContent {
		k.log.Info("Successfully sent ", len(series), " datapoints to Kairos")
		return true
	}

	body, _ := ioutil.ReadAll(rsp.Body)
	k.log.Error("Failed to post to Kairos @", apiURL,
		" status was ", rsp.Status,
		" rsp body was ", string(body),
		" payload was ", string(payload))
	return false
}

func (k *Kairos) dialTimeout(network, addr string) (net.Conn, error) {
	return net.DialTimeout(network, addr, k.timeout)
}
