package manago

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type Client struct {
	Url    string
	ApiKey string
}

func (cl *Client) Call(path string, params url.Values, result interface{}) error {
	params.Add("api_key", cl.ApiKey)

	netClient := &http.Client{
		Timeout: time.Second * 3,
	}

	absUrl, err := url.Parse(cl.Url)
	if err != nil {
		return err
	}

	relUrl, err := absUrl.Parse(path)
	if err != nil {
		return err
	}

	resp, err := netClient.PostForm(relUrl.String(), params)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 203 {
		return fmt.Errorf("Client Call failed, non success status received: \n%v", resp.Status)
	}

	if resp.Header.Get("Content-Type") != "application/json" {
		return fmt.Errorf("Client Call failed, recieved wrong content type: %s, application/json expected.", resp.Header.Get("Content-Type"))
	}

	decode := json.NewDecoder(resp.Body)
	err = decode.Decode(result)
	if err != nil {
		return err
	}

	return nil
}
