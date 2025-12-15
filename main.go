package main

import (
	"encoding/json"
	"fmt"

	"gomod.pri/golib/apollo"
)

/*
	{
	    "a": 123,
	    "b": "ABC",
	    "c": {
	        "dd": "DE",
	        "ee": [
	            {
	                "f": "F"
	            },
	            {
	                "f": "F1"
	            }
	        ]
	    },
	    "g": [
	        {
	            "g1": "G1"
	        },
	        {
	            "g1": "G2"
	        }
	    ]
	}
*/
type Config struct {
	A int    `json:"a"`
	B string `json:"b"`
	C struct {
		DD string `json:"dd"`
		EE []struct {
			F string `json:"f"`
		} `json:"ee"`
	} `json:"c"`
	G []struct {
		G1 string `json:"g1"`
	} `json:"g"`
}

func main() {
	cli, err := apollo.NewClient(&apollo.Config{
		AppID:        "Debt",
		Cluster:      "default",
		Addr:         "http://config.apollo.host:8080",
		PrivateSpace: "test02.json",
	})
	if err != nil {
		panic(err)
	}
	var config Config
	jsonData := cli.GetPrivateJson()
	err = json.Unmarshal([]byte(jsonData), &config)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%v\n", config)
}
