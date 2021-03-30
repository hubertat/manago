package manago

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type Messenger interface {
	Send(Message) error
	QuickSend(string) error
}

type Slack struct {
	HookUrl string
}

type Message struct {
	Topic    string
	Body     string
	LinkUrl  *string
	LinkText *string
}

func (ms *Message) GetSlackMessage() ([]byte, error) {

	type SlackText struct {
		Type  string `json:"type,omitempty"`
		Text  string `json:"text"`
		Emoji bool   `json:"emoji,omitempty"`
	}
	type SlackAccessory struct {
		Type     string     `json:"type,omitempty"`
		Text     *SlackText `json:"text,omitempty"`
		Value    string     `json:"value,omitempty"`
		Url      string     `json:"url,omitempty"`
		ActionId string     `json:"action_id,omitempty"`
	}

	type SlackBlock struct {
		Type      string          `json:"type"`
		Text      *SlackText      `json:"text,omitempty"`
		Accessory *SlackAccessory `json:"accessory,omitempty"`
	}
	type SlackMessage struct {
		Blocks []SlackBlock `json:"blocks,omitempty"`
	}

	msg := SlackMessage{}
	msg.Blocks = []SlackBlock{
		SlackBlock{Type: "divider"},
	}

	if len(ms.Topic) > 0 {
		topicBlock := &SlackText{Type: "plain_text", Text: ms.Topic, Emoji: true}
		msg.Blocks = append(msg.Blocks, SlackBlock{Type: "header", Text: topicBlock})
	}

	if len(ms.Body) > 0 {
		bodyBlock := &SlackText{Type: "plain_text", Text: ms.Body, Emoji: true}
		msg.Blocks = append(msg.Blocks, SlackBlock{Type: "section", Text: bodyBlock})

	}

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
