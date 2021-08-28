package chatwootapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"path"

	log "github.com/sirupsen/logrus"
)

type ChatwootAPI struct {
	BaseURL     string
	AccountID   int
	InboxID     int
	AccessToken string

	Client *http.Client
}

func CreateChatwootAPI(baseURL string, accountID int, inboxID int, accessToken string) *ChatwootAPI {
	return &ChatwootAPI{
		BaseURL:     baseURL,
		AccountID:   accountID,
		InboxID:     inboxID,
		AccessToken: accessToken,
		Client:      &http.Client{},
	}
}

func (api *ChatwootAPI) DoRequest(req *http.Request) (*http.Response, error) {
	req.Header.Add("API_ACCESS_TOKEN", api.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	return api.Client.Do(req)
}

func (api *ChatwootAPI) MakeUri(endpoint string) string {
	url, err := url.Parse(api.BaseURL)
	if err != nil {
		panic(err)
	}
	url.Path = path.Join(url.Path, fmt.Sprintf("api/v1/accounts/%d", api.AccountID), endpoint)
	return url.String()
}

func (api *ChatwootAPI) ContactIDWithEmail(email string) (int, error) {
	req, err := http.NewRequest(http.MethodGet, api.MakeUri("contacts/search"), nil)
	if err != nil {
		log.Error(err)
		return 0, err
	}

	q := req.URL.Query()
	q.Add("q", email)
	q.Add("sort", "email")
	req.URL.RawQuery = q.Encode()

	resp, err := api.DoRequest(req)
	if err != nil {
		log.Error(err)
		return 0, err
	}
	if resp.StatusCode != 200 {
		return 0, errors.New(fmt.Sprintf("GET contacts/search returned non-200 status code: %d", resp.StatusCode))
	}

	decoder := json.NewDecoder(resp.Body)
	var contactsPayload ContactsPayload
	err = decoder.Decode(&contactsPayload)
	if err != nil {
		return 0, err
	}
	for _, contact := range contactsPayload.Payload {
		if contact.Email == email {
			return contact.ID, nil
		}
	}

	return 0, errors.New("Couldn't find user with email!")
}

func (api *ChatwootAPI) CreateConversation(sourceID string, contactID int) (*Conversation, error) {
	values := map[string]interface{}{
		"source_id":  sourceID,
		"inbox_id":   api.InboxID,
		"contact_id": contactID,
		"status":     "open",
	}
	jsonValue, _ := json.Marshal(values)
	req, err := http.NewRequest(http.MethodPost, api.MakeUri("conversations"), bytes.NewBuffer(jsonValue))
	if err != nil {
		log.Error(err)
		return nil, err
	}

	resp, err := api.DoRequest(req)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, errors.New(fmt.Sprintf("POST conversations returned non-200 status code: %d", resp.StatusCode))
	}

	decoder := json.NewDecoder(resp.Body)
	var conversation Conversation
	err = decoder.Decode(&conversation)
	if err != nil {
		return nil, err
	}
	return &conversation, nil
}

func (api *ChatwootAPI) SendTextMessage(conversationID int, content string) (*Message, error) {
	values := map[string]interface{}{"content": content, "message_type": "incoming", "private": false}
	jsonValue, _ := json.Marshal(values)
	req, err := http.NewRequest(http.MethodPost, api.MakeUri(fmt.Sprintf("conversations/%d/messages", conversationID)), bytes.NewBuffer(jsonValue))
	if err != nil {
		log.Error(err)
		return nil, err
	}

	resp, err := api.DoRequest(req)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, errors.New(fmt.Sprintf("POST conversations returned non-200 status code: %d", resp.StatusCode))
	}

	decoder := json.NewDecoder(resp.Body)
	var message Message
	err = decoder.Decode(&message)
	if err != nil {
		return nil, err
	}
	return &message, nil
}

func (api *ChatwootAPI) SendAttachmentMessage(conversationID int, filename string, fileData io.Reader) (*Message, error) {
	bodyBuf := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuf)

	contentFieldWriter, err := bodyWriter.CreateFormField("content")
	if err != nil {
		log.Error(err)
		return nil, err
	}
	contentFieldWriter.Write([]byte{})

	privateFieldWriter, err := bodyWriter.CreateFormField("private")
	if err != nil {
		log.Error(err)
		return nil, err
	}
	privateFieldWriter.Write([]byte("false"))

	messageTypeFieldWriter, err := bodyWriter.CreateFormField("message_type")
	if err != nil {
		log.Error(err)
		return nil, err
	}
	messageTypeFieldWriter.Write([]byte("incoming"))

	fileWriter, err := bodyWriter.CreateFormFile("attachments", filename)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	// Copy the file data into the form.
	io.Copy(fileWriter, fileData)

	contentType := bodyWriter.FormDataContentType()
	bodyWriter.Close()

	req, err := http.NewRequest(http.MethodPost, api.MakeUri(fmt.Sprintf("conversations/%d/messages", conversationID)), bodyBuf)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	req.Header.Add("API_ACCESS_TOKEN", api.AccessToken)
	req.Header.Set("Content-Type", contentType)

	resp, err := api.Client.Do(req)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	data, _ := ioutil.ReadAll(resp.Body)
	log.Debug(string(data))
	if resp.StatusCode != 200 {
		return nil, errors.New(fmt.Sprintf("POST conversations returned non-200 status code: %d", resp.StatusCode))
	}

	decoder := json.NewDecoder(resp.Body)
	var message Message
	err = decoder.Decode(&message)
	if err != nil {
		return nil, err
	}
	return &message, nil
}

func (api *ChatwootAPI) DownloadAttachment(url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	resp, err := api.DoRequest(req)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, errors.New(fmt.Sprintf("GET attachment returned non-200 status code: %d", resp.StatusCode))
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return data, err
}

func (api *ChatwootAPI) DeleteMessage(conversationID int, messageID int) error {
	req, err := http.NewRequest(http.MethodDelete, api.MakeUri(fmt.Sprintf("conversations/%d/messages/%d", conversationID, messageID)), nil)
	if err != nil {
		log.Error(err)
		return err
	}

	resp, err := api.DoRequest(req)
	if err != nil {
		log.Error(err)
		return err
	}
	if resp.StatusCode != 200 {
		return errors.New(fmt.Sprintf("GET attachment returned non-200 status code: %d", resp.StatusCode))
	}

	return nil
}