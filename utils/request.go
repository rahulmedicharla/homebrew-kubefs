package utils

import (
	"net/http"
	"time"
	"encoding/json"
	"bytes"
	"fmt"
	"github.com/rahulmedicharla/kubefs/types"
	"io/ioutil"
)

var client *http.Client

func GetHttpClient(){
	client = &http.Client{Timeout: 10 * time.Second}
}

func PostRequest(url string, headers map[string]string, paylod map[string]interface{}) (int, types.ApiResponse , error){
	postBody, err := json.Marshal(paylod)
	var apiResponse types.ApiResponse

	if err != nil {
		PrintError(fmt.Sprintf("Error marshalling payload: %v", err))
		return types.ERROR, apiResponse, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(postBody))
	if err != nil {
		PrintError(fmt.Sprintf("Error creating request: %v", err))
		return types.ERROR, apiResponse, err
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return types.ERROR, apiResponse	, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        PrintError(fmt.Sprintf("Error reading response: %v", err))
		return types.ERROR, apiResponse, err
    }

	err = json.Unmarshal(body, &apiResponse)
	if err != nil {
		PrintError(fmt.Sprintf("Error unmarshalling response: %v", err))
		return types.ERROR, apiResponse, err
	}

	return types.SUCCESS, apiResponse, nil
}

func DeleteRequest(url string, headers map[string]string) (int, error){
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		PrintError(fmt.Sprintf("Error creating request: %v", err))
		return types.ERROR, err
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		PrintError(fmt.Sprintf("Error deleting resource: %v", err))
		return types.ERROR, err
	}
	defer resp.Body.Close()

	return types.SUCCESS, nil
}