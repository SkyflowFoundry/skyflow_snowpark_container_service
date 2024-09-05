package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	saUtil "github.com/skyflowapi/skyflow-go/serviceaccount/util"
	Skyflow "github.com/skyflowapi/skyflow-go/skyflow/client"
	"github.com/skyflowapi/skyflow-go/skyflow/common"
)

// TokenRequest unmarshals request body
type TokenRequest struct {
	Data [][]interface{} `json:"data"`
}

var bearerToken = ""

func getSkyflowBearerToken() (string, error) {
	filePath := "credentials.json"
	if saUtil.IsExpired(bearerToken) {
		newToken, err := saUtil.GenerateBearerToken(filePath)
		if err != nil {
			return "", err
		} else {
			bearerToken = newToken.AccessToken
			return bearerToken, nil
		}
	}
	log.Println("From global variable: " + bearerToken)
	return bearerToken, nil
}

func getTokenArray(tokenReq TokenRequest) ([]string, error) {
	numberOfColumns := len(tokenReq.Data[0])
	numberOfRows := len(tokenReq.Data)

	finalTokens := make([]string, 0, numberOfRows*(numberOfColumns-1))
	for rowIndex := 0; rowIndex < numberOfRows; rowIndex++ {
		for columnIndex := 1; columnIndex < numberOfColumns; columnIndex++ {
			tokenVal := tokenReq.Data[rowIndex][columnIndex].(string)
			finalTokens = append(finalTokens, tokenVal)
		}
	}
	return finalTokens, nil
}

func detokenize(c *gin.Context) {
	var tokenReq TokenRequest

	if err := c.ShouldBindJSON(&tokenReq); err != nil {
		log.Printf("error binding JSON, err: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	tokens, err := getTokenArray(tokenReq)
	if err != nil {
		log.Printf("error retrieving token array, err: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process tokens"})
		return
	}

	// Configuration setup
	configuration := common.Configuration{
		VaultID:       "<REPLACE_ME>",
		VaultURL:      "<REPLACE_ME>",
		TokenProvider: getSkyflowBearerToken,
	}

	// Initialize Skyflow client and perform detokenization
	skyflowClient := Skyflow.Init(configuration)

	responseArray := make([]interface{}, 0, len(tokens))

	for i := 0; i < len(tokens); i += 25 {
		end := i + 25
		if end > len(tokens) {
			end = len(tokens)
		}

		recordsArray := make([]interface{}, end-i)
		for j, token := range tokens[i:end] {
			recordsArray[j] = map[string]interface{}{"token": token}
		}

		records := map[string]interface{}{"records": recordsArray}
		options := common.DetokenizeOptions{ContinueOnError: false}
		log.Printf("Records: %+v", records)

		res, err2 := skyflowClient.Detokenize(records, options)
		log.Println(res)
		if err2 != nil {
			log.Printf("error during detokenization, err: %v", err2)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Detokenization failed"})
			return
		}

		for j, record := range res.Records {
			responseArray = append(responseArray, []interface{}{i + j, record.Value})
		}
	}

	log.Printf("Response data: %+v", responseArray)
	c.JSON(http.StatusOK, gin.H{"data": responseArray})
}

func main() {
	router := gin.Default()
	router.POST("/detokenize", detokenize)

	router.Run(":8080")
}
