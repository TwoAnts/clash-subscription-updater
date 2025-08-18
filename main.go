package main

import (
	"bytes"
	"clash-subscription-updater/overrider"
	"clash-subscription-updater/updater"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime/debug"

	"github.com/jasonlvhit/gocron"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func printVersion() {
	fmt.Println(version())
}

func loadConfig() error {
	defaultTarget := os.Getenv("HOME") + "/.config/clash/config.yaml"
	defaultControllerUrl := "http://127.0.0.1:9090"
	pflag.StringP("clash-config", "f", defaultTarget, "config file of clash")
	pflag.StringP("controller-url", "c", defaultControllerUrl, "controller url")
	pflag.StringP("controller-url-secret", "s", "", "controller secret")
	pflag.StringP("subscription-url", "t", "", "subscription url")
	pflag.IntP("interval", "i", 60, "interval to fetch configuration (minutes)")
	pflag.BoolP("help", "h", false, "show this message")
	pflag.Bool("override", false, "override the existed config file")
	pflag.BoolP("version", "v", false, "show current version")
	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)

	viper.SetConfigName("clash-subscription-updater")
	viper.AddConfigPath("$HOME/.config")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		p := os.Getenv("HOME") + "/.config"
		return fmt.Errorf("load clash-subscription-updater.yaml from . or %s failed: %w", p, err)
	}
	return nil
}

func main() {
	log.SetOutput(os.Stdout)
	if err := loadConfig(); err != nil {
		log.Printf("load config failed: %v", err)
		return
	}
	log.Printf("load config: %s", viper.ConfigFileUsed())
	if viper.GetBool("help") {
		pflag.PrintDefaults()
		return
	}
	if viper.GetBool("version") {
		fmt.Printf("version: ")
		printVersion()
		fmt.Println()
		return
	}
	url := viper.GetString("subscription-url")
	if url == "" {
		log.Fatal("subscription url is required")
		pflag.PrintDefaults()
		return
	}
	controllerUrl := viper.GetString("controller-url")

	configPath := viper.GetString("clash-config")
	u := updater.NewHttpUpdater(url, configPath, viper.GetInt("interval"))

	var proxies []overrider.Proxy
	var rules []overrider.Rule
	pxs := viper.Get("proxies")
	if pxs != nil {
		c := pxs.([]any)
		proxies = make([]overrider.Proxy, len(c))
		for i, p := range c {
			proxy := overrider.Proxy{}
			mapstructure.Decode(p, &proxy)
			proxies[i] = proxy
		}
		u.SetProxies(proxies)
	}

	rs := viper.GetStringSlice("rules")
	if len(rs) > 0 {
		rules = make([]overrider.Rule, len(rs))
		for i, r := range rs {
			rules[i] = overrider.Rule(r)
		}
		u.SetRules(rules)
	}

	var task = func() {
		changed, err := u.Update()
		if err != nil {
			log.Printf("error fetch config %s: %v", url, err)
		} else if changed {
			log.Printf("updated to %s | patch: proxies(+%d) rules(+%d)", configPath, len(proxies), len(rules))
			if len(controllerUrl) > 0 {
				err = notify(controllerUrl, viper.GetString("controller-url-secret"))
				if err != nil {
					log.Printf("notify config reload failed: %v ", err)
				}
			}
		}
	}
	task()
	s := gocron.NewScheduler()
	s.Every(uint64(viper.GetInt("interval"))).Minute().Do(task)
	<-s.Start()
}

func version() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	var rev string
	var date string
	modified := false
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			rev = setting.Value
		case "vcs.time":
			date = setting.Value
		case "vcs.modified":
			modified = true
		}
	}
	if modified {
		rev += "(CHANGED)"
	}
	return fmt.Sprintf("%s %s", rev, date)
}

func notify(controlUrl string, secret string) error {
	url := controlUrl + "/configs"
	body := []byte(`{"path":""}`)

	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("gen put request to %s failed: %w", url, err)
	}
	q := req.URL.Query()
	q.Add("force", "true")
	req.URL.RawQuery = q.Encode()

	req.Header.Set("Content-Type", "application/json")
	if len(secret) > 0 {
		req.Header.Set("Authorization", "Bearer "+secret)
	}

	c := &http.Client{}
	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("send Put req failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("put req failed:%d %s", resp.StatusCode, resp.Status)
	}

	return nil
}
