# clash subscription updater
> Update the clash `config.yaml` peroidly with optional patch

## Build
```shell
env GOOS=linux GOARCH=amd64 go build -o ./clash-subscription-updater
```

## Usage
```shell
-f, --clash-config string            config file of clash (default "/root/.config/clash/config.yaml")
-c, --controller-url string          controller url (default "http://127.0.0.1:9090")
-s, --controller-url-secret string   controller secret
-h, --help                           show this message
-i, --interval int                   interval to fetch configuration (minutes) (default 60)
    --override                       override the existed config file
-v, --version                        show current version

```

It will init a config file in `$HOME/.config/clash-subscription-updater.yaml`
you can add additional clash configs in the file to patch(prepend) to the subscription.

for example
```yaml
clash-config: /home/fengkx/.config/clash/config.yaml
controller-url: http://127.0.0.1:9090
controller-url-secret: "secret"
interval: 60
override: true
proxies:
- name: NeteaseMusic
  port: 9726
  server: 127.0.0.1
  type: http
rules:
- DOMAIN-SUFFIX,163.com,NeteaseMusic,
subscription-url: https://clash-rule-set-flatten.vercel.app/flat?url=xxxxxxxxx
```
`proxies` and `rules` will prepend to existed field

Only `proxies` and `rules` can be patched for now.

or
```shell
./clash-subscription-updater -f xxx/config.yaml -c controller-url -s secret -i 60 -t sub-url
# if needs proxy
HTTP_PROXY=proxy-url HTTPS_PROXY=proxy-url ./clash-subscription-updater -f xxx/config.yaml -c controller-url -s secret -i 60 -t sub-url
```
