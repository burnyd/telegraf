package suzieq

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/plugins/inputs"
)

type SuzieQ struct {
	Url       string `toml:"url"`
	Token     string `toml:"token"`
	EnableTLS bool   `toml:"enable_tls"`
	Port      int    `toml:"port"`
	Log       telegraf.Logger
	Interval  *time.Ticker `toml:"interval"`
	done      chan bool
}

type NetworkRoutes []struct {
	Action     string   `json:"action"`
	Hostname   string   `json:"hostname"`
	Ipvers     int64    `json:"ipvers"`
	Namespace  string   `json:"namespace"`
	NexthopIps []string `json:"nexthopIps"`
	Oifs       []string `json:"oifs"`
	Preference int64    `json:"preference"`
	Prefix     string   `json:"prefix"`
	Protocol   string   `json:"protocol"`
	Source     string   `json:"source"`
	Timestamp  int64    `json:"timestamp"`
	Vrf        string   `json:"vrf"`
}

func (c *SuzieQ) Description() string {
	return "a SuzieQ Plugin"
}

func (c *SuzieQ) SampleConfig() string {
	return `
  ## Indicate if everything is fine
  Add values later here.
 `
}

// Init is for setup, and validating config.
func (c *SuzieQ) Init() error {
	return nil
}

func (c *SuzieQ) Start(acc telegraf.Accumulator) error {
	c.Interval = time.NewTicker(time.Second * 10) // Example: adjust interval every 10 seconds
	return nil
}

func (c *SuzieQ) Stop() {
	c.Interval.Stop()
}

func (c *SuzieQ) Gather(acc telegraf.Accumulator) error {
	req, err := http.NewRequest("GET", "https://"+c.Url+":"+strconv.Itoa(c.Port)+"/api/v2/route/show?access_token="+c.Token, nil)
	if err != nil {
		fmt.Println(err)
	}
	req.Header.Add("Accept", "application/json")
	//To do tls certs in case of if else

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{Transport: tr}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
	}
	defer resp.Body.Close()

	responseData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Cannot marshall data", err)
	}

	JsonResponse := NetworkRoutes{}
	//Unmarshall it
	err = json.Unmarshal(responseData, &JsonResponse)
	// Show how type values work.
	var fields = make(map[string]interface{}, len(NetworkRoutes{}))
	var tags = map[string]string{}
	for _, v := range JsonResponse {
		tags["Device"] = v.Hostname
		tags["VRF"] = v.Vrf
		tags["RoutingProtocol"] = v.Protocol
		tags["Timestamp"] = strconv.Itoa(int(v.Timestamp))
		for _, nh := range v.NexthopIps {
			tags["nexthops"] = nh
		}
		for _, oi := range v.Oifs {
			tags["OutgoingInterfaces"] = oi
		}
		fields["Prefix"] = v.Prefix
		acc.AddFields("NetworkRoutes", fields, tags)
	}
	return nil
}

func init() {
	inputs.Add("suzieq", func() telegraf.Input {
		return &SuzieQ{}
	})
}
