import requests

proxies = {
  'http': 'http://127.0.0.1:8080',
}

try:
    response = requests.get('http://admin:admin@baidu.com/path1/path2.txt?a=1&b=2#anchor', proxies=proxies, timeout=3)
    print(response.status_code)
    print(response.content)
except Exception as e:
    print(e)
