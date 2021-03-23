package manago

import (
	"fmt"
	"encoding/json"
	"net/http"
	"net/url"
	"bytes"
	"time"
)

type Slack struct {
	HookUrl		string
	
}

type Message struct {
	Topic	string
	Body	string
}

func (ms *Message) GetSlackMessage() ([]byte, error) {
	
	type SlackMessage struct {
		Text		string	`json:"text"`
	}

	msg := SlackMessage{}
	msg.Text = ms.Topic + " " + ms.Body

	return json.Marshal(msg)
}

func (sl *Slack) Send(msg Message) error {
	reqUrl, err := url.Parse(sl.HookUrl)
	if err != nil {
		return fmt.Errorf("Parsing Api Url failed: %v", err)
	}

	jsonMsg, err := msg.GetSlackMessage()
	if err != nil {
		return fmt.Errorf("Parsing message to json failed: %v\n", err)
	}
	
	request, err := http.NewRequest("POST", reqUrl.String(), bytes.NewBuffer(jsonMsg))
	if err != nil {
		return fmt.Errorf("Preparing request failed: %v", err)
	}

	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: time.Second * 3,
	}

	resp, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("Failed doing request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		return fmt.Errorf("Received non success response: %s", resp.Status)
	}

	

	// decode := json.NewDecoder(resp.Body)
	// err = decode.Decode(em.LastState)
	// if err != nil {
	// 	return fmt.Errorf("Decoding json failed: %v", err)
	// } 

	return nil
}

func (sl *Slack) QuickSend(text string) error {
	message := Message{Body: text}
	return sl.Send(message)
}