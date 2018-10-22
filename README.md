# PrGoxy

```
HTTP(s)/1.0 Proxy in Golang
```

#### Usage
```
go run PrGoxy.go
```

#### Config File
```
{
    "proxy":{
        "lhost":"127.0.0.1",
        "lport":8080
    },
    "block":{
        "hosts":[
            "127.0.0.2"
        ],
        "sites":[
            "google.com",
            "youtube.com"
        ]
    },
    "redirect":{
        "acm.hit.edu.cn":"jwts.hit.edu.cn"
    },
    "cache":false
}
```

#### Reference
* https://www.ietf.org/rfc/rfc2068.txt
* https://www.ietf.org/rfc/rfc2817.txt

#### TODO
- [x] Block specific websites
- [x] Block specific users
- [x] Redirection
- [x] Supporting for cache
- [x] Use If-Modify-Since to ensure objects in cache is latest
- [ ] Random case to Bypass blocking
- [ ] Password sniffer
- [ ] Support for HTTP/1.1
