package updater

import (
	"bytes"
	"clash-subscription-updater/overrider"
	"errors"
	"io"
	"net/http"
	"os"

	"gopkg.in/yaml.v2"
)

type HttpUpdater struct {
	url             string
	interval        int
	target          string
	overrideRules   []overrider.Rule
	overrideProxies []overrider.Proxy
	lastContent     []byte // 用于判断内容是否有更新
}

func NewHttpUpdater(url string, target string, interval int) HttpUpdater {
	return HttpUpdater{url: url, interval: interval, target: target}
}

func (u *HttpUpdater) SetRules(rules []overrider.Rule) {
	u.overrideRules = rules
}

func (u *HttpUpdater) SetProxies(proxies []overrider.Proxy) {
	u.overrideProxies = proxies
}

func (u *HttpUpdater) Update() (bool, error) {
	_, err := os.Stat(u.target)
	if os.IsNotExist(err) {
		// 避免主动创建配置文件
		return false, nil
	}

	resp, err := http.Get(u.url)
	if err != nil {
		return false, err
	}
	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	var c map[string]any
	yaml.Unmarshal(buf, &c)

	var patchedProxies []any
	for _, p := range u.overrideProxies {
		patchedProxies = append(patchedProxies, p)
	}
	if p, ok := c["proxies"]; ok && p != nil {
		proxies, ok := p.([]any)
		if !ok {
			return false, errors.New("parse proxies failed")
		}
		patchedProxies = append(patchedProxies, proxies...)
	}

	var patchedRules []any
	for _, r := range u.overrideRules {
		patchedRules = append(patchedRules, r)
	}
	if r, ok := c["rules"]; ok && r != nil {
		rules, ok := r.([]any)
		if !ok {
			return false, errors.New("parse rules failed")
		}
		patchedRules = append(patchedRules, rules...)
	}

	var proxyGroups []any
	if g, ok := c["proxy-groups"]; ok && g != nil {
		proxyGroups, ok = g.([]any)
		if !ok {
			return false, errors.New("parse proxy-groups failed")
		}
	}

	f, err := os.Open(u.target)
	if err != nil {
		return false, err
	}
	defer f.Close()
	c["proxies"] = patchedProxies
	if len(patchedProxies) == 0 {
		delete(c, "proxies")
	}
	c["proxy-groups"] = proxyGroups
	if len(proxyGroups) == 0 {
		delete(c, "proxy-groups")
	}
	c["rules"] = patchedRules
	if len(patchedRules) == 0 {
		delete(c, "rules")
	}
	out, err := yaml.Marshal(c)
	if err != nil {
		return false, err
	}
	os.WriteFile(u.target, out, 0644)
	// 判断内容是否变化
	if bytes.Equal(u.lastContent, buf) {
		return false, nil
	}
	u.lastContent = buf
	return true, nil
}
