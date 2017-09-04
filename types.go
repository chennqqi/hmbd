package main

type CR struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

type FileResp struct {
	MD5  string `json:"md5"`
	SHA1 string `json:"sha1"`
}
