package client

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHTTClient(t *testing.T) {

	t.Run("client execute script", func(t *testing.T) {

		client := HTTPExecutorClient{URI: DefaultURI}
		e, err := client.Execute(exampleCollection)

		assert.NoError(t, err)
		assert.NotEqual(t, "", e)

		fmt.Printf("%+v\n", e)
		t.Fail()

	})

}

const exampleCollection = `{
	"info": {
		"_postman_id": "3d9a6be2-bd3e-4cf7-89ca-354103aab4a7",
		"name": "KubeTest",
		"schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
	},
	"item": [
		{
			"name": "Test",
			"event": [
				{
					"listen": "test",
					"script": {
						"exec": [
							"    pm.test(\"Successful GET request\", function () {",
							"        pm.expect(pm.response.code).to.be.oneOf([200, 201, 202]);",
							"    });"
						],
						"type": "text/javascript"
					}
				}
			],
			"request": {
				"method": "GET",
				"header": [],
				"url": {
					"raw": "http://127.0.0.1:8082",
					"protocol": "http",
					"host": [
						"127",
						"0",
						"0",
						"1"
					],
					"port": "%s"
	
				},
				"host": ["localhost"]
			},
			"response": []
		}
	]
}`
