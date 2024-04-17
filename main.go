package main

import (
	"fmt"
	"log"
	"os"

	jsoniter "github.com/json-iterator/go"
	"github.com/valyala/fasthttp"
)

type Mode uint8

const (
	Replace Mode = iota
	Add
)

const mode = Replace

var json = jsoniter.ConfigFastest

var client = fasthttp.Client{
	MaxConnsPerHost: 99999,
}

var documents = map[string]string{
	"game-assets": "324",
	"logo":        "325",
	"audio":       "373",
}

type ViewerData struct {
	Asset struct {
		Filename string `json:"filename"`
	} `json:"asset"`
}

type SearchResult struct {
	Token string `json:"token"`
}

type SearchData struct {
	Data  []SearchResult `json:"data"`
	Total int            `json:"total"`
}

func DownloadFile(token, document, base string) {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI("https://fankit.supercell.com/api/viewer/data/" + token + "?document_id=" + document)
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	err := client.Do(req, resp)
	if err != nil {
		log.Println("error getting file viewer: ", err)
		return
	}

	data, err := resp.BodyUncompressed()
	if err != nil {
		log.Println("error getting file viewer: ", err)
		return
	}

	var vd ViewerData
	err = json.Unmarshal(data, &vd)
	if err != nil {
		log.Println("error decoding file viewer: ", err, string(data))
		return
	}
	if mode == Add {
		if _, err = os.Stat(base + vd.Asset.Filename); err == nil {
			log.Println("skipping", vd.Asset.Filename)
			return
		}
	}

	req.SetRequestURI("https://brand.supercell.com/api/screen/download/" + token)

	err = client.Do(req, resp)
	if err != nil {
		log.Println("error getting file url: ", err)
		return
	}

	req.SetRequestURIBytes(resp.Header.Peek("location"))

	err = client.Do(req, resp)
	if err != nil {
		log.Println("error downloading file: ", err)
		return
	}

	data, err = resp.BodyUncompressed()
	if err != nil {
		log.Println("error downloading file: ", err)
		return
	}

	os.WriteFile(base+vd.Asset.Filename, data, 0777)
	log.Println("downloaded", vd.Asset.Filename)
}

func DownloadAll(folder, id string) {
	os.Mkdir(folder, 0777)

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI("https://fankit.supercell.com/api/assets/search/" + id + "?limit=1")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	err := client.Do(req, resp)
	if err != nil {
		log.Println("error getting file count for", folder, err)
		return
	}

	data, err := resp.BodyUncompressed()
	if err != nil {
		log.Println("error getting file count for", folder, err)
		return
	}

	var sd SearchData
	err = json.Unmarshal(data, &sd)
	if err != nil {
		log.Println("error decoding file count for", folder, err, string(data))
		return
	}

	req.SetRequestURI(fmt.Sprintf("https://fankit.supercell.com/api/assets/search/%s?limit=%d", id, sd.Total))

	err = client.Do(req, resp)
	if err != nil {
		log.Println("error getting files of", folder, err)
		return
	}

	data, err = resp.BodyUncompressed()
	if err != nil {
		log.Println("error getting files of", folder, err)
		return
	}

	err = json.Unmarshal(data, &sd)
	if err != nil {
		log.Println("error getting files of", folder, err, string(data))
		return
	}

	log.Printf("Downloading %d files from %s\n", len(sd.Data), folder)
	folder += "/"
	for _, file := range sd.Data {
		DownloadFile(file.Token, id, folder) // don't spawn goroutine for higher stability, otherwise a lot of connections were failing
	}
	log.Println(folder, "is done")
}

func main() {
	for folder, id := range documents {
		go DownloadAll(folder, id)
	}

	<-make(chan struct{})
}
