hmbd
=============

This repository contains a **Dockerfile** of [hmbd](https://github.com/chennqqi/hmbd/) for [Docker](https://www.docker.io/)'s [trusted build](https://index.docker.io/u/sort/hmbd/) published to the public [DockerHub](https://index.docker.io/).

### Dependencies

-	[malice/alpine](https://hub.docker.com/r/malice/alpine/)

### Installation

1.	Install [Docker](https://www.docker.io/).
2.	Download [hmbd](https://github.com/chennqqi/hmbd/) for [Docker](https://www.docker.io/)'s [trusted build](https://index.docker.io/u/sort/hmbd/) published to the public [DockerHub](https://index.docker.io/).

### Usage

copy license
	
	mkdir -p /opt/hmb/license
	cp hmb.lic /opt/hmb/license

build

	git clone https://github.com/chennqqi/hmbd.git
	cd hmbd
	docker build -t xxx/hmbd .

run as webservice

	docker run -e HM_LICENSE_PATH=/opt/hmb/license/hmb.lic -v /opt/hmb/license:/opt/hmb/license -d -p 8080:8080 xxx/hmbd web

	curl -F 'filename=@testshell.php' localhost:8080/file?timeout=10s?callback=http://api.xxx.com/result

	curl -F 'zipname=@testshell.zip' localhost:8080/zip?timeout=60s?callback=http://api.xxx.com/result


`timeout` set scan max timeout

`callback` set result call back
	if you want set callback once, but keep for all, set to docker run
	add to docker run env ` -e HMBD_CALLBACK=http://api.xxx.com/result`
	priority:
		httprequest param>run param>docker -e option
	

version

	docker run xxx/hmbd version

update

	docker run xxx/hmbd update


## Sample Output

### JSON:
scan as a zip

```json
	{
		  "suspious_list": [],
		  "black_list": [
		    {
		      "judger": "FEATURE",
		      "advice": "DEL",
		      "type": "一句话后门",
		      "name": "/dev/shm/scan_089524826/file277982036/scan_019678371",
		      "md5": "8d6428492359c27b163648a5888da9da"
		    },
		    {
		      "judger": "FEATURE",
		      "advice": "DEL",
		      "type": "一句话后门",
		      "name": "/dev/shm/scan_089524826/shell.php",
		      "md5": "8d6428492359c27b163648a5888da9da"
		    },
		    {
		      "judger": "FEATURE",
		      "advice": "DEL",
		      "type": "一句话后门",
		      "name": "/dev/shm/scan_089524826/shell1.php",
		      "md5": "8d6428492359c27b163648a5888da9da"
		    }
		  ],
		  "app_version": "1.0.3 hmb#linux-amd64.c339720",
		  "rule_version": "6",
		  "cost": 0,
		  "end_time": "2017-09-22T10:30:31.036382428+08:00",
		  "start_time": "2017-09-22T10:30:30.868960028+08:00",
		  "b_count": 3,
		  "w_count": 0,
		  "s_count": 0,
		  "cloud_valid": true,
		  "jw_count": 0,
		  "jb_count": 0,
		  "m_count": 0,
		  "f_total": 3
		}
```

scan as a file


```json
{
  "suspious_list": [],
  "black_list": [
    {
      "judger": "FEATURE",
      "advice": "DEL",
      "type": "一句话后门",
      "name": "/scan_881052458",
      "md5": "8d6428492359c27b163648a5888da9da"
    }
  ],
  "app_version": "1.0.3 hmb#linux-amd64.c339720",
  "rule_version": "6",
  "cost": 0,
  "end_time": "2017-09-22T10:27:17.932094512+08:00",
  "start_time": "2017-09-22T10:27:17.764498329+08:00",
  "b_count": 1,
  "w_count": 0,
  "s_count": 0,
  "cloud_valid": true,
  "jw_count": 0,
  "jb_count": 0,
  "m_count": 0,
  "f_total": 1
}
```

Documentation
-------------

-	[To write results to ElasticSearch](https://github.com/malice-plugins/clamav/blob/master/docs/elasticsearch.md)
-	[To create a ClamAV scan micro-service](https://github.com/malice-plugins/clamav/blob/master/docs/web.md)
-	[To post results to a webhook](https://github.com/malice-plugins/clamav/blob/master/docs/callback.md)
-	[To update the AV definitions](https://github.com/malice-plugins/clamav/blob/master/docs/update.md)

### Issues

Find a bug? Want more features? Find something missing in the documentation? Let me know! Please don't hesitate to [file an issue](https://github.com/chennqqi/hmbd/issues/new).

### CHANGELOG

See [`CHANGELOG.md`](https://github.com/chennqqi/hmbd/blob/master/CHANGELOG.md)

### License

MIT 
