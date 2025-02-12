package utils 

import (
	"net/http"
	"time"
	"encoding/json"
	"bytes"
	"fmt"
	"github.com/rahulmedicharla/kubefs/types"
	"io/ioutil"
	"os"
	"io"
)

var client *http.Client

func GetHttpClient(){
	client = &http.Client{Timeout: 10 * time.Second}
}
func PostRequest(url string, headers map[string]string, paylod map[string]interface{}) (*types.ApiResponse , error){
	postBody, err := json.Marshal(paylod)
	var apiResponse types.ApiResponse

	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(postBody))
	if err != nil {
		return nil, err
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
		return nil, err
    }

	err = json.Unmarshal(body, &apiResponse)
	if err != nil {
		return nil, err
	}

	return &apiResponse, nil
}

func DeleteRequest(url string, headers map[string]string) error{
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func DownloadZip(url string, name string) error {
	err := RunCommand(fmt.Sprintf("(cd %s && rm -rf deploy)", name), false, true)
	if err != nil {
		return err
	}

	resp, err := client.Get(url)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	out, err := os.Create(fmt.Sprintf("%s/helm.zip", name))
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	err = RunCommand(fmt.Sprintf("(cd %s && unzip helm.zip -d deploy && rm -rf helm.zip deploy/__MACOSX && echo '')", name), false, true)
	if err != nil {
		return err
	}

	return nil
}