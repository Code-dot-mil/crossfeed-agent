package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	// "github.com/lib/pq"
)

type BDInventory struct {
	Total  int
	Assets []map[string](interface{})
}

func scanBitDiscovery(args []string) {
	log.SetPrefix("[bitdiscovery] ")
	fetchOpenPorts()
}

func fetchOpenPorts() {
	var (
		scanID  = getTimestamp(true)
		outPath = "bitdiscovery/" + scanID + ".txt"
	)

	offset := 0
	limit := 1000
	var allAssets []map[string](interface{})
	for {
		requestBody := "[{\"column\": \"bd.original_hostname\", \"type\": \"ends with\",\"value\": \".mil\"},{\"column\": \"ports.ports\",\"type\": \"has any value\"}]"
		var bdResponse BDInventory
		fetchExternalAPI(fmt.Sprintf("https://bitdiscovery.com/api/1.0/inventory?offset=%d&limit=%d&inventory=true", offset, limit),
			http.MethodPost,
			bytes.NewBuffer([]byte(requestBody)),
			map[string]string{
				"Authorization": config.BD_API_KEY,
				"Content-Type":  "application/json",
			},
			&bdResponse)

		allAssets = append(allAssets, bdResponse.Assets...)

		if len(bdResponse.Assets) < limit {
			break
		}
		offset += limit
	}

	var results []byte

	for _, asset := range allAssets {
		assetJSON := make(map[string](interface{}))
		if asset["bd.original_hostname"] != nil {
			assetJSON["domain"] = asset["bd.original_hostname"]
		} else {
			assetJSON["domain"] = asset["bd.hostname"]
		}
		assetJSON["ip"] = asset["bd.ip_address"]
		assetJSON["ports"] = asset["ports.ports"]
		assetJSON["services"] = asset["ports.services"]
		assetJSON["banners"] = asset["ports.banners"]
		if asset["screenshot.screenshot"] != "no" {
			assetJSON["screenshot"] = asset["screenshot.screenshot"]
		}
		bytes, err := json.Marshal(assetJSON)
		handleError(err)
		bytes = append(bytes, '\n')
		results = append(results, bytes...)
	}

	uploadToS3(outPath, results)
	notifyStorageService(outPath)
}
