package main

import (
	"encoding/json"
	"log"
)

func main() {
	data := []byte(`{"RequestContext":{"elb":{"targetGroupArn":""}},"httpMethod":"GET","path":"","isBase64Encoded":false,"body":""}`)
	m := make(map[string]interface{})
	err := json.Unmarshal(data, &m)
	if err != nil {
		log.Fatal(err)
	}
}
